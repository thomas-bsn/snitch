package httpapi

import (
	"encoding/base64"
	"net/http"
)

// The roles that decide access, as stored in users.role and exposed by the contract.
const (
	roleSuperAdministrator = "super_admin"
	roleAdministrator      = "admin"
)

// isAdministrator is the one place that decides whether a role may use the administration: Administrator
// and Super-administrator are treated alike for everything that is not reserved to the owner.
func isAdministrator(role string) bool {
	return role == roleAdministrator || role == roleSuperAdministrator
}

// isSuperAdministrator is the one place that decides who may do what only the owner of the installation
// does: exporting the master key, configuring Discord and setting the registration approval (and, with the
// member management, handling the accounts of Administrators).
func isSuperAdministrator(role string) bool { return role == roleSuperAdministrator }

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
