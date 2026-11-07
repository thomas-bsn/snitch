//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"snitch/internal/audioplans"
	"snitch/internal/httpapi/contract"
	"strings"
	"testing"
	"time"
)

func testAudioPublication(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	h := newVoiceHarness(t, pool)
	h.transcribe()
	var run int64
	if err := pool.QueryRow(ctx, `INSERT INTO clip_analysis_runs(clip_id,scope,status) VALUES($1,'full','active') RETURNING id`, h.clipID).Scan(&run); err != nil {
		t.Fatal(err)
	}
	var input audioplans.Input
	_ = json.Unmarshal(audioplans.ContractFixture(), &input)
	input.Decisions = []contract.AudioTrackDecision{{TrackIndex: 2, Action: "include", Forced: true, ActorId: 1, Reason: "fixture"}, {TrackIndex: 7, Action: "include", Forced: true, ActorId: 1, Reason: "fixture"}}
	// The fixture's human decisions are explicit, not inferred from stream order.
	for i := range input.Decisions {
		input.Decisions[i].DecidedAt = time.Now().UTC()
	}
	store := audioplans.New(pool)
	// stage creates a plan whose children are done fixtures; tracks maps index to text.
	stage := func(reason string, tracks map[int]string, complete bool) (audioplans.Plan, audioplans.Revision) {
		in := input
		in.Decisions = append([]contract.AudioTrackDecision(nil), input.Decisions...)
		for i := range in.Decisions {
			in.Decisions[i].Reason = reason
		}
		p, err := store.CreatePlan(ctx, run, in)
		if err != nil {
			t.Fatal(err)
		}
		revision := audioplans.Revision{ClipId: h.clipID, PlanId: &p.Id, TimeBase: "clip"}
		for _, track := range []int{7, 2} {
			if track == 2 && !complete {
				continue
			}
			var job int64
			version := fmt.Sprintf("publication-%s-%d", reason, track)
			if err = pool.QueryRow(ctx, `INSERT INTO jobs(clip_id,step,input_version) VALUES($1,$2,$3) RETURNING id`, h.clipID, fmt.Sprintf("transcription.audio.%d", track), version).Scan(&job); err != nil {
				t.Fatal(err)
			}
			if err = store.BindChild(ctx, audioplans.Child{PlanId: p.Id, TrackIndex: track, Stage: "transcription", JobId: job, InputVersion: version}); err != nil {
				t.Fatal(err)
			}
			if _, err = pool.Exec(ctx, `UPDATE jobs SET status='done',result=jsonb_build_object('audio_stream_index',$2::int,'segments',jsonb_build_array(jsonb_build_object('start',1,'end',2,'text',$3::text))) WHERE id=$1`, job, track, tracks[track]); err != nil {
				t.Fatal(err)
			}
			revision.Segments = append(revision.Segments, contract.TranscriptManifestSegment{Start: 1, End: 2, Sources: []contract.TranscriptSource{{TrackIndex: track, JobId: job, SegmentIndex: 0}}})
			if got := h.clipTranscript(h.adminUser()); len(got.Transcription.Segments) > 0 && got.Transcription.Segments[0].Text == tracks[track] {
				t.Fatal("child published")
			}
		}
		staged, err := store.StageRevision(ctx, revision)
		if err != nil {
			t.Fatal(err)
		}
		return p, staged
	}
	texts := func() []string {
		out := []string{}
		for _, s := range h.clipTranscript(h.adminUser()).Transcription.Segments {
			out = append(out, s.Text)
		}
		return out
	}
	p, staged := stage("one", map[int]string{2: "voice 2", 7: "voice 7"}, true)
	preview, err := store.PreviewPublication(ctx, p.Id)
	if err != nil {
		t.Fatal(err)
	}
	if err = store.PublishRevision(ctx, staged.Id, preview, false); err != nil {
		t.Fatal(err)
	}
	if err = store.PublishRevision(ctx, staged.Id, preview, false); err != nil {
		t.Fatal("idempotent publication", err)
	}
	if got := texts(); len(got) != 2 || got[0] != "voice 2" || got[1] != "voice 7" {
		t.Fatal("complete ordered revision", got)
	}
	// Stale revisions reject writes; the visible one accepts a correction.
	path := segmentTextPath(h.clipID, 1)
	h.adminCall("PUT", path, map[string]any{"text": "x", "expected_version": 0, "expected_revision_id": "plan:0:stale"}, 409)
	h.adminCall("PUT", path, map[string]any{"text": "voix corrigée", "expected_version": 0, "expected_revision_id": staged.Id}, 200)
	var search struct {
		Total int `json:"total"`
	}
	if err = json.Unmarshal(h.adminCall("GET", "/api/search?mode=exact&q=corrig%C3%A9e", nil, 200), &search); err != nil || search.Total != 1 {
		t.Fatal("corrected text searchable", search, err)
	}
	if body := string(h.adminCall("GET", "/api/admin/segment-corrections/export", nil, 200)); !strings.Contains(body, "voix corrigée") || !strings.Contains(body, staged.Id) {
		t.Fatal("export lacks correction or revision provenance", body)
	}
	// An incomplete replacement is refused and leaves the old revision and its correction visible.
	p2, partial := stage("two", map[int]string{7: "new 7"}, false)
	preview2, err := store.PreviewPublication(ctx, p2.Id)
	if err != nil || !preview2.ConfirmationRequired {
		t.Fatal(preview2, err)
	}
	if err = store.PublishRevision(ctx, partial.Id, preview2, true); !errors.Is(err, audioplans.ErrIncomplete) {
		t.Fatal("incomplete publication", err)
	}
	if got := texts(); len(got) != 2 || got[1] != "voix corrigée" {
		t.Fatal("old revision must remain", got)
	}
	// Complete replacement: unconfirmed is refused, then a correction after the preview forces a new preview.
	p3, full := stage("three", map[int]string{2: "new 2", 7: "new 7"}, true)
	preview3, err := store.PreviewPublication(ctx, p3.Id)
	if err != nil {
		t.Fatal(err)
	}
	if err = store.PublishRevision(ctx, full.Id, preview3, false); !errors.Is(err, audioplans.ErrConfirmationRequired) {
		t.Fatal("confirmation", err)
	}
	h.adminCall("PUT", path, map[string]any{"text": "tardive", "expected_version": 1, "expected_revision_id": staged.Id}, 200)
	if err = store.PublishRevision(ctx, full.Id, preview3, true); !errors.Is(err, audioplans.ErrPublicationChanged) {
		t.Fatal("late correction must require a new preview", err)
	}
	if got := texts(); got[1] != "tardive" {
		t.Fatal("late correction overwritten", got)
	}
	preview3, _ = store.PreviewPublication(ctx, p3.Id)
	if err = store.PublishRevision(ctx, full.Id, preview3, true); err != nil {
		t.Fatal(err)
	}
	if got := texts(); got[0] != "new 2" || got[1] != "new 7" {
		t.Fatal("replacement", got)
	}
	h.adminCall("PUT", path, map[string]any{"text": "x", "expected_version": 0, "expected_revision_id": staged.Id}, 409)
}
