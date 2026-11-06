package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"snitch/internal/library"
)

type metadataLocation struct {
	Country  string `json:"country"`
	Timezone string `json:"timezone"`
}

func (s *Server) getMetadataLocation(w http.ResponseWriter, r *http.Request) {
	var v metadataLocation
	if err := s.Pool.QueryRow(r.Context(), `SELECT country,timezone FROM clip_metadata_settings WHERE id=true`).Scan(&v.Country, &v.Timezone); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, 200, v)
}
func (s *Server) putMetadataLocation(w http.ResponseWriter, r *http.Request) {
	var v metadataLocation
	if !decode(w, r, &v) {
		return
	}
	v.Country = strings.ToUpper(strings.TrimSpace(v.Country))
	if len(v.Country) != 2 || v.Country[0] < 'A' || v.Country[0] > 'Z' || v.Country[1] < 'A' || v.Country[1] > 'Z' {
		fail(w, 400, "invalid_filter")
		return
	}
	if _, err := time.LoadLocation(v.Timezone); err != nil || v.Timezone == "" || v.Timezone == "Local" {
		fail(w, 400, "invalid_filter")
		return
	}
	if _, err := s.Pool.Exec(r.Context(), `UPDATE clip_metadata_settings SET country=$1,timezone=$2 WHERE id=true`, v.Country, v.Timezone); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, 200, v)
}

// detectedMetadata keeps what was detected automatically (container tags, file name) the first time a field is
// edited by hand, so the edit can be undone. A field is restorable only when its *Available flag is set.
type detectedMetadata struct {
	GameAvailable       bool       `json:"game_available"`
	Game                *string    `json:"game"`
	GameOrigin          *string    `json:"game_origin"`
	RecordedAtAvailable bool       `json:"recorded_at_available"`
	RecordedAt          *time.Time `json:"recorded_at"`
	RecordedAtOrigin    *string    `json:"recorded_at_origin"`
}

func decodeDetected(raw []byte) *detectedMetadata {
	if len(raw) == 0 {
		return nil
	}
	var d detectedMetadata
	if json.Unmarshal(raw, &d) != nil {
		return nil
	}
	return &d
}

// maxClipTitleRunes bounds the display title in characters, not bytes.
const maxClipTitleRunes = 200

// spoofingRune is a character a title must not carry: control characters, line/paragraph separators and the
// bidirectional overrides and isolates that reorder or hide the text around them. Joiners stay (emoji sequences).
func spoofingRune(r rune) bool {
	return unicode.IsControl(r) || r == '\u2028' || r == '\u2029' || (r >= '\u202a' && r <= '\u202e') || (r >= '\u2066' && r <= '\u2069')
}

// cleanClipTitle trims a display title and refuses control characters and over-long values; "" clears it.
func cleanClipTitle(raw string) (*string, bool) {
	v := strings.TrimSpace(raw)
	if v == "" {
		return nil, true
	}
	if !utf8.ValidString(v) || utf8.RuneCountInString(v) > maxClipTitleRunes || strings.IndexFunc(v, spoofingRune) >= 0 {
		return nil, false
	}
	return &v, true
}

func (s *Server) putClipMetadata(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		fail(w, 400, "invalid_clip_id")
		return
	}
	var in struct {
		Game                *string    `json:"game"`
		RecordedAt          *time.Time `json:"recorded_at"`
		GameChanged         bool       `json:"game_changed"`
		RecordedAtChanged   bool       `json:"recorded_at_changed"`
		DisplayTitle        *string    `json:"display_title"`
		DisplayTitleChanged bool       `json:"display_title_changed"`
		GameReset           bool       `json:"game_reset"`
		RecordedAtReset     bool       `json:"recorded_at_reset"`
	}
	if !decode(w, r, &in) {
		return
	}
	if (in.GameReset && in.GameChanged) || (in.RecordedAtReset && in.RecordedAtChanged) {
		fail(w, 400, "invalid_metadata_request")
		return
	}
	if in.Game != nil {
		v := strings.TrimSpace(*in.Game)
		if len([]rune(v)) > 200 {
			fail(w, 400, "invalid_filter")
			return
		}
		if v == "" {
			in.Game = nil
		} else {
			in.Game = &v
		}
	}
	var title *string
	if in.DisplayTitleChanged {
		raw := ""
		if in.DisplayTitle != nil {
			raw = *in.DisplayTitle
		}
		var ok bool
		if title, ok = cleanClipTitle(raw); !ok {
			fail(w, 400, "invalid_clip_title")
			return
		}
	}
	ctx := r.Context()
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer tx.Rollback(ctx)
	var game, gameOrigin, timeOrigin *string
	var captured *time.Time
	var gameEdited, timeEdited bool
	var rawDetected []byte
	err = tx.QueryRow(ctx, `SELECT game,game_origin,game_edited,capture_time,capture_time_origin,capture_time_edited,detected_metadata FROM clips WHERE id=$1 FOR UPDATE`, id).
		Scan(&game, &gameOrigin, &gameEdited, &captured, &timeOrigin, &timeEdited, &rawDetected)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "clip_not_found")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	detected := decodeDetected(rawDetected)
	if detected == nil {
		detected = &detectedMetadata{}
	}
	// The first manual edit of a field saves its automatically detected value.
	if in.GameChanged && !gameEdited && !detected.GameAvailable {
		detected.GameAvailable, detected.Game, detected.GameOrigin = true, game, gameOrigin
	}
	if in.RecordedAtChanged && !timeEdited && !detected.RecordedAtAvailable {
		detected.RecordedAtAvailable, detected.RecordedAt, detected.RecordedAtOrigin = true, captured, timeOrigin
	}
	switch {
	case in.GameReset:
		if !detected.GameAvailable {
			fail(w, 409, "metadata_reset_unavailable")
			return
		}
		game, gameOrigin, gameEdited = detected.Game, detected.GameOrigin, false
		// Back on the automatic value: the snapshot is stale as soon as a rescan may change it, so the next
		// manual edit takes a fresh one.
		detected.GameAvailable, detected.Game, detected.GameOrigin = false, nil, nil
	case in.GameChanged:
		game, gameOrigin, gameEdited = in.Game, ptr("manual"), true
	}
	switch {
	case in.RecordedAtReset:
		if !detected.RecordedAtAvailable {
			fail(w, 409, "metadata_reset_unavailable")
			return
		}
		captured, timeOrigin, timeEdited = detected.RecordedAt, detected.RecordedAtOrigin, false
		detected.RecordedAtAvailable, detected.RecordedAt, detected.RecordedAtOrigin = false, nil, nil
	case in.RecordedAtChanged:
		captured, timeOrigin, timeEdited = in.RecordedAt, ptr("manual"), true
	}
	var encoded []byte
	if detected.GameAvailable || detected.RecordedAtAvailable {
		if encoded, err = json.Marshal(detected); err != nil {
			s.internal(w, err)
			return
		}
	}
	if _, err = tx.Exec(ctx, `UPDATE clips SET game=$2,game_origin=$3,game_edited=$4,capture_time=$5,capture_time_origin=$6,capture_time_edited=$7,display_title=CASE WHEN $8 THEN $9 ELSE display_title END,detected_metadata=$10 WHERE id=$1`,
		id, game, gameOrigin, gameEdited, captured, timeOrigin, timeEdited, in.DisplayTitleChanged, title, encoded); err != nil {
		s.internal(w, err)
		return
	}
	if err = tx.Commit(ctx); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, 200, map[string]string{"status": "ok"})
}

func ptr[T any](v T) *T { return &v }

// Reprocess one clip per request so the server's write timeout cannot cut off
// a large library halfway through a single response. The caller advances by ID.
func (s *Server) reprocessClipMetadata(w http.ResponseWriter, r *http.Request) {
	// A large clip can take longer than the server's ordinary response deadline
	// to hash. The request remains cancellable when the caller disconnects.
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(30 * time.Minute))
	after := int64(0)
	if raw := r.URL.Query().Get("after"); raw != "" {
		var err error
		after, err = strconv.ParseInt(raw, 10, 64)
		if err != nil || after < 0 {
			fail(w, 400, "invalid_filter")
			return
		}
	}
	var id int64
	var name string
	var expectedHash []byte
	err := s.Pool.QueryRow(r.Context(), `SELECT id,original_name,content_sha256 FROM clips WHERE id>$1 ORDER BY id LIMIT 1`, after).Scan(&id, &name, &expectedHash)
	if errors.Is(err, pgx.ErrNoRows) {
		reply(w, 200, map[string]any{"updated": 0, "unavailable": 0, "next": after, "done": true})
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	result := map[string]any{"updated": 0, "unavailable": 0, "next": id, "done": false}
	if len(expectedHash) == 0 {
		result["unavailable"] = 1
		reply(w, 200, result)
		return
	}
	// The hash is checked inside the selector so a changed copy is skipped and
	// the next intact copy can be tried instead of giving up on the whole clip.
	var matchedInfo os.FileInfo
	path, err := s.findClipMedia(r.Context(), id, func(path string) bool {
		beforeHash, statErr := os.Stat(path)
		if statErr != nil {
			return false
		}
		actual, hashErr := library.HashFile(r.Context(), path)
		afterHash, statErr := os.Stat(path)
		if hashErr != nil || statErr != nil || beforeHash.Size() != afterHash.Size() || !beforeHash.ModTime().Equal(afterHash.ModTime()) || !bytes.Equal(actual[:], expectedHash) {
			return false
		}
		matchedInfo = afterHash
		return true
	})
	if errors.Is(err, os.ErrNotExist) {
		result["unavailable"] = 1
		reply(w, 200, result)
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	before, err := os.Stat(path)
	if err != nil || matchedInfo == nil || before.Size() != matchedInfo.Size() || !before.ModTime().Equal(matchedInfo.ModTime()) {
		result["unavailable"] = 1
		reply(w, 200, result)
		return
	}
	meta, err := library.ProbeFile(r.Context(), s.Config.ProbePath, path)
	if err != nil {
		result["unavailable"] = 1
		reply(w, 200, result)
		return
	}
	afterProbe, err := os.Stat(path)
	if err != nil || before.Size() != afterProbe.Size() || !before.ModTime().Equal(afterProbe.ModTime()) {
		result["unavailable"] = 1
		reply(w, 200, result)
		return
	}
	game, gameOrigin, date, dateOrigin, err := library.CaptureMetadata(r.Context(), s.Pool, name, meta.ContainerTags)
	if err != nil {
		s.internal(w, err)
		return
	}
	_, err = s.Pool.Exec(r.Context(), `UPDATE clips SET game=CASE WHEN game_edited THEN game ELSE NULLIF($2,'') END, game_origin=CASE WHEN game_edited THEN game_origin ELSE NULLIF($3,'') END, capture_time=CASE WHEN capture_time_edited THEN capture_time ELSE $4 END, capture_time_origin=CASE WHEN capture_time_edited THEN capture_time_origin ELSE NULLIF($5,'') END WHERE id=$1`, id, game, gameOrigin, date, dateOrigin)
	if err != nil {
		s.internal(w, err)
		return
	}
	result["updated"] = 1
	reply(w, 200, result)
}
