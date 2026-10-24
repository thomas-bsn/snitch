//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
	"receipts/internal/config"
	"receipts/internal/setup"
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
		MasterKey: bytes.Repeat([]byte{1}, 32), DisplayName: "Receipts",
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
}
