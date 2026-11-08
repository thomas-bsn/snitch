//go:build integration

package httpapi

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func testVoiceEngineRegistry(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	withSecondCatalogModel(t, pool)
	for _, engine := range []string{"nemo-titanet", "speechbrain-ecapa"} {
		if _, err := pool.Exec(t.Context(), `INSERT INTO voice_model_catalog(id,kind,engine,repo,revision) VALUES($1,'fingerprint',$1,$2,$3)`, engine, "synthetic/"+engine, "0123456789abcdef0123456789abcdef01234567"); err != nil {
			t.Fatalf("catalogue rejects approved engine %s: %v", engine, err)
		}
	}

	pub, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	encode := base64.StdEncoding.EncodeToString
	keys, _ := json.Marshal(map[string]string{"publisher": encode(pub)})
	t.Setenv(engineTrustEnv, string(keys))
	m := testEngineManifest()
	m.Engine, m.Kind, m.Architecture = "pyannote", "diarization", "pyannote"
	envelope := func() map[string]string {
		raw, _ := json.Marshal(m)
		return map[string]string{"key_id": "publisher", "payload": encode(raw), "signature": encode(ed25519.Sign(private, raw))}
	}
	input := envelope()
	invalid := envelope()
	invalid["signature"] = "invalid"
	h.adminCall(http.MethodPost, "/api/admin/voice-engines", invalid, http.StatusBadRequest)
	member := h.newUser("engine-member")
	h.userCall(member, http.MethodPost, "/api/admin/voice-engines", input, http.StatusForbidden)
	var release engineRelease
	if err := json.Unmarshal(h.adminCall(http.MethodPost, "/api/admin/voice-engines", input, http.StatusCreated), &release); err != nil {
		t.Fatal(err)
	}
	h.adminCall(http.MethodPost, "/api/admin/voice-engines", input, http.StatusConflict)
	var key, payload, signature string
	if err := pool.QueryRow(t.Context(), `SELECT key_id,signed_payload,signature FROM voice_engine_registry WHERE digest=$1`, release.Digest).Scan(&key, &payload, &signature); err != nil {
		t.Fatal(err)
	}
	if key != "publisher" || payload != input["payload"] || signature != input["signature"] {
		t.Fatal("signature provenance lost")
	}
	var registry struct {
		Engines []engineRelease `json:"engines"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/admin/voice-engines", nil, http.StatusOK), &registry); err != nil || len(registry.Engines) != 1 {
		t.Fatalf("registry: %+v %v", registry, err)
	}
	body := map[string]any{"model_id": secondID, "idempotency_key": "runtime-order-0001", "runtime_digest": release.Digest}
	var order modelOrder
	if err := json.Unmarshal(h.adminCall(http.MethodPost, "/api/admin/workers/"+h.fixtureWorkerID()+"/voice-model-orders", body, http.StatusCreated), &order); err != nil {
		t.Fatal(err)
	}
	var pinned engineManifest
	if err := json.Unmarshal(order.Runtime, &pinned); err != nil || pinned.Version != m.Version {
		t.Fatalf("runtime not pinned: %s %v", order.Runtime, err)
	}
	e := &engineOrderHarness{voiceHarness: h}
	engines := e.delivered()
	if len(engines) != 1 {
		t.Fatalf("dependent engine: %+v", engines)
	}
	e.ack(engines[0].OrderID)
	e.report(engines[0].OrderID, release.Digest, "ready", "", http.StatusNoContent)
	m.Version = "2.0.0"
	h.adminCall(http.MethodPost, "/api/admin/voice-engines", envelope(), http.StatusCreated)
	if got := h.workerOrders(); len(got) != 1 || string(got[0].Runtime) != string(order.Runtime) {
		t.Fatalf("new release changed queued order: %+v", got)
	}
	// Acknowledged orders may drain, but revocation cancels unacknowledged deliveries.
	other := h.order(h.otherWorker(), secondID, "runtime-order-0002", http.StatusCreated)
	h.ack(order.OrderID, true, "", http.StatusNoContent)
	h.adminCall(http.MethodDelete, "/api/admin/voice-engines/"+release.Digest, nil, http.StatusOK)
	if got := h.workerOrders(); len(got) != 1 || got[0].OrderID != order.OrderID {
		t.Fatalf("acknowledged order lost: %+v", got)
	}
	h.adminCall(http.MethodPost, "/api/admin/voice-engines", input, http.StatusConflict)
	body["idempotency_key"] = "runtime-revoked-0003"
	h.adminCall(http.MethodPost, "/api/admin/workers/"+h.otherWorker()+"/voice-model-orders", body, http.StatusConflict)
	// The second release has a pending delivery: revoke it and prove cancellation is persisted.
	var digest string
	if err := pool.QueryRow(t.Context(), `SELECT digest FROM voice_engine_registry WHERE version='2.0.0'`).Scan(&digest); err != nil {
		t.Fatal(err)
	}
	h.adminCall(http.MethodDelete, "/api/admin/voice-engines/"+digest, nil, http.StatusOK)
	var state string
	if err := pool.QueryRow(t.Context(), `SELECT status FROM worker_model_orders WHERE id=$1`, other.OrderID).Scan(&state); err != nil || state != "cancelled" {
		t.Fatalf("pending order not cancelled: %s %v", state, err)
	}
}
