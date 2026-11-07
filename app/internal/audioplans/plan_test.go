package audioplans_test

import (
	"encoding/json"
	"testing"

	"snitch/internal/audioplans"
	"snitch/internal/httpapi/contract"
)

func TestPlanCanonicalIdentity(t *testing.T) {
	input := fixture(t)
	a, err := audioplans.CanonicalInput(input)
	if err != nil {
		t.Fatal(err)
	}
	input.TrackIndices = []int{7, 2, 7}
	b, err := audioplans.CanonicalInput(input)
	if err != nil || a.Version != b.Version {
		t.Fatalf("selection order must not change identity: %v", err)
	}
	input.PipelineVersion = "pipeline-v2"
	c, err := audioplans.CanonicalInput(input)
	if err != nil || c.Version == a.Version {
		t.Fatal("pipeline change must invalidate input", err)
	}
	input.TrackIndices = []int{99}
	if _, err := audioplans.CanonicalInput(input); err == nil {
		t.Fatal("unknown ffprobe index accepted")
	}
}

func fixture(t *testing.T) audioplans.Input {
	t.Helper()
	var v audioplans.Input
	if err := json.Unmarshal(audioplans.ContractFixture(), &v); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestRevisionTemporalOrderAndMultipleSources(t *testing.T) {
	id := int64(42)
	wi := 0
	source := func(track int, job int64) contract.TranscriptSource {
		return contract.TranscriptSource{TrackIndex: track, JobId: job, SegmentIndex: 3}
	}
	a, b := source(7, 15), source(2, 14)
	aw, bw := a, b
	aw.WordIndex = &wi
	bw.WordIndex = &wi
	input := audioplans.Revision{ClipId: 6, PlanId: &id, TimeBase: "clip", Segments: []contract.TranscriptManifestSegment{
		{Start: 4, End: 5, Sources: []contract.TranscriptSource{a}},
		{Start: 1, End: 2, Sources: []contract.TranscriptSource{a, b}, Words: []contract.TranscriptManifestWord{{Start: 1.1, End: 1.9, Sources: []contract.TranscriptSource{aw, bw}}}},
	}}
	got, err := audioplans.CanonicalRevision(input)
	if err != nil {
		t.Fatal(err)
	}
	if got.Segments[0].Start != 1 || got.Segments[0].Sources[0].TrackIndex != 2 || len(got.Segments[0].Words[0].Sources) != 2 || got.Segments[1].Index != 1 {
		t.Fatal(got)
	}
	input.Segments[0], input.Segments[1] = input.Segments[1], input.Segments[0]
	replay, err := audioplans.CanonicalRevision(input)
	if err != nil || got.Id != replay.Id {
		t.Fatal("order affects revision", err)
	}
	input.Segments[0].Words[0].End = 9
	if _, err = audioplans.CanonicalRevision(input); err == nil {
		t.Fatal("word outside clip segment accepted")
	}
}

func TestRevisionTieOrderIncludesAllProvenance(t *testing.T) {
	id := int64(42)
	// A source can contribute to more than one output chunk. All provenance must
	// break ties, even when the first source and timestamps are identical.
	a := contract.TranscriptSource{TrackIndex: 2, JobId: 14, SegmentIndex: 0}
	b := contract.TranscriptSource{TrackIndex: 7, JobId: 15, SegmentIndex: 0}
	c := b
	c.SegmentIndex = 1
	input := audioplans.Revision{ClipId: 6, PlanId: &id, TimeBase: "clip", Segments: []contract.TranscriptManifestSegment{
		{Start: 1, End: 2, Sources: []contract.TranscriptSource{a, c}},
		{Start: 1, End: 2, Sources: []contract.TranscriptSource{a, b}},
	}}
	first, err := audioplans.CanonicalRevision(input)
	if err != nil {
		t.Fatal(err)
	}
	input.Segments[0], input.Segments[1] = input.Segments[1], input.Segments[0]
	second, err := audioplans.CanonicalRevision(input)
	if err != nil || first.Id != second.Id {
		t.Fatal("tied provenance depends on input order", err)
	}
}

func TestRevisionTimeTiesUseRealTrackOrder(t *testing.T) {
	id := int64(42)
	input := audioplans.Revision{ClipId: 6, PlanId: &id, TimeBase: "clip", Segments: []contract.TranscriptManifestSegment{
		{Start: 1, End: 2, Sources: []contract.TranscriptSource{{TrackIndex: 7, JobId: 10, SegmentIndex: 0}}},
		{Start: 1, End: 2, Sources: []contract.TranscriptSource{{TrackIndex: 2, JobId: 99, SegmentIndex: 0}}},
	}}
	got, err := audioplans.CanonicalRevision(input)
	if err != nil || got.Segments[0].Sources[0].TrackIndex != 2 {
		t.Fatal("equal timestamps must order by real track before job identity", got, err)
	}
}
