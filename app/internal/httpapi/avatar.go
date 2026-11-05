package httpapi

import (
	"bytes"
	"context"
	"errors"
	"image"
	_ "image/jpeg"
	"image/png"
	"io"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5"
)

// Where an account's displayed photo comes from. The choice is persisted in avatar_preferences; the two
// possible images (a personal one and the one downloaded from Discord) live side by side in account_avatars,
// so choosing one source never destroys the other.
const (
	avatarCustom  = "custom"
	avatarDiscord = "discord"
	avatarNone    = "none"
)

const maxAvatarBytes = 5 << 20

var errInvalidAvatar = errors.New("invalid avatar")

// normalizeAvatar validates an uploaded or downloaded image and returns only re-encoded pixels: the central
// square at 256x256, as PNG, without any metadata.
func normalizeAvatar(data []byte) ([]byte, error) {
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || (format != "jpeg" && format != "png") || cfg.Width < 1 || cfg.Height < 1 || int64(cfg.Width)*int64(cfg.Height) > 16000000 {
		return nil, errInvalidAvatar
	}
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, errInvalidAvatar
	}
	side := min(cfg.Width, cfg.Height)
	out := image.NewNRGBA(image.Rect(0, 0, 256, 256))
	for y := 0; y < 256; y++ {
		for x := 0; x < 256; x++ {
			out.Set(x, y, src.At(src.Bounds().Min.X+(cfg.Width-side)/2+x*side/256, src.Bounds().Min.Y+(cfg.Height-side)/2+y*side/256))
		}
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, out); err != nil {
		return nil, err
	}
	return encoded.Bytes(), nil
}

// lockAvatarAccount serializes every avatar transition of one account (upload, removal, source switch, Discord
// synchronization, unlink): they all take this row lock first, so they apply one after the other and a
// synchronization can never interleave with a personal upload.
func lockAvatarAccount(ctx context.Context, tx pgx.Tx, id int64) error {
	var locked int64
	if err := tx.QueryRow(ctx, `SELECT id FROM users WHERE id=$1 FOR UPDATE`, id).Scan(&locked); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `INSERT INTO avatar_preferences(user_id) VALUES($1) ON CONFLICT DO NOTHING`, id)
	return err
}

// setAvatarSource records the displayed source and invalidates the browser's copy.
func setAvatarSource(ctx context.Context, tx pgx.Tx, id int64, source string) error {
	_, err := tx.Exec(ctx, `UPDATE avatar_preferences SET source=$2,version=version+1 WHERE user_id=$1`, id, source)
	return err
}

func hasAvatarImage(ctx context.Context, tx pgx.Tx, id int64, source string) (bool, error) {
	var ok bool
	err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM account_avatars WHERE user_id=$1 AND source=$2)`, id, source).Scan(&ok)
	return ok, err
}

// adoptDiscordAvatar makes Discord the displayed source of an account that has just been created or linked,
// unless it already has a personal photo, which always wins.
func adoptDiscordAvatar(ctx context.Context, tx pgx.Tx, id int64) error {
	if err := lockAvatarAccount(ctx, tx, id); err != nil {
		return err
	}
	custom, err := hasAvatarImage(ctx, tx, id, avatarCustom)
	if err != nil || custom {
		return err
	}
	return setAvatarSource(ctx, tx, id, avatarDiscord)
}

// dropDiscordAvatar is the Discord side of an unlink: the downloaded image and the synchronization state go
// away (the identity they came from is no longer the account's), the personal photo stays, and an account that
// displayed the Discord image falls back to its personal photo, or to initials.
func dropDiscordAvatar(ctx context.Context, tx pgx.Tx, id int64) error {
	if err := lockAvatarAccount(ctx, tx, id); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM account_avatars WHERE user_id=$1 AND source='discord'`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE avatar_preferences SET discord_checked_at=NULL,discord_next_attempt_at=NULL,discord_failures=0,discord_last_error=NULL WHERE user_id=$1`, id); err != nil {
		return err
	}
	var source string
	if err := tx.QueryRow(ctx, `SELECT source FROM avatar_preferences WHERE user_id=$1`, id).Scan(&source); err != nil || source != avatarDiscord {
		return err
	}
	custom, err := hasAvatarImage(ctx, tx, id, avatarCustom)
	if err != nil {
		return err
	}
	next := avatarNone
	if custom {
		next = avatarCustom
	}
	return setAvatarSource(ctx, tx, id, next)
}

func (s *Server) accountAvatar(w http.ResponseWriter, r *http.Request) {
	id := r.Context().Value(sessionKey{}).(session).User.ID
	switch r.Method {
	case http.MethodGet:
		s.serveAccountAvatar(w, r, id)
	case http.MethodDelete:
		// Removing the personal photo: the account goes back to Discord when it is linked and has an image
		// from there, to initials otherwise.
		err := s.avatarTx(r.Context(), id, func(tx pgx.Tx) error {
			if _, err := tx.Exec(r.Context(), `DELETE FROM account_avatars WHERE user_id=$1 AND source='custom'`, id); err != nil {
				return err
			}
			var source string
			if err := tx.QueryRow(r.Context(), `SELECT source FROM avatar_preferences WHERE user_id=$1`, id).Scan(&source); err != nil || source != avatarCustom {
				return err
			}
			next := avatarNone
			if ok, err := hasAvatarImage(r.Context(), tx, id, avatarDiscord); err != nil {
				return err
			} else if ok {
				next = avatarDiscord
			}
			return setAvatarSource(r.Context(), tx, id, next)
		})
		if err != nil {
			s.internal(w, err)
			return
		}
		w.WriteHeader(204)
	case http.MethodPut:
		data, err := io.ReadAll(io.LimitReader(r.Body, maxAvatarBytes+1))
		if err != nil || len(data) > maxAvatarBytes {
			fail(w, 413, "avatar_too_large")
			return
		}
		encoded, err := normalizeAvatar(data)
		if errors.Is(err, errInvalidAvatar) {
			fail(w, 400, "invalid_avatar")
			return
		}
		if err != nil {
			s.internal(w, err)
			return
		}
		// A personal upload is an explicit choice: it becomes the displayed source at once.
		err = s.avatarTx(r.Context(), id, func(tx pgx.Tx) error {
			if _, err := tx.Exec(r.Context(), `INSERT INTO account_avatars(user_id,source,png) VALUES($1,'custom',$2) ON CONFLICT(user_id,source) DO UPDATE SET png=EXCLUDED.png`, id, encoded); err != nil {
				return err
			}
			return setAvatarSource(r.Context(), tx, id, avatarCustom)
		})
		if err != nil {
			s.internal(w, err)
			return
		}
		w.WriteHeader(204)
	}
}

// avatarTx runs one avatar transition atomically, under the account's lock.
func (s *Server) avatarTx(ctx context.Context, id int64, work func(pgx.Tx) error) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := lockAvatarAccount(ctx, tx, id); err != nil {
		return err
	}
	if err := work(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Server) serveAccountAvatar(w http.ResponseWriter, r *http.Request, id int64) {
	var data []byte
	var source string
	var version int64
	err := s.Pool.QueryRow(r.Context(), `SELECT a.png,p.source,p.version FROM avatar_preferences p JOIN account_avatars a ON a.user_id=p.user_id AND a.source=p.source WHERE p.user_id=$1`, id).Scan(&data, &source, &version)
	if errors.Is(err, pgx.ErrNoRows) {
		w.Header().Set("Cache-Control", "private, no-store")
		w.WriteHeader(404)
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	// The browser revalidates at every use, so a refreshed photo shows up everywhere (other tabs and
	// devices included) without clearing the cache, and an unchanged one costs a 304.
	etag := `"` + strconv.FormatInt(id, 10) + "-" + source + "-" + strconv.FormatInt(version, 10) + `"`
	w.Header().Set("Cache-Control", "private, no-cache")
	w.Header().Set("ETag", etag)
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	_, _ = w.Write(data)
}

// The validated account link is resolved on every read. No photo is copied onto the Personne.
func (s *Server) personAvatar(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		fail(w, 400, "invalid_request")
		return
	}
	var userID int64
	err = s.Pool.QueryRow(r.Context(), `SELECT u.id FROM persons p JOIN users u ON u.id=p.user_id WHERE p.id=$1 AND u.approved AND u.deleted_at IS NULL`, id).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		w.Header().Set("Cache-Control", "private, no-store")
		w.WriteHeader(404)
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	s.serveAccountAvatar(w, r, userID)
}
