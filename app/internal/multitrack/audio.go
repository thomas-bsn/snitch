// Package multitrack is the versioned boundary between per-track Workers and
// the plan assembler. All result times use the clip timeline.
package multitrack

import (
	"encoding/json"
	"math"
	"snitch/internal/audioplans"
	"snitch/internal/httpapi/contract"
)

const Capability = "audio.multitrack.v1"
const Prefix = "mt1:"

type Task struct {
	PlanID       int64                          `json:"plan_id"`
	InputVersion string                         `json:"input_version"`
	TrackIndex   int                            `json:"track_index"`
	StartSeconds float64                        `json:"start_seconds"`
	Intervals    []contract.AudioSpeechInterval `json:"intervals"`
}
type TrackResult struct {
	TrackIndex   int             `json:"track_index"`
	JobID        int64           `json:"job_id"`
	InputVersion string          `json:"input_version"`
	Output       json.RawMessage `json:"output"`
}

// Result contains every retained child, never a partially successful set. The
// publisher owns merging, immutable revision staging and visible publication.
type Result struct {
	Plan   audioplans.Plan `json:"plan"`
	Tracks []TrackResult   `json:"tracks"`
	Empty  bool            `json:"empty"`
}

// ValidateOutput rejects legacy/local-time or foreign-plan results before they
// can enter publication. Worker completion remains replay-safe.
func ValidateOutput(raw []byte, plan audioplans.Plan, track int) error {
	var output struct {
		PlanID   int64  `json:"audio_plan_id"`
		Version  string `json:"audio_plan_version"`
		Track    *int   `json:"audio_stream_index"`
		TimeBase string `json:"time_base"`
		Segments *[]struct {
			Start float64 `json:"start"`
			End   float64 `json:"end"`
			Words []struct {
				Start float64 `json:"start"`
				End   float64 `json:"end"`
			} `json:"words"`
		} `json:"segments"`
	}
	if err := json.Unmarshal(raw, &output); err != nil {
		return err
	}
	if output.PlanID != plan.Id || output.Version != plan.InputVersion || output.Track == nil || *output.Track != track || output.TimeBase != "clip" || output.Segments == nil {
		return audioplans.ErrConflict
	}
	for _, segment := range *output.Segments {
		if !validTime(segment.Start, segment.End) {
			return audioplans.ErrConflict
		}
		for _, word := range segment.Words {
			if !validTime(word.Start, word.End) || word.Start < segment.Start || word.End > segment.End {
				return audioplans.ErrConflict
			}
		}
	}
	return nil
}
func validTime(start, end float64) bool {
	return !math.IsNaN(start) && !math.IsNaN(end) && !math.IsInf(start, 0) && !math.IsInf(end, 0) && start >= 0 && end >= start
}
