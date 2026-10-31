package httpapi

import (
	"bytes"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

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
func (s *Server) putClipMetadata(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		fail(w, 400, "invalid_clip_id")
		return
	}
	var in struct {
		Game              *string    `json:"game"`
		RecordedAt        *time.Time `json:"recorded_at"`
		GameChanged       bool       `json:"game_changed"`
		RecordedAtChanged bool       `json:"recorded_at_changed"`
	}
	if !decode(w, r, &in) {
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
	tag, err := s.Pool.Exec(r.Context(), `UPDATE clips SET game=CASE WHEN $4 THEN $2 ELSE game END,game_origin=CASE WHEN $4 THEN 'manual' ELSE game_origin END,game_edited=game_edited OR $4,capture_time=CASE WHEN $5 THEN $3 ELSE capture_time END,capture_time_origin=CASE WHEN $5 THEN 'manual' ELSE capture_time_origin END,capture_time_edited=capture_time_edited OR $5 WHERE id=$1`, id, in.Game, in.RecordedAt, in.GameChanged, in.RecordedAtChanged)
	if err != nil {
		s.internal(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		fail(w, 404, "clip_not_found")
		return
	}
	reply(w, 200, map[string]string{"status": "ok"})
}

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
