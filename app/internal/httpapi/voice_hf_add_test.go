//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"snitch/internal/auth"

	"github.com/jackc/pgx/v5/pgxpool"
)

type inspectionView struct {
	Repo              string   `json:"repo"`
	Revision          string   `json:"revision"`
	License           string   `json:"license"`
	Gated             bool     `json:"gated"`
	Engine            string   `json:"engine"`
	ProposedKind      *string  `json:"proposed_kind"`
	CompatibleKinds   []string `json:"compatible_kinds"`
	Supported         bool     `json:"supported"`
	UnsupportedReason string   `json:"unsupported_reason"`
	ExistingIDs       []string `json:"existing_ids"`
}

// A Super-administrator pastes a repository: Snitch reads its public metadata from a (fake) Hugging Face,
// proposes the type only when reliable, derives the engine itself, pins the commit and records the reference.
func testVoiceModelAddFromHF(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	ctx := context.Background()
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM voice_model_candidate WHERE model_id NOT IN ('pyannote-community-1','nvidia-sortformer-4spk-v2.1','pyannote-wespeaker-resnet34')`)
		_, _ = pool.Exec(ctx, `DELETE FROM voice_model_catalog WHERE id NOT IN ('pyannote-community-1','nvidia-sortformer-4spk-v2.1','pyannote-wespeaker-resnet34')`)
	})

	const commit = "0123456789abcdef0123456789abcdef01234567"
	repos := map[string]string{
		"nvidia/synthetic-sortformer": `{"id":"nvidia/synthetic-sortformer","sha":"%s","gated":false,"library_name":"nemo","tags":["nemo","speaker-diarization"],"siblings":[{"rfilename":"model.nemo"}],"cardData":{"license":"apache-2.0"}}`,
		"acme/unsure-pyannote":        `{"id":"acme/unsure-pyannote","sha":"%s","gated":"auto","library_name":"pyannote-audio","tags":["pyannote-audio"],"cardData":{"license":"mit"}}`,
		"acme/transformers-only":      `{"id":"acme/transformers-only","sha":"%s","library_name":"transformers","tags":["transformers","speaker-diarization"]}`,
		"acme/private-model":          "403",
		"acme/lying-sha":              `{"id":"acme/lying-sha","sha":"ffffffffffffffffffffffffffffffffffffffff","library_name":"nemo","tags":["nemo"]}`,
	}
	var hits atomic.Int32
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.Header.Get("Authorization") != "" {
			t.Error("a credential reached the fake Hugging Face")
		}
		p := strings.TrimPrefix(r.URL.Path, "/api/models/")
		rev := commit
		if before, after, ok := strings.Cut(p, "/revision/"); ok {
			p, rev = before, after
		}
		body, known := repos[p]
		switch {
		case !known:
			http.NotFound(w, r)
		case body == "403":
			w.WriteHeader(http.StatusForbidden)
		default:
			_, _ = fmt.Fprint(w, strings.ReplaceAll(body, "%s", rev))
		}
	}))
	t.Cleanup(fake.Close)
	h.api.hf = &hfInspector{host: strings.TrimPrefix(fake.URL, "http://"), scheme: "http", timeout: 2 * time.Second, maxBytes: 1 << 16, allowInternal: true}

	inspect := func(source string, want int) inspectionView {
		var v inspectionView
		raw := h.adminCall(http.MethodPost, "/api/admin/voice-models/inspection", map[string]any{"source": source}, want)
		if want == http.StatusOK {
			if err := json.Unmarshal(raw, &v); err != nil {
				t.Fatal(err)
			}
		}
		return v
	}
	add := func(body map[string]any, want int) catalogModel {
		var m catalogModel
		raw := h.adminCall(http.MethodPost, "/api/admin/voice-models", body, want)
		if want == http.StatusCreated {
			if err := json.Unmarshal(raw, &m); err != nil {
				t.Fatal(err)
			}
		}
		return m
	}

	// A known Sortformer repository: type proposed, NeMo engine derived, nothing typed by hand.
	known := inspect("nvidia/synthetic-sortformer", http.StatusOK)
	if known.ProposedKind == nil || *known.ProposedKind != "diarization" || known.Engine != "nemo" || !known.Supported || known.Revision != commit || known.License != "apache-2.0" || known.Gated {
		t.Fatalf("known repository: %+v", known)
	}
	if byURL := inspect("http://"+strings.TrimPrefix(fake.URL, "http://")+"/nvidia/synthetic-sortformer", http.StatusOK); byURL.Repo != known.Repo {
		t.Fatalf("a URL of the host gives the same repository: %+v", byURL)
	}
	// The user confirms the proposed type; the commit seen is the commit pinned; the engine is not an input.
	h.adminCall(http.MethodPost, "/api/admin/voice-models", map[string]any{"source": "nvidia/synthetic-sortformer", "revision": commit, "kind": "diarization", "engine": "pyannote"}, http.StatusBadRequest)
	created := add(map[string]any{"source": "nvidia/synthetic-sortformer", "revision": commit, "kind": "diarization"}, http.StatusCreated)
	if created.Kind != "diarization" || created.Engine != "nemo" || created.Revision != commit || created.Repo != "nvidia/synthetic-sortformer" || !created.Supported || !created.Enabled || created.License != "apache-2.0" {
		t.Fatalf("created: %+v", created)
	}
	if listed := h.voiceModels().model(t, created.ID); listed.Revision != commit || listed.Engine != "nemo" {
		t.Fatalf("listed: %+v", listed)
	}
	add(map[string]any{"source": "nvidia/synthetic-sortformer", "revision": commit, "kind": "diarization"}, http.StatusConflict)
	if again := inspect("nvidia/synthetic-sortformer", http.StatusOK); len(again.ExistingIDs) != 1 || again.ExistingIDs[0] != created.ID {
		t.Fatalf("an existing reference is reported: %+v", again)
	}
	h.adminCall(http.MethodPut, "/api/admin/voice-models/candidate", map[string]any{"model_id": created.ID}, http.StatusOK)
	h.adminCall(http.MethodDelete, "/api/admin/voice-models/candidate", nil, http.StatusNoContent)
	// A type the metadata declare reliably cannot be overridden.
	add(map[string]any{"source": "nvidia/synthetic-sortformer", "revision": commit, "kind": "fingerprint"}, http.StatusConflict)

	// Compatible, type not reliable: nothing is proposed, the user picks one of the three types.
	unsure := inspect("acme/unsure-pyannote", http.StatusOK)
	if unsure.ProposedKind != nil || unsure.Engine != "pyannote" || !unsure.Supported || !unsure.Gated || strings.Join(unsure.CompatibleKinds, ",") != "diarization,fingerprint" {
		t.Fatalf("unsure repository: %+v", unsure)
	}
	add(map[string]any{"source": "acme/unsure-pyannote", "revision": commit}, http.StatusBadRequest)
	add(map[string]any{"source": "acme/unsure-pyannote", "revision": commit, "kind": "banana"}, http.StatusBadRequest)
	picked := add(map[string]any{"source": "acme/unsure-pyannote", "revision": commit, "kind": "fingerprint"}, http.StatusCreated)
	if picked.Kind != "fingerprint" || picked.Engine != "pyannote" || !picked.Gated || !picked.Supported {
		t.Fatalf("picked: %+v", picked)
	}

	// Incompatible: explained, recorded as unsupported, never installable even with a type chosen.
	bad := inspect("acme/transformers-only", http.StatusOK)
	if bad.Supported || bad.UnsupportedReason == "" || bad.Engine != "" {
		t.Fatalf("incompatible repository: %+v", bad)
	}
	refused := add(map[string]any{"source": "acme/transformers-only", "revision": commit, "kind": "diarization"}, http.StatusCreated)
	if refused.Supported || refused.UnsupportedReason == "" || refused.Engine != "" {
		t.Fatalf("an unsupported reference is flagged: %+v", refused)
	}
	h.adminCall(http.MethodPut, "/api/admin/voice-models/candidate", map[string]any{"model_id": refused.ID}, http.StatusBadRequest)
	h.adminCall(http.MethodPost, "/api/admin/voice-models/"+refused.ID+"/activate", nil, http.StatusBadRequest)
	// A compatible engine that cannot run the chosen type is unsupported as well.
	mismatch := add(map[string]any{"source": "acme/unsure-pyannote", "revision": commit, "kind": "separation"}, http.StatusCreated)
	if mismatch.Supported || mismatch.Kind != "separation" {
		t.Fatalf("separation has no adapter yet: %+v", mismatch)
	}

	// Unknown repository, Hugging Face rights, misleading answer, bad input, foreign hosts.
	inspect("acme/does-not-exist", http.StatusNotFound)
	inspect("acme/private-model", http.StatusUnprocessableEntity)
	add(map[string]any{"source": "acme/lying-sha", "revision": commit, "kind": "diarization"}, http.StatusBadGateway)
	add(map[string]any{"source": "acme/unsure-pyannote", "revision": "main", "kind": "diarization"}, http.StatusBadRequest)
	before := hits.Load()
	for _, source := range []string{"", "gpt2", "https://evil.example/acme/unsure-pyannote", "http://127.0.0.1:1/acme/x", "https://huggingface.co@evil.example/a/b", "file:///etc/passwd"} {
		inspect(source, http.StatusBadRequest)
	}
	if hits.Load() != before {
		t.Fatalf("a refused source must not leave Snitch: %d requests", hits.Load()-before)
	}

	// Only a Super-administrator inspects and adds.
	if _, err := pool.Exec(ctx, `INSERT INTO users(username,password_hash,role,approved) VALUES('operator',$1,'admin',true)`, auth.Password("synthetic-password")); err != nil {
		t.Fatal(err)
	}
	jar, _ := cookiejar.New(nil)
	operator := &voiceHarness{t: t, api: h.api, pool: pool, ts: h.ts, admin: &http.Client{Jar: jar}}
	operator.adminCall(http.MethodPost, "/api/auth/login", map[string]string{"username": "operator", "password": "synthetic-password"}, http.StatusOK)
	var me struct {
		CSRF string `json:"csrf_token"`
	}
	if err := json.Unmarshal(operator.adminCall(http.MethodGet, "/api/auth/me", nil, http.StatusOK), &me); err != nil {
		t.Fatal(err)
	}
	operator.csrf = me.CSRF
	operator.adminCall(http.MethodPost, "/api/admin/voice-models/inspection", map[string]any{"source": "acme/unsure-pyannote"}, http.StatusForbidden)
	operator.adminCall(http.MethodPost, "/api/admin/voice-models", map[string]any{"source": "acme/unsure-pyannote", "revision": commit, "kind": "diarization"}, http.StatusForbidden)
}
