//go:build integration

package httpapi

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// A folder named after a game gets that game's cover, fetched once; any other name keeps the folder icon.
func testFolderLogo(t *testing.T, pool *pgxpool.Pool) {
	var searches atomic.Int32
	steam := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/storesearch/":
			searches.Add(1)
			term := r.URL.Query().Get("term")
			if term == "Among Us 3d Demo" || term == "Among Us" {
				_, _ = w.Write([]byte(`{"items":[{"type":"app","name":"The Wolf Among Us","id":1},{"type":"app","name":"Among Us","id":945360}]}`))
				return
			}
			_, _ = w.Write([]byte(`{"items":[{"type":"app","name":"Something else","id":2}]}`))
		case "/steam/apps/945360/library_600x900.jpg":
			w.Header().Set("Content-Type", "image/jpeg")
			_, _ = w.Write([]byte("\xff\xd8synthetic-cover"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer steam.Close()

	h := newVoiceHarness(t, pool)
	h.api.Config.FolderLogos, h.api.Config.SteamStoreURL, h.api.Config.SteamCDNURL = true, steam.URL, steam.URL
	get := func(name string, want int) []byte {
		return h.adminCall(http.MethodGet, "/api/library/folder-logo?name="+url.QueryEscape(name), nil, want)
	}

	// The title wins over its neighbours, and a build qualifier ("3d Demo") does not matter.
	if body := get("Among Us 3d Demo", http.StatusOK); string(body) != "\xff\xd8synthetic-cover" {
		t.Fatalf("the cover of the game is served: %q", body)
	}
	before := searches.Load()
	get("Among Us 3d Demo", http.StatusOK)
	get("among us", http.StatusOK) // same key: no second call
	if searches.Load() != before {
		t.Fatalf("a logo already kept is not asked again")
	}

	// A name that is no game is refused, and remembered so that Steam is not asked at each visit.
	get("Common", http.StatusNotFound)
	asked := searches.Load()
	get("Common", http.StatusNotFound)
	if searches.Load() != asked {
		t.Fatalf("a miss is remembered")
	}
	get("1.", http.StatusNotFound) // too short once reduced: never sent out
	if searches.Load() != asked {
		t.Fatalf("a short name is not sent to Steam")
	}

	// Switched off, nothing is served and nothing leaves the instance.
	h.api.Config.FolderLogos = false
	get("Among Us", http.StatusNotFound)
}
