package audioplans_test

import (
	"math/rand"
	"testing"

	"snitch/internal/audioplans"
	"snitch/internal/httpapi/contract"
)

func report(t *testing.T) audioplans.EvidenceReport {
	t.Helper()
	r, err := audioplans.ParseEvidence(audioplans.EvidenceFixture())
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// speech builds the ASR segments of one track: two phrases inside the 1.93 s event.
func speech(track int, bounds ...float64) []audioplans.SourceSegment {
	out := []audioplans.SourceSegment{}
	for i := 0; i+1 < len(bounds); i++ {
		out = append(out, audioplans.SourceSegment{Source: contract.TranscriptSource{TrackIndex: track, JobId: int64(track * 10), SegmentIndex: i}, Start: bounds[i], End: bounds[i+1]})
	}
	return out
}

func all(groups ...[]audioplans.SourceSegment) []audioplans.SourceSegment {
	out := []audioplans.SourceSegment{}
	for _, g := range groups {
		out = append(out, g...)
	}
	return out
}

func tracksOf(s contract.TranscriptManifestSegment) []int {
	out := []int{}
	for _, source := range s.Sources {
		out = append(out, source.TrackIndex)
	}
	return out
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestExactCopyIsOneRetainedPassageWithEveryProvenance(t *testing.T) {
	got, err := audioplans.MergeAcoustic(all(speech(7, 0.2, 1.0, 1.8), speech(2, 0.2, 1.0, 1.8)), report(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Segments) != 2 {
		t.Fatalf("expected one passage per phrase, got %d", len(got.Segments))
	}
	for _, s := range got.Segments {
		if !equalInts(tracksOf(s), []int{2, 7}) || s.Sources[0].JobId != 20 || s.Sources[1].JobId != 70 {
			t.Fatalf("identity must be the lowest real track, with all provenances: %+v", s.Sources)
		}
		if s.Grouping == nil || len(*s.Grouping) != 1 || (*s.Grouping)[0].Strategy != contract.TranscriptGroupingStrategy("exact_pcm") || (*s.Grouping)[0].Source.TrackIndex != 7 {
			t.Fatalf("grouping must record the exact PCM proof: %+v", s.Grouping)
		}
	}
	if got.Retained[audioplans.SourceKey{Track: 7, Job: 70, Segment: 1}].TrackIndex != 2 {
		t.Fatal("track 7 segment must point at its retained representative")
	}
}

func TestGainCopyIsGroupedAndRecordsTheGain(t *testing.T) {
	got, err := audioplans.MergeAcoustic(all(speech(2, 0.2, 1.8), speech(11, 0.2, 1.8)), report(t))
	if err != nil || len(got.Segments) != 1 {
		t.Fatal(got, err)
	}
	g := (*got.Segments[0].Grouping)[0]
	if g.Strategy != contract.TranscriptGroupingStrategy("gain") || g.Gain == nil || *g.Gain < 0.49 || *g.Gain > 0.51 || g.StrategyVersion != "1" {
		t.Fatalf("%+v", g)
	}
}

func TestUnprovenCopiesAndOtherVoicesAreAllKept(t *testing.T) {
	// 9: same words by another voice; 12: AAC copy; 13: speech plus a game bed. None is proven.
	input := all(speech(2, 0.2, 1.8), speech(9, 0.2, 1.8), speech(12, 0.2, 1.8), speech(13, 0.2, 1.8))
	got, err := audioplans.MergeAcoustic(input, report(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Segments) != 4 || len(got.Suppressed) != 0 {
		t.Fatalf("uncertain evidence must never suppress: %+v", got.Segments)
	}
	for _, s := range got.Segments {
		if s.Grouping != nil || len(s.Sources) != 1 {
			t.Fatal("kept passage grew a provenance", s)
		}
	}
}

func TestArrivalOrderNeverChangesTheDecision(t *testing.T) {
	r := report(t)
	input := all(speech(2, 0.2, 1.0, 1.8), speech(7, 0.2, 1.0, 1.8), speech(11, 0.2, 1.0, 1.8), speech(9, 0.2, 1.0, 1.8))
	id := int64(5)
	var want string
	for seed := int64(0); seed < 20; seed++ {
		rng := rand.New(rand.NewSource(seed))
		shuffled := append([]audioplans.SourceSegment(nil), input...)
		rng.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
		evidence := audioplans.EvidenceReport{SchemaVersion: r.SchemaVersion, Comparisons: append([]audioplans.Comparison(nil), r.Comparisons...)}
		rng.Shuffle(len(evidence.Comparisons), func(i, j int) {
			evidence.Comparisons[i], evidence.Comparisons[j] = evidence.Comparisons[j], evidence.Comparisons[i]
		})
		got, err := audioplans.MergeAcoustic(shuffled, evidence)
		if err != nil {
			t.Fatal(err)
		}
		revision, err := audioplans.CanonicalRevision(audioplans.Revision{ClipId: 6, PlanId: &id, TimeBase: "clip", Segments: got.Segments})
		if err != nil {
			t.Fatal(err)
		}
		if want == "" {
			want = revision.Id
			if len(revision.Segments) != 4 { // 2 grouped phrases (tracks 2,7,11) + 2 phrases of track 9
				t.Fatalf("unexpected grouping: %d", len(revision.Segments))
			}
		}
		if revision.Id != want {
			t.Fatalf("seed %d changed the revision identity", seed)
		}
	}
}

func TestSimilarityIsNotTransitive(t *testing.T) {
	// Evidence 2~7 and 7~11 only: 11 is not proven against the retained representative 2.
	r := report(t)
	kept := []audioplans.Comparison{}
	for _, c := range r.Comparisons {
		if c.Left.TrackIndex == 2 && c.Right.TrackIndex == 11 {
			continue
		}
		kept = append(kept, c)
	}
	got, err := audioplans.MergeAcoustic(all(speech(2, 0.2, 1.8), speech(7, 0.2, 1.8), speech(11, 0.2, 1.8)), audioplans.EvidenceReport{SchemaVersion: 1, Comparisons: kept})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Segments) != 2 {
		t.Fatalf("track 11 must stay: %+v", got.Segments)
	}
	for _, s := range got.Segments {
		if equalInts(tracksOf(s), []int{2, 7}) || equalInts(tracksOf(s), []int{11}) {
			continue
		}
		t.Fatalf("unexpected group %v", tracksOf(s))
	}
}

func TestPartialOverlapKeepsWhatIsNotProven(t *testing.T) {
	r := report(t)
	partial := audioplans.EvidenceReport{SchemaVersion: 1}
	for _, c := range r.Comparisons {
		if c.Left.TrackIndex == 2 && c.Right.TrackIndex == 7 {
			c.Matched = []audioplans.EvidenceRange{{LeftStart: 0, LeftEnd: 1.0, RightStart: 0, RightEnd: 1.0}}
			partial.Comparisons = append(partial.Comparisons, c)
		}
	}
	// The second phrase straddles the proven range: both copies remain, whole.
	got, err := audioplans.MergeAcoustic(all(speech(2, 0.2, 1.0, 1.8), speech(7, 0.2, 1.0, 1.8)), partial)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Segments) != 3 || len(got.Suppressed) != 1 {
		t.Fatalf("only the fully proven phrase groups: %+v", got.Segments)
	}
	straddling := all(speech(2, 0.2, 1.8), speech(7, 0.2, 1.8))
	got, _ = audioplans.MergeAcoustic(straddling, partial)
	if len(got.Segments) != 2 || len(got.Suppressed) != 0 {
		t.Fatal("a segment only partly proven must not be cut or dropped", got.Segments)
	}
}

func TestNothingOnTheRetainedSideKeepsTheSource(t *testing.T) {
	// Track 2's recognition missed the phrase: dropping track 7's words would lose them.
	got, err := audioplans.MergeAcoustic(all(speech(7, 0.2, 1.8)), report(t))
	if err != nil || len(got.Segments) != 1 || len(got.Suppressed) != 0 {
		t.Fatal(got, err)
	}
	got, err = audioplans.MergeAcoustic(all(speech(2, 0.2, 0.6), speech(7, 0.2, 1.8)), report(t))
	if err != nil || len(got.Segments) != 2 {
		t.Fatal("a half-covered image must not group", got.Segments, err)
	}
}

func TestStreamOffsetsAreHonouredThroughMatchedClipTimes(t *testing.T) {
	// Track 7 carries the same speech 0.5 s later on the clip timeline (a +500 ms stream offset).
	evidence := audioplans.EvidenceReport{SchemaVersion: 1, Comparisons: []audioplans.Comparison{{
		Left: audioplans.EvidenceEvent{TrackIndex: 2}, Right: audioplans.EvidenceEvent{TrackIndex: 7},
		Decision: "duplicate", Strategy: ptr("exact_pcm"), StrategyVersion: ptr("1"),
		Matched: []audioplans.EvidenceRange{{LeftStart: 0.2, LeftEnd: 1.8, RightStart: 0.7, RightEnd: 2.3}},
	}}}
	got, err := audioplans.MergeAcoustic(all(speech(2, 0.2, 1.8), speech(7, 0.7, 2.3)), evidence)
	if err != nil || len(got.Segments) != 1 || !equalInts(tracksOf(got.Segments[0]), []int{2, 7}) {
		t.Fatal(got, err)
	}
	// The same words 0.5 s later without evidence are a different event.
	got, _ = audioplans.MergeAcoustic(all(speech(2, 0.2, 1.8), speech(7, 0.7, 2.3)), audioplans.EvidenceReport{SchemaVersion: 1})
	if len(got.Segments) != 2 {
		t.Fatal("timestamps alone must never group")
	}
}

func ptr(s string) *string { return &s }

func TestOnlyEnabledCurrentRulesCount(t *testing.T) {
	base := func(strategy, version string) audioplans.EvidenceReport {
		return audioplans.EvidenceReport{SchemaVersion: 1, Comparisons: []audioplans.Comparison{{
			Left: audioplans.EvidenceEvent{TrackIndex: 2}, Right: audioplans.EvidenceEvent{TrackIndex: 7},
			Decision: "duplicate", Strategy: ptr(strategy), StrategyVersion: ptr(version),
			Matched: []audioplans.EvidenceRange{{LeftStart: 0, LeftEnd: 2, RightStart: 0, RightEnd: 2}},
		}}}
	}
	input := all(speech(2, 0.2, 1.8), speech(7, 0.2, 1.8))
	for _, c := range []struct{ strategy, version string }{{"codec", "1"}, {"exact_pcm", "2"}, {"mixture", "1"}} {
		got, _ := audioplans.MergeAcoustic(input, base(c.strategy, c.version))
		if len(got.Segments) != 2 {
			t.Fatalf("%s@%s must not suppress", c.strategy, c.version)
		}
	}
	rejected := base("exact_pcm", "1")
	rejected.Comparisons[0].Decision = "uncertain_keep"
	if got, _ := audioplans.MergeAcoustic(input, rejected); len(got.Segments) != 2 {
		t.Fatal("only duplicate decisions group")
	}
	if _, err := audioplans.ParseEvidence([]byte(`{"schema_version":2,"comparisons":[]}`)); err == nil {
		t.Fatal("unknown schema accepted")
	}
}

func TestCanonicalRevisionRefusesUnprovenGrouping(t *testing.T) {
	id := int64(9)
	merged, _ := audioplans.MergeAcoustic(all(speech(2, 0.2, 1.8), speech(7, 0.2, 1.8)), report(t))
	build := func(mutate func(*contract.TranscriptManifestSegment)) error {
		segments := append([]contract.TranscriptManifestSegment(nil), merged.Segments...)
		g := append([]contract.TranscriptGrouping(nil), *segments[0].Grouping...)
		segments[0].Grouping = &g
		mutate(&segments[0])
		_, err := audioplans.CanonicalRevision(audioplans.Revision{ClipId: 1, PlanId: &id, TimeBase: "clip", Segments: segments})
		return err
	}
	if err := build(func(s *contract.TranscriptManifestSegment) {}); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*contract.TranscriptManifestSegment){
		"disabled rule":    func(s *contract.TranscriptManifestSegment) { (*s.Grouping)[0].Strategy = "codec" },
		"stale version":    func(s *contract.TranscriptManifestSegment) { (*s.Grouping)[0].StrategyVersion = "0" },
		"foreign source":   func(s *contract.TranscriptManifestSegment) { (*s.Grouping)[0].Source.TrackIndex = 99 },
		"identity source":  func(s *contract.TranscriptManifestSegment) { (*s.Grouping)[0].Source = s.Sources[0] },
		"duplicate member": func(s *contract.TranscriptManifestSegment) { *s.Grouping = append(*s.Grouping, (*s.Grouping)[0]) },
	} {
		if err := build(mutate); err == nil {
			t.Fatalf("%s accepted", name)
		}
	}
}

func TestStrategyChangeChangesThePlanInputVersion(t *testing.T) {
	a := fixture(t)
	a.MergeVersion = audioplans.MergeVersion()
	b := a
	b.MergeVersion = audioplans.MergeVersion() + "+gain@2"
	ca, err1 := audioplans.CanonicalInput(a)
	cb, err2 := audioplans.CanonicalInput(b)
	if err1 != nil || err2 != nil || ca.Version == cb.Version {
		t.Fatal("a changed acoustic strategy must produce another Plan input version", err1, err2)
	}
	if audioplans.MergeVersion() != "acoustic-dedup/1+exact_pcm@1+gain@1" {
		t.Fatal(audioplans.MergeVersion())
	}
}
