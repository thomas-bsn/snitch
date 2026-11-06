//go:build integration

package httpapi

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// A folder named after a game gets that game's cover, fetched once; qualifiers, scene tags and editions do not
// hide an obvious game, and a system folder is never guessed into one.
func testFolderLogo(t *testing.T, pool *pgxpool.Pool) {
	var searches atomic.Int32
	var mu sync.Mutex
	var terms []string
	steam := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/storesearch/":
			searches.Add(1)
			term := r.URL.Query().Get("term")
			mu.Lock()
			terms = append(terms, term)
			mu.Unlock()
			switch term {
			case "among us":
				_, _ = w.Write([]byte(`{"items":[{"type":"app","name":"The Wolf Among Us","id":1},{"type":"app","name":"Among Us","id":945360},{"type":"app","name":"Among Us 3D","id":3168600}]}`))
			case "bread and fred":
				_, _ = w.Write([]byte(`{"items":[{"type":"app","name":"Bread & Fred","id":1607680,"metascore":"80"},{"type":"app","name":"Bread & Fred Demo","id":2},{"type":"app","name":"Bread & Fred Soundtrack","id":3}]}`))
			case "call of duty black ops 2":
				_, _ = w.Write([]byte(`{"items":[{"type":"app","name":"Call of Duty®: Black Ops II","id":202970,"metascore":"83"},{"type":"app","name":"Call of Duty®: Black Ops II - Apocalypse","id":4}]}`))
			case "earth defense force 4 1":
				_, _ = w.Write([]byte(`{"items":[{"type":"app","name":"EARTH DEFENSE FORCE 4.1 WINGDIVER THE SHOOTER","id":574200},{"type":"app","name":"EARTH DEFENSE FORCE 4.1 The Shadow of New Despair","id":410320,"metascore":"79"}]}`))
			default:
				_, _ = w.Write([]byte(`{"items":[]}`))
			}
		default:
			parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/steam/apps/"), "/")
			if len(parts) != 2 || parts[0] == "" || parts[1] != "library_600x900.jpg" {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "image/jpeg")
			_, _ = w.Write([]byte("\xff\xd8cover-" + parts[0]))
		}
	}))
	defer steam.Close()

	h := newVoiceHarness(t, pool)
	h.api.Config.FolderLogos, h.api.Config.SteamStoreURL, h.api.Config.SteamCDNURL = true, steam.URL, steam.URL
	get := func(name string, want int) []byte {
		return h.adminCall(http.MethodGet, "/api/library/folder-logo?name="+url.QueryEscape(name), nil, want)
	}
	lastTerm := func() string {
		mu.Lock()
		defer mu.Unlock()
		if len(terms) == 0 {
			return ""
		}
		return terms[len(terms)-1]
	}

	// The search uses the reduced name, and the exact title wins over its neighbours.
	if body := get("Among Us 3d Demo", http.StatusOK); string(body) != "\xff\xd8cover-945360" {
		t.Fatalf("the cover of the game is served: %q", body)
	}
	if lastTerm() != "among us" {
		t.Fatalf("the build qualifier is not sent to Steam: %q", lastTerm())
	}
	before := searches.Load()
	get("Among Us 3d Demo", http.StatusOK)
	get("among us", http.StatusOK) // same key: no second call
	if searches.Load() != before {
		t.Fatalf("a logo already kept is not asked again")
	}

	// A scene release name finds the game through its clean title.
	if body := get("Bread.and.fred-Goldberg", http.StatusOK); string(body) != "\xff\xd8cover-1607680" {
		t.Fatalf("a scene release name finds the game: %q", body)
	}
	if lastTerm() != "bread and fred" {
		t.Fatalf("the scene tag is not sent to Steam: %q", lastTerm())
	}

	// Roman numerals in the store title match the digit in the folder name.
	if body := get("Call of Duty Black Ops 2", http.StatusOK); string(body) != "\xff\xd8cover-202970" {
		t.Fatalf("a roman numeral matches its digit: %q", body)
	}

	// A subtitle is accepted, and the main game beats a spinoff the store ranks first.
	if body := get("Earth Defense Force 4.1", http.StatusOK); string(body) != "\xff\xd8cover-410320" {
		t.Fatalf("the main game beats the spinoff: %q", body)
	}

	// A system folder is never guessed into a game: it is not even sent to Steam.
	asked := searches.Load()
	get("Content", http.StatusNotFound)
	get("Clips", http.StatusNotFound)
	get("Base Profile", http.StatusNotFound)
	if searches.Load() != asked {
		t.Fatalf("a system folder is not sent to Steam")
	}

	// A name that is no game is refused, and remembered so that Steam is not asked at each visit.
	get("Zzz Unknown", http.StatusNotFound)
	asked = searches.Load()
	get("Zzz Unknown", http.StatusNotFound)
	if searches.Load() != asked {
		t.Fatalf("a miss is remembered")
	}
	get("1.", http.StatusNotFound) // too short once reduced: never sent out
	if searches.Load() != asked {
		t.Fatalf("a short name is not sent to Steam")
	}

	// An explicit override picks the app for a game the store search no longer indexes.
	h.api.Config.FolderLogoOverrides = map[string]int64{"Fall Guys": 1097150}
	asked = searches.Load()
	if body := get("Fall Guys", http.StatusOK); string(body) != "\xff\xd8cover-1097150" {
		t.Fatalf("the configured app is served: %q", body)
	}
	get("Fall Guys 3d Demo", http.StatusOK) // same folder, same key
	if searches.Load() != asked {
		t.Fatalf("an override never asks the store")
	}

	// Switched off, nothing is served and nothing leaves the instance.
	h.api.Config.FolderLogos = false
	get("Among Us", http.StatusNotFound)
}
