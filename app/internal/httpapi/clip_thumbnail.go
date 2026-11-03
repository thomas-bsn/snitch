package httpapi

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	thumbnailTimeout  = 15 * time.Second
	thumbnailMaxBytes = 512 << 10
	thumbnailWidth    = 320
	// A failed extraction is not retried for this long, so a broken file does not
	// start ffmpeg again on every tile of every page view.
	thumbnailRetryAfter = time.Minute
)

// thumbnailSlots bounds concurrent ffmpeg runs; thumbnailLocks makes the extraction
// of one clip happen once even under concurrent requests (the per-clip locks live on the Server).
var (
	thumbnailSlots = make(chan struct{}, 2)
)

// clipThumbnail serves the small JPEG of an indexed clip. Access is the one of the
// media itself (authorizedClip); the picture is generated at most once per clip and
// then kept in the database.
func (s *Server) clipThumbnail(w http.ResponseWriter, r *http.Request) {
	v, ok := s.authorizedClip(w, r)
	if !ok {
		return
	}
	jpeg, err := s.thumbnailFor(r.Context(), v.ID, v.DurationSeconds)
	if err != nil {
		fail(w, http.StatusNotFound, "thumbnail_not_found")
		return
	}
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "private, max-age=86400")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	etag := fmt.Sprintf(`"thumb-%d-%d"`, v.ID, len(jpeg))
	w.Header().Set("ETag", etag)
	http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(jpeg))
}

// thumbnailFor returns the stored picture, generating it under a per-clip lock when missing.
func (s *Server) thumbnailFor(ctx context.Context, clipID int64, duration *float64) ([]byte, error) {
	if jpeg, err := s.storedThumbnail(ctx, clipID); err == nil {
		return jpeg, nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	lock, _ := s.thumbnailLocks.LoadOrStore(clipID, &sync.Mutex{})
	mu := lock.(*sync.Mutex)
	mu.Lock()
	defer mu.Unlock()
	if jpeg, err := s.storedThumbnail(ctx, clipID); err == nil {
		return jpeg, nil
	}
	if at, failed := s.thumbnailFails.Load(clipID); failed && time.Since(at.(time.Time)) < thumbnailRetryAfter {
		return nil, errors.New("thumbnail recently failed")
	}
	jpeg, err := s.extractThumbnail(ctx, clipID, duration)
	if err != nil {
		s.thumbnailFails.Store(clipID, time.Now())
		s.Log.Warn("clip thumbnail failed", "clip", clipID, "kind", fmtError(err))
		return nil, err
	}
	s.thumbnailFails.Delete(clipID)
	if _, err = s.Pool.Exec(ctx, `INSERT INTO clip_thumbnails(clip_id,jpeg) SELECT id,$2 FROM clips WHERE id=$1 ON CONFLICT(clip_id) DO NOTHING`, clipID, jpeg); err != nil {
		return nil, err
	}
	return jpeg, nil
}

func (s *Server) storedThumbnail(ctx context.Context, clipID int64) ([]byte, error) {
	var jpeg []byte
	err := s.Pool.QueryRow(ctx, `SELECT jpeg FROM clip_thumbnails WHERE clip_id=$1`, clipID).Scan(&jpeg)
	return jpeg, err
}

// warmThumbnail is the best-effort generation at indexing time: it never blocks the
// indexing and its failure is only logged; the first request would try again.
func (s *Server) warmThumbnail(clipID int64) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*thumbnailTimeout)
	defer cancel()
	var duration *float64
	if err := s.Pool.QueryRow(ctx, `SELECT duration_seconds FROM clips WHERE id=$1`, clipID).Scan(&duration); err != nil {
		return
	}
	if _, err := s.thumbnailFor(ctx, clipID, duration); err != nil {
		s.Log.Debug("clip thumbnail not warmed", "clip", clipID, "kind", fmtError(err))
	}
}

// extractThumbnail takes one frame at 10 % of the duration (one second when the
// duration is unknown or short), scaled to thumbnailWidth, as a JPEG of moderate quality.
func (s *Server) extractThumbnail(ctx context.Context, clipID int64, duration *float64) ([]byte, error) {
	media, err := s.resolveClipMedia(ctx, clipID)
	if err != nil {
		return nil, err
	}
	at := 1.0
	if duration != nil && *duration > 10 {
		at = *duration / 10
	} else if duration != nil && *duration > 0 && *duration < 1 {
		at = 0
	}
	select {
	case thumbnailSlots <- struct{}{}:
		defer func() { <-thumbnailSlots }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	ctx, cancel := context.WithTimeout(ctx, thumbnailTimeout)
	defer cancel()
	ffmpeg := s.Config.FFmpegPath
	if ffmpeg == "" {
		ffmpeg = "ffmpeg"
	}
	out := &limitedBuffer{limit: thumbnailMaxBytes}
	cmd := exec.CommandContext(ctx, ffmpeg, "-hide_banner", "-loglevel", "error", "-nostdin",
		"-ss", strconv.FormatFloat(at, 'f', 3, 64), "-i", media, "-map", "0:v:0", "-frames:v", "1",
		"-vf", fmt.Sprintf("scale=%d:-2", thumbnailWidth), "-q:v", "6", "-f", "image2pipe", "-c:v", "mjpeg", "pipe:1")
	cmd.Stdout = out
	if err = cmd.Run(); err != nil {
		return nil, err
	}
	b := out.Bytes()
	if len(b) < 4 || b[0] != 0xFF || b[1] != 0xD8 {
		return nil, os.ErrInvalid
	}
	return b, nil
}
