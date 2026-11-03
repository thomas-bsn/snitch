package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5"
)

// Disposition and refusal of a local speaker (ADR 0031). A disposition says whether a voice
// is counted at all: "ignored" hides it everywhere (no speaker on its segments, no
// verification, no group, no rapprochement, no statistic) and "random" labels its segments
// "Random" and keeps it out of verification, groups and rapprochement. Both are reversible
// and journaled (voice_disposition_events). A refusal ("ce n'est pas X") removes the
// attribution of that Person on the voice, takes its samples out of the references, and is
// kept in voice_refusals so that matching never proposes that Person for the voice again.
// Setting a disposition needs the same right as a validation: a signed-in user who can see
// the clip. A voice whose human validation stands is a decision, not noise: it is refused.

const (
	dispositionIgnored = "ignored"
	dispositionRandom  = "random"
	// randomSpeakerLabel is what the segments of a random voice show instead of a speaker.
	randomSpeakerLabel = "Random"
)

// notIgnoredVoice keeps an ignored voice out of a query over local_speakers aliased ls (ADR 0031):
// no speaker on its segments, no verification, no group, no rapprochement, no statistic.
const notIgnoredVoice = `NOT EXISTS (SELECT 1 FROM local_speaker_dispositions dsp WHERE dsp.local_speaker_id=ls.id AND dsp.disposition='ignored')`

type dispositionInput struct {
	Disposition string `json:"disposition"`
}

type dispositionView struct {
	LocalSpeakerID int64   `json:"local_speaker_id"`
	Disposition    *string `json:"disposition"`
}

type refusalInput struct {
	PersonID *int64 `json:"person_id"`
}

func (s *Server) registerVoiceDispositionRoutes(mux *http.ServeMux) {
	mux.Handle("PUT /api/local-speakers/{id}/disposition", s.protected(false, http.HandlerFunc(s.putLocalSpeakerDisposition)))
	mux.Handle("DELETE /api/local-speakers/{id}/disposition", s.protected(false, http.HandlerFunc(s.deleteLocalSpeakerDisposition)))
	mux.Handle("PUT /api/voice-groups/{id}/disposition", s.protected(false, http.HandlerFunc(s.putVoiceGroupDisposition)))
	mux.Handle("POST /api/local-speakers/{id}/refusal", s.protected(false, http.HandlerFunc(s.refuseLocalSpeaker)))
	mux.Handle("DELETE /api/local-speakers/{id}/refusals/{person_id}", s.protected(false, http.HandlerFunc(s.cancelLocalSpeakerRefusal)))
}

// setDispositions applies a disposition to each local speaker, or restores them when
// disposition is "". It returns false, with nothing written, when one of them has a
// human validation and a disposition is requested. An unchanged disposition writes nothing.
func setDispositions(ctx context.Context, tx pgx.Tx, ids []int64, disposition string, author authorRef) (bool, error) {
	for _, id := range ids {
		var clipID int64
		var validated bool
		err := tx.QueryRow(ctx, `SELECT ls.clip_id,EXISTS (SELECT 1 FROM speaker_attributions a WHERE a.local_speaker_id=ls.id AND a.validation_state='validated') FROM local_speakers ls WHERE ls.id=$1 FOR UPDATE OF ls`, id).Scan(&clipID, &validated)
		if err != nil {
			return false, err
		}
		if disposition != "" && validated {
			return false, nil
		}
		var old string
		err = tx.QueryRow(ctx, `SELECT disposition FROM local_speaker_dispositions WHERE local_speaker_id=$1`, id).Scan(&old)
		if errors.Is(err, pgx.ErrNoRows) {
			old = ""
		} else if err != nil {
			return false, err
		}
		if old == disposition {
			continue
		}
		if disposition == "" {
			_, err = tx.Exec(ctx, `DELETE FROM local_speaker_dispositions WHERE local_speaker_id=$1`, id)
		} else {
			_, err = tx.Exec(ctx, `INSERT INTO local_speaker_dispositions(local_speaker_id,disposition,author_id,author_username) VALUES($1,$2,$3,$4)
ON CONFLICT (local_speaker_id) DO UPDATE SET disposition=EXCLUDED.disposition,author_id=EXCLUDED.author_id,author_username=EXCLUDED.author_username,updated_at=now()`, id, disposition, author.ID, author.Username)
		}
		if err != nil {
			return false, err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO voice_disposition_events(local_speaker_id,clip_id,old_disposition,new_disposition,author_id,author_username) VALUES($1,$2,$3,$4,$5,$6)`, id, clipID, old, disposition, author.ID, author.Username); err != nil {
			return false, err
		}
	}
	return true, nil
}

func (s *Server) putLocalSpeakerDisposition(w http.ResponseWriter, r *http.Request) {
	sp, ok := s.authorizedSpeaker(w, r)
	if !ok {
		return
	}
	var input dispositionInput
	if !decode(w, r, &input) {
		return
	}
	if input.Disposition != dispositionIgnored && input.Disposition != dispositionRandom {
		fail(w, http.StatusBadRequest, "invalid_disposition")
		return
	}
	ss := r.Context().Value(sessionKey{}).(session)
	if !s.changeDispositions(w, r, []int64{sp.ID}, input.Disposition, ss) {
		return
	}
	reply(w, http.StatusOK, dispositionView{LocalSpeakerID: sp.ID, Disposition: &input.Disposition})
}

func (s *Server) deleteLocalSpeakerDisposition(w http.ResponseWriter, r *http.Request) {
	sp, ok := s.authorizedSpeaker(w, r)
	if !ok {
		return
	}
	ss := r.Context().Value(sessionKey{}).(session)
	if !s.changeDispositions(w, r, []int64{sp.ID}, "", ss) {
		return
	}
	reply(w, http.StatusOK, dispositionView{LocalSpeakerID: sp.ID})
}

// putVoiceGroupDisposition applies a disposition to every voice of a group the caller sees.
// Excluded members are not part of the group any more and are left alone.
func (s *Server) putVoiceGroupDisposition(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "invalid_voice_group_id")
	if !ok {
		return
	}
	var input dispositionInput
	if !decode(w, r, &input) {
		return
	}
	if input.Disposition != dispositionIgnored && input.Disposition != dispositionRandom {
		fail(w, http.StatusBadRequest, "invalid_disposition")
		return
	}
	ss := r.Context().Value(sessionKey{}).(session)
	rows, err := s.Pool.Query(r.Context(), voiceGroupsCTE+` SELECT id,excluded FROM members WHERE group_id=$3 ORDER BY id`, ss.User.ID, isAdministrator(ss.User.Role), id)
	if err != nil {
		s.internal(w, err)
		return
	}
	ids := []int64{}
	found := false
	for rows.Next() {
		var voiceID int64
		var excluded bool
		if err = rows.Scan(&voiceID, &excluded); err != nil {
			rows.Close()
			s.internal(w, err)
			return
		}
		found = true
		if !excluded {
			ids = append(ids, voiceID)
		}
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		s.internal(w, err)
		return
	}
	if !found {
		fail(w, http.StatusNotFound, "voice_group_not_found")
		return
	}
	if !s.changeDispositions(w, r, ids, input.Disposition, ss) {
		return
	}
	reply(w, http.StatusOK, map[string]any{"group_id": id, "disposition": nullableDisposition(input.Disposition), "voices": len(ids)})
}

func nullableDisposition(disposition string) *string {
	if disposition == "" {
		return nil
	}
	return &disposition
}

// changeDispositions writes the dispositions in one transaction gated on the analysis of
// their clips, and re-runs matching so the machine attributions of the voices follow at once.
// It answers the error itself and returns false when nothing was written.
func (s *Server) changeDispositions(w http.ResponseWriter, r *http.Request, ids []int64, disposition string, ss session) bool {
	ctx := r.Context()
	clipIDs := []int64{}
	rows, err := s.Pool.Query(ctx, `SELECT DISTINCT clip_id FROM local_speakers WHERE id=ANY($1) ORDER BY clip_id`, ids)
	if err == nil {
		clipIDs, err = pgx.CollectRows(rows, pgx.RowTo[int64])
	}
	if err != nil {
		s.internal(w, err)
		return false
	}
	tx, ok := s.beginAnalysisGated(w, r, clipIDs...)
	if !ok {
		return false
	}
	defer tx.Rollback(ctx)
	done, err := setDispositions(ctx, tx, ids, disposition, authorRef{ID: &ss.User.ID, Username: ss.User.Username})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			fail(w, http.StatusNotFound, "local_speaker_not_found")
			return false
		}
		s.internal(w, err)
		return false
	}
	if !done {
		fail(w, http.StatusConflict, "validated_voice")
		return false
	}
	if err = tx.Commit(ctx); err != nil {
		s.internal(w, err)
		return false
	}
	s.matchingChanged(ctx, 0)
	return true
}

// beginAnalysisGated opens the transaction of a change to the voices of the clips that waits
// for their analysis, like the deletion of analysis steps. Each clip row is share-locked, in
// id order: a diarization takes it FOR UPDATE while it plans the carry of dispositions and
// refusals and deletes the old voices, so nothing can be committed in between. Under that
// lock, an active run or voice job on a clip refuses the change (409 analysis_active). It
// answers the error itself and returns false with the transaction rolled back; otherwise the
// caller owns the transaction.
func (s *Server) beginAnalysisGated(w http.ResponseWriter, r *http.Request, clipIDs ...int64) (pgx.Tx, bool) {
	ctx := r.Context()
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		s.internal(w, err)
		return nil, false
	}
	for _, clipID := range clipIDs {
		if _, err = tx.Exec(ctx, `SELECT id FROM clips WHERE id=$1 FOR SHARE`, clipID); err != nil {
			tx.Rollback(ctx)
			s.internal(w, err)
			return nil, false
		}
		busy, err := analysisActive(ctx, tx, clipID)
		if err != nil {
			tx.Rollback(ctx)
			s.internal(w, err)
			return nil, false
		}
		if busy {
			tx.Rollback(ctx)
			fail(w, http.StatusConflict, "analysis_active")
			return nil, false
		}
	}
	return tx, true
}

// refuseLocalSpeaker records "ce n'est pas X": the attribution of that Person on the voice
// is removed (human or machine), its validated sample leaves the references, and the
// refusal stops matching from proposing the Person for this voice.
func (s *Server) refuseLocalSpeaker(w http.ResponseWriter, r *http.Request) {
	sp, ok := s.authorizedSpeaker(w, r)
	if !ok {
		return
	}
	var input refusalInput
	if !decode(w, r, &input) {
		return
	}
	if input.PersonID == nil || *input.PersonID < 1 {
		fail(w, http.StatusBadRequest, "invalid_request")
		return
	}
	ss := r.Context().Value(sessionKey{}).(session)
	ctx := r.Context()
	tx, ok := s.beginAnalysisGated(w, r, sp.ClipID)
	if !ok {
		return
	}
	defer tx.Rollback(ctx)
	_, err := tx.Exec(ctx, `SELECT id FROM local_speakers WHERE id=$1 FOR UPDATE`, sp.ID)
	if err != nil {
		s.internal(w, err)
		return
	}
	personID := *input.PersonID
	var name string
	if err = tx.QueryRow(ctx, `SELECT display_name FROM persons WHERE id=$1 FOR SHARE`, personID).Scan(&name); errors.Is(err, pgx.ErrNoRows) {
		fail(w, http.StatusNotFound, "person_not_found")
		return
	} else if err != nil {
		s.internal(w, err)
		return
	}
	// Like a correction, the refusal retires the reference sample rather than deleting it:
	// the history keeps pointing to it (attribution_events.old_sample_id).
	var sampleID *int64
	err = tx.QueryRow(ctx, `UPDATE voice_samples SET retired_at=now(),retired_by=$3,retired_reason='refus_identification' WHERE local_speaker_id=$1 AND person_id=$2 AND retired_at IS NULL RETURNING id`, sp.ID, personID, ss.User.ID).Scan(&sampleID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		s.internal(w, err)
		return
	}
	// The human validation, when it is this Person, is the attribution that matters for the history.
	removed := []string{}
	rows, err := tx.Query(ctx, `DELETE FROM speaker_attributions WHERE local_speaker_id=$1 AND person_id=$2 RETURNING decision_level`, sp.ID, personID)
	if err != nil {
		s.internal(w, err)
		return
	}
	for rows.Next() {
		var level string
		if err = rows.Scan(&level); err != nil {
			rows.Close()
			s.internal(w, err)
			return
		}
		removed = append(removed, level)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		s.internal(w, err)
		return
	}
	oldLevel := "unknown"
	for _, level := range removed {
		if level == string(LevelHuman) || oldLevel == "unknown" {
			oldLevel = level
		}
	}
	if _, err = tx.Exec(ctx, `INSERT INTO voice_refusals(local_speaker_id,person_id,author_id,author_username) VALUES($1,$2,$3,$4)
ON CONFLICT (local_speaker_id,person_id) DO UPDATE SET author_id=EXCLUDED.author_id,author_username=EXCLUDED.author_username,created_at=now()`, sp.ID, personID, ss.User.ID, ss.User.Username); err != nil {
		s.internal(w, err)
		return
	}
	if _, err = tx.Exec(ctx, `INSERT INTO voice_refusal_events(local_speaker_id,clip_id,person_id,person_name,action,author_id,author_username) VALUES($1,$2,$3,$4,'refuse',$5,$6)`, sp.ID, sp.ClipID, personID, name, ss.User.ID, ss.User.Username); err != nil {
		s.internal(w, err)
		return
	}
	if _, err = tx.Exec(ctx, `INSERT INTO attribution_events(local_speaker_id,clip_id,action,old_person_id,old_person_name,old_decision_level,old_sample_id,new_decision_level,author_id,author_username) VALUES($1,$2,'refuse',$3,$4,$5,$6,'unknown',$7,$8)`, sp.ID, sp.ClipID, personID, name, oldLevel, sampleID, ss.User.ID, ss.User.Username); err != nil {
		s.internal(w, err)
		return
	}
	if err = tx.Commit(ctx); err != nil {
		s.internal(w, err)
		return
	}
	// A removed reference or attribution changes what the other voices may match.
	s.matchingChanged(ctx, 0)
	reply(w, http.StatusOK, map[string]any{"local_speaker_id": sp.ID, "person_id": personID, "refused": true})
}

// cancelLocalSpeakerRefusal is "Annuler le refus": the Person is no longer kept out of the
// matching of this voice. It needs the right of the refusal, waits for the analysis like it,
// and is journaled when a refusal existed. The attribution removed by the refusal is not
// restored: matching proposes it again, or a human decides.
func (s *Server) cancelLocalSpeakerRefusal(w http.ResponseWriter, r *http.Request) {
	sp, ok := s.authorizedSpeaker(w, r)
	if !ok {
		return
	}
	personID, err := strconv.ParseInt(r.PathValue("person_id"), 10, 64)
	if err != nil || personID < 1 {
		fail(w, http.StatusBadRequest, "invalid_person_id")
		return
	}
	ss := r.Context().Value(sessionKey{}).(session)
	ctx := r.Context()
	tx, ok := s.beginAnalysisGated(w, r, sp.ClipID)
	if !ok {
		return
	}
	defer tx.Rollback(ctx)
	var name string
	err = tx.QueryRow(ctx, `WITH cancelled AS (DELETE FROM voice_refusals WHERE local_speaker_id=$1 AND person_id=$2 RETURNING person_id)
SELECT COALESCE(p.display_name,'') FROM cancelled c LEFT JOIN persons p ON p.id=c.person_id`, sp.ID, personID).Scan(&name)
	if errors.Is(err, pgx.ErrNoRows) {
		// No refusal: nothing to cancel, nothing journaled.
		reply(w, http.StatusOK, map[string]any{"local_speaker_id": sp.ID, "person_id": personID, "refused": false})
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	if _, err = tx.Exec(ctx, `INSERT INTO voice_refusal_events(local_speaker_id,clip_id,person_id,person_name,action,author_id,author_username) VALUES($1,$2,$3,$4,'cancel',$5,$6)`, sp.ID, sp.ClipID, personID, name, ss.User.ID, ss.User.Username); err != nil {
		s.internal(w, err)
		return
	}
	if err = tx.Commit(ctx); err != nil {
		s.internal(w, err)
		return
	}
	s.matchingChanged(ctx, 0)
	reply(w, http.StatusOK, map[string]any{"local_speaker_id": sp.ID, "person_id": personID, "refused": false})
}
