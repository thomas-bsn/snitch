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
	// A kept track whose voices carry no word has no segment: its voices stay listed, and its
	// coverage says so, instead of vanishing with the words (#273, #342).
	if revision, ok := object["revision"].(map[string]any); ok {
		if plan, ok := revision["plan_id"].(float64); ok {
			rows, err := s.Pool.Query(ctx, `SELECT track_index FROM audio_plan_tracks WHERE plan_id=$1`, int64(plan))
			if err != nil {
				return err
			}
			for rows.Next() {
				var index int
				if err = rows.Scan(&index); err != nil {
					rows.Close()
					return err
				}
				if _, present := groups[index]; !present {
					groups[index] = []any{}
				}
			}
			rows.Close()
			if err = rows.Err(); err != nil {
				return err
			}
		}
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
		if err := s.attachTrackSpeakers(ctx, clipID, part, groupedStates[track], true); err != nil {
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
