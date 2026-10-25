package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"snitch/internal/jobs"
)

// registerVoiceRoutes declares the phase 5 voice routes. Later tickets add their
// own register method in their own file and one call in Handler.
func (s *Server) registerVoiceRoutes(mux *http.ServeMux) {
	mux.Handle("GET /api/admin/voice-pipeline", s.protected(true, http.HandlerFunc(s.getVoicePipeline)))
	mux.Handle("PUT /api/admin/voice-pipeline", s.protected(true, http.HandlerFunc(s.putVoicePipeline)))
	mux.Handle("POST /api/admin/clips/{id}/voice", s.protected(true, http.HandlerFunc(s.scheduleVoice)))
	mux.Handle("GET /api/clips/{id}/voice", s.protected(false, http.HandlerFunc(s.getClipVoice)))
}

type voiceFake struct {
	Speakers   int `json:"speakers"`
	Dimensions int `json:"dimensions"`
}

type voicePipeline struct {
	Provider              string     `json:"provider"`
	Model                 string     `json:"model"`
	ModelVersion          string     `json:"model_version"`
	AccessConfirmed       bool       `json:"access_confirmed"`
	MinFingerprintSeconds float64    `json:"min_fingerprint_seconds"`
	MinQuality            float64    `json:"min_quality"`
	Fake                  *voiceFake `json:"fake,omitempty"`
}

func (s *Server) voiceConfig(ctx context.Context, version string) (voicePipeline, error) {
	var raw []byte
	var config voicePipeline
	if err := s.Pool.QueryRow(ctx, `SELECT config FROM voice_pipeline_versions WHERE version=$1`, version).Scan(&raw); err != nil {
		return config, fmt.Errorf("read voice pipeline: %w", err)
	}
	if err := json.Unmarshal(raw, &config); err != nil {
		return config, fmt.Errorf("decode voice pipeline: %w", err)
	}
	return config, nil
}

func (s *Server) activeVoicePipeline(ctx context.Context) (string, voicePipeline, error) {
	var version string
	if err := s.Pool.QueryRow(ctx, `SELECT version FROM voice_pipeline_active WHERE id=true`).Scan(&version); err != nil {
		return "", voicePipeline{}, err
	}
	config, err := s.voiceConfig(ctx, version)
	return version, config, err
}

func (s *Server) getVoicePipeline(w http.ResponseWriter, r *http.Request) {
	version, config, err := s.activeVoicePipeline(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{"version": version, "config": config})
}

func (s *Server) putVoicePipeline(w http.ResponseWriter, r *http.Request) {
	var p voicePipeline
	if !decode(w, r, &p) {
		return
	}
	p.Model, p.ModelVersion = strings.TrimSpace(p.Model), strings.TrimSpace(p.ModelVersion)
	valid := (p.Provider == "fake" || p.Provider == "pyannote") && p.Model != "" && len(p.Model) <= 100 && p.ModelVersion != "" && len(p.ModelVersion) <= 64 &&
		p.MinFingerprintSeconds >= 0.1 && p.MinFingerprintSeconds <= 60 && p.MinQuality >= 0 && p.MinQuality <= 1
	if valid && p.Provider == "fake" {
		if p.Fake == nil {
			p.Fake = &voiceFake{Speakers: 2, Dimensions: 8}
		}
		valid = p.Fake.Speakers >= 1 && p.Fake.Speakers <= 8 && p.Fake.Dimensions >= 2 && p.Fake.Dimensions <= 512
	} else if valid {
		valid = p.Fake == nil
	}
	if !valid {
		fail(w, http.StatusBadRequest, "invalid_voice_pipeline")
		return
	}
	raw, _ := json.Marshal(p)
	hash := sha256.Sum256(raw)
	version := "voice-" + hex.EncodeToString(hash[:10])
	tx, err := s.Pool.Begin(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	if _, err = tx.Exec(r.Context(), `INSERT INTO voice_pipeline_versions(version,config) VALUES($1,$2) ON CONFLICT(version) DO NOTHING`, version, raw); err != nil {
		s.internal(w, err)
		return
	}
	if _, err = tx.Exec(r.Context(), `UPDATE voice_pipeline_active SET version=$1 WHERE id=true`, version); err != nil {
		s.internal(w, err)
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{"version": version, "config": p})
}

// scheduleVoice is the only way voice processing starts: an explicit admin action
// for one clip whose transcription already exists. It never touches transcription.
func (s *Server) scheduleVoice(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		fail(w, http.StatusBadRequest, "invalid_clip_id")
		return
	}
	var input struct {
		AudioStreamIndex *int `json:"audio_stream_index"`
	}
	if !decode(w, r, &input) {
		return
	}
	tracks, err := s.clipAudioTracks(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, http.StatusNotFound, "clip_not_found")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	index := input.AudioStreamIndex
	if index != nil && !hasAudioTrack(tracks, *index) {
		fail(w, http.StatusBadRequest, "invalid_audio_stream")
		return
	}
	if index == nil {
		selection, selErr := s.audioSelection(r.Context(), id, tracks)
		if selErr != nil {
			s.internal(w, selErr)
			return
		}
		if selection.AudioStreamIndex == nil {
			fail(w, http.StatusConflict, "audio_selection_required")
			return
		}
		index = selection.AudioStreamIndex
	}
	var transcribed bool
	if err = s.Pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM jobs WHERE clip_id=$1 AND step=$2 AND status='done')`, id, fmt.Sprintf("transcription.audio.%d", *index)).Scan(&transcribed); err != nil {
		s.internal(w, err)
		return
	}
	if !transcribed {
		fail(w, http.StatusConflict, "transcription_required")
		return
	}
	version, config, err := s.activeVoicePipeline(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	if config.Provider != "fake" && !config.AccessConfirmed {
		fail(w, http.StatusConflict, "voice_model_access_unconfirmed")
		return
	}
	job, created, err := s.Jobs.Schedule(r.Context(), jobs.ScheduleInput{ClipID: id, Step: fmt.Sprintf("voice.diarization.%d", *index), InputVersion: fmt.Sprintf("d5-a%d-p%s", *index, version), MaxAttempts: 3})
	switch {
	case errors.Is(err, jobs.ErrClipNotFound):
		fail(w, http.StatusNotFound, "clip_not_found")
	case err != nil:
		s.internal(w, err)
	default:
		status := http.StatusOK
		if created {
			status = http.StatusCreated
		}
		reply(w, status, map[string]any{"job": job})
	}
}

type voiceStep struct {
	JobID        int64  `json:"job_id"`
	Step         string `json:"step"`
	Status       string `json:"status"`
	InputVersion string `json:"input_version"`
	Attempts     int    `json:"attempts"`
	MaxAttempts  int    `json:"max_attempts"`
	LastError    string `json:"last_error"`
}

type voiceFingerprintView struct {
	ID             int64   `json:"id"`
	Scope          string  `json:"scope"`
	StartSeconds   float64 `json:"start_seconds"`
	EndSeconds     float64 `json:"end_seconds"`
	DurationSecond float64 `json:"duration_seconds"`
	Quality        float64 `json:"quality"`
	Usable         bool    `json:"usable"`
	UnusableReason string  `json:"unusable_reason"`
	Model          string  `json:"model"`
	ModelVersion   string  `json:"model_version"`
	Dimensions     int     `json:"dimensions"`
}

type localSpeakerView struct {
	ID              int64                  `json:"id"`
	Label           string                 `json:"label"`
	Ordinal         int                    `json:"ordinal"`
	SpeakingSeconds float64                `json:"speaking_seconds"`
	TurnCount       int                    `json:"turn_count"`
	State           string                 `json:"state"`
	Fingerprints    []voiceFingerprintView `json:"fingerprints"`
}

func voiceStatus(jobStatus string) string {
	switch jobStatus {
	case "pending", "retry_wait":
		return "queued"
	case "leased":
		return "running"
	case "done":
		return "done"
	}
	return "failed"
}

func (s *Server) latestVoiceStep(ctx context.Context, clipID int64, pattern string) (*voiceStep, error) {
	var step voiceStep
	err := s.Pool.QueryRow(ctx, `SELECT id,step,status,input_version,attempts,max_attempts,last_error FROM jobs WHERE clip_id=$1 AND step LIKE $2 ORDER BY id DESC LIMIT 1`, clipID, pattern).Scan(&step.JobID, &step.Step, &step.Status, &step.InputVersion, &step.Attempts, &step.MaxAttempts, &step.LastError)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &step, err
}

// getClipVoice returns the current local speakers of a clip with fingerprint
// metadata (never the vectors) and the redacted state of the voice steps.
func (s *Server) getClipVoice(w http.ResponseWriter, r *http.Request) {
	clip, ok := s.authorizedClip(w, r)
	if !ok {
		return
	}
	view := map[string]any{"clip_id": clip.ID, "status": "none", "audio_stream_index": nil, "steps": []voiceStep{}, "speakers": []localSpeakerView{}}
	diarization, err := s.latestVoiceStep(r.Context(), clip.ID, "voice.diarization.%")
	if err != nil {
		s.internal(w, err)
		return
	}
	if diarization == nil {
		reply(w, http.StatusOK, view)
		return
	}
	index, _ := strconv.Atoi(strings.TrimPrefix(diarization.Step, "voice.diarization."))
	steps := []voiceStep{*diarization}
	status := voiceStatus(diarization.Status)
	if diarization.Status == "done" {
		fingerprint, fpErr := s.latestVoiceStep(r.Context(), clip.ID, fmt.Sprintf("voice.fingerprint.%d", index))
		if fpErr != nil {
			s.internal(w, fpErr)
			return
		}
		if fingerprint != nil {
			steps = append(steps, *fingerprint)
			status = voiceStatus(fingerprint.Status)
		}
	}
	speakers := []localSpeakerView{}
	byID := map[int64]int{}
	rows, err := s.Pool.Query(r.Context(), `SELECT id,label,ordinal,speaking_seconds,turn_count FROM local_speakers WHERE clip_id=$1 AND audio_stream_index=$2 ORDER BY ordinal`, clip.ID, index)
	if err != nil {
		s.internal(w, err)
		return
	}
	for rows.Next() {
		sp := localSpeakerView{State: "unidentified", Fingerprints: []voiceFingerprintView{}}
		if err = rows.Scan(&sp.ID, &sp.Label, &sp.Ordinal, &sp.SpeakingSeconds, &sp.TurnCount); err != nil {
			rows.Close()
			s.internal(w, err)
			return
		}
		byID[sp.ID] = len(speakers)
		speakers = append(speakers, sp)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		s.internal(w, err)
		return
	}
	fpRows, err := s.Pool.Query(r.Context(), `SELECT local_speaker_id,id,scope,start_seconds,end_seconds,duration_seconds,quality,usable,unusable_reason,model,model_version,dimensions FROM voice_fingerprints WHERE clip_id=$1 ORDER BY id`, clip.ID)
	if err != nil {
		s.internal(w, err)
		return
	}
	for fpRows.Next() {
		var speakerID int64
		var fp voiceFingerprintView
		if err = fpRows.Scan(&speakerID, &fp.ID, &fp.Scope, &fp.StartSeconds, &fp.EndSeconds, &fp.DurationSecond, &fp.Quality, &fp.Usable, &fp.UnusableReason, &fp.Model, &fp.ModelVersion, &fp.Dimensions); err != nil {
			fpRows.Close()
			s.internal(w, err)
			return
		}
		if at, found := byID[speakerID]; found {
			speakers[at].Fingerprints = append(speakers[at].Fingerprints, fp)
		}
	}
	fpRows.Close()
	if err = fpRows.Err(); err != nil {
		s.internal(w, err)
		return
	}
	view["status"], view["audio_stream_index"], view["steps"], view["speakers"] = status, index, steps, speakers
	reply(w, http.StatusOK, view)
}

// attachLocalSpeakers labels each transcript segment with the local speaker whose
// turns overlap it most. Segments without overlap stay unlabeled.
func (s *Server) attachLocalSpeakers(ctx context.Context, clipID int64, transcript any) error {
	object, ok := transcript.(map[string]any)
	if !ok {
		return nil
	}
	index, indexOK := object["audio_stream_index"].(float64)
	segments, isList := object["segments"].([]any)
	if !indexOK || !isList || len(segments) == 0 {
		return nil
	}
	rows, err := s.Pool.Query(ctx, `SELECT ls.id,ls.label,st.start_seconds,st.end_seconds FROM speaker_turns st JOIN local_speakers ls ON ls.id=st.local_speaker_id WHERE ls.clip_id=$1 AND ls.audio_stream_index=$2 ORDER BY ls.ordinal,st.turn_index`, clipID, int(index))
	if err != nil {
		return err
	}
	defer rows.Close()
	type turn struct {
		id         int64
		label      string
		start, end float64
	}
	turns := []turn{}
	for rows.Next() {
		var t turn
		if err = rows.Scan(&t.id, &t.label, &t.start, &t.end); err != nil {
			return err
		}
		turns = append(turns, t)
	}
	if err = rows.Err(); err != nil || len(turns) == 0 {
		return err
	}
	for _, raw := range segments {
		segment, isMap := raw.(map[string]any)
		if !isMap {
			continue
		}
		start, startOK := segment["start"].(float64)
		end, endOK := segment["end"].(float64)
		if !startOK || !endOK {
			continue
		}
		overlap := map[int64]float64{}
		labels := map[int64]string{}
		order := []int64{}
		for _, t := range turns {
			if shared := math.Min(end, t.end) - math.Max(start, t.start); shared > 0 {
				if _, seen := overlap[t.id]; !seen {
					order = append(order, t.id)
				}
				overlap[t.id] += shared
				labels[t.id] = t.label
			}
		}
		if len(order) == 0 {
			continue
		}
		sort.SliceStable(order, func(i, j int) bool { return overlap[order[i]] > overlap[order[j]] })
		segment["local_speaker_id"], segment["local_speaker_label"] = order[0], labels[order[0]]
	}
	return nil
}
