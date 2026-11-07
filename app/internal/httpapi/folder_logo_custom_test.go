//go:build integration

package httpapi

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// testFolderLogoCustom covers #323: a folder logo chosen by an administrator, its four-image history, the
// automatic fallback, restore as a new version, and the Dossier privé rule. The images are synthetic 2x2
// PNGs; no real media is read.
func testFolderLogoCustom(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarnessClips(t, pool, 1)
	root := t.TempDir()
	for _, dir := range []string{"league/2026", "secret"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	h.adminCall(http.MethodPut, "/api/admin/video-root", map[string]string{"path": root}, http.StatusOK)
	var sourceID int64
	if err := pool.QueryRow(t.Context(), `SELECT id FROM clip_sources WHERE managed_root`).Scan(&sourceID); err != nil {
		t.Fatal(err)
	}
	h.adminCall(http.MethodPut, "/api/admin/library/folders/privacy", map[string]any{"source_id": sourceID, "folder": "secret", "private": true}, http.StatusOK)
	logoAdmin := h.newUser("logo-admin")
	if _, err := pool.Exec(t.Context(), `UPDATE users SET role='admin' WHERE username='logo-admin'`); err != nil {
		t.Fatal(err)
	}
	member := h.newUser("logo-member")

	makePNG := func(red, green, blue uint8) []byte {
		t.Helper()
		var out bytes.Buffer
		img := image.NewRGBA(image.Rect(0, 0, 2, 2))
		for y := 0; y < 2; y++ {
			for x := 0; x < 2; x++ {
				img.Set(x, y, color.RGBA{red, green, blue, 255})
			}
		}
		if err := png.Encode(&out, img); err != nil {
			t.Fatal(err)
		}
		return out.Bytes()
	}
	rawStatus := func(u *voiceUser, method, path string, body []byte, contentType string) (int, []byte) {
		t.Helper()
		req, err := http.NewRequest(method, h.ts.URL+path, bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Origin", h.ts.URL)
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		if u.csrf != "" {
			req.Header.Set("X-CSRF-Token", u.csrf)
		}
		resp, err := u.client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		data, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, data
	}
	raw := func(u *voiceUser, method, path string, body []byte, contentType string, want int) []byte {
		t.Helper()
		status, data := rawStatus(u, method, path, body, contentType)
		if status != want {
			t.Fatalf("%s %s as %s: got %d want %d: %s", method, path, u.name, status, want, data)
		}
		return data
	}
	logoVersion := func(u *voiceUser, path string) int64 {
		t.Helper()
		var page struct {
			Items []struct {
				Path    string `json:"path"`
				Version int64  `json:"logo_version"`
			} `json:"items"`
		}
		if err := json.Unmarshal(h.userCall(u, http.MethodGet, "/api/library/explorer?path=", nil, http.StatusOK), &page); err != nil {
			t.Fatal(err)
		}
		for _, item := range page.Items {
			if item.Path == path {
				return item.Version
			}
		}
		return 0
	}
	history := func(u *voiceUser, path string) folderLogoState {
		t.Helper()
		var state folderLogoState
		if err := json.Unmarshal(raw(u, http.MethodGet, "/api/library/folder-logo/history?path="+path, nil, "", http.StatusOK), &state); err != nil {
			t.Fatal(err)
		}
		return state
	}

	first := makePNG(255, 0, 0)
	// A regular member never changes a logo.
	raw(member, http.MethodPut, "/api/library/folder-logo?path=league", first, "image/png", http.StatusForbidden)
	// The administrator uploads the first custom logo; the custom image is served before any Steam lookup
	// (FOLDER_LOGOS is off here, which proves the custom logo stands alone).
	state := folderLogoState{}
	if err := json.Unmarshal(raw(logoAdmin, http.MethodPut, "/api/library/folder-logo?path=league", first, "image/png", http.StatusOK), &state); err != nil {
		t.Fatal(err)
	}
	if !state.Custom || len(state.Versions) != 1 || state.Versions[0].Version != 1 {
		t.Fatalf("first upload state: %+v", state)
	}
	if got := raw(member, http.MethodGet, "/api/library/folder-logo?name=league&path=league", nil, "", http.StatusOK); !bytes.Equal(got, first) {
		t.Fatalf("the custom logo is served to any signed-in user")
	}
	if got := raw(logoAdmin, http.MethodGet, "/api/library/folder-logo?name=league&path=league&version=1", nil, "", http.StatusOK); !bytes.Equal(got, first) {
		t.Fatalf("a version image is served")
	}
	if got := logoVersion(member, "league"); got != 1 {
		t.Fatalf("the explorer carries the active logo version: %d", got)
	}
	raw(logoAdmin, http.MethodGet, "/api/library/folder-logo?name=league&path=league&version=99", nil, "", http.StatusNotFound)

	// A Dossier privé accepts a logo from the Super-administrateur only, and its image stays hidden from
	// everyone else even through the direct endpoint.
	raw(logoAdmin, http.MethodPut, "/api/library/folder-logo?path=secret", makePNG(0, 255, 0), "image/png", http.StatusForbidden)
	raw(h.adminUser(), http.MethodPut, "/api/library/folder-logo?path=secret", makePNG(0, 255, 0), "image/png", http.StatusOK)
	raw(member, http.MethodGet, "/api/library/folder-logo?name=secret&path=secret", nil, "", http.StatusForbidden)

	// The last four images are kept: five uploads leave versions 5,4,3,2.
	for i := 2; i <= 5; i++ {
		raw(logoAdmin, http.MethodPut, "/api/library/folder-logo?path=league", makePNG(uint8(i*20), 0, 0), "image/png", http.StatusOK)
	}
	state = history(logoAdmin, "league")
	if !state.Custom || len(state.Versions) != 4 || state.Versions[0].Version != 5 || state.Versions[3].Version != 2 {
		t.Fatalf("history is trimmed to the last four: %+v", state)
	}

	// Restoring version 4 copies it as version 6, exactly as Discord does.
	restored := folderLogoState{}
	if err := json.Unmarshal(raw(logoAdmin, http.MethodPost, "/api/library/folder-logo/history", []byte(`{"path":"league","version":4}`), "application/json", http.StatusOK), &restored); err != nil {
		t.Fatal(err)
	}
	if !restored.Custom || restored.Versions[0].Version != 6 {
		t.Fatalf("restoring creates a new version: %+v", restored)
	}
	if got := raw(logoAdmin, http.MethodGet, "/api/library/folder-logo?name=league&path=league", nil, "", http.StatusOK); !bytes.Equal(got, makePNG(80, 0, 0)) {
		t.Fatalf("the restored image is the current one")
	}
	if got := logoVersion(member, "league"); got != 6 {
		t.Fatalf("the restored version reaches the explorer: %d", got)
	}
	raw(logoAdmin, http.MethodPost, "/api/library/folder-logo/history", []byte(`{"path":"league","version":99}`), "application/json", http.StatusNotFound)

	// The automatic logo comes back, without losing the history.
	raw(logoAdmin, http.MethodDelete, "/api/library/folder-logo?path=league", nil, "", http.StatusNoContent)
	state = history(logoAdmin, "league")
	if state.Custom || len(state.Versions) != 4 {
		t.Fatalf("automatic logo keeps the history: %+v", state)
	}
	raw(logoAdmin, http.MethodGet, "/api/library/folder-logo?name=league&path=league", nil, "", http.StatusNotFound)
	if got := logoVersion(member, "league"); got != 0 {
		t.Fatalf("the automatic logo removes the version from the explorer: %d", got)
	}

	// A new upload after the reset is version 7 and makes the custom logo current again.
	state = folderLogoState{}
	if err := json.Unmarshal(raw(logoAdmin, http.MethodPut, "/api/library/folder-logo?path=league", first, "image/png", http.StatusOK), &state); err != nil {
		t.Fatal(err)
	}
	if !state.Custom || state.Versions[0].Version != 7 {
		t.Fatalf("upload after reset: %+v", state)
	}
	if got := logoVersion(member, "league"); got != 7 {
		t.Fatalf("upload after reset reaches the explorer: %d", got)
	}

	// Invalid images, oversized bodies, missing folders and video paths are refused.
	raw(logoAdmin, http.MethodPut, "/api/library/folder-logo?path=league", []byte("not an image"), "image/png", http.StatusBadRequest)
	if status, _ := rawStatus(logoAdmin, http.MethodPut, "/api/library/folder-logo?path=league", make([]byte, folderLogoMaxBytes+1), "image/png"); status != http.StatusRequestEntityTooLarge {
		t.Fatalf("an oversized logo: %d", status)
	}
	raw(logoAdmin, http.MethodPut, "/api/library/folder-logo?path=missing", first, "image/png", http.StatusUnprocessableEntity)
	raw(logoAdmin, http.MethodPut, "/api/library/folder-logo?path=league%2F2026%2Fx.mp4", first, "image/png", http.StatusUnprocessableEntity)
	raw(logoAdmin, http.MethodPut, "/api/library/folder-logo?path=", first, "image/png", http.StatusUnprocessableEntity)
	raw(member, http.MethodGet, "/api/library/folder-logo/history?path=league", nil, "", http.StatusForbidden)
}
