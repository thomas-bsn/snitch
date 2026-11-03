package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// The central Hugging Face token (#194, ADR 0027). The value is sealed in the settings table under hfTokenKey; it is
// never returned by an administrator read, never sent with a Worker pulse, and reaches a Worker only through the
// dedicated Worker endpoints, which are behind the Worker bearer check (revoked or unknown Workers get nothing).
const hfTokenKey = "hf_token_central"

// hfTokenNone is the version of "no central token" (never configured, or unreadable). A Worker never acknowledges it.
const hfTokenNone = "0"

// hfTokenFormat is the shape Hugging Face issues; anything else is refused without echoing it.
var hfTokenFormat = regexp.MustCompile(`^hf_[A-Za-z0-9]{8,200}$`)

// hfTokenState is what the settings row holds, sealed as a whole. The version changes at every save and every
// deletion, so a Worker compares versions for equality only and never has to order them.
type hfTokenState struct {
	Version string `json:"version"`
	Token   string `json:"token"`
}

// hfRead reads the stored state. An absent, empty or unreadable row counts as no token at all (version "0"), so a
// lost master key never makes a Worker purge its copy by mistake. The flag reports an unreadable row.
func (s *Server) hfRead(ctx context.Context) (hfTokenState, bool, error) {
	raw, err := s.Store.Get(ctx, hfTokenKey)
	if err != nil {
		return hfTokenState{}, false, err
	}
	state := hfTokenState{Version: hfTokenNone}
	if raw != "" {
		var parsed hfTokenState
		if json.Unmarshal([]byte(raw), &parsed) == nil && parsed.Version != "" {
			state = parsed
		}
	}
	unreadable := false
	if raw == "" {
		names, err := s.Store.Unreadable(ctx)
		if err != nil {
			return hfTokenState{}, false, err
		}
		for _, name := range names {
			unreadable = unreadable || name == hfTokenKey
		}
	}
	return state, unreadable, nil
}

// hfNextVersion returns a version that differs from the previous one, derived from the clock so that it is never
// reused even when the previous value cannot be read.
func hfNextVersion(previous string) string {
	next := time.Now().UnixNano()
	if old, err := strconv.ParseInt(previous, 10, 64); err == nil && old >= next {
		next = old + 1
	}
	return strconv.FormatInt(next, 10)
}

func hfFingerprint(token string) string {
	if token == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(token))
	return "sha256:" + hex.EncodeToString(sum[:6])
}

func (s *Server) saveHFState(r *http.Request, next hfTokenState) error {
	encoded, err := json.Marshal(next)
	if err != nil {
		return err
	}
	return s.Store.Set(r.Context(), hfTokenKey, string(encoded))
}

// getHFToken tells an administrator whether a central token is configured, its fingerprint, and what each Worker
// acknowledged. It never returns the value.
func (s *Server) getHFToken(w http.ResponseWriter, r *http.Request) {
	state, unreadable, err := s.hfRead(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	rows, err := s.Pool.Query(r.Context(), `SELECT w.id,w.name,COALESCE(h.acked_version,'0'),COALESCE(h.source,''),COALESCE(h.error_code,'') FROM workers w LEFT JOIN worker_hf_token h ON h.worker_id=w.id WHERE w.revoked_at IS NULL ORDER BY w.created_at,w.id`)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer rows.Close()
	workers := []map[string]string{}
	for rows.Next() {
		var id, name, acked, source, errorCode string
		if err = rows.Scan(&id, &name, &acked, &source, &errorCode); err != nil {
			s.internal(w, err)
			return
		}
		// Synchronised when the Worker acknowledged the current version; otherwise pending (not synchronised),
		// or in error when it reported the cause of its last failed application.
		stateName := "pending"
		switch {
		case acked == state.Version:
			stateName = "synced"
		case errorCode != "":
			stateName = "error"
		}
		workers = append(workers, map[string]string{"id": id, "name": name, "source": source, "state": stateName, "error_code": errorCode})
	}
	if err = rows.Err(); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{
		"configured":  state.Token != "",
		"fingerprint": hfFingerprint(state.Token),
		"unreadable":  unreadable,
		"workers":     workers,
	})
}

// putHFToken stores a new central token, replacing the previous one. Only the Super-administrator reaches it.
func (s *Server) putHFToken(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Token string `json:"token"`
	}
	if !decode(w, r, &input) {
		return
	}
	if !hfTokenFormat.MatchString(input.Token) {
		fail(w, http.StatusBadRequest, "invalid_hf_token")
		return
	}
	current, _, err := s.hfRead(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	next := hfTokenState{Version: hfNextVersion(current.Version), Token: input.Token}
	if err = s.saveHFState(r, next); err != nil {
		s.internal(w, err)
		return
	}
	ss := r.Context().Value(sessionKey{}).(session)
	// The value and its fingerprint are never logged: the author and the new version identify the change.
	s.Log.Info("hf token saved", "user_id", ss.User.ID, "version", next.Version)
	w.WriteHeader(http.StatusNoContent)
}

// deleteHFToken removes the central token; Workers purge their central copy at the next version. Their local token,
// if they have one, is kept.
func (s *Server) deleteHFToken(w http.ResponseWriter, r *http.Request) {
	current, unreadable, err := s.hfRead(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	if current.Token == "" && !unreadable {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	next := hfTokenState{Version: hfNextVersion(current.Version)}
	if err = s.saveHFState(r, next); err != nil {
		s.internal(w, err)
		return
	}
	ss := r.Context().Value(sessionKey{}).(session)
	s.Log.Info("hf token deleted", "user_id", ss.User.ID, "version", next.Version)
	w.WriteHeader(http.StatusNoContent)
}

// workerHFTokenVersion is the only thing a Worker learns without the value: the version to compare with its own.
func (s *Server) workerHFTokenVersion(w http.ResponseWriter, r *http.Request) {
	state, _, err := s.hfRead(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]string{"version": state.Version})
}

// workerHFToken hands the value to an authenticated, non-revoked Worker. A removed or absent token is answered
// with a null value, which is the instruction to purge the central copy.
func (s *Server) workerHFToken(w http.ResponseWriter, r *http.Request) {
	state, _, err := s.hfRead(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	var token *string
	if state.Token != "" {
		token = &state.Token
	}
	w.Header().Set("Cache-Control", "no-store")
	reply(w, http.StatusOK, map[string]any{"version": state.Version, "token": token})
}

// ackWorkerHFToken records what a Worker did with a version. A version the Worker is not asked to apply any more
// is refused, so the Worker fetches the current one.
func (s *Server) ackWorkerHFToken(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Version   string `json:"version"`
		Outcome   string `json:"outcome"`
		Source    string `json:"source"`
		ErrorCode string `json:"error_code"`
	}
	if !decode(w, r, &input) {
		return
	}
	validSource := input.Source == "central" || input.Source == "local" || input.Source == "none"
	switch input.Outcome {
	case "applied":
		validSource = validSource && input.Source == "central"
	case "kept_local":
		validSource = validSource && input.Source == "local"
	case "removed":
		validSource = validSource && input.Source == "none"
	case "failed":
		validSource = validSource && input.ErrorCode == "hf_token_store_failed"
	default:
		validSource = false
	}
	if (input.Outcome != "failed" && input.ErrorCode != "") || !validSource {
		fail(w, http.StatusBadRequest, "invalid_hf_token_ack")
		return
	}
	state, _, err := s.hfRead(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	if input.Version != state.Version || (input.Outcome == "applied" && state.Token == "") || (input.Outcome == "removed" && state.Token != "") || (input.Outcome != "removed" && state.Version == hfTokenNone) {
		fail(w, http.StatusConflict, "stale_hf_token_version")
		return
	}
	// The write lands only for a Worker still known and not revoked: one deleted between the bearer check and here is
	// refused like an unknown Worker. A version that moves during this write leaves the Worker pending for the new
	// one, since the admin view compares acknowledged versions for equality only.
	worker := currentWorker(r)
	var tag pgconn.CommandTag
	if input.Outcome == "failed" {
		tag, err = s.Pool.Exec(r.Context(), `INSERT INTO worker_hf_token(worker_id,source,error_code) SELECT w.id,$2::text,$3::text FROM workers w WHERE w.id=$1::text AND w.revoked_at IS NULL ON CONFLICT(worker_id) DO UPDATE SET source=EXCLUDED.source,error_code=EXCLUDED.error_code,updated_at=now()`, worker.ID, input.Source, input.ErrorCode)
	} else {
		tag, err = s.Pool.Exec(r.Context(), `INSERT INTO worker_hf_token(worker_id,acked_version,source,error_code) SELECT w.id,$2::text,$3::text,'' FROM workers w WHERE w.id=$1::text AND w.revoked_at IS NULL ON CONFLICT(worker_id) DO UPDATE SET acked_version=EXCLUDED.acked_version,source=EXCLUDED.source,error_code='',updated_at=now()`, worker.ID, input.Version, input.Source)
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		fail(w, http.StatusUnauthorized, "worker_token_invalid")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
