package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"snitch/internal/jobs"
)

type audioTrackRule struct {
	Title                    *string `json:"title,omitempty"`
	Language                 *string `json:"language,omitempty"`
	ChannelLayout            *string `json:"channel_layout,omitempty"`
	Channels                 *int    `json:"channels,omitempty"`
	FallbackAudioStreamIndex *int    `json:"fallback_audio_stream_index,omitempty"`
}
type audioTrack struct {
	Codec           string   `json:"codec"`
	DurationSeconds *float64 `json:"duration_seconds"`
	StartSeconds    *float64 `json:"start_seconds"`
	Index           int      `json:"index"`
	Type            string   `json:"type"`
	Title           string   `json:"title"`
	Language        string   `json:"language"`
	Channels        int      `json:"channels"`
	ChannelLayout   string   `json:"channel_layout"`
}
type audioSelection struct {
	Status           string `json:"status"`
	AudioStreamIndex *int   `json:"audio_stream_index,omitempty"`
	Source           string `json:"selection_source,omitempty"`
	RuleVersion      int64  `json:"rule_version,omitempty"`
}

func (s *Server) putSourceAudioRule(w http.ResponseWriter, r *http.Request) {
	id, ok := sourceID(w, r)
	if !ok {
		return
	}
	var rule audioTrackRule
	if !decode(w, r, &rule) {
		return
	}
	if rule.Title == nil && rule.Language == nil && rule.ChannelLayout == nil && rule.Channels == nil && rule.FallbackAudioStreamIndex == nil {
		fail(w, 400, "empty_audio_track_rule")
		return
	}
	for _, value := range []*string{rule.Title, rule.Language, rule.ChannelLayout} {
		if value != nil {
			normalized := strings.TrimSpace(*value)
			if normalized == "" {
				fail(w, 400, "invalid_audio_track_rule")
				return
			}
			*value = normalized
		}
	}
	if rule.Channels != nil && *rule.Channels < 1 {
		fail(w, 400, "invalid_audio_track_rule")
		return
	}
	if rule.FallbackAudioStreamIndex != nil && *rule.FallbackAudioStreamIndex < 0 {
		fail(w, 400, "invalid_audio_track_rule")
		return
	}
	var exists bool
	if err := s.Pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM clip_sources WHERE id=$1)`, id).Scan(&exists); err != nil {
		s.internal(w, err)
		return
	}
	if !exists {
		fail(w, 404, "source_not_found")
		return
	}
	raw, _ := json.Marshal(rule)
	var version int64
	err := s.Pool.QueryRow(r.Context(), `INSERT INTO source_audio_track_rules(source_id,rule) VALUES($1,$2) ON CONFLICT(source_id) DO UPDATE SET rule=EXCLUDED.rule,version=source_audio_track_rules.version+1,updated_at=now() RETURNING version`, id, raw).Scan(&version)
	if err != nil {
		s.internal(w, err)
		return
	}
	skipped := s.reprocessSource(r.Context(), id)
	reply(w, 200, map[string]any{"rule": rule, "version": version, "corrected_clips_skipped": skipped})
}

// reprocessSource retranscribes the ready clips of a source after a rule change. A clip
// that carries human corrections is skipped, never retranscribed silently: the administrator
// retranscribes it explicitly (confirmation). It returns the number of clips skipped.
func (s *Server) reprocessSource(ctx context.Context, sourceID int64) int {
	rows, err := s.Pool.Query(ctx, `SELECT DISTINCT sf.clip_id FROM source_files sf WHERE sf.source_id=$1 AND sf.clip_id IS NOT NULL AND sf.status='ready'`, sourceID)
	if err != nil {
		return 0
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()
	corrected, err := s.correctedClipSegments(ctx)
	if err != nil {
		return 0
	}
	skipped := 0
	for _, id := range ids {
		if corrected[id] > 0 {
			skipped++
			continue
		}
		_, _, _ = s.scheduleResolvedTranscription(ctx, id, nil, false)
	}
	return skipped
}
func (s *Server) putClipAudioTrack(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		fail(w, 400, "invalid_clip_id")
		return
	}
	var input struct {
		AudioStreamIndex int `json:"audio_stream_index"`
	}
	if !decode(w, r, &input) {
		return
	}
	tracks, err := s.clipAudioTracks(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "clip_not_found")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	if !hasAudioTrack(tracks, input.AudioStreamIndex) {
		fail(w, 400, "audio_stream_not_found")
		return
	}
	if !s.correctionsGuard(w, r, id) {
		return
	}
	_, err = s.Pool.Exec(r.Context(), `INSERT INTO clip_audio_track_overrides(clip_id,audio_stream_index) VALUES($1,$2) ON CONFLICT(clip_id) DO UPDATE SET audio_stream_index=EXCLUDED.audio_stream_index,version=clip_audio_track_overrides.version+1,updated_at=now()`, id, input.AudioStreamIndex)
	if err != nil {
		s.internal(w, err)
		return
	}
	_, _, _ = s.scheduleResolvedTranscription(r.Context(), id, &input.AudioStreamIndex, false)
	s.reconcileAnalysis(r.Context(), id)
	reply(w, 200, map[string]any{"audio_stream_index": input.AudioStreamIndex})
}
func (s *Server) deleteClipAudioTrack(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		fail(w, 400, "invalid_clip_id")
		return
	}
	if !s.correctionsGuard(w, r, id) {
		return
	}
	if _, err = s.Pool.Exec(r.Context(), `DELETE FROM clip_audio_track_overrides WHERE clip_id=$1`, id); err != nil {
		s.internal(w, err)
		return
	}
	_, _, _ = s.scheduleResolvedTranscription(r.Context(), id, nil, false)
	s.reconcileAnalysis(r.Context(), id)
	reply(w, 200, map[string]string{"status": "removed"})
}
func (s *Server) clipAudioTracks(ctx context.Context, id int64) ([]audioTrack, error) {
	var raw []byte
	err := s.Pool.QueryRow(ctx, `SELECT tracks FROM clips WHERE id=$1`, id).Scan(&raw)
	if err != nil {
		return nil, err
	}
	var tracks []audioTrack
	err = json.Unmarshal(raw, &tracks)
	return tracks, err
}
func (s *Server) audioSelection(ctx context.Context, id int64, tracks []audioTrack) (audioSelection, error) {
	var index int
	var overrideVersion int64
	err := s.Pool.QueryRow(ctx, `SELECT audio_stream_index,version FROM clip_audio_track_overrides WHERE clip_id=$1`, id).Scan(&index, &overrideVersion)
	if err == nil {
		if hasAudioTrack(tracks, index) {
			return audioSelection{Status: "selected", AudioStreamIndex: &index, Source: "clip_override", RuleVersion: overrideVersion}, nil
		}
		return audioSelection{Status: "selection_required"}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return audioSelection{}, err
	}
	rows, err := s.Pool.Query(ctx, `SELECT sar.rule,sar.version FROM source_audio_track_rules sar JOIN source_files sf ON sf.source_id=sar.source_id WHERE sf.clip_id=$1 AND sf.status='ready' ORDER BY sar.source_id`, id)
	if err != nil {
		return audioSelection{}, err
	}
	defer rows.Close()
	selected := -1
	version := int64(0)
	sourceCount := 0
	for rows.Next() {
		var raw []byte
		var v int64
		if err = rows.Scan(&raw, &v); err != nil {
			return audioSelection{}, err
		}
		var rule audioTrackRule
		if json.Unmarshal(raw, &rule) != nil {
			continue
		}
		n, ok := resolveAudioTrack(tracks, rule)
		if !ok {
			return audioSelection{Status: "selection_required"}, nil
		}
		if selected >= 0 && selected != n {
			return audioSelection{Status: "selection_required"}, nil
		}
		selected = n
		version += v
		sourceCount++
	}
	if err = rows.Err(); err != nil {
		return audioSelection{}, err
	}
	if selected >= 0 {
		return audioSelection{Status: "selected", AudioStreamIndex: &selected, Source: "source_rule", RuleVersion: version}, nil
	}
	n, ok := resolveAudioTrack(tracks, audioTrackRule{})
	if !ok {
		return audioSelection{Status: "selection_required"}, nil
	}
	return audioSelection{Status: "selected", AudioStreamIndex: &n, Source: "single_audio_track"}, nil
}
func (s *Server) scheduleResolvedTranscription(ctx context.Context, id int64, explicit *int, requireReady bool) (jobs.Job, bool, error) {
	plan, err := s.planTranscription(ctx, id, explicit, requireReady)
	if err != nil || !plan.ready {
		return jobs.Job{}, false, err
	}
	return s.Jobs.Schedule(ctx, jobs.ScheduleInput{ClipID: id, Step: plan.step, InputVersion: plan.version, MaxAttempts: 3})
}

// transcriptionPlan is the transcription job a clip needs now: its step and input version.
// ready is false when the clip media is not ready (nothing to plan yet).
type transcriptionPlan struct {
	step, version string
	ready         bool
}

func (s *Server) planTranscription(ctx context.Context, id int64, explicit *int, requireReady bool) (transcriptionPlan, error) {
	tracks, err := s.clipAudioTracks(ctx, id)
	if err != nil {
		return transcriptionPlan{}, err
	}
	selection, err := s.audioSelection(ctx, id, tracks)
	if err != nil {
		return transcriptionPlan{}, err
	}
	index := selection.AudioStreamIndex
	if explicit != nil {
		if !hasAudioTrack(tracks, *explicit) {
			return transcriptionPlan{}, errors.New("audio stream not found")
		}
		index = explicit
	}
	if index == nil {
		return transcriptionPlan{}, errors.New("audio selection required")
	}
	var ready bool
	if err = s.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM uploaded_media WHERE clip_id=$1) OR EXISTS(SELECT 1 FROM source_files WHERE clip_id=$1 AND status='ready')`, id).Scan(&ready); err != nil {
		return transcriptionPlan{}, err
	}
	if requireReady && !ready {
		return transcriptionPlan{}, errors.New("clip not ready")
	}
	if !ready {
		return transcriptionPlan{}, nil
	}
	vocabulary, err := s.activeVocabulary(ctx)
	if err != nil {
		return transcriptionPlan{}, err
	}
	vocabVersion := vocabularyVersion(vocabulary)
	vocabularyJSON, err := json.Marshal(vocabulary)
	if err != nil {
		return transcriptionPlan{}, err
	}
	if _, err = s.Pool.Exec(ctx, `INSERT INTO transcription_vocabulary_versions(version,terms) VALUES($1,$2) ON CONFLICT(version) DO NOTHING`, vocabVersion, vocabularyJSON); err != nil {
		return transcriptionPlan{}, err
	}
	pipelineVersion, err := s.activeTranscriptionPipeline(ctx)
	if err != nil {
		return transcriptionPlan{}, err
	}
	return transcriptionPlan{step: fmt.Sprintf("transcription.audio.%d", *index), version: fmt.Sprintf("t4-a%d-r%d-v%s-p%s", *index, selection.RuleVersion, vocabVersion, pipelineVersion), ready: true}, nil
}

func (s *Server) activeTranscriptionPipeline(ctx context.Context) (string, error) {
	var version string
	if err := s.Pool.QueryRow(ctx, `SELECT version FROM transcription_pipeline_active WHERE id=true`).Scan(&version); err != nil {
		return "", err
	}
	return version, nil
}
func hasAudioTrack(tracks []audioTrack, index int) bool {
	for _, t := range tracks {
		if t.Index == index && t.Type == "audio" {
			return true
		}
	}
	return false
}
func resolveAudioTrack(tracks []audioTrack, rule audioTrackRule) (int, bool) {
	matches := []int{}
	for _, t := range tracks {
		if t.Type != "audio" {
			continue
		}
		if rule.Title != nil && !strings.EqualFold(t.Title, *rule.Title) {
			continue
		}
		if rule.Language != nil && !strings.EqualFold(t.Language, *rule.Language) {
			continue
		}
		if rule.ChannelLayout != nil && !strings.EqualFold(t.ChannelLayout, *rule.ChannelLayout) {
			continue
		}
		if rule.Channels != nil && t.Channels != *rule.Channels {
			continue
		}
		matches = append(matches, t.Index)
	}
	if len(matches) == 1 {
		return matches[0], true
	}
	if len(matches) == 0 && rule.FallbackAudioStreamIndex != nil && hasAudioTrack(tracks, *rule.FallbackAudioStreamIndex) {
		return *rule.FallbackAudioStreamIndex, true
	}
	if len(matches) == 0 {
		aud := []int{}
		for _, t := range tracks {
			if t.Type == "audio" {
				aud = append(aud, t.Index)
			}
		}
		if len(aud) == 1 {
			return aud[0], true
		}
	}
	return 0, false
}

func tracksFromClip(value any) []audioTrack {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	var tracks []audioTrack
	_ = json.Unmarshal(raw, &tracks)
	return tracks
}
