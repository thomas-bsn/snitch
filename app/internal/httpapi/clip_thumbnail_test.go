//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// The thumbnail of an indexed clip: same access as the media, generated once, kept, removed with the clip.
func testClipThumbnail(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	h := newVoiceHarnessClips(t, pool, 2)
	runs := func() int {
		log, _ := os.ReadFile(h.ffmpegLog)
		return strings.Count(string(log), "mjpeg")
	}
	count := func(clip int64) int {
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM clip_thumbnails WHERE clip_id=$1`, clip).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	path := func(clip int64) string { return "/api/clips/" + strconv64(clip) + "/thumbnail" }

	// At indexing: each clip gets its picture once, without any request.
	deadline := time.Now().Add(10 * time.Second)
	for (count(h.clipIDs[0]) == 0 || count(h.clipIDs[1]) == 0) && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if count(h.clipIDs[0]) != 1 || count(h.clipIDs[1]) != 1 || runs() != 2 {
		t.Fatalf("indexing must generate one thumbnail per clip: rows %d/%d, ffmpeg runs %d", count(h.clipIDs[0]), count(h.clipIDs[1]), runs())
	}
	image := h.adminCall(http.MethodGet, path(h.clipIDs[0]), nil, http.StatusOK)
	if !bytes.HasPrefix(image, []byte{0xFF, 0xD8}) || runs() != 2 {
		t.Fatalf("a stored thumbnail is served as a JPEG without a new extraction (runs %d)", runs())
	}
	req, _ := http.NewRequest(http.MethodGet, h.ts.URL+path(h.clipIDs[0]), nil)
	resp, err := h.admin.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.Header.Get("Content-Type") != "image/jpeg" || resp.Header.Get("ETag") == "" || resp.Header.Get("Cache-Control") == "" {
		t.Fatalf("missing type or cache headers: %v", resp.Header)
	}

	// Lazy generation for an already indexed clip, once even under concurrent requests.
	if _, err = pool.Exec(ctx, `DELETE FROM clip_thumbnails WHERE clip_id=$1`, h.clipIDs[0]); err != nil {
		t.Fatal(err)
	}
	before := runs()
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if body := h.adminCall(http.MethodGet, path(h.clipIDs[0]), nil, http.StatusOK); !bytes.HasPrefix(body, []byte{0xFF, 0xD8}) {
				t.Error("concurrent request did not get a JPEG")
			}
		}()
	}
	wg.Wait()
	if runs()-before != 1 {
		t.Fatalf("one ffmpeg run expected under concurrency, got %d", runs()-before)
	}

	// Same access as the media: a private clip is hidden from another member, not from the administrator.
	alice := h.newUser("thumbnail-member")
	if _, err = pool.Exec(ctx, `UPDATE clips SET visibility='shared' WHERE id=$1`, h.clipIDs[0]); err != nil {
		t.Fatal(err)
	}
	h.userCall(alice, http.MethodGet, path(h.clipIDs[0]), nil, http.StatusOK)
	if _, err = pool.Exec(ctx, `UPDATE clips SET visibility='private' WHERE id=$1`, h.clipIDs[0]); err != nil {
		t.Fatal(err)
	}
	mediaStatus, _ := h.userStatus(alice, http.MethodGet, "/api/clips/"+strconv64(h.clipIDs[0])+"/media", nil)
	thumbStatus, _ := h.userStatus(alice, http.MethodGet, path(h.clipIDs[0]), nil)
	if mediaStatus != http.StatusNotFound || thumbStatus != mediaStatus {
		t.Fatalf("private clip: media %d, thumbnail %d", mediaStatus, thumbStatus)
	}
	h.adminCall(http.MethodGet, path(h.clipIDs[0]), nil, http.StatusOK)
	anonymous, err := http.Get(h.ts.URL + path(h.clipIDs[0]))
	if err != nil {
		t.Fatal(err)
	}
	anonymous.Body.Close()
	if anonymous.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous thumbnail: %d", anonymous.StatusCode)
	}

	// A failed extraction answers 404 cleanly and stores nothing.
	failing := filepath.Join(t.TempDir(), "ffmpeg-fails")
	if err = os.WriteFile(failing, []byte("#!/bin/sh\nexit 1\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	h.api.Config.FFmpegPath = failing
	if _, err = pool.Exec(ctx, `DELETE FROM clip_thumbnails WHERE clip_id=$1`, h.clipIDs[1]); err != nil {
		t.Fatal(err)
	}
	h.adminCall(http.MethodGet, path(h.clipIDs[1]), nil, http.StatusNotFound)
	if count(h.clipIDs[1]) != 0 {
		t.Fatal("a failed extraction must store nothing")
	}

	// The thumbnail goes away with the clip.
	if count(h.clipIDs[0]) != 1 {
		t.Fatal("expected a stored thumbnail before deleting the clip")
	}
	if _, err = pool.Exec(ctx, `DELETE FROM source_files WHERE clip_id=$1`, h.clipIDs[0]); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `DELETE FROM clips WHERE id=$1`, h.clipIDs[0]); err != nil {
		t.Fatal(err)
	}
	if count(h.clipIDs[0]) != 0 {
		t.Fatal("deleting a clip must delete its thumbnail")
	}
}
