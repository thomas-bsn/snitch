package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// audioDiagnosticInputPayload is what a Worker needs to examine one real stream. The timeline offset and the
// stream duration come from the immutable plan inventory, never from the container (ADR 0048).
type audioDiagnosticInputPayload struct {
	PlanID                int64    `json:"plan_id"`
	TrackIndex            int      `json:"track_index"`
	InputVersion          string   `json:"input_version"`
	DetectorVersion       string   `json:"detector_version"`
	DetectorConfigVersion string   `json:"detector_config_version"`
	StartSeconds          float64  `json:"start_seconds"`
	DurationSeconds       *float64 `json:"duration_seconds"`
}

func (s *Service) audioDiagnosticInput(ctx context.Context, input *Input) error {
	index, err := strconv.Atoi(strings.TrimPrefix(input.Job.Step, "audio.diagnostic."))
	if err != nil || index < 0 {
		return fmt.Errorf("audio diagnostic step has no track index")
	}
	var raw []byte
	payload := audioDiagnosticInputPayload{TrackIndex: index}
	err = s.pool.QueryRow(ctx, `SELECT p.id,p.input_version,p.input FROM audio_plan_children c JOIN audio_plans p ON p.id=c.plan_id
 WHERE c.job_id=$1 AND c.stage='diagnostic' AND c.track_index=$2`, input.Job.ID, index).Scan(&payload.PlanID, &payload.InputVersion, &raw)
	if err != nil {
		return fmt.Errorf("read audio diagnostic plan binding: %w", err)
	}
	var plan struct {
		DetectorVersion       string `json:"detector_version"`
		DetectorConfigVersion string `json:"detector_config_version"`
		Inventory             []struct {
			Index           int      `json:"index"`
			StartSeconds    float64  `json:"start_seconds"`
			DurationSeconds *float64 `json:"duration_seconds"`
		} `json:"inventory"`
	}
	if err = json.Unmarshal(raw, &plan); err != nil {
		return fmt.Errorf("decode audio diagnostic plan: %w", err)
	}
	payload.DetectorVersion, payload.DetectorConfigVersion = plan.DetectorVersion, plan.DetectorConfigVersion
	for _, track := range plan.Inventory {
		if track.Index == index {
			payload.StartSeconds, payload.DurationSeconds = track.StartSeconds, track.DurationSeconds
		}
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	input.AudioDiagnostic = encoded
	return nil
}
