package audioplans

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"

	"snitch/internal/httpapi/contract"
)

func sourceLess(a, b contract.TranscriptSource) bool {
	if a.TrackIndex != b.TrackIndex {
		return a.TrackIndex < b.TrackIndex
	}
	if a.JobId != b.JobId {
		return a.JobId < b.JobId
	}
	if a.SegmentIndex != b.SegmentIndex {
		return a.SegmentIndex < b.SegmentIndex
	}
	aw, bw := -1, -1
	if a.WordIndex != nil {
		aw = *a.WordIndex
	}
	if b.WordIndex != nil {
		bw = *b.WordIndex
	}
	return aw < bw
}

// provenanceOrder compares numeric real track/source identities before the full
// payload tie-breaker. Job creation order cannot outrank the real track index.
func provenanceOrder(a, b []contract.TranscriptSource) int {
	for i := 0; i < len(a) && i < len(b); i++ {
		if sourceLess(a[i], b[i]) {
			return -1
		}
		if sourceLess(b[i], a[i]) {
			return 1
		}
	}
	if len(a) < len(b) {
		return -1
	}
	if len(a) > len(b) {
		return 1
	}
	return 0
}

// canonicalLess provides a total order after timestamps, including every source
// and word. Input array positions never participate in revision identity.
func canonicalLess(a, b any) bool {
	left, _ := json.Marshal(a)
	right, _ := json.Marshal(b)
	return bytes.Compare(left, right) < 0
}
func validTimes(start, end float64) bool {
	return finite(start) && finite(end) && start >= 0 && end >= start
}
func canonicalSources(sources []contract.TranscriptSource, word bool) error {
	if len(sources) == 0 {
		return ErrConflict
	}
	for _, s := range sources {
		if s.TrackIndex < 0 || s.JobId < 1 || s.SegmentIndex < 0 || word != (s.WordIndex != nil) || (word && *s.WordIndex < 0) {
			return ErrConflict
		}
	}
	sort.Slice(sources, func(i, j int) bool { return sourceLess(sources[i], sources[j]) })
	for i := 1; i < len(sources); i++ {
		if !sourceLess(sources[i-1], sources[i]) {
			return ErrConflict
		}
	}
	return nil
}

// CanonicalRevision orders provenance and segments deterministically. Published
// indices refer to this order; original source indices are never rewritten.
func CanonicalRevision(input Revision) (Revision, error) {
	raw, err := json.Marshal(input)
	if err != nil {
		return Revision{}, err
	}
	var r Revision
	if err = json.Unmarshal(raw, &r); err != nil {
		return r, err
	}
	if r.PlanId == nil || *r.PlanId < 1 || r.LegacyJobId != nil || r.ClipId < 1 || r.TimeBase != "clip" {
		return r, ErrConflict
	}
	for i := range r.Segments {
		s := &r.Segments[i]
		s.Index = 0
		if !validTimes(s.Start, s.End) {
			return r, ErrConflict
		}
		if err = canonicalSources(s.Sources, false); err != nil {
			return r, err
		}
		for j := range s.Words {
			w := &s.Words[j]
			w.Index = 0
			if !validTimes(w.Start, w.End) || w.Start < s.Start || w.End > s.End {
				return r, ErrConflict
			}
			if err = canonicalSources(w.Sources, true); err != nil {
				return r, err
			}
			for _, source := range w.Sources {
				found := false
				for _, parent := range s.Sources {
					if source.TrackIndex == parent.TrackIndex && source.JobId == parent.JobId && source.SegmentIndex == parent.SegmentIndex {
						found = true
						break
					}
				}
				if !found {
					return r, ErrConflict
				}
			}
		}
		sort.Slice(s.Words, func(i, j int) bool {
			a, b := s.Words[i], s.Words[j]
			if a.Start != b.Start {
				return a.Start < b.Start
			}
			if a.End != b.End {
				return a.End < b.End
			}
			if order := provenanceOrder(a.Sources, b.Sources); order != 0 {
				return order < 0
			}
			return canonicalLess(a, b)
		})
		if s.Words == nil {
			s.Words = []contract.TranscriptManifestWord{}
		}
		for j := range s.Words {
			s.Words[j].Index = j
		}
	}
	sort.Slice(r.Segments, func(i, j int) bool {
		a, b := r.Segments[i], r.Segments[j]
		if a.Start != b.Start {
			return a.Start < b.Start
		}
		if a.End != b.End {
			return a.End < b.End
		}
		if order := provenanceOrder(a.Sources, b.Sources); order != 0 {
			return order < 0
		}
		return canonicalLess(a, b)
	})
	if r.Segments == nil {
		r.Segments = []contract.TranscriptManifestSegment{}
	}
	for i := range r.Segments {
		r.Segments[i].Index = i
	}
	r.Empty = len(r.Segments) == 0
	r.Id = ""
	raw, err = json.Marshal(r)
	if err != nil {
		return r, err
	}
	sum := sha256.Sum256(raw)
	r.Id = fmt.Sprintf("plan:%d:%s", *r.PlanId, hex.EncodeToString(sum[:]))
	return r, nil
}

// StageRevision validates real source segments/words against completed bound jobs
// and stores an immutable candidate. This does NOT publish or invalidate anything.
// #340 owns publication: completeness, loss preview and atomic pointer replacement.
func (s *Store) StageRevision(ctx context.Context, input Revision) (Revision, error) {
	r, err := CanonicalRevision(input)
	if err != nil {
		return r, err
	}
	var clipID int64
	if err = s.db.QueryRow(ctx, `SELECT clip_id FROM audio_plans WHERE id=$1`, *r.PlanId).Scan(&clipID); err != nil {
		return r, err
	}
	if clipID != r.ClipId {
		return r, ErrConflict
	}
	// Validate the whole manifest in one database round trip, including words.
	// Long clips must not make one network request per source word.
	sources := []contract.TranscriptSource{}
	for _, segment := range r.Segments {
		sources = append(sources, segment.Sources...)
		for _, word := range segment.Words {
			sources = append(sources, word.Sources...)
		}
	}
	sourceJSON, err := json.Marshal(sources)
	if err != nil {
		return r, err
	}
	var valid bool
	err = s.db.QueryRow(ctx, `SELECT NOT EXISTS(
 SELECT 1 FROM jsonb_to_recordset($2::jsonb) AS source(track_index int,job_id bigint,segment_index int,word_index int)
 WHERE NOT EXISTS(SELECT 1 FROM audio_plan_children c JOIN jobs j ON j.id=c.job_id
 WHERE c.plan_id=$1 AND c.track_index=source.track_index AND c.stage='transcription' AND j.id=source.job_id AND j.status='done'
 AND (j.result->>'audio_stream_index')::int=source.track_index
 AND jsonb_typeof(j.result->'segments'->source.segment_index)='object'
 AND (source.word_index IS NULL OR jsonb_typeof(j.result->'segments'->source.segment_index->'words'->source.word_index)='object')))
`, *r.PlanId, sourceJSON).Scan(&valid)
	if err != nil {
		return r, err
	}
	if !valid {
		return r, ErrConflict
	}
	raw, err := json.Marshal(r)
	if err != nil {
		return r, err
	}
	_, err = s.db.Exec(ctx, `INSERT INTO transcript_revisions(id,clip_id,plan_id,manifest) VALUES($1,$2,$3,$4) ON CONFLICT(id) DO NOTHING`, r.Id, r.ClipId, *r.PlanId, raw)
	return r, err
}
