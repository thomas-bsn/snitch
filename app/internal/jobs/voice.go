package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

// ErrStaleResult marks a voice result computed against a diarization that has
// since been replaced.
var ErrStaleResult = errors.New("stale voice result")

// VoiceInputVersion returns the pipeline version embedded in a voice step's
// input version ("d5-a2-pvoice-xxxx" or "f5-a2-pvoice-xxxx-d12").
func VoiceInputVersion(inputVersion string) (pipeline string, diarizationJobID int64, ok bool) {
	marker := strings.Index(inputVersion, "-p")
	if marker < 0 {
		return "", 0, false
	}
	pipeline = inputVersion[marker+2:]
	if cut := strings.LastIndex(pipeline, "-d"); cut >= 0 && strings.HasPrefix(inputVersion, "f5-") {
		id, err := strconv.ParseInt(pipeline[cut+2:], 10, 64)
		if err != nil {
			return "", 0, false
		}
		return pipeline[:cut], id, true
	}
	return pipeline, 0, true
}

type voiceInputPayload struct {
	PipelineVersion  string          `json:"pipeline_version"`
	Config           json.RawMessage `json:"config"`
	Diarization      json.RawMessage `json:"diarization,omitempty"`
	FingerprintModel json.RawMessage `json:"fingerprint_model,omitempty"`
}

// voiceInput gives a leased voice step its pinned configuration and, for
// fingerprints, the speakers and turns it must embed.
func (s *Service) voiceInput(ctx context.Context, input *Input) error {
	pipeline, diarizationID, ok := VoiceInputVersion(input.Job.InputVersion)
	if !ok {
		return fmt.Errorf("voice input has no pipeline version")
	}
	payload := voiceInputPayload{PipelineVersion: pipeline}
	if err := s.pool.QueryRow(ctx, `SELECT config FROM voice_pipeline_versions WHERE version=$1`, pipeline).Scan(&payload.Config); err != nil {
		return fmt.Errorf("read voice pipeline snapshot: %w", err)
	}
	if diarizationID > 0 {
		// The embedding model is independent of the diarization engine. Send the
		// exact catalog reference, including for candidate benchmark jobs.
		var config map[string]json.RawMessage
		if err := json.Unmarshal(payload.Config, &config); err != nil {
			return fmt.Errorf("decode voice pipeline snapshot: %w", err)
		}
		if string(config["provider"]) != `"fake"` {
			var reference []byte
			err := s.pool.QueryRow(ctx, `SELECT jsonb_build_object('repo',c.repo,'revision',c.revision,'engine',c.engine)
FROM voice_model_catalog c WHERE c.id=COALESCE((SELECT requested_model_id FROM jobs WHERE id=$1),(SELECT model_id FROM voice_active_model WHERE kind='fingerprint'))`, input.Job.ID).Scan(&reference)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("read fingerprint catalog reference: %w", err)
			}
			if err == nil {
				payload.FingerprintModel = reference
			}
		}
		var raw []byte
		err := s.pool.QueryRow(ctx, `SELECT COALESCE(jsonb_agg(jsonb_build_object('label',ls.label,'turns',COALESCE((SELECT jsonb_agg(jsonb_build_object('index',st.turn_index,'start',st.start_seconds,'end',st.end_seconds) ORDER BY st.turn_index) FROM speaker_turns st WHERE st.local_speaker_id=ls.id),'[]'::jsonb)) ORDER BY ls.ordinal),'[]'::jsonb) FROM local_speakers ls WHERE ls.diarization_job_id=$1`, diarizationID).Scan(&raw)
		if err != nil {
			return fmt.Errorf("read diarization for fingerprints: %w", err)
		}
		if offset, offsetErr := PlanTrackOffset(ctx, s.pool, input.Job.ID); offsetErr != nil {
			return fmt.Errorf("read plan track offset: %w", offsetErr)
		} else if offset != 0 {
			// Stored turns are in clip time; the worker reads the stream from its own origin.
			var speakers []map[string]any
			if err = json.Unmarshal(raw, &speakers); err != nil {
				return fmt.Errorf("decode diarization for fingerprints: %w", err)
			}
			for _, speaker := range speakers {
				turns, _ := speaker["turns"].([]any)
				for _, turn := range turns {
					if m, ok := turn.(map[string]any); ok {
						for _, key := range []string{"start", "end"} {
							if v, isNumber := m[key].(float64); isNumber {
								m[key] = v - offset
							}
						}
					}
				}
			}
			if raw, err = json.Marshal(speakers); err != nil {
				return err
			}
		}
		diarization, _ := json.Marshal(map[string]any{"job_id": diarizationID, "speakers": json.RawMessage(raw)})
		payload.Diarization = diarization
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	input.Voice = encoded
	return nil
}

// PlanTrackOffset is the start offset, in seconds, of the audio stream a plan child works on, as
// the plan inventory recorded it. A worker measures a stream from its own first sample while the
// transcript manifest of a plan is in clip time: voice times are converted exactly once, on
// storage (add) and on the way back to a worker (subtract). A job that is not a bound child, or
// a stream whose offset is unknown, has none.
func PlanTrackOffset(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, jobID int64) (float64, error) {
	var offset float64
	err := q.QueryRow(ctx, `SELECT COALESCE((SELECT (t->>'start_seconds')::float8 FROM jsonb_array_elements(p.input->'inventory') t WHERE (t->>'index')::int=c.track_index),0)
FROM audio_plan_children c JOIN audio_plans p ON p.id=c.plan_id WHERE c.job_id=$1`, jobID).Scan(&offset)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	return offset, err
}
