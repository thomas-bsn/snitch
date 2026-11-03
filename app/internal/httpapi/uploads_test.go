//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"snitch/internal/config"
	"snitch/internal/setup"
)

func testResumableUploads(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	uploadRoot := t.TempDir()
	probe := syntheticProbeFixture(t)
	code, err := setup.Issue(context.Background(), pool)
	if err != nil {
		t.Fatal(err)
	}
	api := New(pool, config.Config{
		MasterKey: bytes.Repeat([]byte{1}, 32),
		ProbePath: probe, UploadRoot: uploadRoot, UploadMaxBytes: 128, UploadChunkBytes: 16 * 1024,
		UploadSessionTTL: time.Hour,
	}, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	if err := api.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer api.Close()
	ts := httptest.NewUnstartedServer(api.Handler())
	api.Config.PublicURL = "http://" + ts.Listener.Addr().String()
	ts.Start()
	defer ts.Close()
	jar, _ := cookiejar.New(nil)
	admin := &http.Client{Jar: jar}
	request := func(client *http.Client, method, path string, body []byte, contentType, contentRange, csrf string, want int) []byte {
		t.Helper()
		req, err := http.NewRequest(method, ts.URL+path, bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Origin", ts.URL)
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		if contentRange != "" {
			req.Header.Set("Content-Range", contentRange)
		}
		if csrf != "" {
			req.Header.Set("X-CSRF-Token", csrf)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != want {
			t.Fatalf("%s %s: got %d want %d: %s", method, path, resp.StatusCode, want, data)
		}
		return data
	}
	request(http.DefaultClient, http.MethodPost, "/api/uploads", []byte(`{"filename":"clip.mp4","size_bytes":25}`), "application/json", "", "", http.StatusUnauthorized)
	request(admin, http.MethodPost, "/api/setup", []byte(`{"code":"`+code+`","username":"administrator","password":"synthetic-password"}`), "application/json", "", "", http.StatusCreated)
	var session struct {
		CSRF string `json:"csrf_token"`
	}
	if err := json.Unmarshal(request(admin, http.MethodGet, "/api/auth/me", nil, "", "", "", http.StatusOK), &session); err != nil {
		t.Fatal(err)
	}
	content := []byte("synthetic-video-upload-123")
	createBody, _ := json.Marshal(map[string]any{"filename": "../../synthetic video.mp4", "size_bytes": len(content), "visibility": "shared"})
	var created struct {
		ID        string `json:"id"`
		Name      string `json:"original_name"`
		Offset    int64  `json:"offset_bytes"`
		ChunkSize int64  `json:"chunk_size_bytes"`
	}
	if err := json.Unmarshal(request(admin, http.MethodPost, "/api/uploads", createBody, "application/json", "", session.CSRF, http.StatusCreated), &created); err != nil {
		t.Fatal(err)
	}
	if created.ID == "" || created.Name != "synthetic video.mp4" || created.Offset != 0 || created.ChunkSize != 16*1024 {
		t.Fatalf("unsafe or incomplete upload session response: %+v", created)
	}
	path := "/api/uploads/" + created.ID + "/chunks"
	first := content[:8]
	request(admin, http.MethodPut, path, first, "application/octet-stream", "bytes 0-7/26", session.CSRF, http.StatusOK)
	// A retry after a lost response is idempotent when it repeats exactly the accepted bytes.
	request(admin, http.MethodPut, path, first, "application/octet-stream", "bytes 0-7/26", session.CSRF, http.StatusOK)
	request(admin, http.MethodPut, path, content[8:16], "application/octet-stream", "bytes 8-15/26", session.CSRF, http.StatusOK)
	request(admin, http.MethodPut, path, content[20:], "application/octet-stream", "bytes 20-25/26", session.CSRF, http.StatusConflict)
	if _, err := json.Marshal(map[string]string{"filename": "too-large.mp4"}); err != nil {
		t.Fatal(err)
	}
	var uploads struct {
		Items []struct {
			ID     string `json:"id"`
			Offset int64  `json:"offset_bytes"`
			Size   int64  `json:"size_bytes"`
			Status string `json:"status"`
			Name   string `json:"original_name"`
		} `json:"items"`
	}
	if err := json.Unmarshal(request(admin, http.MethodGet, "/api/uploads", nil, "", "", "", http.StatusOK), &uploads); err != nil {
		t.Fatal(err)
	}
	if len(uploads.Items) != 1 || uploads.Items[0].ID != created.ID || uploads.Items[0].Offset != 16 || uploads.Items[0].Size != int64(len(content)) || uploads.Items[0].Status != "uploading" {
		t.Fatalf("upload cannot resume from server offset: %+v", uploads.Items)
	}
	// A fresh client with the same authenticated session can resume after reconnect.
	resumed, _ := cookiejar.New(nil)
	serverURL, _ := url.Parse(ts.URL)
	for _, cookie := range jar.Cookies(serverURL) {
		resumed.SetCookies(serverURL, []*http.Cookie{cookie})
	}
	resumedClient := &http.Client{Jar: resumed}
	request(resumedClient, http.MethodPut, path, content[16:], "application/octet-stream", "bytes 16-25/26", session.CSRF, http.StatusOK)
	request(resumedClient, http.MethodPost, "/api/uploads/"+created.ID+"/complete", []byte(`{}`), "application/json", "", session.CSRF, http.StatusCreated)
	var finalized struct {
		Status string `json:"status"`
		ClipID int64  `json:"clip_id"`
		SHA256 string `json:"sha256"`
	}
	// Completion is idempotent and returns the same clip identity.
	if err := json.Unmarshal(request(admin, http.MethodPost, "/api/uploads/"+created.ID+"/complete", []byte(`{}`), "application/json", "", session.CSRF, http.StatusCreated), &finalized); err != nil {
		t.Fatal(err)
	}
	wantHash := sha256.Sum256(content)
	if finalized.Status != "complete" || finalized.ClipID <= 0 || finalized.SHA256 != hex.EncodeToString(wantHash[:]) {
		t.Fatalf("upload was not finalized as the expected clip: %+v", finalized)
	}
	files, err := os.ReadDir(uploadRoot)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if strings.Contains(file.Name(), "synthetic video") || strings.Contains(file.Name(), "..") || filepath.IsAbs(file.Name()) {
			t.Fatalf("client filename was used as a storage path: %q", file.Name())
		}
	}
	if len(files) != 1 {
		t.Fatalf("completed media not retained under generated storage identity: %v", files)
	}

	tooLarge, _ := json.Marshal(map[string]any{"filename": "large.mp4", "size_bytes": 129})
	request(admin, http.MethodPost, "/api/uploads", tooLarge, "application/json", "", session.CSRF, http.StatusRequestEntityTooLarge)
	var cancelled struct {
		ID string `json:"id"`
	}
	cancelBody, _ := json.Marshal(map[string]any{"filename": "cancel.mp4", "size_bytes": 4, "visibility": "private"})
	if err := json.Unmarshal(request(admin, http.MethodPost, "/api/uploads", cancelBody, "application/json", "", session.CSRF, http.StatusCreated), &cancelled); err != nil {
		t.Fatal(err)
	}
	request(admin, http.MethodDelete, "/api/uploads/"+cancelled.ID, nil, "", "", session.CSRF, http.StatusNoContent)
	if _, err := os.Stat(filepath.Join(uploadRoot, cancelled.ID+".part")); !os.IsNotExist(err) {
		t.Fatalf("cancelled partial upload was not removed: %v", err)
	}
	// Existing shared content may be deduplicated, but a private upload cannot reveal or inherit it.
	duplicateBody, _ := json.Marshal(map[string]any{"filename": "private-duplicate.mp4", "size_bytes": len(content), "visibility": "private"})
	var duplicate struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(request(admin, http.MethodPost, "/api/uploads", duplicateBody, "application/json", "", session.CSRF, http.StatusCreated), &duplicate); err != nil {
		t.Fatal(err)
	}
	request(admin, http.MethodPut, "/api/uploads/"+duplicate.ID+"/chunks", content, "application/octet-stream", "bytes 0-25/26", session.CSRF, http.StatusOK)
	request(admin, http.MethodPost, "/api/uploads/"+duplicate.ID+"/complete", []byte(`{}`), "application/json", "", session.CSRF, http.StatusConflict)
	// A matching shared upload returns the existing shared clip identity without duplicating it.
	sharedDuplicate, _ := json.Marshal(map[string]any{"filename": "shared-duplicate.mp4", "size_bytes": len(content), "visibility": "shared"})
	var duplicateShared struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(request(admin, http.MethodPost, "/api/uploads", sharedDuplicate, "application/json", "", session.CSRF, http.StatusCreated), &duplicateShared); err != nil {
		t.Fatal(err)
	}
	request(admin, http.MethodPut, "/api/uploads/"+duplicateShared.ID+"/chunks", content, "application/octet-stream", "bytes 0-25/26", session.CSRF, http.StatusOK)
	var sharedResult struct {
		ClipID int64  `json:"clip_id"`
		SHA256 string `json:"sha256"`
	}
	if err := json.Unmarshal(request(admin, http.MethodPost, "/api/uploads/"+duplicateShared.ID+"/complete", []byte(`{}`), "application/json", "", session.CSRF, http.StatusCreated), &sharedResult); err != nil {
		t.Fatal(err)
	}
	if sharedResult.ClipID != finalized.ClipID || sharedResult.SHA256 != finalized.SHA256 {
		t.Fatalf("duplicate did not resolve to existing shared clip: %+v", sharedResult)
	}
	if _, err := pool.Exec(context.Background(), `UPDATE clips SET visibility='private' WHERE id=$1`, finalized.ClipID); err != nil {
		t.Fatal(err)
	}
	// A shared request cannot silently expose or promote an existing private clip.
	var privateDuplicate struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(request(admin, http.MethodPost, "/api/uploads", sharedDuplicate, "application/json", "", session.CSRF, http.StatusCreated), &privateDuplicate); err != nil {
		t.Fatal(err)
	}
	request(admin, http.MethodPut, "/api/uploads/"+privateDuplicate.ID+"/chunks", content, "application/octet-stream", "bytes 0-25/26", session.CSRF, http.StatusOK)
	request(admin, http.MethodPost, "/api/uploads/"+privateDuplicate.ID+"/complete", []byte(`{}`), "application/json", "", session.CSRF, http.StatusConflict)
	t.Run("managed root manual scan and named uploads", func(t *testing.T) {
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, "existing.mp4"), []byte("synthetic-existing-video"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(t.TempDir(), filepath.Join(root, "outside")); err != nil {
			t.Fatal(err)
		}
		call := func(method, path string, body any, want int) []byte {
			data, _ := json.Marshal(body)
			return request(admin, method, path, data, "application/json", "", session.CSRF, want)
		}
		call("PUT", "/api/admin/video-root", map[string]string{"path": root}, 200)
		managed, err := api.Library.ManagedRoot(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		var count int
		pool.QueryRow(context.Background(), `SELECT count(*) FROM source_files WHERE source_id=$1`, managed.ID).Scan(&count)
		if count != 0 {
			t.Fatal("setting a root discovered files automatically")
		}
		call("GET", "/api/admin/video-root/browse?path=..", nil, 422)
		call("GET", "/api/admin/video-root/browse?path=outside", nil, 422)
		call("POST", "/api/admin/video-root/folders", map[string]string{"parent": "", "name": "matches"}, 201)
		call("POST", "/api/admin/video-root/scan", map[string]any{"paths": []string{""}, "confirm": false}, 200)
		pool.QueryRow(context.Background(), `SELECT count(*) FROM source_files WHERE source_id=$1`, managed.ID).Scan(&count)
		if count != 0 {
			t.Fatal("preview queued a scan")
		}
		call("POST", "/api/admin/video-root/scan", map[string]any{"paths": []string{"existing.mp4"}, "confirm": true}, 200)
		if err := api.Library.ScanSource(context.Background(), managed.ID); err != nil {
			t.Fatal(err)
		}
		var clip *int64
		pool.QueryRow(context.Background(), `SELECT clip_id FROM source_files WHERE source_id=$1`, managed.ID).Scan(&clip)
		if clip != nil {
			t.Fatal("scan advanced without a Worker")
		}
		upload := func(content, version string, completion int) string {
			var item struct {
				ID string `json:"id"`
			}
			body := map[string]any{"filename": "game.mp4", "size_bytes": len(content), "destination": "matches", "overwrite_version": version}
			if err := json.Unmarshal(call("POST", "/api/uploads", body, 201), &item); err != nil {
				t.Fatal(err)
			}
			request(admin, "PUT", "/api/uploads/"+item.ID+"/chunks", []byte(content), "application/octet-stream", fmt.Sprintf("bytes 0-%d/%d", len(content)-1, len(content)), session.CSRF, 200)
			call("POST", "/api/uploads/"+item.ID+"/complete", map[string]any{}, completion)
			return item.ID
		}
		id := upload("synthetic-named-first", "", 201)
		got, err := os.ReadFile(filepath.Join(root, "matches", "game.mp4"))
		if err != nil || string(got) != "synthetic-named-first" {
			t.Fatalf("named upload: %s %v", got, err)
		}
		call("POST", "/api/uploads/"+id+"/complete", map[string]any{}, 201)
		var conflict struct {
			Version string `json:"version"`
		}
		json.Unmarshal(call("POST", "/api/uploads", map[string]any{"filename": "game.mp4", "size_bytes": 22, "destination": "matches"}, 409), &conflict)
		if conflict.Version == "" {
			t.Fatal("overwrite conflict has no version")
		}
		oldID := int64(0)
		pool.QueryRow(context.Background(), `SELECT clip_id FROM uploads WHERE id=$1`, id).Scan(&oldID)
		second := upload("synthetic-named-replacement", conflict.Version, 201)
		got, err = os.ReadFile(filepath.Join(root, "matches", "game.mp4"))
		if err != nil || string(got) != "synthetic-named-replacement" {
			t.Fatal("replacement missing", err)
		}
		pool.QueryRow(context.Background(), `SELECT count(*) FROM clips WHERE id=$1`, oldID).Scan(&count)
		if count != 0 {
			t.Fatal("replaced clip retained its results")
		}
		call("POST", "/api/uploads/"+second+"/complete", map[string]any{}, 201)
		call("POST", "/api/uploads", map[string]any{"filename": "game.mp4", "size_bytes": 22, "destination": "matches", "overwrite_version": conflict.Version}, 409)
		call("POST", "/api/uploads", map[string]any{"filename": "game.mp4", "size_bytes": 22, "destination": "../outside"}, 422)
	})

	// call and upload are shared by the publication scenarios below. They always
	// reconfigure the managed root first, so each scenario starts from a clean root.
	call := func(method, path string, body any, want int) []byte {
		data, _ := json.Marshal(body)
		return request(admin, method, path, data, "application/json", "", session.CSRF, want)
	}
	startUpload := func(name, folder string, content []byte, version string) string {
		t.Helper()
		var item struct {
			ID string `json:"id"`
		}
		body := map[string]any{"filename": name, "size_bytes": len(content), "destination": folder, "overwrite_version": version}
		if err := json.Unmarshal(call("POST", "/api/uploads", body, 201), &item); err != nil {
			t.Fatal(err)
		}
		request(admin, "PUT", "/api/uploads/"+item.ID+"/chunks", content, "application/octet-stream", fmt.Sprintf("bytes 0-%d/%d", len(content)-1, len(content)), session.CSRF, 200)
		return item.ID
	}
	finish := func(id string, want int) int64 {
		t.Helper()
		data := call("POST", "/api/uploads/"+id+"/complete", map[string]any{}, want)
		var done struct {
			ClipID int64 `json:"clip_id"`
		}
		if want == 201 {
			if err := json.Unmarshal(data, &done); err != nil {
				t.Fatal(err)
			}
		}
		return done.ClipID
	}
	uploadTo := func(name, folder string, content []byte, version string, want int) (string, int64) {
		t.Helper()
		id := startUpload(name, folder, content, version)
		return id, finish(id, want)
	}

	t.Run("publication recovers after an interruption", func(t *testing.T) {
		root := t.TempDir()
		if err := os.Mkdir(filepath.Join(root, "recovery"), 0755); err != nil {
			t.Fatal(err)
		}
		call("PUT", "/api/admin/video-root", map[string]string{"path": root}, 200)
		content := []byte("synthetic-recovery-content")
		id, _ := uploadTo("recover.mp4", "recovery", content, "", 201)
		destination := filepath.Join(root, "recovery", "recover.mp4")
		// Crash after the rename but before the durable commit: the installed
		// content is recognized by its hash, even though staging is already gone.
		if _, err := pool.Exec(context.Background(), `UPDATE upload_destinations SET published=false WHERE upload_id=$1`, id); err != nil {
			t.Fatal(err)
		}
		api.resumeUploadPublications(context.Background())
		var published bool
		if err := pool.QueryRow(context.Background(), `SELECT published FROM upload_destinations WHERE upload_id=$1`, id).Scan(&published); err != nil {
			t.Fatal(err)
		}
		if !published {
			t.Fatal("installed content was not recognized after an interrupted commit")
		}
		// Crash before the rename: the destination is gone and the staging copy
		// is still there. The resume loop must republish it.
		if err := os.Remove(destination); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(context.Background(), `UPDATE upload_destinations SET published=false WHERE upload_id=$1`, id); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(api.Uploads.MediaRoot(), id+".media"), content, 0600); err != nil {
			t.Fatal(err)
		}
		api.resumeUploadPublications(context.Background())
		got, err := os.ReadFile(destination)
		if err != nil || !bytes.Equal(got, content) {
			t.Fatalf("staged content was not republished: %q %v", got, err)
		}
	})

	t.Run("publication to a new path requests a scan", func(t *testing.T) {
		root := t.TempDir()
		if err := os.Mkdir(filepath.Join(root, "fresh"), 0755); err != nil {
			t.Fatal(err)
		}
		call("PUT", "/api/admin/video-root", map[string]string{"path": root}, 200)
		managed, err := api.Library.ManagedRoot(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		uploadTo("fresh.mp4", "fresh", []byte("synthetic-fresh-content"), "", 201)
		var requested bool
		if err := pool.QueryRow(context.Background(), `SELECT scan_requested FROM source_files WHERE source_id=$1 AND relative_path='fresh/fresh.mp4'`, managed.ID).Scan(&requested); err != nil {
			t.Fatal(err)
		}
		if !requested {
			t.Fatal("a publication to a new path must request the scan that indexes it and starts its analysis")
		}
	})

	t.Run("destination changed during a transfer is reported and cancellable", func(t *testing.T) {
		root := t.TempDir()
		if err := os.Mkdir(filepath.Join(root, "clash"), 0755); err != nil {
			t.Fatal(err)
		}
		call("PUT", "/api/admin/video-root", map[string]string{"path": root}, 200)
		content := []byte("synthetic-destination-race")
		id := startUpload("game.mp4", "clash", content, "")
		// Another process installs a different file before completion.
		if err := os.WriteFile(filepath.Join(root, "clash", "game.mp4"), []byte("someone-else"), 0600); err != nil {
			t.Fatal(err)
		}
		finish(id, 409)
		var list struct {
			Items []struct {
				ID               string `json:"id"`
				Status           string `json:"status"`
				Destination      string `json:"destination"`
				Publication      string `json:"publication"`
				PublicationError string `json:"publication_error"`
			} `json:"items"`
		}
		if err := json.Unmarshal(call("GET", "/api/uploads", nil, 200), &list); err != nil {
			t.Fatal(err)
		}
		found := false
		for _, item := range list.Items {
			if item.ID != id {
				continue
			}
			found = true
			if item.Status != "complete" || item.Destination != "clash/game.mp4" || item.Publication != "failed" || item.PublicationError != "destination_changed" {
				t.Fatalf("failed publication is not visible with its stable code: %+v", item)
			}
		}
		if !found {
			t.Fatal("failed publication disappeared from the upload list")
		}
		call("DELETE", "/api/uploads/"+id, nil, 204)
		if err := json.Unmarshal(call("GET", "/api/uploads", nil, 200), &list); err != nil {
			t.Fatal(err)
		}
		for _, item := range list.Items {
			if item.ID == id {
				t.Fatal("cancelled publication is still listed")
			}
		}
		got, err := os.ReadFile(filepath.Join(root, "clash", "game.mp4"))
		if err != nil || string(got) != "someone-else" {
			t.Fatalf("cancelling changed the destination file: %q %v", got, err)
		}
	})

	t.Run("publication permission failure can be retried", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("permission checks are not enforced for root")
		}
		root := t.TempDir()
		locked := filepath.Join(root, "locked")
		if err := os.Mkdir(locked, 0500); err != nil {
			t.Fatal(err)
		}
		call("PUT", "/api/admin/video-root", map[string]string{"path": root}, 200)
		content := []byte("synthetic-permission-content")
		id, _ := uploadTo("locked.mp4", "locked", content, "", 422)
		var list struct {
			Items []struct {
				ID               string `json:"id"`
				Publication      string `json:"publication"`
				PublicationError string `json:"publication_error"`
			} `json:"items"`
		}
		if err := json.Unmarshal(call("GET", "/api/uploads", nil, 200), &list); err != nil {
			t.Fatal(err)
		}
		found := false
		for _, item := range list.Items {
			if item.ID == id {
				found = true
				if item.Publication != "failed" || item.PublicationError != "video_root_permission" {
					t.Fatalf("permission failure was not recorded: %+v", item)
				}
			}
		}
		if !found {
			t.Fatal("permission failure is not listed")
		}
		if err := os.Chmod(locked, 0700); err != nil {
			t.Fatal(err)
		}
		// An explicit retry is the same idempotent completion call.
		call("POST", "/api/uploads/"+id+"/complete", map[string]any{}, 201)
		got, err := os.ReadFile(filepath.Join(locked, "locked.mp4"))
		if err != nil || !bytes.Equal(got, content) {
			t.Fatalf("retry did not publish the file: %q %v", got, err)
		}
	})

	t.Run("concurrent uploads to the same path do not overwrite silently", func(t *testing.T) {
		root := t.TempDir()
		if err := os.Mkdir(filepath.Join(root, "race"), 0755); err != nil {
			t.Fatal(err)
		}
		call("PUT", "/api/admin/video-root", map[string]string{"path": root}, 200)
		first := []byte("synthetic-race-first")
		second := []byte("synthetic-race-second")
		// Both intents are created while the destination is still absent.
		var one, two struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(call("POST", "/api/uploads", map[string]any{"filename": "race.mp4", "size_bytes": len(first), "destination": "race"}, 201), &one); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(call("POST", "/api/uploads", map[string]any{"filename": "race.mp4", "size_bytes": len(second), "destination": "race"}, 201), &two); err != nil {
			t.Fatal(err)
		}
		request(admin, "PUT", "/api/uploads/"+one.ID+"/chunks", first, "application/octet-stream", fmt.Sprintf("bytes 0-%d/%d", len(first)-1, len(first)), session.CSRF, 200)
		request(admin, "PUT", "/api/uploads/"+two.ID+"/chunks", second, "application/octet-stream", fmt.Sprintf("bytes 0-%d/%d", len(second)-1, len(second)), session.CSRF, 200)
		call("POST", "/api/uploads/"+one.ID+"/complete", map[string]any{}, 201)
		call("POST", "/api/uploads/"+two.ID+"/complete", map[string]any{}, 409)
		got, err := os.ReadFile(filepath.Join(root, "race", "race.mp4"))
		if err != nil || !bytes.Equal(got, first) {
			t.Fatalf("concurrent publication replaced the winner: %q %v", got, err)
		}
		var list struct {
			Items []struct {
				ID               string `json:"id"`
				PublicationError string `json:"publication_error"`
			} `json:"items"`
		}
		if err := json.Unmarshal(call("GET", "/api/uploads", nil, 200), &list); err != nil {
			t.Fatal(err)
		}
		for _, item := range list.Items {
			if item.ID == two.ID && item.PublicationError != "destination_changed" {
				t.Fatalf("losing publication is not reported: %+v", item)
			}
		}
		// The loser releases its intent so it no longer blocks a root change.
		call("DELETE", "/api/uploads/"+two.ID, nil, 204)
	})

	t.Run("distinct copies keep the results of the replaced content", func(t *testing.T) {
		root := t.TempDir()
		if err := os.Mkdir(filepath.Join(root, "copies"), 0755); err != nil {
			t.Fatal(err)
		}
		call("PUT", "/api/admin/video-root", map[string]string{"path": root}, 200)
		managed, err := api.Library.ManagedRoot(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		shared := []byte("synthetic-shared-copy")
		_, firstClip := uploadTo("a.mp4", "copies", shared, "", 201)
		_, secondClip := uploadTo("b.mp4", "copies", shared, "", 201)
		if firstClip == 0 || firstClip != secondClip {
			t.Fatalf("identical content was not deduplicated: %d %d", firstClip, secondClip)
		}
		var conflict struct {
			Version string `json:"version"`
		}
		json.Unmarshal(call("POST", "/api/uploads", map[string]any{"filename": "a.mp4", "size_bytes": len(shared), "destination": "copies"}, 409), &conflict)
		if conflict.Version == "" {
			t.Fatal("overwrite conflict has no version")
		}
		_, replacedClip := uploadTo("a.mp4", "copies", []byte("synthetic-replaced-copy"), conflict.Version, 201)
		if replacedClip == firstClip {
			t.Fatal("replacement reused the old clip")
		}
		var count int
		if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM source_files WHERE clip_id=$1`, firstClip).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("the other copy lost its clip: %d reference(s)", count)
		}
		var otherClip int64
		if err := pool.QueryRow(context.Background(), `SELECT clip_id FROM source_files WHERE source_id=$1 AND relative_path='copies/b.mp4'`, managed.ID).Scan(&otherClip); err != nil {
			t.Fatal(err)
		}
		if otherClip != firstClip {
			t.Fatalf("the other copy was rebound: %d", otherClip)
		}
	})

	t.Run("changing the root detaches files and blocks pending publications", func(t *testing.T) {
		first := t.TempDir()
		if err := os.Mkdir(filepath.Join(first, "media"), 0755); err != nil {
			t.Fatal(err)
		}
		call("PUT", "/api/admin/video-root", map[string]string{"path": first}, 200)
		managed, err := api.Library.ManagedRoot(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		_, clip := uploadTo("kept.mp4", "media", []byte("synthetic-root-kept"), "", 201)
		// A pending intent still points into the current root.
		var pending struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(call("POST", "/api/uploads", map[string]any{"filename": "pending.mp4", "size_bytes": 4, "destination": "media"}, 201), &pending); err != nil {
			t.Fatal(err)
		}
		request(admin, "PUT", "/api/uploads/"+pending.ID+"/chunks", []byte("data"), "application/octet-stream", "bytes 0-3/4", session.CSRF, 200)
		second := t.TempDir()
		call("PUT", "/api/admin/video-root", map[string]string{"path": second}, 409)
		call("DELETE", "/api/uploads/"+pending.ID, nil, 204)
		call("PUT", "/api/admin/video-root", map[string]string{"path": second}, 200)
		var status string
		var fileClip *int64
		if err := pool.QueryRow(context.Background(), `SELECT status,clip_id FROM source_files WHERE source_id=$1 AND relative_path='media/kept.mp4'`, managed.ID).Scan(&status, &fileClip); err != nil {
			t.Fatal(err)
		}
		if status != "missing" || fileClip == nil || *fileClip != clip {
			t.Fatalf("root change lost the old file or its results: status=%q clip=%v", status, fileClip)
		}
		var browsing struct {
			Items []struct {
				Path string `json:"path"`
			} `json:"items"`
		}
		if err := json.Unmarshal(call("GET", "/api/admin/video-root/browse?path=", nil, 200), &browsing); err != nil {
			t.Fatal(err)
		}
		if len(browsing.Items) != 0 {
			t.Fatalf("new root was indexed without a request: %+v", browsing.Items)
		}
	})

}
