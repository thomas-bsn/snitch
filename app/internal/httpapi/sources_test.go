//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"receipts/internal/config"
	"receipts/internal/setup"
)

type reconciliationItem struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	ClipID    int64  `json:"clip_id"`
	ErrorCode string `json:"error_code"`
}

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
		ClipRoots: []string{root}, ProbePath: probe, StabilityWindow: 10 * time.Millisecond, ReconcileInterval: 100 * time.Millisecond, UploadRoot: t.TempDir(),
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
	if clip.Name != "synthetic-clip.mp4" || clip.Status != "ready" || clip.SizeBytes != int64(len("synthetic-video-fixture")) || clip.Duration != 4.25 || clip.VideoCodec != "h264" || clip.RecordedAt != "2025-01-02T03:04:05Z" || clip.DateEstimated || len(clip.Tracks) != 3 || clip.Tracks[1].Language != "fra" {
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

func testSourceReconciliation(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	allowed := t.TempDir()
	firstRoot := filepath.Join(allowed, "first")
	secondRoot := filepath.Join(allowed, "second")
	if err := os.MkdirAll(firstRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(secondRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	firstMedia := filepath.Join(firstRoot, "same.mp4")
	if err := os.WriteFile(firstMedia, []byte("same-synthetic-content"), 0o400); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(secondRoot, "copy.mp4"), []byte("same-synthetic-content"), 0o400); err != nil {
		t.Fatal(err)
	}
	for i := range 5 {
		name := fmt.Sprintf("unique-%d.mp4", i)
		if err := os.WriteFile(filepath.Join(firstRoot, name), []byte("unique-synthetic-content-"+name), 0o400); err != nil {
			t.Fatal(err)
		}
	}
	probe := syntheticConcurrentProbeFixture(t)
	code, err := setup.Issue(context.Background(), pool)
	if err != nil {
		t.Fatal(err)
	}
	s := New(pool, config.Config{
		MasterKey: bytes.Repeat([]byte{1}, 32), DisplayName: "Receipts",
		ClipRoots: []string{allowed}, ProbePath: probe, StabilityWindow: 10 * time.Millisecond, ReconcileInterval: time.Hour, UploadRoot: t.TempDir(),
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
	request := func(method, path, body, csrf string, want int) []byte {
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
		resp, err := admin.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var result bytes.Buffer
		_, _ = result.ReadFrom(resp.Body)
		if resp.StatusCode != want {
			t.Fatalf("%s %s: got %d want %d: %s", method, path, resp.StatusCode, want, result.String())
		}
		return result.Bytes()
	}
	request("POST", "/api/setup", `{"code":"`+code+`","username":"administrator","password":"synthetic-password"}`, "", http.StatusCreated)
	var current struct {
		CSRF string `json:"csrf_token"`
	}
	if err := json.Unmarshal(request("GET", "/api/auth/me", "", "", http.StatusOK), &current); err != nil {
		t.Fatal(err)
	}
	type source struct {
		ID int64 `json:"id"`
	}
	createSource := func(name, path string) source {
		t.Helper()
		body, _ := json.Marshal(map[string]string{"name": name, "path": path})
		var created source
		if err := json.Unmarshal(request("POST", "/api/admin/sources", string(body), current.CSRF, http.StatusCreated), &created); err != nil || created.ID == 0 {
			t.Fatalf("create source %s: %v %+v", name, err, created)
		}
		return created
	}
	listFiles := func(id int64) []reconciliationItem {
		t.Helper()
		var result struct {
			Items []reconciliationItem `json:"items"`
		}
		path := fmt.Sprintf("/api/admin/sources/%d/files", id)
		if err := json.Unmarshal(request("GET", path, "", "", http.StatusOK), &result); err != nil {
			t.Fatal(err)
		}
		return result.Items
	}
	scan := func(id int64) {
		t.Helper()
		path := fmt.Sprintf("/api/admin/sources/%d/scan", id)
		request("POST", path, `{}`, current.CSRF, http.StatusAccepted)
	}
	waitFor := func(id int64, check func([]reconciliationItem) bool) []reconciliationItem {
		t.Helper()
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			files := listFiles(id)
			if check(files) {
				return files
			}
			time.Sleep(20 * time.Millisecond)
		}
		t.Fatalf("source %d never reached expected state: %+v", id, listFiles(id))
		return nil
	}
	first := createSource("first", firstRoot)
	second := createSource("second", secondRoot)
	// Start both full scans together to exercise cross-source content identity.
	var scans sync.WaitGroup
	for _, id := range []int64{first.ID, second.ID} {
		scans.Add(1)
		go func(sourceID int64) {
			defer scans.Done()
			scan(sourceID)
		}(id)
	}
	scans.Wait()
	firstItems := waitFor(first.ID, func(items []reconciliationItem) bool {
		ready := 0
		for _, file := range items {
			if file.Status == "ready" {
				ready++
			}
		}
		return ready == 6
	})
	secondItems := waitFor(second.ID, func(items []reconciliationItem) bool { return len(items) == 1 && items[0].Status == "ready" })
	firstID := clipFor(t, firstItems, "same.mp4")
	if copyID := clipFor(t, secondItems, "copy.mp4"); copyID != firstID {
		t.Fatalf("identical content created separate clips: first=%d copy=%d", firstID, copyID)
	}

	// A known, unchanged path does not call the external probe again.
	if err := os.WriteFile(probe, []byte("#!/bin/sh\nexit 1\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	scan(first.ID)
	time.Sleep(100 * time.Millisecond)
	unchanged := listFiles(first.ID)
	if current := itemFor(t, unchanged, "same.mp4"); current.Status != "ready" || current.ClipID != firstID {
		t.Fatalf("unchanged path was unnecessarily re-probed: %+v", current)
	}
	if err := os.WriteFile(probe, []byte(syntheticProbeScript()), 0o700); err != nil {
		t.Fatal(err)
	}

	// A content change at a path receives a new identity and leaves the other provenance intact.
	if err := os.Chmod(firstMedia, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(firstMedia, []byte("different-synthetic-content"), 0o400); err != nil {
		t.Fatal(err)
	}
	changedTime := time.Now().Add(time.Second).UTC()
	if err := os.Chtimes(firstMedia, changedTime, changedTime); err != nil {
		t.Fatal(err)
	}
	scan(first.ID)
	changedItems := waitFor(first.ID, func(items []reconciliationItem) bool {
		return clipForMaybe(items, "same.mp4") != 0 && clipForMaybe(items, "same.mp4") != firstID
	})
	newID := clipFor(t, changedItems, "same.mp4")
	if newID == firstID || clipFor(t, listFiles(second.ID), "copy.mp4") != firstID {
		t.Fatalf("changed content overwrote the existing clip: old=%d new=%d", firstID, newID)
	}

	// Removing the old copy only removes that location; the second source remains readable.
	if err := os.Remove(filepath.Join(secondRoot, "copy.mp4")); err != nil {
		t.Fatal(err)
	}
	scan(second.ID)
	missingCopy := waitFor(second.ID, func(items []reconciliationItem) bool { return len(items) == 1 && items[0].Status == "missing" })
	if clipFor(t, missingCopy, "copy.mp4") != firstID {
		t.Fatal("missing location did not retain its previous clip")
	}
	if clipFor(t, listFiles(first.ID), "same.mp4") != newID {
		t.Fatal("remaining source location no longer resolves to changed clip")
	}

	// Lose at least 20% of known paths in a successful scan: warn but keep each clip record.
	for _, file := range firstItems {
		if strings.HasPrefix(file.Name, "unique-") {
			_ = os.Remove(filepath.Join(firstRoot, file.Name))
		}
	}
	scan(first.ID)
	warned := waitFor(first.ID, func(items []reconciliationItem) bool {
		for _, file := range items {
			if strings.HasPrefix(file.Name, "unique-") && file.Status == "missing" {
				return true
			}
		}
		return false
	})
	if len(warned) != 6 {
		t.Fatalf("missing paths were deleted from history: %+v", warned)
	}
	var health struct {
		Items []struct {
			ID      int64  `json:"id"`
			Health  string `json:"health"`
			Message string `json:"health_message"`
		} `json:"items"`
	}
	if err := json.Unmarshal(request("GET", "/api/admin/sources", "", "", http.StatusOK), &health); err != nil {
		t.Fatal(err)
	}
	for _, current := range health.Items {
		if current.ID == first.ID && (current.Health != "warning" || !strings.Contains(current.Message, "introuvable")) {
			t.Fatalf("large disappearance was not reported: %+v", current)
		}
	}

	// An unavailable root is not a successful empty scan and must not mark paths missing.
	beforeUnavailable := listFiles(first.ID)
	offline := firstRoot + ".offline"
	if err := os.Rename(firstRoot, offline); err != nil {
		t.Fatal(err)
	}
	scan(first.ID)
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if err := json.Unmarshal(request("GET", "/api/admin/sources", "", "", http.StatusOK), &health); err != nil {
			t.Fatal(err)
		}
		found := false
		for _, current := range health.Items {
			found = found || (current.ID == first.ID && current.Health == "unavailable")
		}
		if found {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	afterUnavailable := listFiles(first.ID)
	if len(afterUnavailable) != len(beforeUnavailable) {
		t.Fatalf("unavailable source changed file history: before=%+v after=%+v", beforeUnavailable, afterUnavailable)
	}
	for i, file := range afterUnavailable {
		if file.Status != beforeUnavailable[i].Status || file.ClipID != beforeUnavailable[i].ClipID {
			t.Fatalf("unavailable source changed historical state: before=%+v after=%+v", beforeUnavailable[i], file)
		}
	}
	_ = os.Rename(offline, firstRoot)
}

func clipFor(t *testing.T, files []reconciliationItem, name string) int64 {
	t.Helper()
	for _, file := range files {
		if file.Name == name {
			if file.ClipID == 0 {
				t.Fatalf("%s has no clip identity: %+v", name, file)
			}
			return file.ClipID
		}
	}
	t.Fatalf("%s not found in %+v", name, files)
	return 0
}

func itemFor(t *testing.T, files []reconciliationItem, name string) reconciliationItem {
	t.Helper()
	for _, file := range files {
		if file.Name == name {
			return file
		}
	}
	t.Fatalf("%s not found in %+v", name, files)
	return reconciliationItem{}
}

func clipForMaybe(files []reconciliationItem, name string) int64 {
	for _, file := range files {
		if file.Name == name {
			return file.ClipID
		}
	}
	return 0
}

func syntheticProbeFixture(t *testing.T) string {
	t.Helper()
	probe := filepath.Join(t.TempDir(), "ffprobe-fixture")
	if err := os.WriteFile(probe, []byte(syntheticProbeScript()), 0o700); err != nil {
		t.Fatal(err)
	}
	return probe
}

func syntheticProbeScript() string {
	const metadata = `{"format":{"duration":"4.25","tags":{"creation_time":"2025-01-02T03:04:05Z"}},"streams":[{"index":0,"codec_type":"video","codec_name":"h264","width":640,"height":360,"tags":{"title":"Game"}},{"index":1,"codec_type":"audio","codec_name":"aac","tags":{"language":"fra","title":"Micro"}},{"index":2,"codec_type":"audio","codec_name":"aac","tags":{"language":"fra","title":"Voix"}}]}`
	const noVideo = `{"format":{"duration":"2"},"streams":[{"codec_type":"audio","codec_name":"aac"}]}`
	probeScript := "#!/bin/sh\nfor input do :; done\ncase \"$input\" in *broken.mp4) exit 1;; *no-video.mp4) printf '%s\\n' '" + noVideo + "';; *) printf '%s\\n' '" + metadata + "';; esac\n"
	return probeScript
}

func syntheticConcurrentProbeFixture(t *testing.T) string {
	t.Helper()
	gate := t.TempDir()
	probe := filepath.Join(t.TempDir(), "ffprobe-concurrency-fixture")
	const metadata = `{"format":{"duration":"4.25","tags":{"creation_time":"2025-01-02T03:04:05Z"}},"streams":[{"codec_type":"video","codec_name":"h264","width":640,"height":360}]}`
	first := filepath.Join(gate, "first-probe-started")
	second := filepath.Join(gate, "second-probe-started")
	firstQuoted := "'" + strings.ReplaceAll(first, "'", "'\\''") + "'"
	secondQuoted := "'" + strings.ReplaceAll(second, "'", "'\\''") + "'"
	script := "#!/bin/sh\nfor input do :; done\ncase \"$input\" in\n" +
		"*/same.mp4) touch " + firstQuoted + "; while [ ! -f " + secondQuoted + " ]; do sleep 0.01; done;;\n" +
		"*/copy.mp4) touch " + secondQuoted + "; while [ ! -f " + firstQuoted + " ]; do sleep 0.01; done;;\n" +
		"esac\nprintf '%s\\n' '" + metadata + "'\n"
	if err := os.WriteFile(probe, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	return probe
}
