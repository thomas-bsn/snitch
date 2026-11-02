package httpapi

import (
	"context"
	"net/http"
	"snitch/internal/logging"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
)

// Person corrections (phase 5): a Person created by mistake is merged into
// another, separated from some of its voices, or deleted. Every route is open to any
// signed-in user, like the creation of Persons and the validation of voices. The
// invariants of the validation stay: a merge never replaces a human validation, it only
// re-points it; a separation or a deletion withdraws the validation of the voices it
// names (they become unknown again) and takes their samples out of the references. Each
// change is written to the history of the affected voices (attribution_events) and to
// the Person-level trail (person_events), then matching is recomputed from the stored
// fingerprints, without any worker or video.

// registerPersonCorrectionRoutes declares the Person correction routes.
func (s *Server) registerPersonCorrectionRoutes(mux *http.ServeMux) {
	mux.Handle("POST /api/persons/{id}/merge", s.protected(false, http.HandlerFunc(s.mergePerson)))
	mux.Handle("POST /api/persons/{id}/separation", s.protected(false, http.HandlerFunc(s.separatePerson)))
	mux.Handle("DELETE /api/persons/{id}", s.protected(false, http.HandlerFunc(s.deletePerson)))
	mux.Handle("GET /api/persons/{id}/voices", s.protected(false, http.HandlerFunc(s.listPersonVoices)))
	mux.Handle("GET /api/person-events", s.protected(false, http.HandlerFunc(s.listPersonEvents)))
}

// correctionSummary is the outcome of a correction.
type correctionSummaryView struct {
	// Person is the Person that remains (merge target, separated Person); null after a deletion.
	Person          *personView `json:"person"`
	Attributions    int         `json:"attributions"`
	Samples         int         `json:"samples"`
	MicrophoneRules int         `json:"microphone_rules"`
}

type personIdentity struct {
	ID     int64
	Name   string
	UserID *int64
	// OptedOut is set when the Person deleted their voice profile (voice profile deletion).
	OptedOut bool
}

func personPathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, ok := pathID(w, r, "invalid_person_id")
	if !ok {
		return 0, false
	}
	return id, true
}

// lockPersons locks the named Persons in id order and returns the ones that exist.
// The validated attributions that point to them are locked first, which is the order
// a correction takes (attribution, then Person): the two cannot deadlock.
func lockPersons(ctx context.Context, tx pgx.Tx, ids ...int64) (map[int64]personIdentity, error) {
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	if _, err := tx.Exec(ctx, `SELECT id FROM speaker_attributions WHERE person_id=ANY($1) ORDER BY id FOR UPDATE`, ids); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT id,display_name,user_id,voice_profile_deleted_at IS NOT NULL FROM persons WHERE id=ANY($1) ORDER BY id FOR UPDATE`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]personIdentity{}
	for rows.Next() {
		var p personIdentity
		if err = rows.Scan(&p.ID, &p.Name, &p.UserID, &p.OptedOut); err != nil {
			return nil, err
		}
		out[p.ID] = p
	}
	return out, rows.Err()
}

// withdrawn lists what a correction took away from the microphone track mechanism, so
// the affected clips can be reconciled once the change is committed.
type affectedMicrophones struct {
	clips   map[int64]bool
	sources map[int64]bool
}

func newAffectedMicrophones() *affectedMicrophones {
	return &affectedMicrophones{clips: map[int64]bool{}, sources: map[int64]bool{}}
}

// reconcile re-applies the microphone designations of the affected sources and clips.
// A designation that named a removed Person no longer exists, so a track that was only
// attributed through it ends up without attribution.
func (s *Server) reconcileMicrophones(ctx context.Context, affected *affectedMicrophones, author authorRef) {
	ctx = context.WithoutCancel(ctx)
	for id := range affected.sources {
		if _, err := s.applySource(ctx, id, author); err != nil {
			s.Log.Error("microphone reconcile failed", append([]any{"source_id", id}, logging.Failure(err)...)...)
		}
	}
	for id := range affected.clips {
		if _, err := s.applyMicrophone(ctx, id, author); err != nil {
			s.Log.Error("microphone reconcile failed", append([]any{"clip_id", id}, logging.Failure(err)...)...)
		}
	}
}

func recordPersonEvent(ctx context.Context, tx pgx.Tx, action string, person personIdentity, target *personIdentity, attributions, samples, rules int, author authorRef) error {
	var targetID *int64
	targetName := ""
	if target != nil {
		targetID, targetName = &target.ID, target.Name
	}
	_, err := tx.Exec(ctx, `INSERT INTO person_events(action,person_id,person_name,target_person_id,target_person_name,attribution_count,sample_count,microphone_rule_count,author_id,author_username) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, action, person.ID, person.Name, targetID, targetName, attributions, samples, rules, author.ID, author.Username)
	return err
}

type mergeInput struct {
	TargetPersonID int64 `json:"target_person_id"`
}

// mergePerson folds a Person into another. Their validated attributions and samples
// (retired ones included, they are history) are attached to the target, the microphone
// designations that named the Person follow, and the Person disappears. A Person tied
// to an account is never absorbed: merge the other one into it.
func (s *Server) mergePerson(w http.ResponseWriter, r *http.Request) {
	id, ok := personPathID(w, r)
	if !ok {
		return
	}
	var input mergeInput
	if !decode(w, r, &input) {
		return
	}
	if input.TargetPersonID < 1 {
		fail(w, http.StatusBadRequest, "invalid_request")
		return
	}
	if input.TargetPersonID == id {
		fail(w, http.StatusConflict, "same_person")
		return
	}
	author := s.authorOf(r)
	ctx := r.Context()
	tx, unlock, err := s.beginCorrection(ctx)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer unlock()
	defer tx.Rollback(ctx)
	persons, err := lockPersons(ctx, tx, id, input.TargetPersonID)
	if err != nil {
		s.internal(w, err)
		return
	}
	source, found := persons[id]
	target, targetFound := persons[input.TargetPersonID]
	if !found || !targetFound {
		fail(w, http.StatusNotFound, "person_not_found")
		return
	}
	if source.UserID != nil {
		fail(w, http.StatusConflict, "person_linked_account")
		return
	}
	if s.refuseHiddenVoices(ctx, w, r, tx, source.ID) {
		return
	}
	// A deleted voice profile is an opt-out: merging into it would rebuild a profile, and
	// merging out of it would dissolve the opt-out. Both are refused.
	if source.OptedOut || target.OptedOut {
		fail(w, http.StatusConflict, "voice_profile_deleted")
		return
	}
	// History first, from the state the merge is about to change.
	if _, err = tx.Exec(ctx, `INSERT INTO attribution_events(local_speaker_id,clip_id,action,old_person_id,old_person_name,old_decision_level,old_sample_id,new_person_id,new_person_name,new_decision_level,new_sample_id,author_id,author_username)
		SELECT a.local_speaker_id,ls.clip_id,'merge',$1,$2,a.decision_level,vs.id,$3,$4,a.decision_level,vs.id,$5,$6
		FROM speaker_attributions a JOIN local_speakers ls ON ls.id=a.local_speaker_id LEFT JOIN voice_samples vs ON vs.local_speaker_id=a.local_speaker_id AND vs.retired_at IS NULL
		WHERE a.person_id=$1 AND (a.validation_state='validated' OR a.origin=$7)`, source.ID, source.Name, target.ID, target.Name, author.ID, author.Username, microphoneOrigin); err != nil {
		s.internal(w, err)
		return
	}
	attributions, err := tx.Exec(ctx, `UPDATE speaker_attributions SET person_id=$2,revision=revision+1 WHERE person_id=$1 AND (validation_state='validated' OR origin=$3)`, source.ID, target.ID, microphoneOrigin)
	if err != nil {
		s.internal(w, err)
		return
	}
	// Derived matching rows that named the Person are recomputed below.
	if _, err = tx.Exec(ctx, `DELETE FROM speaker_attributions WHERE person_id=$1`, source.ID); err != nil {
		s.internal(w, err)
		return
	}
	samples, err := tx.Exec(ctx, `UPDATE voice_samples SET person_id=$2 WHERE person_id=$1`, source.ID, target.ID)
	if err != nil {
		s.internal(w, err)
		return
	}
	rules, err := repointMicrophoneRules(ctx, tx, source, target, author)
	if err != nil {
		s.internal(w, err)
		return
	}
	if err = recordPersonEvent(ctx, tx, "merge", source, &target, int(attributions.RowsAffected()), int(samples.RowsAffected()), rules, author); err != nil {
		s.internal(w, err)
		return
	}
	if _, err = tx.Exec(ctx, `DELETE FROM persons WHERE id=$1`, source.ID); err != nil {
		s.internal(w, err)
		return
	}
	if err = tx.Commit(ctx); err != nil {
		s.internal(w, err)
		return
	}
	unlock()
	s.matchingChanged(ctx, 0)
	view, err := scanPerson(s.Pool.QueryRow(ctx, `SELECT `+personColumns+` FROM persons p WHERE p.id=$1`, target.ID))
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, correctionSummaryView{Person: &view, Attributions: int(attributions.RowsAffected()), Samples: int(samples.RowsAffected()), MicrophoneRules: rules})
}

// refuseHiddenVoices answers 409 person_has_hidden_voices and returns true when the
// Person is attributed on a clip the caller cannot see. Merging or deleting a Person
// rewrites every one of its attributions, so a non-administrator may only do it when
// all of them are on clips they can read. Single-voice corrections already follow the
// same visibility rule.
func (s *Server) refuseHiddenVoices(ctx context.Context, w http.ResponseWriter, r *http.Request, tx pgx.Tx, personID int64) bool {
	ss := r.Context().Value(sessionKey{}).(session)
	if isAdministrator(ss.User.Role) {
		return false
	}
	var hidden bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM speaker_attributions a JOIN local_speakers ls ON ls.id=a.local_speaker_id JOIN clips c ON c.id=ls.clip_id WHERE a.person_id=$1 AND NOT (c.visibility='shared' OR c.owner_id=$2))`, personID, ss.User.ID).Scan(&hidden); err != nil {
		s.internal(w, err)
		return true
	}
	if hidden {
		fail(w, http.StatusConflict, "person_has_hidden_voices")
	}
	return hidden
}

// repointMicrophoneRules makes the source and clip designations that name the Person
// name the target instead, with a history entry for each.
func repointMicrophoneRules(ctx context.Context, tx pgx.Tx, from, to personIdentity, author authorRef) (int, error) {
	total := 0
	tag, err := tx.Exec(ctx, `INSERT INTO microphone_track_events(scope,source_id,action,old_rule,new_rule,author_id,author_username)
		SELECT 'source',source_id,'set',
		 jsonb_strip_nulls(jsonb_build_object('audio_stream_index',audio_stream_index,'title',title,'person',jsonb_build_object('id',$1::bigint,'display_name',$2::text))),
		 jsonb_strip_nulls(jsonb_build_object('audio_stream_index',audio_stream_index,'title',title,'person',jsonb_build_object('id',$3::bigint,'display_name',$4::text))),
		 $5,$6 FROM source_microphone_tracks WHERE person_id=$1`, from.ID, from.Name, to.ID, to.Name, author.ID, author.Username)
	if err != nil {
		return 0, err
	}
	total += int(tag.RowsAffected())
	if _, err = tx.Exec(ctx, `UPDATE source_microphone_tracks SET person_id=$2,version=version+1,updated_by=$3,updated_at=now() WHERE person_id=$1`, from.ID, to.ID, author.ID); err != nil {
		return 0, err
	}
	tag, err = tx.Exec(ctx, `INSERT INTO microphone_track_events(scope,clip_id,action,old_rule,new_rule,author_id,author_username)
		SELECT 'clip',clip_id,'set',
		 jsonb_build_object('audio_stream_index',audio_stream_index,'person',jsonb_build_object('id',$1::bigint,'display_name',$2::text)),
		 jsonb_build_object('audio_stream_index',audio_stream_index,'person',jsonb_build_object('id',$3::bigint,'display_name',$4::text)),
		 $5,$6 FROM clip_microphone_overrides WHERE person_id=$1`, from.ID, from.Name, to.ID, to.Name, author.ID, author.Username)
	if err != nil {
		return 0, err
	}
	total += int(tag.RowsAffected())
	if _, err = tx.Exec(ctx, `UPDATE clip_microphone_overrides SET person_id=$2,version=version+1,updated_by=$3,updated_at=now() WHERE person_id=$1`, from.ID, to.ID, author.ID); err != nil {
		return 0, err
	}
	return total, nil
}

// withdrawAttributions takes the human validations of a Person out, for the given
// voices (all of them when speakerIDs is nil). The voices become unknown again and
// their samples leave the references (they are erased: the sample belongs to the
// validation). It records one history entry per voice.
func withdrawAttributions(ctx context.Context, tx pgx.Tx, person personIdentity, speakerIDs []int64, action string, author authorRef, affected *affectedMicrophones) (attributions, samples int, err error) {
	filter := `a.person_id=$1 AND a.validation_state='validated' AND ($2::bigint[] IS NULL OR a.local_speaker_id=ANY($2))`
	if _, err = tx.Exec(ctx, `INSERT INTO attribution_events(local_speaker_id,clip_id,action,old_person_id,old_person_name,old_decision_level,old_sample_id,new_decision_level,author_id,author_username)
		SELECT a.local_speaker_id,ls.clip_id,$3,$1,$4,'human',vs.id,'unknown',$5,$6
		FROM speaker_attributions a JOIN local_speakers ls ON ls.id=a.local_speaker_id LEFT JOIN voice_samples vs ON vs.local_speaker_id=a.local_speaker_id AND vs.retired_at IS NULL
		WHERE `+filter, person.ID, speakerIDs, action, person.Name, author.ID, author.Username); err != nil {
		return 0, 0, err
	}
	rows, err := tx.Query(ctx, `SELECT ls.id,ls.clip_id,ls.kind FROM speaker_attributions a JOIN local_speakers ls ON ls.id=a.local_speaker_id WHERE `+filter, person.ID, speakerIDs)
	if err != nil {
		return 0, 0, err
	}
	ids := []int64{}
	for rows.Next() {
		var id, clipID int64
		var kind string
		if err = rows.Scan(&id, &clipID, &kind); err != nil {
			rows.Close()
			return 0, 0, err
		}
		ids = append(ids, id)
		if kind == "microphone" {
			affected.clips[clipID] = true
		}
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return 0, 0, err
	}
	if len(ids) == 0 {
		return 0, 0, nil
	}
	tag, err := tx.Exec(ctx, `DELETE FROM voice_samples WHERE person_id=$1 AND local_speaker_id=ANY($2) AND retired_at IS NULL`, person.ID, ids)
	if err != nil {
		return 0, 0, err
	}
	samples = int(tag.RowsAffected())
	del, err := tx.Exec(ctx, `DELETE FROM speaker_attributions WHERE person_id=$1 AND validation_state='validated' AND local_speaker_id=ANY($2)`, person.ID, ids)
	if err != nil {
		return 0, 0, err
	}
	return int(del.RowsAffected()), samples, nil
}

type separationInput struct {
	LocalSpeakerIDs []int64 `json:"local_speaker_ids"`
}

// separatePerson detaches voices from a Person that was validated on them by mistake:
// the human validation is withdrawn, the sample is erased from the references and the voice is
// unknown again, ready for a new validation. The Person itself stays. Only voices of
// clips the caller can see are named; anything else is reported as not found.
func (s *Server) separatePerson(w http.ResponseWriter, r *http.Request) {
	id, ok := personPathID(w, r)
	if !ok {
		return
	}
	var input separationInput
	if !decode(w, r, &input) {
		return
	}
	unique := map[int64]bool{}
	for _, sp := range input.LocalSpeakerIDs {
		if sp < 1 {
			fail(w, http.StatusBadRequest, "invalid_request")
			return
		}
		unique[sp] = true
	}
	if len(unique) == 0 || len(unique) > 100 {
		fail(w, http.StatusBadRequest, "invalid_request")
		return
	}
	speakerIDs := make([]int64, 0, len(unique))
	for sp := range unique {
		speakerIDs = append(speakerIDs, sp)
	}
	ss := r.Context().Value(sessionKey{}).(session)
	author := s.authorOf(r)
	ctx := r.Context()
	var visible int
	if err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM local_speakers ls JOIN clips c ON c.id=ls.clip_id WHERE ls.id=ANY($1) AND (c.visibility='shared' OR c.owner_id=$2 OR $3)`, speakerIDs, ss.User.ID, isAdministrator(ss.User.Role)).Scan(&visible); err != nil {
		s.internal(w, err)
		return
	}
	if visible != len(speakerIDs) {
		fail(w, http.StatusNotFound, "local_speaker_not_found")
		return
	}
	tx, unlock, err := s.beginCorrection(ctx)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer unlock()
	defer tx.Rollback(ctx)
	persons, err := lockPersons(ctx, tx, id)
	if err != nil {
		s.internal(w, err)
		return
	}
	person, found := persons[id]
	if !found {
		fail(w, http.StatusNotFound, "person_not_found")
		return
	}
	var owned int
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM speaker_attributions WHERE person_id=$1 AND validation_state='validated' AND local_speaker_id=ANY($2)`, id, speakerIDs).Scan(&owned); err != nil {
		s.internal(w, err)
		return
	}
	if owned != len(speakerIDs) {
		fail(w, http.StatusConflict, "not_attributed_to_person")
		return
	}
	affected := newAffectedMicrophones()
	attributions, samples, err := withdrawAttributions(ctx, tx, person, speakerIDs, "split", author, affected)
	if err != nil {
		s.internal(w, err)
		return
	}
	if err = recordPersonEvent(ctx, tx, "split", person, nil, attributions, samples, 0, author); err != nil {
		s.internal(w, err)
		return
	}
	if err = tx.Commit(ctx); err != nil {
		s.internal(w, err)
		return
	}
	unlock()
	s.reconcileMicrophones(ctx, affected, author)
	s.matchingChanged(ctx, 0)
	view, err := scanPerson(s.Pool.QueryRow(ctx, `SELECT `+personColumns+` FROM persons p WHERE p.id=$1`, id))
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, correctionSummaryView{Person: &view, Attributions: attributions, Samples: samples})
}

// deletePerson removes a Person created by mistake: its validations are withdrawn
// (voices unknown again), its samples erased, the microphone designations that name
// it removed (recorded in their own history) and the machine attributions that named
// it recomputed. Clips and fingerprints are untouched. A Person tied to an account is
// refused: its voice profile is deleted from the profile instead.
func (s *Server) deletePerson(w http.ResponseWriter, r *http.Request) {
	id, ok := personPathID(w, r)
	if !ok {
		return
	}
	author := s.authorOf(r)
	ctx := r.Context()
	tx, unlock, err := s.beginCorrection(ctx)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer unlock()
	defer tx.Rollback(ctx)
	persons, err := lockPersons(ctx, tx, id)
	if err != nil {
		s.internal(w, err)
		return
	}
	person, found := persons[id]
	if !found {
		fail(w, http.StatusNotFound, "person_not_found")
		return
	}
	if person.UserID != nil {
		fail(w, http.StatusConflict, "person_linked_account")
		return
	}
	if s.refuseHiddenVoices(ctx, w, r, tx, id) {
		return
	}
	affected := newAffectedMicrophones()
	attributions, samples, err := withdrawAttributions(ctx, tx, person, nil, "delete_person", author, affected)
	if err != nil {
		s.internal(w, err)
		return
	}
	// Whatever else named the Person: samples that were already retired, machine rows.
	if _, err = tx.Exec(ctx, `DELETE FROM voice_samples WHERE person_id=$1`, id); err != nil {
		s.internal(w, err)
		return
	}
	// A machine attribution from the microphone track disappears with the designation;
	// its speaker is reconciled after the commit.
	micRows, err := tx.Query(ctx, `SELECT ls.id,ls.clip_id FROM speaker_attributions a JOIN local_speakers ls ON ls.id=a.local_speaker_id WHERE a.person_id=$1 AND a.origin=$2`, id, microphoneOrigin)
	if err != nil {
		s.internal(w, err)
		return
	}
	for micRows.Next() {
		var speakerID, clipID int64
		if err = micRows.Scan(&speakerID, &clipID); err != nil {
			micRows.Close()
			s.internal(w, err)
			return
		}
		affected.clips[clipID] = true
	}
	micRows.Close()
	if _, err = tx.Exec(ctx, `INSERT INTO attribution_events(local_speaker_id,clip_id,action,old_person_id,old_person_name,old_decision_level,new_decision_level,author_id,author_username)
		SELECT a.local_speaker_id,ls.clip_id,'microphone_unassign',$1,$2,a.decision_level,'unknown',$3,$4 FROM speaker_attributions a JOIN local_speakers ls ON ls.id=a.local_speaker_id WHERE a.person_id=$1 AND a.origin=$5`, id, person.Name, author.ID, author.Username, microphoneOrigin); err != nil {
		s.internal(w, err)
		return
	}
	if _, err = tx.Exec(ctx, `DELETE FROM speaker_attributions WHERE person_id=$1`, id); err != nil {
		s.internal(w, err)
		return
	}
	rules, err := removeMicrophoneRules(ctx, tx, person, author, affected)
	if err != nil {
		s.internal(w, err)
		return
	}
	if err = recordPersonEvent(ctx, tx, "delete", person, nil, attributions, samples, rules, author); err != nil {
		s.internal(w, err)
		return
	}
	if _, err = tx.Exec(ctx, `DELETE FROM persons WHERE id=$1`, id); err != nil {
		s.internal(w, err)
		return
	}
	if err = tx.Commit(ctx); err != nil {
		s.internal(w, err)
		return
	}
	unlock()
	s.reconcileMicrophones(ctx, affected, author)
	s.matchingChanged(ctx, 0)
	reply(w, http.StatusOK, correctionSummaryView{Attributions: attributions, Samples: samples, MicrophoneRules: rules})
}

// removeMicrophoneRules deletes the designations that name the Person (a rule cannot
// outlive its owner), recording each removal, and notes the clips and sources to re-apply.
func removeMicrophoneRules(ctx context.Context, tx pgx.Tx, person personIdentity, author authorRef, affected *affectedMicrophones) (int, error) {
	total := 0
	tag, err := tx.Exec(ctx, `INSERT INTO microphone_track_events(scope,source_id,action,old_rule,new_rule,author_id,author_username)
		SELECT 'source',source_id,'remove',
		 jsonb_strip_nulls(jsonb_build_object('audio_stream_index',audio_stream_index,'title',title,'person',jsonb_build_object('id',$1::bigint,'display_name',$2::text))),
		 NULL,$3,$4 FROM source_microphone_tracks WHERE person_id=$1`, person.ID, person.Name, author.ID, author.Username)
	if err != nil {
		return 0, err
	}
	total += int(tag.RowsAffected())
	rows, err := tx.Query(ctx, `DELETE FROM source_microphone_tracks WHERE person_id=$1 RETURNING source_id`, person.ID)
	if err != nil {
		return 0, err
	}
	for rows.Next() {
		var id int64
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return 0, err
		}
		affected.sources[id] = true
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return 0, err
	}
	tag, err = tx.Exec(ctx, `INSERT INTO microphone_track_events(scope,clip_id,action,old_rule,new_rule,author_id,author_username)
		SELECT 'clip',clip_id,'remove',
		 jsonb_build_object('audio_stream_index',audio_stream_index,'person',jsonb_build_object('id',$1::bigint,'display_name',$2::text)),
		 NULL,$3,$4 FROM clip_microphone_overrides WHERE person_id=$1`, person.ID, person.Name, author.ID, author.Username)
	if err != nil {
		return 0, err
	}
	total += int(tag.RowsAffected())
	rows, err = tx.Query(ctx, `DELETE FROM clip_microphone_overrides WHERE person_id=$1 RETURNING clip_id`, person.ID)
	if err != nil {
		return 0, err
	}
	for rows.Next() {
		var id int64
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return 0, err
		}
		affected.clips[id] = true
	}
	rows.Close()
	return total, rows.Err()
}

type personEventView struct {
	ID                  int64      `json:"id"`
	Action              string     `json:"action"`
	Person              personRef  `json:"person"`
	Target              *personRef `json:"target"`
	AttributionCount    int        `json:"attribution_count"`
	SampleCount         int        `json:"sample_count"`
	MicrophoneRuleCount int        `json:"microphone_rule_count"`
	Author              *authorRef `json:"author"`
	CreatedAt           time.Time  `json:"created_at"`
}

// listPersonEvents returns the trail of corrections, newest first, optionally those
// that concern one Person (as the corrected one or as the merge target). It also
// answers for Persons that no longer exist.
func (s *Server) listPersonEvents(w http.ResponseWriter, r *http.Request) {
	var personID *int64
	if raw := r.URL.Query().Get("person_id"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id < 1 {
			fail(w, http.StatusBadRequest, "invalid_person_id")
			return
		}
		personID = &id
	}
	rows, err := s.Pool.Query(r.Context(), `SELECT id,action,person_id,person_name,target_person_id,target_person_name,attribution_count,sample_count,microphone_rule_count,author_id,author_username,created_at FROM person_events WHERE $1::bigint IS NULL OR person_id=$1 OR target_person_id=$1 ORDER BY id DESC LIMIT 200`, personID)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer rows.Close()
	items := []personEventView{}
	for rows.Next() {
		var e personEventView
		var targetID, authorID *int64
		var targetName, username string
		if err = rows.Scan(&e.ID, &e.Action, &e.Person.ID, &e.Person.DisplayName, &targetID, &targetName, &e.AttributionCount, &e.SampleCount, &e.MicrophoneRuleCount, &authorID, &username, &e.CreatedAt); err != nil {
			s.internal(w, err)
			return
		}
		if targetID != nil {
			e.Target = &personRef{ID: *targetID, DisplayName: targetName}
		}
		if authorID != nil || username != "" {
			e.Author = &authorRef{ID: authorID, Username: username}
		}
		items = append(items, e)
	}
	if err = rows.Err(); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{"items": items})
}

// beginCorrection opens the transaction of a correction under the matching locks, in the
// order matching and the voice profile erasure take them (process mutex, then the
// database lock), so a run in flight finishes first and cannot write a row naming a
// Person that is being merged or deleted. unlock is idempotent: call it after the
// commit, before the matching is run again.
func (s *Server) beginCorrection(ctx context.Context) (pgx.Tx, func(), error) {
	voiceMatchingLock.Lock()
	var once sync.Once
	unlock := func() { once.Do(voiceMatchingLock.Unlock) }
	tx, err := s.beginLocked(ctx, lockVoiceMatching)
	if err != nil {
		unlock()
		return nil, unlock, err
	}
	return tx, unlock, nil
}
