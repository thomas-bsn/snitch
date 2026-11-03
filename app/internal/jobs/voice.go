package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
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
	PipelineVersion string          `json:"pipeline_version"`
	Config          json.RawMessage `json:"config"`
	Diarization     json.RawMessage `json:"diarization,omitempty"`
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
		var raw []byte
		err := s.pool.QueryRow(ctx, `SELECT COALESCE(jsonb_agg(jsonb_build_object('label',ls.label,'turns',COALESCE((SELECT jsonb_agg(jsonb_build_object('index',st.turn_index,'start',st.start_seconds,'end',st.end_seconds) ORDER BY st.turn_index) FROM speaker_turns st WHERE st.local_speaker_id=ls.id),'[]'::jsonb)) ORDER BY ls.ordinal),'[]'::jsonb) FROM local_speakers ls WHERE ls.diarization_job_id=$1`, diarizationID).Scan(&raw)
		if err != nil {
			return fmt.Errorf("read diarization for fingerprints: %w", err)
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
