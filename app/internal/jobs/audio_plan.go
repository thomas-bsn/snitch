package jobs

import (
	"context"
	"encoding/json"
	"fmt"

	"snitch/internal/audioplans"
	"snitch/internal/httpapi/contract"
	"snitch/internal/multitrack"
)

// audioPlanInput loads immutable plan snapshots rather than parsing legacy
// version strings or consulting the current pipeline/vocabulary.
func (s *Service) audioPlanInput(ctx context.Context, input *Input) error {
	var planID int64
	var trackIndex int
	if err := s.pool.QueryRow(ctx, `SELECT plan_id,track_index FROM audio_plan_children WHERE job_id=$1 AND stage='transcription' AND input_version=$2`, input.Job.ID, input.Job.InputVersion).Scan(&planID, &trackIndex); err != nil {
		return err
	}
	state, err := audioplans.New(s.pool).GetPlan(ctx, planID)
	if err != nil {
		return err
	}
	if err = s.pool.QueryRow(ctx, `SELECT terms FROM transcription_vocabulary_versions WHERE version=$1`, state.Plan.Input.VocabularyVersion).Scan(&input.Vocabulary); err != nil {
		// pgx's JSON decoder supports the typed vocabulary slice.
		return fmt.Errorf("read audio plan vocabulary: %w", err)
	}
	var raw []byte
	if err = s.pool.QueryRow(ctx, `SELECT config FROM transcription_pipeline_versions WHERE version=$1`, state.Plan.Input.PipelineVersion).Scan(&raw); err != nil {
		return err
	}
	var pipeline map[string]any
	if err = json.Unmarshal(raw, &pipeline); err != nil {
		return err
	}
	task := multitrack.Task{PlanID: planID, InputVersion: state.Plan.InputVersion, TrackIndex: trackIndex, Intervals: []contract.AudioSpeechInterval{}}
	for _, track := range state.Plan.Input.Inventory {
		if track.Index == trackIndex {
			task.StartSeconds = track.StartSeconds
		}
	}
	for _, diagnostic := range state.Diagnostics {
		if diagnostic.TrackIndex == trackIndex {
			task.Intervals = diagnostic.Intervals
		}
	}
	input.AudioPlan = &task
	input.PipelineVersion = state.Plan.Input.PipelineVersion
	input.Pipeline, err = json.Marshal(pipeline)
	return err
}
