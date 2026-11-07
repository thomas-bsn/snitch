package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"snitch/internal/jobs"
)

// registerVoiceRoutes declares the phase 5 voice routes. Later phase 5 steps add their
// own register method in their own file and one call in Handler.
func (s *Server) registerVoiceRoutes(mux *http.ServeMux) {
	mux.Handle("GET /api/admin/voice-pipeline", s.protected(true, http.HandlerFunc(s.getVoicePipeline)))
	mux.Handle("PUT /api/admin/voice-pipeline", s.protected(true, http.HandlerFunc(s.putVoicePipeline)))
	mux.Handle("GET /api/admin/voice-overview", s.protected(true, http.HandlerFunc(s.getVoiceOverview)))
	s.registerVoiceModelRoutes(mux)
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
	var entry *voiceCatalogEntry
	valid := (p.Provider == "fake" || voiceRealProvider(p.Provider)) && p.Model != "" && len(p.Model) <= 100 && p.ModelVersion != "" && len(p.ModelVersion) <= 64 &&
		p.MinFingerprintSeconds >= 0.1 && p.MinFingerprintSeconds <= 60 && p.MinQuality >= 0 && p.MinQuality <= 1
	if valid && p.Provider == "fake" {
		if p.Fake == nil {
			p.Fake = &voiceFake{Speakers: 2, Dimensions: 8}
		}
		valid = p.Fake.Speakers >= 1 && p.Fake.Speakers <= 8 && p.Fake.Dimensions >= 2 && p.Fake.Dimensions <= 512
	} else if valid {
		// A real model is chosen from the catalogue, never typed: its pinned revision is the version.
		found, known, lookupErr := voiceCatalogByRepo(r.Context(), s.Pool, "diarization", p.Model)
		if lookupErr != nil {
			s.internal(w, lookupErr)
			return
		}
		valid = p.Fake == nil && known && found.choosable() && p.ModelVersion == found.Revision && p.Provider == found.Engine
		entry = &found
	}
	if !valid {
		fail(w, http.StatusBadRequest, "invalid_voice_pipeline")
		return
	}
	// A catalogue model follows the same rule as "activate": a connected Worker must hold it ready.
	version, config, err := s.switchVoicePipeline(r.Context(), entry, func(voicePipeline) voicePipeline { return p })
	if errors.Is(err, errVoiceModelNotReady) {
		fail(w, http.StatusConflict, "voice_model_not_ready")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{"version": version, "config": config})
}

// saveVoicePipelineIn stores an immutable snapshot and makes it the active version for new voice steps.
func saveVoicePipelineIn(ctx context.Context, db pgQuerier, p voicePipeline) (string, error) {
	raw, _ := json.Marshal(p)
	hash := sha256.Sum256(raw)
	version := "voice-" + hex.EncodeToString(hash[:10])
	if _, err := db.Exec(ctx, `INSERT INTO voice_pipeline_versions(version,config) VALUES($1,$2) ON CONFLICT(version) DO NOTHING`, version, raw); err != nil {
		return "", err
	}
	if _, err := db.Exec(ctx, `UPDATE voice_pipeline_active SET version=$1 WHERE id=true`, version); err != nil {
		return "", err
	}
	return version, nil
}

// scheduleVoice is the only way voice processing starts: an explicit admin action
// for one clip whose transcription already exists. It never touches transcription.
func (s *Server) scheduleVoice(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "invalid_clip_id")
	if !ok {
		return
	}
	var input struct {
		AudioStreamIndex          *int `json:"audio_stream_index"`
		ConfirmReplaceValidations bool `json:"confirm_replace_validations"`
	}
	if !decode(w, r, &input) {
		return
	}
	if s.refuseWhileAnalyzing(w, r, id) {
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
	// A designated microphone track is attributed directly and never diarized; the
	// designation is (re)applied first so a clip ingested after the rule is covered.
	if _, err = s.applyMicrophone(r.Context(), id, s.authorOf(r)); err != nil {
		s.internal(w, err)
		return
	}
	var microphone bool
	if err = s.Pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM local_speakers WHERE clip_id=$1 AND audio_stream_index=$2 AND kind='microphone')`, id, *index).Scan(&microphone); err != nil {
		s.internal(w, err)
		return
	}
	if microphone {
		fail(w, http.StatusConflict, "microphone_track_not_diarized")
		return
	}
	// A relaunch replaces the speakers of the stream: the explicit confirmation of the
	// administrator is required when it would make human work obsolete.
	if !input.ConfirmReplaceValidations {
		stake, found, stakeErr := s.voiceRediarizationStake(r.Context(), id, *index)
		if stakeErr != nil {
			s.internal(w, stakeErr)
			return
		}
		if found {
			reply(w, http.StatusConflict, map[string]any{"error": "voice_rediarization_confirmation_required", "stake": stake})
			return
		}
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
	version, _, err := s.activeVoicePipeline(r.Context())
	if err != nil {
		s.internal(w, err)
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

// voiceRediarizationStake is what a relaunched diarization would make obsolete for a
// clip: the speakers a human validated and their Voice samples (kept, linked to the
// audio, but no longer attached to the visible speakers), and the speaker corrections
// of the transcript (obsolete).
type voiceRediarizationStake struct {
	ValidatedSpeakers  int `json:"validated_speakers"`
	VoiceSamples       int `json:"voice_samples"`
	SpeakerCorrections int `json:"speaker_corrections"`
}

func (s *Server) voiceRediarizationStake(ctx context.Context, clipID int64, index int) (voiceRediarizationStake, bool, error) {
	var stake voiceRediarizationStake
	err := s.Pool.QueryRow(ctx, `SELECT count(DISTINCT a.local_speaker_id),count(DISTINCT vs.id)
FROM speaker_attributions a JOIN local_speakers ls ON ls.id=a.local_speaker_id
LEFT JOIN voice_samples vs ON vs.local_speaker_id=ls.id AND vs.retired_at IS NULL
WHERE ls.clip_id=$1 AND ls.audio_stream_index=$2 AND ls.kind='diarized' AND a.validation_state='validated'`, clipID, index).Scan(&stake.ValidatedSpeakers, &stake.VoiceSamples)
	if err != nil {
		return stake, false, err
	}
	corrections, _, err := s.clipCorrections(ctx, s.Pool, clipID)
	if err != nil {
		return stake, false, err
	}
	stake.SpeakerCorrections = corrections.SpeakerSegments
	return stake, stake.ValidatedSpeakers+stake.VoiceSamples+stake.SpeakerCorrections > 0, nil
}

// voiceHistoryEntry is a replaced diarization result of an audio stream, with its provenance.
type voiceHistoryEntry struct {
	JobID            int64     `json:"job_id"`
	Model            string    `json:"model"`
	ModelVersion     string    `json:"model_version"`
	CompletedAt      time.Time `json:"completed_at"`
	SpeakerCount     int       `json:"speaker_count"`
	ValidatedSpeaker int       `json:"validated_speakers"`
}

// voiceHistory lists the diarization results of the stream that the visible one replaced,
// most recent first. The visible result is the latest done diarization (the rule of
// projectFingerprints); a relaunch in progress does not change it.
func (s *Server) voiceHistory(ctx context.Context, clipID int64, index int) ([]voiceHistoryEntry, error) {
	rows, err := s.Pool.Query(ctx, `SELECT j.id,COALESCE(j.result->>'model',''),COALESCE(j.result->>'model_version',''),j.updated_at,
 CASE WHEN jsonb_typeof(j.result->'speakers')='array' THEN jsonb_array_length(j.result->'speakers') ELSE 0 END,
 (SELECT count(*) FROM local_speakers ls WHERE ls.diarization_job_id=j.id AND ls.kind='archived')
FROM jobs j WHERE j.clip_id=$1 AND j.step=$2 AND j.status='done'
 AND j.id<>(SELECT id FROM jobs WHERE clip_id=$1 AND step=$2 AND status='done' ORDER BY updated_at DESC,id DESC LIMIT 1)
ORDER BY j.updated_at DESC,j.id DESC`, clipID, fmt.Sprintf("voice.diarization.%d", index))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []voiceHistoryEntry{}
	for rows.Next() {
		var e voiceHistoryEntry
		if err = rows.Scan(&e.JobID, &e.Model, &e.ModelVersion, &e.CompletedAt, &e.SpeakerCount, &e.ValidatedSpeaker); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
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

// DecisionLevel is the decision level of an attribution. The values are part of the
// API contract and of the database (decision_level columns).
type DecisionLevel string

const (
	LevelHuman      DecisionLevel = "human"
	LevelAutomatic  DecisionLevel = "automatic"
	LevelSuggestion DecisionLevel = "suggestion"
	LevelAmbiguous  DecisionLevel = "ambiguous"
	LevelUnknown    DecisionLevel = "unknown"
)

// pathID reads the positive integer {id} of the route, answering 400 with code when
// it is malformed.
func pathID(w http.ResponseWriter, r *http.Request, code string) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		fail(w, http.StatusBadRequest, code)
		return 0, false
	}
	return id, true
}

type localSpeakerView struct {
	ID    int64  `json:"id"`
	Label string `json:"label"`
	// AudioStreamIndex is the real stream the voice was found on: its provenance (#342).
	AudioStreamIndex int                    `json:"audio_stream_index"`
	Kind             string                 `json:"kind"`
	Ordinal          int                    `json:"ordinal"`
	SpeakingSeconds  float64                `json:"speaking_seconds"`
	TurnCount        int                    `json:"turn_count"`
	State            string                 `json:"state"`
	DecisionLevel    DecisionLevel          `json:"decision_level"`
	Person           *personRef             `json:"person"`
	Attribution      *attributionView       `json:"attribution"`
	Fingerprints     []voiceFingerprintView `json:"fingerprints"`
	// Disposition is "ignored" or "random" when the voice is set aside (ADR 0031), else absent.
	Disposition *string `json:"disposition"`
	// RefusedPersons are the Persons refused for the voice (ADR 0031), each one cancellable.
	RefusedPersons []personRef `json:"refused_persons"`
}

// applyAttribution fills the identity fields of a local speaker from its effective
// attribution. A suggestion or an ambiguous voice stays "unidentified".
func (sp *localSpeakerView) applyAttribution(a *attributionView) {
	sp.State, sp.DecisionLevel, sp.Person, sp.Attribution = "unidentified", LevelUnknown, nil, nil
	if a == nil {
		return
	}
	sp.DecisionLevel, sp.Person, sp.Attribution = a.DecisionLevel, a.Person, a
	if a.DecisionLevel == LevelHuman || a.DecisionLevel == LevelAutomatic {
		sp.State = "identified"
	}
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
	err := s.Pool.QueryRow(ctx, `SELECT id,step,status,input_version,attempts,max_attempts,last_error FROM jobs WHERE clip_id=$1 AND step LIKE $2 AND requested_model_id IS NULL ORDER BY id DESC LIMIT 1`, clipID, pattern).Scan(&step.JobID, &step.Step, &step.Status, &step.InputVersion, &step.Attempts, &step.MaxAttempts, &step.LastError)
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
	view := map[string]any{"clip_id": clip.ID, "status": "none", "audio_stream_index": nil, "steps": []voiceStep{}, "speakers": []localSpeakerView{}, "history": []voiceHistoryEntry{}}
	// A clip whose visible transcript is a plan revision shows the voices of every kept track (#342).
	if planID, found, planErr := s.visiblePlanOf(r.Context(), clip.ID); planErr != nil {
		s.internal(w, planErr)
		return
	} else if found {
		planView, viewErr := s.planClipVoice(r.Context(), clip.ID, planID)
		if viewErr != nil {
			s.internal(w, viewErr)
			return
		}
		reply(w, http.StatusOK, planView)
		return
	}
	diarization, err := s.latestVoiceStep(r.Context(), clip.ID, "voice.diarization.%")
	if err != nil {
		s.internal(w, err)
		return
	}
	if diarization == nil {
		// A microphone track is attributed without any diarization step.
		attributions, attrErr := s.effectiveAttributionPointers(r.Context(), clip.ID)
		if attrErr != nil {
			s.internal(w, attrErr)
			return
		}
		microphone, micErr := s.microphoneSpeakerViews(r.Context(), clip.ID, attributions)
		if micErr != nil {
			s.internal(w, micErr)
			return
		}
		if err = s.attachRefusedPersons(r.Context(), clip.ID, microphone); err != nil {
			s.internal(w, err)
			return
		}
		view["speakers"] = microphone
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
	speakers, err := s.voiceSpeakers(r.Context(), clip.ID, []int{index})
	if err != nil {
		s.internal(w, err)
		return
	}
	history, err := s.voiceHistory(r.Context(), clip.ID, index)
	if err != nil {
		s.internal(w, err)
		return
	}
	view["status"], view["audio_stream_index"], view["steps"], view["speakers"], view["history"] = status, index, steps, speakers, history
	reply(w, http.StatusOK, view)
}

// voiceSpeakers lists the voices of the given streams, then the microphone voice first, with their
// fingerprint metadata (never the vectors) and refused Persons.
func (s *Server) voiceSpeakers(ctx context.Context, clipID int64, indexes []int) ([]localSpeakerView, error) {
	attributions, err := s.effectiveAttributionPointers(ctx, clipID)
	if err != nil {
		return nil, err
	}
	speakers := []localSpeakerView{}
	rows, err := s.Pool.Query(ctx, `SELECT ls.id,ls.label,ls.ordinal,ls.speaking_seconds,ls.turn_count,ls.kind,d.disposition,ls.audio_stream_index FROM local_speakers ls LEFT JOIN local_speaker_dispositions d ON d.local_speaker_id=ls.id WHERE ls.clip_id=$1 AND ls.audio_stream_index=ANY($2) AND ls.kind<>'archived' ORDER BY array_position($2::int[],ls.audio_stream_index),ls.ordinal`, clipID, indexes)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		sp := localSpeakerView{Fingerprints: []voiceFingerprintView{}}
		if err = rows.Scan(&sp.ID, &sp.Label, &sp.Ordinal, &sp.SpeakingSeconds, &sp.TurnCount, &sp.Kind, &sp.Disposition, &sp.AudioStreamIndex); err != nil {
			rows.Close()
			return nil, err
		}
		sp.applyAttribution(attributions[sp.ID])
		speakers = append(speakers, sp)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, err
	}
	// The microphone speaker lives on its own stream: list it first, once.
	microphone, err := s.microphoneSpeakerViews(ctx, clipID, attributions)
	if err != nil {
		return nil, err
	}
	listed := map[int64]bool{}
	for _, sp := range speakers {
		listed[sp.ID] = true
	}
	for i := len(microphone) - 1; i >= 0; i-- {
		if !listed[microphone[i].ID] {
			speakers = append([]localSpeakerView{microphone[i]}, speakers...)
		}
	}
	byID := map[int64]int{}
	for i, sp := range speakers {
		byID[sp.ID] = i
	}
	fpRows, err := s.Pool.Query(ctx, `SELECT local_speaker_id,id,scope,start_seconds,end_seconds,duration_seconds,quality,usable,unusable_reason,model,model_version,dimensions FROM voice_fingerprints WHERE clip_id=$1 AND NOT benchmark ORDER BY id`, clipID)
	if err != nil {
		return nil, err
	}
	for fpRows.Next() {
		var speakerID int64
		var fp voiceFingerprintView
		if err = fpRows.Scan(&speakerID, &fp.ID, &fp.Scope, &fp.StartSeconds, &fp.EndSeconds, &fp.DurationSecond, &fp.Quality, &fp.Usable, &fp.UnusableReason, &fp.Model, &fp.ModelVersion, &fp.Dimensions); err != nil {
			fpRows.Close()
			return nil, err
		}
		if at, found := byID[speakerID]; found {
			speakers[at].Fingerprints = append(speakers[at].Fingerprints, fp)
		}
	}
	fpRows.Close()
	if err = fpRows.Err(); err != nil {
		return nil, err
	}
	if err = s.attachRefusedPersons(ctx, clipID, speakers); err != nil {
		return nil, err
	}
	return speakers, nil
}

// attachRefusedPersons lists on each voice the Persons refused for it (ADR 0031).
func (s *Server) attachRefusedPersons(ctx context.Context, clipID int64, speakers []localSpeakerView) error {
	byID := map[int64]int{}
	for i := range speakers {
		speakers[i].RefusedPersons = []personRef{}
		byID[speakers[i].ID] = i
	}
	rows, err := s.Pool.Query(ctx, `SELECT vr.local_speaker_id,p.id,p.display_name FROM voice_refusals vr JOIN local_speakers ls ON ls.id=vr.local_speaker_id JOIN persons p ON p.id=vr.person_id WHERE ls.clip_id=$1 ORDER BY p.display_name,p.id`, clipID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var speakerID int64
		var p personRef
		if err = rows.Scan(&speakerID, &p.ID, &p.DisplayName); err != nil {
			return err
		}
		if at, found := byID[speakerID]; found {
			speakers[at].RefusedPersons = append(speakers[at].RefusedPersons, p)
		}
	}
	return rows.Err()
}

// attachLocalSpeakers labels each transcript segment with its local speaker: the one a
// user designated for the segment (phase 6; states carries the overrides), otherwise
// the one whose turns overlap it most. Segments without either stay unlabeled. It also
// lists the local speakers of the audio stream (transcription.local_speakers) so a
// reader can offer them as the choice of a correction.
func (s *Server) attachLocalSpeakers(ctx context.Context, clipID int64, transcript any, states map[int]segmentState) error {
	return s.attachTrackSpeakers(ctx, clipID, transcript, states, false)
}

// attachTrackSpeakers is attachLocalSpeakers; keepEmpty lists and covers the voices of a track
// that has no segment at all (a plan's kept track whose voices carry no word, #342).
func (s *Server) attachTrackSpeakers(ctx context.Context, clipID int64, transcript any, states map[int]segmentState, keepEmpty bool) error {
	object, ok := transcript.(map[string]any)
	if !ok {
		return nil
	}
	if revision, ok := object["revision"].(map[string]any); ok {
		if id, _ := revision["id"].(string); strings.HasPrefix(id, "plan:") {
			return s.attachRevisionSpeakers(ctx, clipID, object, states)
		}
	}
	index, indexOK := object["audio_stream_index"].(float64)
	segments, isList := object["segments"].([]any)
	if !indexOK || !isList || (len(segments) == 0 && !keepEmpty) {
		return nil
	}
	roster, err := s.Pool.Query(ctx, `SELECT ls.id,ls.label,COALESCE(d.disposition,'') FROM local_speakers ls LEFT JOIN local_speaker_dispositions d ON d.local_speaker_id=ls.id WHERE ls.clip_id=$1 AND ls.audio_stream_index=$2 AND ls.kind<>'archived' ORDER BY ls.ordinal,ls.id`, clipID, int(index))
	if err != nil {
		return err
	}
	// An ignored voice is no speaker at all (ADR 0031): it is neither listed nor labelled.
	// A random voice is labelled "Random" wherever it is shown.
	dispositions := map[int64]string{}
	speakerLabels := map[int64]string{}
	speakerList := []map[string]any{}
	for roster.Next() {
		var id int64
		var label, disposition string
		if err = roster.Scan(&id, &label, &disposition); err != nil {
			roster.Close()
			return err
		}
		dispositions[id] = disposition
		if disposition == dispositionIgnored {
			continue
		}
		if disposition == dispositionRandom {
			label = randomSpeakerLabel
		}
		speakerLabels[id] = label
		speakerList = append(speakerList, map[string]any{"id": id, "label": label})
	}
	roster.Close()
	if err = roster.Err(); err != nil || len(speakerList) == 0 {
		return err
	}
	object["local_speakers"] = speakerList
	rows, err := s.Pool.Query(ctx, `SELECT ls.id,ls.label,st.start_seconds,st.end_seconds FROM speaker_turns st JOIN local_speakers ls ON ls.id=st.local_speaker_id WHERE ls.clip_id=$1 AND ls.audio_stream_index=$2 AND `+notIgnoredVoice+` ORDER BY ls.ordinal,st.turn_index`, clipID, int(index))
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
	if err = rows.Err(); err != nil {
		return err
	}
	attributions, err := s.effectiveAttributions(ctx, clipID)
	if err != nil {
		return err
	}
	coverageTurns := make([]coverageTurn, len(turns))
	turnLabels := map[int64]string{}
	for i, t := range turns {
		coverageTurns[i] = coverageTurn{SpeakerID: t.id, Start: t.start, End: t.end}
		turnLabels[t.id] = t.label
	}
	coverageSegments := []coverageSegment{}
	for position, raw := range segments {
		if segment, isMap := raw.(map[string]any); isMap {
			start, startOK := segment["start"].(float64)
			end, endOK := segment["end"].(float64)
			if startOK && endOK {
				cs := coverageSegment{Start: start, End: end}
				if st, found := states[position]; found && st.StaleAt == nil && st.SpeakerOverrideID != nil {
					cs.OverrideSpeakerID = st.SpeakerOverrideID
				}
				coverageSegments = append(coverageSegments, cs)
			}
		}
	}
	object["speaker_coverage"] = speakerCoverage(coverageTurns, coverageSegments)
	for position, raw := range segments {
		segment, isMap := raw.(map[string]any)
		if !isMap {
			continue
		}
		start, startOK := segment["start"].(float64)
		end, endOK := segment["end"].(float64)
		if !startOK || !endOK {
			continue
		}
		order, overlap := segmentOverlaps(coverageTurns, start, end)
		var speakerID int64
		var speakerLabel string
		if winner, found := dominantSpeaker(order, overlap); found {
			speakerID, speakerLabel = winner, turnLabels[winner]
		}
		segment["local_speaker_corrected"] = false
		if st, found := states[position]; found && st.StaleAt == nil && st.SpeakerOverrideID != nil {
			if label, known := speakerLabels[*st.SpeakerOverrideID]; known {
				speakerID, speakerLabel = *st.SpeakerOverrideID, label
				segment["local_speaker_corrected"] = true
			}
		}
		if speakerID == 0 || dispositions[speakerID] == dispositionIgnored {
			continue
		}
		if dispositions[speakerID] == dispositionRandom {
			speakerLabel = randomSpeakerLabel
		}
		segment["local_speaker_id"], segment["local_speaker_label"] = speakerID, speakerLabel
		segment["local_speaker_decision_level"] = LevelUnknown
		if a, found := attributions[speakerID]; found {
			segment["local_speaker_decision_level"] = a.DecisionLevel
			if a.Person != nil {
				segment["local_speaker_person_id"], segment["local_speaker_person_name"] = a.Person.ID, a.Person.DisplayName
			}
		}
	}
	return nil
}
