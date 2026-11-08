package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

// setWorkerEngineGroup changes only the administrator's grouping, never runtime state or orders.
func (s *Server) setWorkerEngineGroup(w http.ResponseWriter, r *http.Request) {
	var input struct {
		GroupName *string `json:"group_name"`
	}
	if !decode(w, r, &input) {
		return
	}
	if input.GroupName == nil {
		fail(w, http.StatusBadRequest, "invalid_request")
		return
	}
	name := strings.TrimSpace(*input.GroupName)
	if utf8.RuneCountInString(name) > 80 || strings.ContainsFunc(name, unicode.IsControl) {
		fail(w, http.StatusBadRequest, "invalid_request")
		return
	}
	tx, err := s.Pool.Begin(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	var id string
	err = tx.QueryRow(r.Context(), `SELECT id FROM workers WHERE id=$1 AND revoked_at IS NULL FOR UPDATE`, r.PathValue("id")).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, http.StatusNotFound, "not_found")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	if name == "" {
		_, err = tx.Exec(r.Context(), `DELETE FROM worker_engine_groups WHERE worker_id=$1`, id)
	} else {
		_, err = tx.Exec(r.Context(), `INSERT INTO worker_engine_groups(worker_id,group_name) VALUES($1,$2) ON CONFLICT(worker_id) DO UPDATE SET group_name=excluded.group_name`, id, name)
	}
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
