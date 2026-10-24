//go:build integration

package httpapi

import (
	"bytes"
	"context"
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
	"receipts/internal/config"
	"receipts/internal/setup"
)

func testClipLibrary(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	clipRoot := t.TempDir()
	sourceContent := []byte("synthetic-folder-video")
	if err := os.WriteFile(filepath.Join(clipRoot, "source-fixture.mp4"), sourceContent, 0o400); err != nil {
		t.Fatal(err)
	}
	code, err := setup.Issue(context.Background(), pool)
	if err != nil {
		t.Fatal(err)
	}
	api := New(pool, config.Config{MasterKey: bytes.Repeat([]byte{1}, 32), DisplayName: "Receipts", ClipRoots: []string{clipRoot}, StabilityWindow: 0, UploadRoot: t.TempDir(), ProbePath: syntheticProbeFixture(t), UploadMaxBytes: 1024, UploadChunkBytes: 1024, UploadSessionTTL: time.Hour}, slog.New(slog.NewJSONHandler(io.Discard, nil)))
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
	call := func(c *http.Client, method, path string, body []byte, contentType, csrf, rng string, want int) (*http.Response, []byte) {
		t.Helper()
		req, e := http.NewRequest(method, ts.URL+path, bytes.NewReader(body))
		if e != nil {
			t.Fatal(e)
		}
		req.Header.Set("Origin", ts.URL)
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		if method == http.MethodPut && strings.HasSuffix(path, "/chunks") {
			req.Header.Set("Content-Range", fmt.Sprintf("bytes 0-%d/%d", len(body)-1, len(body)))
		}
		if csrf != "" {
			req.Header.Set("X-CSRF-Token", csrf)
		}
		if rng != "" {
			req.Header.Set("Range", rng)
		}
		resp, e := c.Do(req)
		if e != nil {
			t.Fatal(e)
		}
		data, e := io.ReadAll(resp.Body)
		resp.Body.Close()
		if e != nil {
			t.Fatal(e)
		}
		if resp.StatusCode != want {
			t.Fatalf("%s %s: got %d want %d: %s", method, path, resp.StatusCode, want, data)
		}
		return resp, data
	}
	call(admin, "POST", "/api/setup", []byte(`{"code":"`+code+`","username":"administrator","password":"synthetic-password"}`), "application/json", "", "", 201)
	var session struct {
		CSRF string `json:"csrf_token"`
	}
	_, data := call(admin, "GET", "/api/auth/me", nil, "", "", "", 200)
	if err := json.Unmarshal(data, &session); err != nil {
		t.Fatal(err)
	}
	content := []byte("synthetic-browser-video")
	upload := func(visibility string) int64 {
		video := content
		if visibility == "private" {
			video = []byte("private-synthetic-browser-video")
		}
		body, _ := json.Marshal(map[string]any{"filename": "fixture.mp4", "size_bytes": len(video), "visibility": visibility})
		_, data := call(admin, "POST", "/api/uploads", body, "application/json", session.CSRF, "", 201)
		var item struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(data, &item); err != nil {
			t.Fatal(err)
		}
		call(admin, "PUT", "/api/uploads/"+item.ID+"/chunks", video, "application/octet-stream", session.CSRF, "", 200)
		_, data = call(admin, "POST", "/api/uploads/"+item.ID+"/complete", []byte(`{}`), "application/json", session.CSRF, "", 201)
		var clip struct {
			ClipID int64 `json:"clip_id"`
		}
		if err := json.Unmarshal(data, &clip); err != nil {
			t.Fatal(err)
		}
		return clip.ClipID
	}
	sharedID := upload("shared")
	privateID := upload("private")
	_, data = call(admin, "POST", "/api/admin/sources", mustJSON(t, map[string]string{"name": "library-fixture", "path": clipRoot}), "application/json", session.CSRF, "", 201)
	var source struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(data, &source); err != nil {
		t.Fatal(err)
	}
	var sourceClipID int64
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		_, data = call(admin, "GET", fmt.Sprintf("/api/admin/sources/%d/files", source.ID), nil, "", "", "", 200)
		var files struct {
			Items []struct {
				Status string `json:"status"`
				ClipID *int64 `json:"clip_id"`
			} `json:"items"`
		}
		if err := json.Unmarshal(data, &files); err != nil {
			t.Fatal(err)
		}
		if len(files.Items) > 0 && files.Items[0].Status == "ready" && files.Items[0].ClipID != nil {
			sourceClipID = *files.Items[0].ClipID
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if sourceClipID == 0 {
		t.Fatal("synthetic source file did not become ready")
	}
	_, data = call(admin, "GET", "/api/clips/"+strconv64(sourceClipID), nil, "", "", "", 200)
	var sourceDetails struct {
		Status  string   `json:"status"`
		Sources []string `json:"sources"`
	}
	if err := json.Unmarshal(data, &sourceDetails); err != nil || sourceDetails.Status != "ready" || len(sourceDetails.Sources) != 1 || sourceDetails.Sources[0] != "library-fixture" {
		t.Fatalf("source clip details missing provenance: %s %v", data, err)
	}
	call(http.DefaultClient, "GET", "/api/clips", nil, "", "", "", 401)
	_, data = call(admin, "GET", "/api/clips?page=1&page_size=1&visibility=shared", nil, "", "", "", 200)
	var page struct {
		Items []struct {
			ID         int64  `json:"id"`
			Visibility string `json:"visibility"`
		} `json:"items"`
		Total int `json:"total"`
	}
	if err := json.Unmarshal(data, &page); err != nil {
		t.Fatal(err)
	}
	if page.Total != 2 || len(page.Items) != 1 || page.Items[0].Visibility != "shared" {
		t.Fatalf("unexpected shared page: %+v", page)
	}
	_, data = call(admin, "GET", "/api/clips?page=2&page_size=1&visibility=shared", nil, "", "", "", 200)
	if err := json.Unmarshal(data, &page); err != nil || len(page.Items) != 1 || page.Items[0].ID != sharedID {
		t.Fatalf("unexpected second page: %s %v", data, err)
	}
	call(admin, "GET", "/api/clips/"+strconv64(privateID), nil, "", "", "", 200)
	if _, err := pool.Exec(context.Background(), `INSERT INTO users(username,role,approved) VALUES('other-member','user',true)`); err != nil {
		t.Fatal(err)
	}
	var otherID int64
	if err := pool.QueryRow(context.Background(), `SELECT id FROM users WHERE username='other-member'`).Scan(&otherID); err != nil {
		t.Fatal(err)
	}
	otherJar, _ := cookiejar.New(nil)
	otherURL, _ := url.Parse(ts.URL)
	otherClient := &http.Client{Jar: otherJar}
	issued := httptest.NewRecorder()
	if err := api.issueSession(issued, httptest.NewRequest("GET", ts.URL, nil), otherID); err != nil {
		t.Fatal(err)
	}
	otherJar.SetCookies(otherURL, issued.Result().Cookies())
	_, data = call(otherClient, "GET", "/api/clips?page=1&page_size=10", nil, "", "", "", 200)
	if strings.Contains(string(data), `"id":`+strconv64(privateID)) {
		t.Fatalf("other member library leaked private clip: %s", data)
	}
	call(otherClient, "GET", "/api/clips/"+strconv64(privateID), nil, "", "", "", 404)
	call(otherClient, "GET", "/api/clips/"+strconv64(privateID)+"/media", nil, "", "", "bytes=0-1", 404)
	var mediaKey string
	if err := pool.QueryRow(context.Background(), `SELECT storage_key FROM uploaded_media WHERE clip_id=$1`, sharedID).Scan(&mediaKey); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(api.Uploads.MediaRoot(), mediaKey+".media")); err != nil {
		t.Fatalf("uploaded media file missing: %v root=%s key=%s", err, api.Uploads.MediaRoot(), mediaKey)
	}
	resp, bytes := call(admin, "GET", "/api/clips/"+strconv64(sharedID)+"/media", nil, "", "", "bytes=3-7", 206)
	if string(bytes) != string(content[3:8]) || resp.Header.Get("Content-Range") != "bytes 3-7/23" || resp.Header.Get("Content-Type") != "video/mp4" {
		t.Fatalf("unexpected range: %q %q", bytes, resp.Header.Get("Content-Range"))
	}
	resp, fullBytes := call(admin, "GET", "/api/clips/"+strconv64(sharedID)+"/media", nil, "", "", "", 200)
	if string(fullBytes) != string(content) || resp.Header.Get("Content-Disposition") == "" || !strings.Contains(resp.Header.Get("Content-Disposition"), "fixture.mp4") {
		t.Fatalf("unexpected full media response: %q %q", fullBytes, resp.Header.Get("Content-Disposition"))
	}
	resp, sourceBytes := call(admin, "GET", "/api/clips/"+strconv64(sourceClipID)+"/media", nil, "", "", "bytes=2-6", 206)
	if string(sourceBytes) != string(sourceContent[2:7]) || resp.Header.Get("Content-Range") != fmt.Sprintf("bytes 2-6/%d", len(sourceContent)) {
		t.Fatalf("unexpected folder source range: %q %q", sourceBytes, resp.Header.Get("Content-Range"))
	}
	resp, _ = call(admin, "GET", "/api/clips/"+strconv64(sharedID)+"/media", nil, "", "", "bytes=999-", 416)
	if resp.Header.Get("Content-Range") != "bytes */23" {
		t.Fatalf("missing unsatisfied range: %q", resp.Header.Get("Content-Range"))
	}
	if strings.Contains(string(data), "/data/media") {
		t.Fatal("library response leaked a storage path")
	}
}

func strconv64(v int64) string { return fmt.Sprintf("%d", v) }

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
