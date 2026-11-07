package httpapi

import (
	"context"
	"sort"
)

// The manifest's per-segment track is authoritative. Never let overlapping turns
// from a different track win an attribution or inflate its ASR coverage.
func (s *Server) attachRevisionSpeakers(ctx context.Context, clipID int64, object map[string]any, states map[int]segmentState) error {
	groups := map[int][]any{}
	groupedStates := map[int]map[int]segmentState{}
	for position, raw := range object["segments"].([]any) {
		segment, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		track, ok := segment["audio_stream_index"].(float64)
		if !ok {
			continue
		}
		index := int(track)
		if groupedStates[index] == nil {
			groupedStates[index] = map[int]segmentState{}
		}
		if st, ok := states[position]; ok {
			groupedStates[index][len(groups[index])] = st
		}
		groups[index] = append(groups[index], segment)
	}
	roster := []any{}
	coverage := []any{}
	// Inventory order keeps output deterministic without depending on map iteration.
	tracks := make([]int, 0, len(groups))
	for track := range groups {
		tracks = append(tracks, track)
	}
	sort.Ints(tracks)
	for _, track := range tracks {
		part := map[string]any{"audio_stream_index": float64(track), "segments": groups[track]}
		if err := s.attachLocalSpeakers(ctx, clipID, part, groupedStates[track]); err != nil {
			return err
		}
		if speakers, ok := part["local_speakers"].([]map[string]any); ok {
			for _, speaker := range speakers {
				speaker["audio_stream_index"] = track
				roster = append(roster, speaker)
			}
		}
		if value, ok := part["speaker_coverage"]; ok {
			coverage = append(coverage, map[string]any{"audio_stream_index": track, "coverage": value})
		}
	}
	object["local_speakers"] = roster
	object["speaker_coverage_by_track"] = coverage
	return nil
}
