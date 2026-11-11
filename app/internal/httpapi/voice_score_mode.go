package httpapi

import (
	"context"
	"errors"
	"net/http"
	"time"
)

// Score mode setting (ADR 0064, #391): which score matching uses, the cumulative Reference (default) or the best
// comparable sample. Every change is a new version with its author, and re-runs matching over the stored
// fingerprints; recording the same mode again records nothing.

const lockVoiceScoreMode = "snitch.voice_score_mode"

type scoreModeVersion struct {
	Version   int       `json:"version"`
	Mode      scoreMode `json:"mode"`
	Author    *string   `json:"author"`
	CreatedAt time.Time `json:"created_at"`
}

type scoreModeSettings struct {
	scoreModeVersion
	History []scoreModeVersion `json:"history"`
}

func (s *Server) registerVoiceScoreModeRoutes(mux *http.ServeMux) {
	mux.Handle("GET /api/admin/voice-score-mode", s.protected(true, http.HandlerFunc(s.getVoiceScoreMode)))
	mux.Handle("PUT /api/admin/voice-score-mode", s.protected(true, http.HandlerFunc(s.putVoiceScoreMode)))
}

// loadScoreMode reads the active mode (highest version) and the twenty latest versions.
func loadScoreMode(ctx context.Context, q pgQuerier) (scoreModeSettings, error) {
	var out scoreModeSettings
	rows, err := q.Query(ctx, `SELECT version,mode,author_username,created_at FROM voice_score_modes ORDER BY version DESC LIMIT 20`)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	out.History = []scoreModeVersion{}
	for rows.Next() {
		var v scoreModeVersion
		if err = rows.Scan(&v.Version, &v.Mode, &v.Author, &v.CreatedAt); err != nil {
			return out, err
		}
		out.History = append(out.History, v)
	}
	if err = rows.Err(); err != nil {
		return out, err
	}
	if len(out.History) == 0 {
		return out, errors.New("no voice score mode")
	}
	out.scoreModeVersion = out.History[0]
	return out, nil
}

func (s *Server) getVoiceScoreMode(w http.ResponseWriter, r *http.Request) {
	out, err := loadScoreMode(r.Context(), s.Pool)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, out)
}

func (s *Server) putVoiceScoreMode(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Mode scoreMode `json:"mode"`
	}
	if !decode(w, r, &input) {
		return
	}
	if !input.Mode.valid() {
		fail(w, http.StatusBadRequest, "invalid_voice_score_mode")
		return
	}
	ctx := context.WithoutCancel(r.Context())
	author := s.authorOf(r)
	tx, err := s.beginLocked(ctx, lockVoiceScoreMode)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer tx.Rollback(ctx)
	var version int
	var current scoreMode
	if err = tx.QueryRow(ctx, `SELECT version,mode FROM voice_score_modes ORDER BY version DESC LIMIT 1`).Scan(&version, &current); err != nil {
		s.internal(w, err)
		return
	}
	changed := current != input.Mode
	if changed {
		if _, err = tx.Exec(ctx, `INSERT INTO voice_score_modes(version,mode,author_id,author_username) VALUES($1,$2,$3,$4)`, version+1, input.Mode, author.ID, author.Username); err != nil {
			s.internal(w, err)
			return
		}
	}
	if err = tx.Commit(ctx); err != nil {
		s.internal(w, err)
		return
	}
	// Another score re-examines every stored fingerprint, never the videos.
	if changed {
		s.matchingChanged(ctx, 0) // in the background (#392)
	}
	out, err := loadScoreMode(ctx, s.Pool)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, out)
}
