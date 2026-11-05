//go:build integration

package httpapi

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// syntheticHuggingFace is the fake Hugging Face of the browser journey: three synthetic repositories (known
// Sortformer, compatible with an unreliable type, incompatible).
func syntheticHuggingFace(t *testing.T) *hfInspector {
	t.Helper()
	const commit = "cd03eee90fbec18297ac31b8c21546e596b7f71c"
	repos := map[string]string{
		"nvidia/synthetic-sortformer": `{"id":"nvidia/synthetic-sortformer","sha":"` + commit + `","gated":false,"library_name":"nemo","tags":["nemo","speaker-diarization"],"siblings":[{"rfilename":"model.nemo"}],"cardData":{"license":"apache-2.0"}}`,
		"acme/unsure-pyannote":        `{"id":"acme/unsure-pyannote","sha":"` + commit + `","gated":"auto","library_name":"pyannote-audio","tags":["pyannote-audio"],"cardData":{"license":"mit"}}`,
		"acme/transformers-only":      `{"id":"acme/transformers-only","sha":"` + commit + `","library_name":"transformers","tags":["transformers","speaker-diarization"]}`,
	}
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		repo, _, _ := strings.Cut(strings.TrimPrefix(r.URL.Path, "/api/models/"), "/revision/")
		body, ok := repos[repo]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = fmt.Fprint(w, body)
	}))
	t.Cleanup(fake.Close)
	return &hfInspector{host: strings.TrimPrefix(fake.URL, "http://"), scheme: "http", timeout: 2 * time.Second, maxBytes: 1 << 16, allowInternal: true}
}
