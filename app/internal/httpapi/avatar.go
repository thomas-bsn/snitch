package httpapi

import (
	"bytes"
	"errors"
	"image"
	_ "image/jpeg"
	"image/png"
	"io"
	"net/http"

	"github.com/jackc/pgx/v5"
)

func (s *Server) accountAvatar(w http.ResponseWriter, r *http.Request) {
	id := r.Context().Value(sessionKey{}).(session).User.ID
	switch r.Method {
	case http.MethodGet:
		var data []byte
		err := s.Pool.QueryRow(r.Context(), `SELECT png FROM account_avatars WHERE user_id=$1`, id).Scan(&data)
		if errors.Is(err, pgx.ErrNoRows) {
			w.WriteHeader(404)
			return
		}
		if err != nil {
			s.internal(w, err)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(data)
	case http.MethodDelete:
		if _, err := s.Pool.Exec(r.Context(), `DELETE FROM account_avatars WHERE user_id=$1`, id); err != nil {
			s.internal(w, err)
			return
		}
		w.WriteHeader(204)
	case http.MethodPut:
		data, err := io.ReadAll(r.Body)
		if err != nil || len(data) > 5<<20 {
			fail(w, 413, "avatar_too_large")
			return
		}
		cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
		if err != nil || (format != "jpeg" && format != "png") || cfg.Width < 1 || cfg.Height < 1 || int64(cfg.Width)*int64(cfg.Height) > 16000000 {
			fail(w, 400, "invalid_avatar")
			return
		}
		src, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			fail(w, 400, "invalid_avatar")
			return
		}
		// Crop the central square and store only re-encoded pixels, without metadata.
		side := min(cfg.Width, cfg.Height)
		out := image.NewNRGBA(image.Rect(0, 0, 256, 256))
		for y := 0; y < 256; y++ {
			for x := 0; x < 256; x++ {
				out.Set(x, y, src.At(src.Bounds().Min.X+(cfg.Width-side)/2+x*side/256, src.Bounds().Min.Y+(cfg.Height-side)/2+y*side/256))
			}
		}
		var encoded bytes.Buffer
		if err := png.Encode(&encoded, out); err != nil {
			s.internal(w, err)
			return
		}
		if _, err := s.Pool.Exec(r.Context(), `INSERT INTO account_avatars(user_id,png) VALUES($1,$2) ON CONFLICT(user_id) DO UPDATE SET png=EXCLUDED.png`, id, encoded.Bytes()); err != nil {
			s.internal(w, err)
			return
		}
		w.WriteHeader(204)
	}
}
