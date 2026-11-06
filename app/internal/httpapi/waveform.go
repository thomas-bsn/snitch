package httpapi

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"snitch/internal/jobs"
)

// Peaks come with a transcription or a diarization result (ADR 0034). They are
// optional: a result without them, or with malformed ones, is accepted as before.
const (
	maxTranscriptionResultBytes = 4 << 20
	maxWaveformPeaks            = 4000
	maxWaveformLevel            = 255
)

// waveformJobPrefixes are the steps whose audio preparation produces peaks for one track.
var waveformJobPrefixes = []string{"transcription.audio.", "voice.diarization."}

func waveformStep(step string) bool {
	for _, prefix := range waveformJobPrefixes {
		if strings.HasPrefix(step, prefix) {
			return true
		}
	}
	return false
}

// validWaveformPeaks reads the peaks of a result. ok is false when the result has none
// or they are malformed; the caller then stores nothing.
func validWaveformPeaks(raw json.RawMessage) (peaks []int, ok bool) {
	var probe struct {
		Peaks json.RawMessage `json:"peaks"`
	}
	if json.Unmarshal(raw, &probe) != nil || len(probe.Peaks) == 0 {
		return nil, false
	}
	if json.Unmarshal(probe.Peaks, &peaks) != nil || len(peaks) == 0 || len(peaks) > maxWaveformPeaks {
		return nil, false
	}
	for _, level := range peaks {
		if level < 0 || level > maxWaveformLevel {
			return nil, false
		}
	}
	return peaks, true
}

// withoutPeaks keeps the peaks out of jobs.result: they are stored per track instead.
func withoutPeaks(raw json.RawMessage) json.RawMessage {
	var object map[string]json.RawMessage
	if json.Unmarshal(raw, &object) != nil || object == nil {
		return raw
	}
	if _, has := object["peaks"]; !has {
		return raw
	}
	delete(object, "peaks")
	stripped, err := json.Marshal(object)
	if err != nil {
		return raw
	}
	return stripped
}

// waveformProjection stores the peaks of a completed job for its audio track, inside the
// completion transaction. It returns nil when there is nothing to store.
func waveformProjection(step string, raw json.RawMessage) func(ctx context.Context, tx pgx.Tx, job jobs.Job) error {
	if !waveformStep(step) {
		return nil
	}
	peaks, ok := validWaveformPeaks(raw)
	if !ok {
		return nil
	}
	index, err := strconv.Atoi(step[strings.LastIndex(step, ".")+1:])
	if err != nil || index < 0 {
		return nil
	}
	encoded, err := json.Marshal(peaks)
	if err != nil {
		return nil
	}
	return func(ctx context.Context, tx pgx.Tx, job jobs.Job) error {
		// Job ids grow with creation: an older job completing late never replaces newer peaks.
		_, err := tx.Exec(ctx, `INSERT INTO clip_waveforms(clip_id,audio_stream_index,job_id,peaks) VALUES($1,$2,$3,$4::jsonb) ON CONFLICT(clip_id,audio_stream_index) DO UPDATE SET job_id=EXCLUDED.job_id,peaks=EXCLUDED.peaks,created_at=now() WHERE clip_waveforms.job_id<EXCLUDED.job_id`, job.ClipID, index, job.ID, string(encoded))
		return err
	}
}

// chainProjections runs two projections of the same completion in order.
func chainProjections(first, second func(ctx context.Context, tx pgx.Tx, job jobs.Job) error) func(ctx context.Context, tx pgx.Tx, job jobs.Job) error {
	switch {
	case first == nil:
		return second
	case second == nil:
		return first
	}
	return func(ctx context.Context, tx pgx.Tx, job jobs.Job) error {
		if err := first(ctx, tx, job); err != nil {
			return err
		}
		return second(ctx, tx, job)
	}
}

// clipWaveforms lists the stored peaks of a clip, one entry per audio track.
func (s *Server) clipWaveforms(ctx context.Context, clipID int64) ([]map[string]any, error) {
	rows, err := s.Pool.Query(ctx, `SELECT audio_stream_index,job_id,peaks FROM clip_waveforms WHERE clip_id=$1 ORDER BY audio_stream_index`, clipID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	waveforms := []map[string]any{}
	for rows.Next() {
		var index int
		var jobID int64
		var peaks json.RawMessage
		if err := rows.Scan(&index, &jobID, &peaks); err != nil {
			return nil, err
		}
		waveforms = append(waveforms, map[string]any{"audio_stream_index": index, "job_id": jobID, "peaks": peaks})
	}
	return waveforms, rows.Err()
}
