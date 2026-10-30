package httpapi

import (
	"net/http"
	"slices"
)

// supportedLanguages lists the interface languages an account can choose.
var supportedLanguages = []string{"en", "fr"}

type accountPreferences struct {
	Language *string `json:"language"`
}

func (s *Server) getAccountPreferences(w http.ResponseWriter, r *http.Request) {
	ss := r.Context().Value(sessionKey{}).(session)
	var language *string
	if err := s.Pool.QueryRow(r.Context(), `SELECT language FROM users WHERE id=$1`, ss.User.ID).Scan(&language); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, 200, accountPreferences{Language: language})
}

func (s *Server) putAccountPreferences(w http.ResponseWriter, r *http.Request) {
	ss := r.Context().Value(sessionKey{}).(session)
	var in accountPreferences
	if !decode(w, r, &in) {
		return
	}
	if in.Language == nil {
		fail(w, 400, "invalid_request")
		return
	}
	if !slices.Contains(supportedLanguages, *in.Language) {
		fail(w, 400, "invalid_language")
		return
	}
	if _, err := s.Pool.Exec(r.Context(), `UPDATE users SET language=$1 WHERE id=$2`, *in.Language, ss.User.ID); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, 200, accountPreferences{Language: in.Language})
}

// unlinkDiscord removes the identity only when the account already has a local password.
// The predicate is repeated in the UPDATE so a stale UI state cannot lock out an account.
func (s *Server) unlinkDiscord(w http.ResponseWriter, r *http.Request) {
	ss := r.Context().Value(sessionKey{}).(session)
	tag, err := s.Pool.Exec(r.Context(), `UPDATE users SET discord_id=NULL WHERE id=$1 AND discord_id IS NOT NULL AND password_hash IS NOT NULL`, ss.User.ID)
	if err != nil {
		s.internal(w, err)
		return
	}
	if tag.RowsAffected() == 1 {
		reply(w, 200, map[string]string{"status": "ok"})
		return
	}
	var linked, canUnlink bool
	if err := s.Pool.QueryRow(r.Context(), `SELECT discord_id IS NOT NULL,password_hash IS NOT NULL FROM users WHERE id=$1`, ss.User.ID).Scan(&linked, &canUnlink); err != nil {
		s.internal(w, err)
		return
	}
	if !linked {
		fail(w, 409, "discord_not_linked")
		return
	}
	if !canUnlink {
		fail(w, 409, "discord_local_login_required")
		return
	}
	fail(w, 409, "discord_unlink_unavailable")
}
