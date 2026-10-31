package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// registerVoiceProfileRoutes declares the account link and voice profile deletion
// routes (phase 5).
func (s *Server) registerVoiceProfileRoutes(mux *http.ServeMux) {
	mux.Handle("GET /api/me/person", s.protected(false, http.HandlerFunc(s.getMyPerson)))
	mux.Handle("PUT /api/admin/persons/{id}/account", s.protected(true, http.HandlerFunc(s.linkPersonAccount)))
	mux.Handle("DELETE /api/admin/persons/{id}/account", s.protected(true, http.HandlerFunc(s.unlinkPersonAccount)))
	mux.Handle("DELETE /api/persons/{id}/voice-profile", s.protected(false, http.HandlerFunc(s.deleteVoiceProfile)))
}

// getMyPerson returns the Person linked to the signed-in account. The clips where this
// Person is identified are listed by the library filter, which applies visibility.
func (s *Server) getMyPerson(w http.ResponseWriter, r *http.Request) {
	ss := r.Context().Value(sessionKey{}).(session)
	v, err := scanPerson(s.Pool.QueryRow(r.Context(), `SELECT `+personColumns+` FROM persons p WHERE p.user_id=$1`, ss.User.ID))
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, http.StatusNotFound, "no_linked_person")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, v)
}

func (s *Server) linkPersonAccount(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "invalid_person_id")
	if !ok {
		return
	}
	var input struct {
		Username string `json:"username"`
	}
	if !decode(w, r, &input) {
		return
	}
	username := strings.TrimSpace(input.Username)
	if username == "" || len(username) > 100 {
		fail(w, http.StatusBadRequest, "invalid_account")
		return
	}
	ss := r.Context().Value(sessionKey{}).(session)
	ctx := r.Context()
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer tx.Rollback(ctx)
	var personName string
	var current *int64
	err = tx.QueryRow(ctx, `SELECT display_name,user_id FROM persons WHERE id=$1 FOR UPDATE`, id).Scan(&personName, &current)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, http.StatusNotFound, "person_not_found")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	var userID int64
	if err = tx.QueryRow(ctx, `SELECT id FROM users WHERE username=$1`, username).Scan(&userID); errors.Is(err, pgx.ErrNoRows) {
		fail(w, http.StatusNotFound, "user_not_found")
		return
	} else if err != nil {
		s.internal(w, err)
		return
	}
	// A replacement is the Administrator's explicit decision: keep who the link leaves.
	var prevID *int64
	var prevName string
	if current != nil && *current != userID {
		prevID = current
		if err = tx.QueryRow(ctx, `SELECT COALESCE((SELECT username FROM users WHERE id=$1),'')`, *current).Scan(&prevName); err != nil {
			s.internal(w, err)
			return
		}
	}
	if _, err = tx.Exec(ctx, `UPDATE persons SET user_id=$2,updated_at=now() WHERE id=$1`, id, userID); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			fail(w, http.StatusConflict, "account_already_linked")
			return
		}
		s.internal(w, err)
		return
	}
	if err = recordPersonLinkEvent(ctx, tx, personLinkEvent{Action: "link", PersonID: id, PersonName: personName, UserID: &userID, Username: username, PreviousUserID: prevID, PreviousUsername: prevName, ActorID: &ss.User.ID, ActorUsername: ss.User.Username}); err != nil {
		s.internal(w, err)
		return
	}
	if err = tx.Commit(ctx); err != nil {
		s.internal(w, err)
		return
	}
	s.replyPerson(w, r, id)
}

func (s *Server) unlinkPersonAccount(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "invalid_person_id")
	if !ok {
		return
	}
	ss := r.Context().Value(sessionKey{}).(session)
	ctx := r.Context()
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer tx.Rollback(ctx)
	var personName string
	var current *int64
	err = tx.QueryRow(ctx, `SELECT display_name,user_id FROM persons WHERE id=$1 FOR UPDATE`, id).Scan(&personName, &current)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, http.StatusNotFound, "person_not_found")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	if current == nil {
		// Nothing to unlink: the call is idempotent and leaves no trace.
		s.replyPerson(w, r, id)
		return
	}
	var name string
	if err = tx.QueryRow(ctx, `SELECT COALESCE((SELECT username FROM users WHERE id=$1),'')`, *current).Scan(&name); err != nil {
		s.internal(w, err)
		return
	}
	if _, err = tx.Exec(ctx, `UPDATE persons SET user_id=NULL,updated_at=now() WHERE id=$1`, id); err != nil {
		s.internal(w, err)
		return
	}
	if err = recordPersonLinkEvent(ctx, tx, personLinkEvent{Action: "unlink", PersonID: id, PersonName: personName, UserID: current, Username: name, ActorID: &ss.User.ID, ActorUsername: ss.User.Username}); err != nil {
		s.internal(w, err)
		return
	}
	if err = tx.Commit(ctx); err != nil {
		s.internal(w, err)
		return
	}
	s.replyPerson(w, r, id)
}

func (s *Server) replyPerson(w http.ResponseWriter, r *http.Request, id int64) {
	v, err := scanPerson(s.Pool.QueryRow(r.Context(), `SELECT `+personColumns+` FROM persons p WHERE p.id=$1`, id))
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, v)
}

type voiceProfileDeletion struct {
	Samples      int       `json:"samples"`
	Fingerprints int       `json:"fingerprints"`
	Attributions int       `json:"attributions"`
	DeletedAt    time.Time `json:"deleted_at"`
}

// deleteVoiceProfile erases the voice profile of a Person: every validated sample
// (reference, active or retired), the fingerprints of the voices attributed to them
// and every attribution naming them, which become unknown. The clips, the Person and
// the other Persons' data are untouched. The opt-out (persons.voice_profile_deleted_at)
// is committed with the erasure and keeps matching from rebuilding the profile.
// Allowed to the account linked to the Person and to administrators.
func (s *Server) deleteVoiceProfile(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "invalid_person_id")
	if !ok {
		return
	}
	ss := r.Context().Value(sessionKey{}).(session)
	ctx := context.WithoutCancel(r.Context())
	result, microphoneClips, status, code, err := s.eraseVoiceProfile(ctx, id, ss)
	if err != nil {
		s.internal(w, err)
		return
	}
	if status != 0 {
		fail(w, status, code)
		return
	}
	// The deleted profile can no longer be a candidate; microphone designations naming
	// the Person are re-resolved and no longer attributed.
	s.matchingChanged(ctx, 0)
	author := s.authorOf(r)
	for _, clipID := range microphoneClips {
		if _, applyErr := s.applyMicrophone(ctx, clipID, author); applyErr != nil {
			s.Log.Error("microphone track update failed after voice profile deletion", "clip_id", clipID, "error", applyErr)
		}
	}
	reply(w, http.StatusOK, result)
}

func (s *Server) eraseVoiceProfile(ctx context.Context, personID int64, ss session) (voiceProfileDeletion, []int64, int, string, error) {
	var out voiceProfileDeletion
	// Same order as matching: process mutex, then the database lock, so a matching run
	// in flight finishes before the erasure and cannot write after it.
	voiceMatchingLock.Lock()
	defer voiceMatchingLock.Unlock()
	tx, err := s.beginLocked(ctx, lockVoiceMatching)
	if err != nil {
		return out, nil, 0, "", err
	}
	defer tx.Rollback(ctx)
	var name string
	var linked *int64
	var already *time.Time
	if err = tx.QueryRow(ctx, `SELECT display_name,user_id,voice_profile_deleted_at FROM persons WHERE id=$1 FOR UPDATE`, personID).Scan(&name, &linked, &already); errors.Is(err, pgx.ErrNoRows) {
		return out, nil, http.StatusNotFound, "person_not_found", nil
	} else if err != nil {
		return out, nil, 0, "", err
	}
	if !isAdministrator(ss.User.Role) && (linked == nil || *linked != ss.User.ID) {
		return out, nil, http.StatusForbidden, "forbidden", nil
	}
	// Voices whose effective attribution (human validation first) is this Person: their
	// fingerprints are the Person's voice data.
	rows, err := tx.Query(ctx, `SELECT DISTINCT ls.id,ls.clip_id,ls.kind FROM speaker_attributions a JOIN local_speakers ls ON ls.id=a.local_speaker_id
WHERE a.person_id=$1 AND (a.validation_state='validated' OR NOT EXISTS (SELECT 1 FROM speaker_attributions v WHERE v.local_speaker_id=a.local_speaker_id AND v.validation_state='validated'))`, personID)
	if err != nil {
		return out, nil, 0, "", err
	}
	var speakerIDs, microphoneClips []int64
	for rows.Next() {
		var speakerID, clipID int64
		var kind string
		if err = rows.Scan(&speakerID, &clipID, &kind); err != nil {
			rows.Close()
			return out, nil, 0, "", err
		}
		speakerIDs = append(speakerIDs, speakerID)
		if kind == "microphone" {
			microphoneClips = append(microphoneClips, clipID)
		}
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return out, nil, 0, "", err
	}
	samples, err := tx.Exec(ctx, `DELETE FROM voice_samples WHERE person_id=$1`, personID)
	if err != nil {
		return out, nil, 0, "", err
	}
	fingerprints, err := tx.Exec(ctx, `DELETE FROM voice_fingerprints WHERE local_speaker_id=ANY($1)`, speakerIDs)
	if err != nil {
		return out, nil, 0, "", err
	}
	// The history of each voice keeps the fact, never any voice data.
	attributions, err := tx.Exec(ctx, `WITH d AS (DELETE FROM speaker_attributions WHERE person_id=$1 RETURNING local_speaker_id,decision_level)
INSERT INTO attribution_events(local_speaker_id,clip_id,action,old_person_id,old_person_name,old_decision_level,new_decision_level,author_id,author_username)
SELECT d.local_speaker_id,ls.clip_id,'delete_profile',$1,$2,d.decision_level,CASE WHEN EXISTS (SELECT 1 FROM speaker_attributions v WHERE v.local_speaker_id=d.local_speaker_id AND v.validation_state='validated' AND v.person_id<>$1) THEN 'human' ELSE 'unknown' END,$3,$4
FROM d JOIN local_speakers ls ON ls.id=d.local_speaker_id`, personID, name, ss.User.ID, ss.User.Username)
	if err != nil {
		return out, nil, 0, "", err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM speaker_attribution_candidates WHERE person_id=$1`, personID); err != nil {
		return out, nil, 0, "", err
	}
	if err = tx.QueryRow(ctx, `UPDATE persons SET voice_profile_deleted_at=COALESCE(voice_profile_deleted_at,now()),updated_at=now() WHERE id=$1 RETURNING voice_profile_deleted_at`, personID).Scan(&out.DeletedAt); err != nil {
		return out, nil, 0, "", err
	}
	out.Samples, out.Fingerprints, out.Attributions = int(samples.RowsAffected()), int(fingerprints.RowsAffected()), int(attributions.RowsAffected())
	if already == nil || out.Samples+out.Fingerprints+out.Attributions > 0 {
		if _, err = tx.Exec(ctx, `INSERT INTO voice_profile_deletions(person_id,person_name,linked_user_id,deleted_by,deleted_by_username,sample_count,fingerprint_count,attribution_count) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, personID, name, linked, ss.User.ID, ss.User.Username, out.Samples, out.Fingerprints, out.Attributions); err != nil {
			return out, nil, 0, "", err
		}
	}
	return out, microphoneClips, 0, "", tx.Commit(ctx)
}
