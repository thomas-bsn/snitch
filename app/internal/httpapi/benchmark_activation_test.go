//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func testAIBenchmarkActivation(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	h := newVoiceHarnessClips(t, pool, 1)
	base := "/api/admin/benchmarks/activation"
	preview := func(stage, model string, want int) (p struct {
		Current       string `json:"current_model"`
		AlreadyActive bool   `json:"already_active"`
		WorkersReady  int    `json:"workers_ready"`
		Reanalysis    string `json:"reanalysis"`
	}) {
		t.Helper()
		q := url.Values{"stage": {stage}, "model": {model}}
		raw := h.adminCall(http.MethodGet, base+"?"+q.Encode(), nil, want)
		_ = json.Unmarshal(raw, &p)
		return p
	}
	activeTranscription := func() string {
		var m string
		if err := pool.QueryRow(ctx, `SELECT v.config->>'model' FROM transcription_pipeline_active a JOIN transcription_pipeline_versions v ON v.version=a.version WHERE a.id=true`).Scan(&m); err != nil {
			t.Fatal(err)
		}
		return m
	}

	// Only administrators change the model.
	member := h.newUser("activation-member")
	if status, _ := h.userStatus(member, http.MethodGet, base+"?stage=transcription&model=x", nil); status != http.StatusForbidden {
		t.Fatalf("a member must be refused, got %d", status)
	}

	// The fingerprint has its own transition, a chain is not a model, and a name is checked.
	preview("fingerprint", "x", http.StatusUnprocessableEntity)
	preview("e2e", "x", http.StatusUnprocessableEntity)
	preview("transcription", "", http.StatusUnprocessableEntity)

	before := activeTranscription()
	p := preview("transcription", "vendor/model-new", http.StatusOK)
	if p.Current != before || p.AlreadyActive || p.Reanalysis != "/admin/transcription" {
		t.Fatalf("the preview names the model it replaces and where to reanalyse: %+v (active %q)", p, before)
	}
	if got := preview("transcription", before, http.StatusOK); !got.AlreadyActive {
		t.Fatal("the model already in production is reported as such")
	}

	body := func(model, expected string, confirm bool) map[string]any {
		return map[string]any{"stage": "transcription", "model": model, "expected_current": expected, "confirm": confirm}
	}
	jobsBefore := h.count(`SELECT count(*) FROM jobs`)
	h.adminCall(http.MethodPost, base, body("vendor/model-new", before, false), http.StatusBadRequest)       // never without an explicit confirmation
	h.adminCall(http.MethodPost, base, body("vendor/model-new", "another-model", true), http.StatusConflict) // another change happened since the preview
	h.adminCall(http.MethodPost, base, body(before, before, true), http.StatusConflict)                      // nothing to change
	if activeTranscription() != before {
		t.Fatal("a refused activation changes nothing")
	}
	// Only an administrator confirms, and the run named must have measured this model for this stage.
	if status, _ := h.userStatus(member, http.MethodPost, base, body("vendor/model-new", before, true)); status != http.StatusForbidden {
		t.Fatalf("a member must not confirm, got %d", status)
	}
	h.adminCall(http.MethodPost, base, map[string]any{"stage": "transcription", "model": "vendor/model-new", "expected_current": before, "confirm": true, "run_id": 999999}, http.StatusUnprocessableEntity)
	// A benchmark under way reads the live profile: no model change meanwhile.
	var dataset int64
	if err := pool.QueryRow(ctx, `INSERT INTO benchmark_datasets(name,author_name,stages) VALUES('activation','t','{transcription}') RETURNING id`).Scan(&dataset); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO benchmark_runs(dataset_id,dataset_name,stage,status,model_ids) VALUES($1,'activation','transcription','running','{vendor/model-new,other}')`, dataset); err != nil {
		t.Fatal(err)
	}
	h.adminCall(http.MethodPost, base, body("vendor/model-new", before, true), http.StatusConflict)
	if activeTranscription() != before {
		t.Fatal("a refused activation changes nothing")
	}
	var finished int64
	if err := pool.QueryRow(ctx, `UPDATE benchmark_runs SET status='completed',finished_at=now() WHERE dataset_id=$1 RETURNING id`, dataset).Scan(&finished); err != nil {
		t.Fatal(err)
	}
	h.adminCall(http.MethodPost, base, map[string]any{"stage": "transcription", "model": "vendor/model-new", "expected_current": before, "confirm": true, "run_id": finished}, http.StatusOK)
	if n := h.count(`SELECT count(*) FROM benchmark_activations WHERE run_id=$1`, finished); n != 1 {
		t.Fatalf("the change is traced to the run that measured the model, got %d", n)
	}
	h.adminCall(http.MethodPost, base, body("vendor/model-new", before, true), http.StatusConflict) // replay
	if activeTranscription() != "vendor/model-new" {
		t.Fatalf("the model is now the new one, got %q", activeTranscription())
	}
	var language string
	if err := pool.QueryRow(ctx, `SELECT v.config->>'language' FROM transcription_pipeline_active a JOIN transcription_pipeline_versions v ON v.version=a.version WHERE a.id=true`).Scan(&language); err != nil || language == "" {
		t.Fatalf("the rest of the profile is kept: %q %v", language, err)
	}
	// An activation is never a reanalysis: nothing is scheduled, nothing of a human is touched.
	if n := h.count(`SELECT count(*) FROM jobs`); n != jobsBefore {
		t.Fatalf("activating a model schedules nothing: %d jobs then %d", jobsBefore, n)
	}
	if n := h.count(`SELECT count(*) FROM benchmark_activations WHERE stage='transcription' AND to_model='vendor/model-new' AND from_model=$1 AND author_name<>''`, before); n != 1 { // one change, one row
		t.Fatalf("the change is recorded with its author, got %d", n)
	}

	// Diarization: a catalogue model that a Worker holds.
	if _, err := pool.Exec(ctx, `INSERT INTO voice_model_catalog(id,kind,engine,repo,revision) VALUES('bench-diar-act','diarization','pyannote','fixture/bench-diar-act','0123456789abcdef0123456789abcdef01234567') ON CONFLICT DO NOTHING`); err != nil {
		t.Fatal(err)
	}
	h.reportModel(communityID, "ready", "", 1, 1, http.StatusNoContent)
	h.putPipeline(map[string]any{"provider": "pyannote", "model": activeDiarization, "model_version": communityRevision, "min_fingerprint_seconds": 1.0, "min_quality": 0.5})
	diar := func(model, expected string, confirm bool, want int) {
		t.Helper()
		h.adminCall(http.MethodPost, base, map[string]any{"stage": "diarization", "model": model, "expected_current": expected, "confirm": confirm}, want)
	}
	pd := preview("diarization", "bench-diar-act", http.StatusOK)
	if pd.Current == "" || pd.WorkersReady != 0 {
		t.Fatalf("the current diarization model is a catalogue id and no Worker holds the new one yet: %+v", pd)
	}
	diar("bench-diar-act", pd.Current, true, http.StatusConflict) // a model no Worker holds is never activated (voice_model_not_ready)
	h.reportModel("bench-diar-act", "ready", "", 1, 1, http.StatusNoContent)
	if preview("diarization", "bench-diar-act", http.StatusOK).WorkersReady != 1 {
		t.Fatal("the Worker that holds the model is counted")
	}
	diar("bench-diar-act", pd.Current, true, http.StatusOK)
	var repo string
	if err := pool.QueryRow(ctx, `SELECT v.config->>'model' FROM voice_pipeline_active a JOIN voice_pipeline_versions v ON v.version=a.version WHERE a.id=true`).Scan(&repo); err != nil || repo != "fixture/bench-diar-act" {
		t.Fatalf("the voice pipeline runs the new model: %q %v", repo, err)
	}
	diar("not-in-the-catalogue", "x", true, http.StatusUnprocessableEntity)
}
