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
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	maxExcerptSeconds = 10.0
	maxExcerptBytes   = 8 << 20
)

// excerptSlots bounds concurrent ffmpeg runs: extracts are cut on demand from the
// clip file and never stored.
var excerptSlots = make(chan struct{}, 2)

type limitedBuffer struct {
	bytes.Buffer
	limit int
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if b.Len()+len(p) > b.limit {
		return 0, errors.New("excerpt too large")
	}
	return b.Buffer.Write(p)
}

// localSpeakerExcerpt serves a short WAV extract of one local speaker: its longest
// turn (or the turn named by ?turn=<index>), capped at ten seconds. Access follows
// the clip visibility through authorizedSpeaker; anonymous callers never reach it.
func (s *Server) localSpeakerExcerpt(w http.ResponseWriter, r *http.Request) {
	sp, ok := s.authorizedSpeaker(w, r)
	if !ok {
		return
	}
	var start, end float64
	var err error
	if raw := r.URL.Query().Get("turn"); raw != "" {
		turn, parseErr := strconv.Atoi(raw)
		if parseErr != nil || turn < 0 {
			fail(w, http.StatusBadRequest, "invalid_turn")
			return
		}
		err = s.Pool.QueryRow(r.Context(), `SELECT start_seconds,end_seconds FROM speaker_turns WHERE local_speaker_id=$1 AND turn_index=$2`, sp.ID, turn).Scan(&start, &end)
	} else {
		err = s.Pool.QueryRow(r.Context(), `SELECT start_seconds,end_seconds FROM speaker_turns WHERE local_speaker_id=$1 ORDER BY end_seconds-start_seconds DESC,turn_index LIMIT 1`, sp.ID).Scan(&start, &end)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, http.StatusNotFound, "excerpt_not_found")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	duration := end - start
	if duration > maxExcerptSeconds {
		duration = maxExcerptSeconds
	}
	if duration < 0.2 {
		duration = 0.2
	}
	media, err := s.resolveClipMedia(r.Context(), sp.ClipID)
	if errors.Is(err, os.ErrNotExist) {
		fail(w, http.StatusNotFound, "media_not_found")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	select {
	case excerptSlots <- struct{}{}:
		defer func() { <-excerptSlots }()
	case <-r.Context().Done():
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	ffmpeg := s.Config.FFmpegPath
	if ffmpeg == "" {
		ffmpeg = "ffmpeg"
	}
	out := &limitedBuffer{limit: maxExcerptBytes}
	cmd := exec.CommandContext(ctx, ffmpeg, "-hide_banner", "-loglevel", "error", "-nostdin",
		"-ss", fmt.Sprintf("%.3f", start), "-t", fmt.Sprintf("%.3f", duration), "-i", media,
		"-map", fmt.Sprintf("0:%d", sp.Index), "-vn", "-ac", "1", "-ar", "16000", "-c:a", "pcm_s16le", "-f", "wav", "pipe:1")
	cmd.Stdout = out
	if err = cmd.Run(); err != nil || out.Len() == 0 {
		s.Log.Error("voice excerpt failed", "kind", fmtError(err))
		fail(w, http.StatusBadGateway, "excerpt_unavailable")
		return
	}
	w.Header().Set("Content-Type", "audio/wav")
	w.Header().Set("Cache-Control", "private, max-age=300")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="speaker-%d.wav"`, sp.ID))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(out.Bytes()))
}
