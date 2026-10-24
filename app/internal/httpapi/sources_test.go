//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"receipts/internal/config"
	"receipts/internal/setup"
)

func testFolderSources(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	root := t.TempDir()
	media := filepath.Join(root, "synthetic-clip.mp4")
	if err := os.WriteFile(media, []byte("synthetic-video-fixture"), 0o400); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "broken.mp4"), []byte("truncated-synthetic-video"), 0o400); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "no-video.mp4"), []byte("synthetic-audio-only"), 0o400); err != nil {
		t.Fatal(err)
	}
	probe := syntheticProbeFixture(t)
	code, err := setup.Issue(context.Background(), pool)
	if err != nil {
		t.Fatal(err)
	}
	s := New(pool, config.Config{
		MasterKey: bytes.Repeat([]byte{1}, 32), DisplayName: "Receipts",
		ClipRoots: []string{root}, ProbePath: probe, StabilityWindow: 10 * time.Millisecond, ReconcileInterval: 100 * time.Millisecond,
	}, slog.New(slog.NewJSONHandler(os.Stderr, nil)))
	if err := s.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ts := httptest.NewUnstartedServer(s.Handler())
	s.Config.PublicURL = "http://" + ts.Listener.Addr().String()
	ts.Start()
	defer ts.Close()
	jar, _ := cookiejar.New(nil)
	admin := &http.Client{Jar: jar}
	anonymous := &http.Client{}
	request := func(client *http.Client, method, path, body, csrf string, want int) []byte {
		t.Helper()
		req, err := http.NewRequest(method, ts.URL+path, strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Origin", ts.URL)
		req.Header.Set("Content-Type", "application/json")
		if csrf != "" {
			req.Header.Set("X-CSRF-Token", csrf)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var data bytes.Buffer
		_, _ = data.ReadFrom(resp.Body)
		if resp.StatusCode != want {
			t.Fatalf("%s %s: got %d want %d: %s", method, path, resp.StatusCode, want, data.String())
		}
		return data.Bytes()
	}
	request(anonymous, "GET", "/api/admin/sources", "", "", http.StatusUnauthorized)
	request(admin, "POST", "/api/setup", `{"code":"`+code+`","username":"administrator","password":"synthetic-password"}`, "", http.StatusCreated)
	var session struct {
		CSRF string `json:"csrf_token"`
	}
	if err := json.Unmarshal(request(admin, "GET", "/api/auth/me", "", "", http.StatusOK), &session); err != nil {
		t.Fatal(err)
	}
	request(admin, "POST", "/api/admin/sources", `{"name":"outside","path":"/"}`, session.CSRF, http.StatusBadRequest)
	body := `{"name":"captures","path":"` + strings.ReplaceAll(root, `\`, `\\`) + `"}`
	var source struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(request(admin, "POST", "/api/admin/sources", body, session.CSRF, http.StatusCreated), &source); err != nil || source.ID == 0 {
		t.Fatalf("source response: %v %+v", err, source)
	}
	sourcePath := "/api/admin/sources/" + strconv.FormatInt(source.ID, 10)
	var files struct {
		Items []struct {
			Name          string  `json:"name"`
			Status        string  `json:"status"`
			ErrorCode     string  `json:"error_code"`
			SizeBytes     int64   `json:"size_bytes"`
			Duration      float64 `json:"duration_seconds"`
			VideoCodec    string  `json:"video_codec"`
			RecordedAt    string  `json:"recorded_at"`
			DateEstimated bool    `json:"recorded_at_estimated"`
			Tracks        []struct {
				Type     string `json:"type"`
				Codec    string `json:"codec"`
				Title    string `json:"title"`
				Language string `json:"language"`
			} `json:"tracks"`
		} `json:"items"`
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if err := json.Unmarshal(request(admin, "GET", sourcePath+"/files", "", "", http.StatusOK), &files); err != nil {
			t.Fatal(err)
		}
		ready, unreadable := 0, 0
		for _, item := range files.Items {
			if item.Status == "ready" {
				ready++
			}
			if item.Status == "unreadable" {
				unreadable++
			}
		}
		if ready == 1 && unreadable == 2 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if len(files.Items) != 3 {
		t.Fatalf("file list: %+v", files.Items)
	}
	byName := map[string]struct {
		Name          string  `json:"name"`
		Status        string  `json:"status"`
		ErrorCode     string  `json:"error_code"`
		SizeBytes     int64   `json:"size_bytes"`
		Duration      float64 `json:"duration_seconds"`
		VideoCodec    string  `json:"video_codec"`
		RecordedAt    string  `json:"recorded_at"`
		DateEstimated bool    `json:"recorded_at_estimated"`
		Tracks        []struct {
			Type     string `json:"type"`
			Codec    string `json:"codec"`
			Title    string `json:"title"`
			Language string `json:"language"`
		} `json:"tracks"`
	}{}
	for _, item := range files.Items {
		byName[item.Name] = item
	}
	clip := byName["synthetic-clip.mp4"]
	if clip.Name != "synthetic-clip.mp4" || clip.Status != "ready" || clip.SizeBytes != int64(len("synthetic-video-fixture")) || clip.Duration != 4.25 || clip.VideoCodec != "h264" || clip.RecordedAt != "2025-01-02T03:04:05Z" || clip.DateEstimated || len(clip.Tracks) != 2 || clip.Tracks[1].Language != "fra" {
		t.Fatalf("unexpected clip metadata: %+v", clip)
	}
	if byName["broken.mp4"].Status != "unreadable" || byName["broken.mp4"].ErrorCode != "video_unreadable" || byName["no-video.mp4"].Status != "unreadable" || byName["no-video.mp4"].ErrorCode != "video_stream_missing" {
		t.Fatalf("invalid videos did not have visible errors: %+v", byName)
	}
	if err := os.WriteFile(filepath.Join(root, "event-driven.mp4"), []byte("new-synthetic-video"), 0o400); err != nil {
		t.Fatal(err)
	}
	deadline = time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if err := json.Unmarshal(request(admin, "GET", sourcePath+"/files", "", "", http.StatusOK), &files); err != nil {
			t.Fatal(err)
		}
		eventReady := false
		for _, item := range files.Items {
			eventReady = eventReady || (item.Name == "event-driven.mp4" && item.Status == "ready")
		}
		if eventReady {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	oldRoot := root + ".offline"
	if err := os.Rename(root, oldRoot); err != nil {
		t.Fatal(err)
	}
	time.Sleep(300 * time.Millisecond) // Let the removed directory's watch expire before recreating it.
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "periodic-reconcile.mp4"), []byte("periodic-synthetic-video"), 0o400); err != nil {
		t.Fatal(err)
	}
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if err := json.Unmarshal(request(admin, "GET", sourcePath+"/files", "", "", http.StatusOK), &files); err != nil {
			t.Fatal(err)
		}
		found := false
		for _, item := range files.Items {
			found = found || (item.Name == "periodic-reconcile.mp4" && item.Status == "ready")
		}
		if found {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if got, err := os.ReadFile(filepath.Join(oldRoot, "synthetic-clip.mp4")); err != nil || string(got) != "synthetic-video-fixture" {
		t.Fatalf("source original changed: %q %v", got, err)
	}
}

func syntheticProbeFixture(t *testing.T) string {
	t.Helper()
	probe := filepath.Join(t.TempDir(), "ffprobe-fixture")
	const metadata = `{"format":{"duration":"4.25","tags":{"creation_time":"2025-01-02T03:04:05Z"}},"streams":[{"codec_type":"video","codec_name":"h264","width":640,"height":360,"tags":{"title":"Game"}},{"codec_type":"audio","codec_name":"aac","tags":{"language":"fra","title":"Micro"}}]}`
	const noVideo = `{"format":{"duration":"2"},"streams":[{"codec_type":"audio","codec_name":"aac"}]}`
	probeScript := "#!/bin/sh\nfor input do :; done\ncase \"$input\" in *broken.mp4) exit 1;; *no-video.mp4) printf '%s\\n' '" + noVideo + "';; *) printf '%s\\n' '" + metadata + "';; esac\n"
	if err := os.WriteFile(probe, []byte(probeScript), 0o700); err != nil {
		t.Fatal(err)
	}
	return probe
}
