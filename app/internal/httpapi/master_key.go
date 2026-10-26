package httpapi

import (
	"encoding/base64"
	"net/http"
)

// isAdministrator is the one place that decides whether a role may use the administration.
// The Super-administrator role (ticket #82) will be accepted here as well, exactly like Administrator.
func isAdministrator(role string) bool { return role == "admin" }

// isSuperAdministrator is the one place that decides who may do what only the owner of the installation does
// (exporting the master key, later also Discord and the registration policy). Until the role exists, every
// Administrator qualifies; ticket #82 narrows this single function.
func isSuperAdministrator(role string) bool { return isAdministrator(role) }

// exportMasterKey hands the master key to the person who must keep it separately from the database.
// The key is never logged: only the fact and the author are.
func (s *Server) exportMasterKey(w http.ResponseWriter, r *http.Request) {
	ss := r.Context().Value(sessionKey{}).(session)
	s.Log.Info("master key exported", "user_id", ss.User.ID, "origin", s.Config.MasterKeyOrigin)
	reply(w, 200, map[string]string{
		"master_key": base64.StdEncoding.EncodeToString(s.Config.MasterKey),
		"origin":     s.Config.MasterKeyOrigin,
		"warning":    "master_key_not_replayable",
	})
}
