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

// trialInput gives a diarization trial its pinned model and options. A trial has no pipeline version: it runs
// the catalogue model chosen for it, never the active configuration.
func (s *Service) trialInput(ctx context.Context, input *Input) error {
	id, err := strconv.ParseInt(strings.TrimPrefix(input.Job.Step, "voicetrial.diarization."), 10, 64)
	if err != nil {
		return fmt.Errorf("voice trial step: %w", err)
	}
	var repo, revision string
	var stream int
	var num, minimum, maximum *int
	if err = s.pool.QueryRow(ctx, `SELECT model_repo,model_revision,audio_stream_index,num_speakers,min_speakers,max_speakers FROM voice_diarization_trials WHERE id=$1 AND clip_id=$2`, id, input.Job.ClipID).Scan(&repo, &revision, &stream, &num, &minimum, &maximum); err != nil {
		return fmt.Errorf("read voice trial: %w", err)
	}
	config := map[string]any{"provider": "pyannote", "model": repo, "model_version": revision}
	for key, value := range map[string]*int{"num_speakers": num, "min_speakers": minimum, "max_speakers": maximum} {
		if value != nil {
			config[key] = *value
		}
	}
	encoded, err := json.Marshal(map[string]any{"config": config, "trial": map[string]any{"audio_stream_index": stream}})
	if err != nil {
		return err
	}
	input.Voice = encoded
	return nil
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
