package httpapi

import (
	"net/http"
	"slices"
)

// supportedLanguages lists the interface languages an account can choose.
var supportedLanguages = []string{"en", "fr"}

// isAdministrator is the single place deciding whether a role may use the administration
// features. Every "is an administrator" check goes through it so that adding the
// super-administrator role later changes one function only.
func isAdministrator(role string) bool {
	return role == "admin"
}

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
