//go:build integration

package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// The Super-administrator moves an existing catalogue model to an on-demand family, or back to its embedded
// contract, without deleting and adding it again (ADR 0056). The family must exist and cover the task of the
// model, and nothing changes while an order or an installation of the model is open.
func testVoiceCatalogFamilyEdition(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	ctx := t.Context()
	for _, row := range [][]string{
		{"family-edit-nemotron", "diarization", "nemo", "synthetic/nemotron"},
		{"family-edit-pyannote", "diarization", "pyannote", "synthetic/pyannote"},
		{"family-edit-titanet", "fingerprint", "nemo-titanet", "synthetic/titanet"},
	} {
		if _, err := pool.Exec(ctx, `INSERT INTO voice_model_catalog(id,kind,engine,repo,revision) VALUES($1,$2,$3,$4,'0123456789abcdef0123456789abcdef01234567')`, row[0], row[1], row[2], row[3]); err != nil {
			t.Fatal(err)
		}
	}
	const nemotron = "/api/admin/voice-models/family-edit-nemotron"
	set := func(path string, family any, want int) catalogModel {
		t.Helper()
		var m catalogModel
		raw := h.adminCall(http.MethodPatch, path, map[string]any{"family": family}, want)
		if want == http.StatusOK {
			if err := json.Unmarshal(raw, &m); err != nil {
				t.Fatal(err)
			}
		}
		return m
	}
	stored := func(id string) string {
		var family string
		if err := pool.QueryRow(ctx, `SELECT family FROM voice_model_catalog WHERE id=$1`, id).Scan(&family); err != nil {
			t.Fatal(err)
		}
		return family
	}

	// An existing NeMo diarization model moves to the family, then back, with its identity untouched.
	if m := set(nemotron, "nemo", http.StatusOK); m.Family != "nemo" || m.Repo != "synthetic/nemotron" || m.Engine != "nemo" {
		t.Fatalf("family set: %+v", m)
	}
	if stored("family-edit-nemotron") != "nemo" {
		t.Fatal("family not stored")
	}
	// A patch that does not name the family leaves it alone.
	h.adminCall(http.MethodPatch, nemotron, map[string]any{"label": "Nemotron"}, http.StatusOK)
	if stored("family-edit-nemotron") != "nemo" {
		t.Fatal("an unrelated edit changed the family")
	}
	if m := set(nemotron, "", http.StatusOK); m.Family != "" || stored("family-edit-nemotron") != "" {
		t.Fatalf("family cleared: %+v", m)
	}

	// Refusals write nothing: unknown family, a family that does not cover the task, an embedded adapter.
	set(nemotron, "does-not-exist", http.StatusBadRequest)
	set(nemotron, "wespeaker", http.StatusBadRequest)
	set("/api/admin/voice-models/family-edit-pyannote", "nemo", http.StatusBadRequest)
	set("/api/admin/voice-models/family-edit-titanet", 12, http.StatusBadRequest)
	if stored("family-edit-nemotron") != "" || stored("family-edit-pyannote") != "" {
		t.Fatal("a refusal changed a family")
	}
	set("/api/admin/voice-models/someone-else", "nemo", http.StatusNotFound)
	var before catalogModel
	if err := json.Unmarshal(h.adminCall(http.MethodPatch, "/api/admin/voice-models/family-edit-titanet", map[string]any{"label": "TitaNet"}, http.StatusOK), &before); err != nil || !before.SignedRelease {
		t.Fatalf("a signed-only model without family needs a signed release: %+v %v", before, err)
	}
	if m := set("/api/admin/voice-models/family-edit-titanet", "speechbrain", http.StatusOK); m.Family != "speechbrain" || m.SignedRelease {
		t.Fatalf("a fingerprint model of another family: %+v", m)
	}

	// An open order, or an installation in progress, keeps the model where it is.
	worker := h.fixtureWorkerID()
	if _, err := pool.Exec(ctx, `INSERT INTO worker_model_orders(id,worker_id,kind,action,model_id,revision,idempotency_key,author_name,status)
		VALUES('ord_family_edit_open_0001',$1,'diarization','install','family-edit-nemotron','0123456789abcdef0123456789abcdef01234567','family-edit-open','synthetic','pending')`, worker); err != nil {
		t.Fatal(err)
	}
	set(nemotron, "nemo", http.StatusConflict)
	if _, err := pool.Exec(ctx, `UPDATE worker_model_orders SET status='done'`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO worker_voice_models(worker_id,model_id,state) VALUES($1,'family-edit-nemotron','downloading')`, worker); err != nil {
		t.Fatal(err)
	}
	set(nemotron, "nemo", http.StatusConflict)
	if _, err := pool.Exec(ctx, `UPDATE worker_voice_models SET state='ready'`); err != nil {
		t.Fatal(err)
	}
	if m := set(nemotron, "nemo", http.StatusOK); m.Family != "nemo" {
		t.Fatalf("settled model: %+v", m)
	}
}
