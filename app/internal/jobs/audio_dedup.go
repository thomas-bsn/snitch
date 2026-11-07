package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"snitch/internal/audioplans"
)

// DedupStepPrefix is the plan-level acoustic comparison job (audio.dedup.<plan id>, ADR 0054).
const DedupStepPrefix = "audio.dedup."

type dedupSegment struct {
	Index int     `json:"index"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}
type dedupTrack struct {
	TrackIndex   int            `json:"track_index"`
	JobID        int64          `json:"job_id"`
	StartSeconds float64        `json:"start_seconds"`
	Segments     []dedupSegment `json:"segments"`
}

// audioDedupInput gives the Worker what it needs to compare the retained tracks of a plan: for each completed
// transcription child, its segments in clip time and the stream's start offset. The audio itself comes from the
// clip media the Worker already downloads. Nothing here reads text.
func (s *Service) audioDedupInput(ctx context.Context, input *Input) error {
	planID, err := strconv.ParseInt(strings.TrimPrefix(input.Job.Step, DedupStepPrefix), 10, 64)
	if err != nil {
		return fmt.Errorf("audio dedup step: %w", err)
	}
	state, err := audioplans.New(s.pool).GetPlan(ctx, planID)
	if err != nil {
		return err
	}
	task := struct {
		PlanID       int64        `json:"plan_id"`
		InputVersion string       `json:"input_version"`
		Tracks       []dedupTrack `json:"tracks"`
	}{PlanID: planID, InputVersion: state.Plan.InputVersion, Tracks: []dedupTrack{}}
	for _, child := range state.Children {
		if child.Stage != "transcription" || child.Status != "done" {
			continue
		}
		track := dedupTrack{TrackIndex: child.TrackIndex, JobID: child.JobId, Segments: []dedupSegment{}}
		for _, inventory := range state.Plan.Input.Inventory {
			if inventory.Index == child.TrackIndex {
				track.StartSeconds = inventory.StartSeconds
			}
		}
		rows, err := s.pool.Query(ctx, `SELECT (e.ord-1)::int,(e.value->>'start')::float8,(e.value->>'end')::float8 FROM jobs j, jsonb_array_elements(j.result->'segments') WITH ORDINALITY e(value,ord) WHERE j.id=$1 ORDER BY e.ord`, child.JobId)
		if err != nil {
			return err
		}
		for rows.Next() {
			var segment dedupSegment
			if err = rows.Scan(&segment.Index, &segment.Start, &segment.End); err != nil {
				rows.Close()
				return err
			}
			track.Segments = append(track.Segments, segment)
		}
		rows.Close()
		if err = rows.Err(); err != nil {
			return err
		}
		task.Tracks = append(task.Tracks, track)
	}
	input.AudioDedup, err = json.Marshal(task)
	return err
}
