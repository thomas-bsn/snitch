//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/jackc/pgx/v5/pgxpool"
	"snitch/internal/audioplans"
	"snitch/internal/httpapi/contract"
)

func testAudioPlans(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	h := newVoiceHarness(t, pool)
	h.transcribe()
	var runID int64
	if err := pool.QueryRow(ctx, `INSERT INTO clip_analysis_runs(clip_id,scope,status) VALUES($1,'full','completed') RETURNING id`, h.clipID).Scan(&runID); err != nil {
		t.Fatal(err)
	}
	var input audioplans.Input
	if err := json.Unmarshal(audioplans.ContractFixture(), &input); err != nil {
		t.Fatal(err)
	}
	store := audioplans.New(pool)
	plan, err := store.CreatePlan(ctx, runID, input)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := store.CreatePlan(ctx, runID, input)
	if err != nil || replay.Id != plan.Id {
		t.Fatal("plan retry", replay, err)
	}
	input.MergeVersion = "merge-v2"
	changed, err := store.CreatePlan(ctx, runID, input)
	if err != nil || changed.Id == plan.Id {
		t.Fatal("changed plan", err)
	}
	input.TrackIndices = []int{2}
	selected, err := store.CreatePlan(ctx, runID, input)
	if err != nil || selected.Id == changed.Id || len(selected.Input.TrackIndices) != 1 {
		t.Fatal("selection must create a new persisted identity", selected, err)
	}
	if _, err = pool.Exec(ctx, `UPDATE audio_plans SET input='{}' WHERE id=$1`, plan.Id); err == nil {
		t.Fatal("persisted plan was mutable")
	}
	state, err := store.GetPlan(ctx, plan.Id)
	if err != nil || len(state.Diagnostics) != 2 || state.Diagnostics[0].Status != "unexamined" {
		t.Fatal("pending diagnostics", state, err)
	}
	raw := h.adminCall("GET", fmt.Sprintf("/api/admin/clips/%d/audio-plans/%d", h.clipID, plan.Id), nil, 200)
	assertAudioContract(t, "AudioPlanState", raw)
	h.adminCall("GET", fmt.Sprintf("/api/admin/clips/%d/audio-plans/0", h.clipID), nil, 400)
	h.adminCall("GET", fmt.Sprintf("/api/admin/clips/%d/audio-plans/999999", h.clipID), nil, 404)
	var got audioplans.State
	if err = json.Unmarshal(raw, &got); err != nil || got.Plan.Id != plan.Id {
		t.Fatal("HTTP plan", string(raw), err)
	}
	var diagnosticJob int64
	if err = pool.QueryRow(ctx, `INSERT INTO jobs(clip_id,step,input_version) VALUES($1,'audio.diagnostic.2',$2) RETURNING id`, h.clipID, plan.InputVersion).Scan(&diagnosticJob); err != nil {
		t.Fatal(err)
	}
	if err = store.BindChild(ctx, audioplans.Child{PlanId: plan.Id, TrackIndex: 2, Stage: "diagnostic", JobId: diagnosticJob, InputVersion: plan.InputVersion}); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `UPDATE jobs SET status='retry_wait',attempts=1 WHERE id=$1`, diagnosticJob); err != nil {
		t.Fatal(err)
	}
	diagnostic := audioplans.Diagnostic{PlanId: plan.Id, TrackIndex: 2, JobId: &diagnosticJob, InputVersion: plan.InputVersion, DetectorVersion: input.DetectorVersion, DetectorConfigVersion: input.DetectorConfigVersion, Attempt: 1, Status: "ambiguous", Cause: "insufficient_evidence"}
	if err = store.RecordDiagnostic(ctx, diagnostic); err != nil {
		t.Fatal(err)
	}
	if err = store.RecordDiagnostic(ctx, diagnostic); err != nil {
		t.Fatal("diagnostic replay", err)
	}
	conflicting := diagnostic
	conflicting.Status = "no_speech"
	if err = store.RecordDiagnostic(ctx, conflicting); err == nil {
		t.Fatal("observation overwritten")
	}
	if _, err = pool.Exec(ctx, `UPDATE jobs SET status='done',attempts=2 WHERE id=$1`, diagnosticJob); err != nil {
		t.Fatal(err)
	}
	diagnostic.Attempt = 2
	diagnostic.Status = "no_speech"
	diagnostic.Cause = "silence"
	if err = store.RecordDiagnostic(ctx, diagnostic); err != nil {
		t.Fatal(err)
	}
	state, err = store.GetPlan(ctx, plan.Id)
	if err != nil || state.Diagnostics[0].Attempt != 2 || state.Diagnostics[0].Status != "no_speech" || state.Diagnostics[1].Status != "unexamined" {
		t.Fatal("latest diagnostic per track", state, err)
	}
	var clip map[string]any
	if err = json.Unmarshal(h.adminCall("GET", fmt.Sprintf("/api/clips/%d", h.clipID), nil, 200), &clip); err != nil {
		t.Fatal(err)
	}
	transcript := clip["transcription"].(map[string]any)
	rev := transcript["revision"].(map[string]any)
	if rev["id"] != fmt.Sprintf("legacy:%.0f", transcript["transcription_job_id"]) || len(rev["segments"].([]any)) != 3 {
		t.Fatal("legacy projection", rev)
	}
}

func testAudioPlanChildren(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	h := newVoiceHarness(t, pool)
	h.transcribe()
	var run int64
	if err := pool.QueryRow(ctx, `INSERT INTO clip_analysis_runs(clip_id,scope,status) VALUES($1,'full','completed') RETURNING id`, h.clipID).Scan(&run); err != nil {
		t.Fatal(err)
	}
	var input audioplans.Input
	_ = json.Unmarshal(audioplans.ContractFixture(), &input)
	store := audioplans.New(pool)
	p, err := store.CreatePlan(ctx, run, input)
	if err != nil {
		t.Fatal(err)
	}
	h.adminCall("PUT", segmentTextPath(h.clipID, 0), map[string]any{"text": "protected correction", "expected_version": 0}, 200)
	var version string
	if err = pool.QueryRow(ctx, `SELECT input_version FROM jobs WHERE clip_id=$1 AND status='done' AND step='transcription.audio.2'`, h.clipID).Scan(&version); err != nil {
		t.Fatal(err)
	}
	var job int64
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if err = tx.QueryRow(ctx, `INSERT INTO jobs(clip_id,step,input_version) VALUES($1,'transcription.audio.2',$2) RETURNING id`, h.clipID, version).Scan(&job); err != nil {
		t.Fatal(err)
	}
	child := audioplans.Child{PlanId: p.Id, TrackIndex: 2, Stage: "transcription", JobId: job, InputVersion: version}
	if err = audioplans.New(tx).BindChild(ctx, child); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if err = store.BindChild(ctx, child); err != nil {
		t.Fatal("idempotent child", err)
	}
	claimed, ok := h.claim()
	if !ok || claimed.ID != job {
		t.Fatal("child not claimable", claimed)
	}
	payload := map[string]any{"provider": "fixture", "model": "deterministic-v1", "version": "deterministic-v1", "language": "fr", "audio_stream_index": 2, "segments": []map[string]any{{"start": 0.2, "end": 0.9, "text": "child only", "words": []map[string]any{{"start": 0.2, "end": 0.5, "word": "child"}, {"start": 0.5, "end": 0.9, "word": "only"}}}}}
	h.complete(claimed, payload, 200)
	h.complete(claimed, payload, 200) // lost response retry
	if h.count(`SELECT count(*) FROM embedding_tasks WHERE transcription_job_id=$1`, job) != 0 {
		t.Fatal("unpublished child scheduled visible indexing")
	}
	if got := h.clipTranscript(h.adminUser()); got.Transcription.Segments[0].Text != "protected correction" {
		t.Fatal("child invalidated correction", got)
	}
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"index":0,"embedding":[1,0]}]}`))
	}))
	defer provider.Close()
	h.api.Config.EmbeddingsURL, h.api.Config.EmbeddingsModel = provider.URL, "fixture"
	var profile int64
	if err = pool.QueryRow(ctx, `INSERT INTO embedding_profiles(model,dimensions,state) VALUES('fixture',2,'active') RETURNING id`).Scan(&profile); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `UPDATE embedding_index_state SET active_profile_id=$1 WHERE id=1`, profile); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO transcript_embeddings(profile_id,transcription_job_id,clip_id,segment_index,segment_start,segment_end,text,embedding) SELECT $1,id,$2,1,1.25,2.1,'second','[1,0]' FROM audio_visible_legacy_transcripts WHERE clip_id=$2`, profile, h.clipID); err != nil {
		t.Fatal(err)
	}
	var search correctedSearchPage
	if err = json.Unmarshal(h.adminCall("GET", "/api/search?mode=hybrid&q=zzz", nil, 200), &search); err != nil || search.Total != 1 || search.Items[0].Context != "second" || search.Items[0].MatchType != "semantic" {
		t.Fatal("child hid legacy semantic search", search, err)
	}
	transcript := h.adminCall("GET", fmt.Sprintf("/api/clips/%d", h.clipID), nil, 200)
	if bytes.Contains(transcript, []byte("child only")) {
		t.Fatal("unpublished child replaced visible transcript")
	}
	state, err := store.GetPlan(ctx, p.Id)
	if err != nil || len(state.Children) != 1 || state.Children[0].Status != "done" {
		t.Fatal(state, err)
	}
	wordIndex := 0
	revision := audioplans.Revision{PlanId: &p.Id, ClipId: h.clipID, TimeBase: "clip", Segments: []contract.TranscriptManifestSegment{{Index: 0, Start: 0.2, End: 0.9, Sources: []contract.TranscriptSource{{TrackIndex: 2, JobId: job, SegmentIndex: 0}}, Words: []contract.TranscriptManifestWord{{Start: 0.2, End: 0.5, Sources: []contract.TranscriptSource{{TrackIndex: 2, JobId: job, SegmentIndex: 0, WordIndex: &wordIndex}}}}}}}
	staged, err := store.StageRevision(ctx, revision)
	if err != nil {
		t.Fatal(err)
	}
	again, err := store.StageRevision(ctx, revision)
	if err != nil || again.Id != staged.Id {
		t.Fatal("revision replay", err)
	}
	got, err := store.GetRevision(ctx, staged.Id)
	if err != nil || got.Segments[0].Sources[0].JobId != job || got.Segments[0].Words[0].Sources[0].WordIndex == nil || *got.Segments[0].Words[0].Sources[0].WordIndex != 0 {
		t.Fatal(got, err)
	}
	wordIndex = 8
	if _, err = store.StageRevision(ctx, revision); err == nil {
		t.Fatal("nonexistent word provenance accepted")
	}
	wordIndex = 0
	revision.Segments[0].Sources[0].SegmentIndex = 8
	if _, err = store.StageRevision(ctx, revision); err == nil {
		t.Fatal("nonexistent provenance accepted")
	}
}

func assertAudioContract(t *testing.T, name string, raw []byte) {
	t.Helper()
	spec, err := openapi3.NewLoader().LoadFromFile("../../../contracts/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var value any
	if err = json.Unmarshal(raw, &value); err != nil {
		t.Fatal(err)
	}
	if err = spec.Components.Schemas[name].Value.VisitJSON(value); err != nil {
		t.Fatalf("%s contract: %v", name, err)
	}
}
