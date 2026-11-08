//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"snitch/internal/audioplans"
	"snitch/internal/httpapi/contract"
)

// testAudioDedup publishes a manifest merged from REAL adapter evidence (the committed
// report over synthetic speech) through the real publication path and public API (#343).
func testAudioDedup(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	h := newVoiceHarness(t, pool)
	h.transcribe()
	var run int64
	if err := pool.QueryRow(ctx, `INSERT INTO clip_analysis_runs(clip_id,scope,status) VALUES($1,'full','active') RETURNING id`, h.clipID).Scan(&run); err != nil {
		t.Fatal(err)
	}
	var input audioplans.Input
	_ = json.Unmarshal(audioplans.ContractFixture(), &input)
	input.Decisions = []contract.AudioTrackDecision{{TrackIndex: 2, Action: "include", Forced: true, ActorId: 1, Reason: "fixture", DecidedAt: time.Now().UTC()}, {TrackIndex: 7, Action: "include", Forced: true, ActorId: 1, Reason: "fixture", DecidedAt: time.Now().UTC()}}
	input.MergeVersion = audioplans.MergeVersion()
	store := audioplans.New(pool)
	report, err := audioplans.ParseEvidence(audioplans.EvidenceFixture())
	if err != nil {
		t.Fatal(err)
	}
	count := func(table string) (n int) {
		if err := pool.QueryRow(ctx, "SELECT count(*) FROM "+table).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return
	}
	samples, speakers := count("voice_samples"), count("local_speakers")

	// stage records both tracks' ASR (track 7 words differ slightly: it is a copy, not a text twin)
	// and stages the manifest produced by the merge rules with the given evidence.
	stage := func(version string, evidence audioplans.EvidenceReport) (audioplans.Plan, audioplans.Revision, audioplans.MergeResult) {
		in := input
		in.MergeVersion = version
		p, err := store.CreatePlan(ctx, run, in)
		if err != nil {
			t.Fatal(err)
		}
		texts := map[int][2]string{2: {"go to the north", "gate now"}, 7: {"go to the northern", "gate now please"}}
		sources := []audioplans.SourceSegment{}
		for _, track := range []int{7, 2} {
			var job int64
			v := fmt.Sprintf("dedup-%s-%d", version, track)
			if err = pool.QueryRow(ctx, `INSERT INTO jobs(clip_id,step,input_version) VALUES($1,$2,$3) RETURNING id`, h.clipID, fmt.Sprintf("transcription.audio.%d", track), v).Scan(&job); err != nil {
				t.Fatal(err)
			}
			if err = store.BindChild(ctx, audioplans.Child{PlanId: p.Id, TrackIndex: track, Stage: "transcription", JobId: job, InputVersion: v}); err != nil {
				t.Fatal(err)
			}
			if _, err = pool.Exec(ctx, `UPDATE jobs SET status='done',result=jsonb_build_object('audio_stream_index',$2::int,'segments',jsonb_build_array(
 jsonb_build_object('start',0.2,'end',1.0,'text',$3::text),jsonb_build_object('start',1.0,'end',1.8,'text',$4::text))) WHERE id=$1`, job, track, texts[track][0], texts[track][1]); err != nil {
				t.Fatal(err)
			}
			for i, bounds := range [][2]float64{{0.2, 1.0}, {1.0, 1.8}} {
				sources = append(sources, audioplans.SourceSegment{Source: contract.TranscriptSource{TrackIndex: track, JobId: job, SegmentIndex: i}, Start: bounds[0], End: bounds[1]})
			}
		}
		// Evidence events carry the worker's job ids; the merge uses real track indices.
		merged, err := audioplans.MergeAcoustic(sources, evidence)
		if err != nil {
			t.Fatal(err)
		}
		staged, err := store.StageRevision(ctx, audioplans.Revision{ClipId: h.clipID, PlanId: &p.Id, TimeBase: "clip", Segments: merged.Segments})
		if err != nil {
			t.Fatal(err)
		}
		return p, staged, merged
	}

	p, staged, merged := stage(audioplans.MergeVersion(), report)
	if len(merged.Segments) != 2 || len(merged.Suppressed) != 2 {
		t.Fatal("exact copy must group both phrases", merged.Segments)
	}
	preview, err := store.PreviewPublication(ctx, p.Id)
	if err != nil {
		t.Fatal(err)
	}
	if err = store.PublishRevision(ctx, staged.Id, preview, false); err != nil {
		t.Fatal(err)
	}

	type view struct {
		Index    int                           `json:"index"`
		Text     string                        `json:"text"`
		Sources  []contract.TranscriptSource   `json:"sources"`
		Grouping []contract.TranscriptGrouping `json:"grouping"`
	}
	var clip struct {
		Transcription struct {
			Segments []view `json:"segments"`
		} `json:"transcription"`
	}
	read := func() []view {
		clip.Transcription.Segments = nil
		if err := json.Unmarshal(h.adminCall("GET", fmt.Sprintf("/api/clips/%d", h.clipID), nil, 200), &clip); err != nil {
			t.Fatal(err)
		}
		return clip.Transcription.Segments
	}
	segments := read()
	if len(segments) != 2 || segments[0].Text != "go to the north" || segments[1].Text != "gate now" {
		t.Fatal("a proven duplicate is published once, with the retained text", segments)
	}
	for _, s := range segments {
		if len(s.Sources) != 2 || s.Sources[0].TrackIndex != 2 || s.Sources[1].TrackIndex != 7 || len(s.Grouping) != 1 || s.Grouping[0].Source.TrackIndex != 7 || s.Grouping[0].Strategy != "exact_pcm" || s.Grouping[0].MatchedEnd <= s.Grouping[0].MatchedStart {
			t.Fatal("every provenance and the acoustic proof must be readable", s)
		}
	}
	// Search sees the passage once, and the grouped copy's own wording is not indexed as a second passage.
	var found struct {
		Total int `json:"total"`
	}
	for query, want := range map[string]int{"north": 1, "gate": 1} {
		found.Total = -1
		if err = json.Unmarshal(h.adminCall("GET", "/api/search?mode=exact&q="+query, nil, 200), &found); err != nil || found.Total != want {
			t.Fatal("exact search", query, found, err)
		}
	}
	// A correction belongs to the retained source and stays there.
	h.adminCall("PUT", segmentTextPath(h.clipID, 0), map[string]any{"text": "go to the north gate", "expected_version": 0, "expected_revision_id": staged.Id}, 200)
	if got := read(); got[0].Text != "go to the north gate" || got[0].Sources[0].TrackIndex != 2 {
		t.Fatal(got)
	}
	// Dedup never touches Locuteurs locaux, Personnes or validated voice samples.
	if count("voice_samples") != samples || count("local_speakers") != speakers {
		t.Fatal("deduplication changed voice data")
	}

	// A strategy change is another input version and goes through the same replacement control:
	// the corrected clip needs an explicit confirmation, and the correction is not moved.
	changed := audioplans.MergeVersion() + "+gain@2"
	p2, second, _ := stage(changed, audioplans.EvidenceReport{SchemaVersion: 1})
	if p2.InputVersion == p.InputVersion {
		t.Fatal("strategy change kept the plan input version")
	}
	preview2, err := store.PreviewPublication(ctx, p2.Id)
	if err != nil || !preview2.ConfirmationRequired || preview2.CorrectionCount != 1 {
		t.Fatal("replacement control", preview2, err)
	}
	if err = store.PublishRevision(ctx, second.Id, preview2, false); !errors.Is(err, audioplans.ErrConfirmationRequired) {
		t.Fatal(err)
	}
	if got := read(); len(got) != 2 || got[0].Text != "go to the north gate" {
		t.Fatal("refused replacement must keep the corrected revision", got)
	}
	if err = store.PublishRevision(ctx, second.Id, preview2, true); err != nil {
		t.Fatal(err)
	}
	got := read()
	if len(got) != 4 {
		t.Fatal("without evidence nothing is suppressed", got)
	}
	for _, s := range got {
		if s.Text == "go to the north gate" || len(s.Grouping) != 0 {
			t.Fatal("the stale correction must not move to another source", s)
		}
	}
}
