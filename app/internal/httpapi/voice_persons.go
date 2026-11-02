package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// registerPersonRoutes declares the Person and attribution routes (phase 5).
// Every route is open to any signed-in user: all users can create Persons, validate
// and correct. Visibility of the underlying clip is always enforced.
func (s *Server) registerPersonRoutes(mux *http.ServeMux) {
	mux.Handle("GET /api/persons", s.protected(false, http.HandlerFunc(s.listPersons)))
	mux.Handle("POST /api/persons", s.protected(false, http.HandlerFunc(s.createPerson)))
	mux.Handle("GET /api/persons/{id}", s.protected(false, http.HandlerFunc(s.getPerson)))
	mux.Handle("GET /api/persons/{id}/impact", s.protected(false, http.HandlerFunc(s.getPersonImpact)))
	mux.Handle("POST /api/local-speakers/{id}/validation", s.protected(false, http.HandlerFunc(s.validateLocalSpeaker)))
	mux.Handle("POST /api/local-speakers/{id}/correction", s.protected(false, http.HandlerFunc(s.correctLocalSpeaker)))
	mux.Handle("GET /api/local-speakers/{id}", s.protected(false, http.HandlerFunc(s.getLocalSpeaker)))
	mux.Handle("GET /api/local-speakers/{id}/history", s.protected(false, http.HandlerFunc(s.localSpeakerHistory)))
	mux.Handle("GET /api/local-speakers/{id}/excerpt", s.protected(false, http.HandlerFunc(s.localSpeakerExcerpt)))
}

type personRef struct {
	ID          int64  `json:"id"`
	DisplayName string `json:"display_name"`
}

type authorRef struct {
	ID       *int64 `json:"id"`
	Username string `json:"username"`
}

// attributionView is the contract shape of one attribution row. Later phase 5 steps
// (matching, microphone track, merge) reuse it unchanged.
type attributionView struct {
	ID              int64         `json:"id"`
	LocalSpeakerID  int64         `json:"local_speaker_id"`
	Person          *personRef    `json:"person"`
	DecisionLevel   DecisionLevel `json:"decision_level"`
	ValidationState string        `json:"validation_state"`
	Confidence      *float64      `json:"confidence"`
	Origin          string        `json:"origin"`
	Author          *authorRef    `json:"author"`
	Revision        int           `json:"revision"`
	DecidedAt       time.Time     `json:"decided_at"`
	// Candidates are the Persons a machine attribution hesitated between (ambiguous)
	// or its alternatives; always empty for a human validation.
	Candidates []candidateView `json:"candidates"`
}

type candidateView struct {
	Rank   int       `json:"rank"`
	Score  float64   `json:"score"`
	Person personRef `json:"person"`
}

// attributionColumns must stay aligned with scanAttribution. Callers join persons as p.
const attributionColumns = `a.id,a.local_speaker_id,a.decision_level,a.validation_state,a.confidence,a.origin,a.author_id,a.author_username,a.revision,a.decided_at,p.id,p.display_name`
const attributionFrom = `speaker_attributions a LEFT JOIN persons p ON p.id=a.person_id`

func scanAttribution(row pgx.Row) (attributionView, error) {
	var v attributionView
	var authorID, personID *int64
	var username string
	var personName *string
	if err := row.Scan(&v.ID, &v.LocalSpeakerID, &v.DecisionLevel, &v.ValidationState, &v.Confidence, &v.Origin, &authorID, &username, &v.Revision, &v.DecidedAt, &personID, &personName); err != nil {
		return v, err
	}
	if personID != nil && personName != nil {
		v.Person = &personRef{ID: *personID, DisplayName: *personName}
	}
	if authorID != nil || username != "" {
		v.Author = &authorRef{ID: authorID, Username: username}
	}
	v.Candidates = []candidateView{}
	return v, nil
}

// loadCandidates fills the candidates of machine attributions in one query.
func (s *Server) loadCandidates(ctx context.Context, byAttribution map[int64]*attributionView) error {
	ids := make([]int64, 0, len(byAttribution))
	for id := range byAttribution {
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil
	}
	rows, err := s.Pool.Query(ctx, `SELECT c.attribution_id,c.rank,c.score,p.id,p.display_name FROM speaker_attribution_candidates c JOIN persons p ON p.id=c.person_id WHERE c.attribution_id=ANY($1) ORDER BY c.attribution_id,c.rank`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var attributionID int64
		var c candidateView
		if err = rows.Scan(&attributionID, &c.Rank, &c.Score, &c.Person.ID, &c.Person.DisplayName); err != nil {
			return err
		}
		if v := byAttribution[attributionID]; v != nil {
			v.Candidates = append(v.Candidates, c)
		}
	}
	return rows.Err()
}

// effectiveAttributions returns, per local speaker of a clip, the attribution that
// applies: the human-validated row when there is one, otherwise the machine row.
func (s *Server) effectiveAttributions(ctx context.Context, clipID int64) (map[int64]attributionView, error) {
	rows, err := s.Pool.Query(ctx, `SELECT DISTINCT ON (a.local_speaker_id) `+attributionColumns+` FROM `+attributionFrom+` JOIN local_speakers ls ON ls.id=a.local_speaker_id WHERE ls.clip_id=$1 ORDER BY a.local_speaker_id,(a.validation_state='validated') DESC`, clipID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	pointers := map[int64]*attributionView{}
	machine := map[int64]*attributionView{}
	for rows.Next() {
		v, scanErr := scanAttribution(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		pointers[v.LocalSpeakerID] = &v
		if v.ValidationState == "unvalidated" {
			machine[v.ID] = &v
		}
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	if err = s.loadCandidates(ctx, machine); err != nil {
		return nil, err
	}
	out := make(map[int64]attributionView, len(pointers))
	for id, v := range pointers {
		out[id] = *v
	}
	return out, nil
}

func (s *Server) validatedAttribution(ctx context.Context, speakerID int64) (*attributionView, error) {
	v, err := scanAttribution(s.Pool.QueryRow(ctx, `SELECT `+attributionColumns+` FROM `+attributionFrom+` WHERE a.local_speaker_id=$1 AND a.validation_state='validated'`, speakerID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}

type personView struct {
	ID                   int64     `json:"id"`
	DisplayName          string    `json:"display_name"`
	UserID               *int64    `json:"user_id"`
	ValidatedSampleCount int       `json:"validated_sample_count"`
	LocalSpeakerCount    int       `json:"local_speaker_count"`
	CreatedAt            time.Time `json:"created_at"`
	// VoiceProfileDeletedAt is set once the Person deleted their voice profile; it
	// holds no voice data and keeps matching from rebuilding the profile.
	VoiceProfileDeletedAt *time.Time `json:"voice_profile_deleted_at"`
}

const personColumns = `p.id,p.display_name,p.user_id,p.created_at,
 (SELECT count(*) FROM voice_samples vs WHERE vs.person_id=p.id AND vs.retired_at IS NULL),
 (SELECT count(*) FROM speaker_attributions a WHERE a.person_id=p.id AND a.validation_state='validated'),p.voice_profile_deleted_at`

func scanPerson(row pgx.Row) (personView, error) {
	var v personView
	err := row.Scan(&v.ID, &v.DisplayName, &v.UserID, &v.CreatedAt, &v.ValidatedSampleCount, &v.LocalSpeakerCount, &v.VoiceProfileDeletedAt)
	return v, err
}

func cleanPersonName(name string) (string, bool) {
	name = strings.TrimSpace(name)
	return name, name != "" && len([]rune(name)) <= 100
}

const personSearchClause = `($1='' OR p.display_name ILIKE '%'||replace(replace(replace($1,'\','\\'),'%','\%'),'_','\_')||'%')`

// listPersons reads Persons by name. Without `page` and `page_size` it keeps the
// historical response (500 first Persons, no total); with either it pages and adds
// the total of the matches.
func (s *Server) listPersons(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	query := strings.TrimSpace(params.Get("q"))
	if len([]rune(query)) > 100 {
		fail(w, http.StatusBadRequest, "invalid_query")
		return
	}
	paged := params.Has("page") || params.Has("page_size")
	page, size := 1, 500
	if paged {
		size = 20
	}
	if v := params.Get("page"); params.Has("page") {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 100000 {
			fail(w, http.StatusBadRequest, "invalid_page")
			return
		}
		page = n
	}
	if v := params.Get("page_size"); params.Has("page_size") {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 100 {
			fail(w, http.StatusBadRequest, "invalid_page")
			return
		}
		size = n
	}
	// Items and total are read in one snapshot: a Person created in between cannot make
	// them disagree.
	tx, err := s.Pool.BeginTx(r.Context(), pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		s.internal(w, err)
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()
	rows, err := tx.Query(r.Context(), `SELECT `+personColumns+` FROM persons p WHERE `+personSearchClause+` ORDER BY lower(p.display_name),p.id LIMIT $2 OFFSET $3`, query, size, (page-1)*size)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer rows.Close()
	items := []personView{}
	for rows.Next() {
		v, scanErr := scanPerson(rows)
		if scanErr != nil {
			s.internal(w, scanErr)
			return
		}
		items = append(items, v)
	}
	if err = rows.Err(); err != nil {
		s.internal(w, err)
		return
	}
	if !paged {
		reply(w, http.StatusOK, map[string]any{"items": items})
		return
	}
	var total int
	if err = tx.QueryRow(r.Context(), `SELECT count(*) FROM persons p WHERE `+personSearchClause, query).Scan(&total); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{"items": items, "total": total})
}

func (s *Server) createPerson(w http.ResponseWriter, r *http.Request) {
	var input struct {
		DisplayName string `json:"display_name"`
	}
	if !decode(w, r, &input) {
		return
	}
	name, ok := cleanPersonName(input.DisplayName)
	if !ok {
		fail(w, http.StatusBadRequest, "invalid_person")
		return
	}
	ss := r.Context().Value(sessionKey{}).(session)
	var id int64
	if err := s.Pool.QueryRow(r.Context(), `INSERT INTO persons(display_name,created_by) VALUES($1,$2) RETURNING id`, name, ss.User.ID).Scan(&id); err != nil {
		s.internal(w, err)
		return
	}
	v, err := scanPerson(s.Pool.QueryRow(r.Context(), `SELECT `+personColumns+` FROM persons p WHERE p.id=$1`, id))
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusCreated, v)
}

func (s *Server) getPerson(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "invalid_person_id")
	if !ok {
		return
	}
	v, err := scanPerson(s.Pool.QueryRow(r.Context(), `SELECT `+personColumns+` FROM persons p WHERE p.id=$1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, http.StatusNotFound, "person_not_found")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, v)
}

type speakerRef struct {
	ID, ClipID int64
	Index      int
	Label      string
}

// authorizedSpeaker resolves a local speaker only when its clip is visible to the
// caller; an invisible speaker is indistinguishable from a missing one.
func (s *Server) authorizedSpeaker(w http.ResponseWriter, r *http.Request) (speakerRef, bool) {
	ss := r.Context().Value(sessionKey{}).(session)
	id, ok := pathID(w, r, "invalid_local_speaker_id")
	if !ok {
		return speakerRef{}, false
	}
	ref := speakerRef{ID: id}
	err := s.Pool.QueryRow(r.Context(), `SELECT ls.clip_id,ls.audio_stream_index,ls.label FROM local_speakers ls JOIN clips c ON c.id=ls.clip_id WHERE ls.id=$1 AND (c.visibility='shared' OR c.owner_id=$2 OR $3)`, id, ss.User.ID, isAdministrator(ss.User.Role)).Scan(&ref.ClipID, &ref.Index, &ref.Label)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, http.StatusNotFound, "local_speaker_not_found")
		return speakerRef{}, false
	}
	if err != nil {
		s.internal(w, err)
		return speakerRef{}, false
	}
	return ref, true
}

type attributionInput struct {
	PersonID         *int64  `json:"person_id"`
	NewPersonName    *string `json:"new_person_name"`
	ExpectedPersonID *int64  `json:"expected_person_id"`
}

func (s *Server) validateLocalSpeaker(w http.ResponseWriter, r *http.Request) {
	s.changeAttribution(w, r, false)
}

func (s *Server) correctLocalSpeaker(w http.ResponseWriter, r *http.Request) {
	s.changeAttribution(w, r, true)
}

type attributionConflict struct {
	Error       string           `json:"error"`
	Attribution *attributionView `json:"attribution,omitempty"`
}

// changeAttribution implements the first validation (correct=false) and the explicit
// correction (correct=true). The first validation is arbitrated by the partial unique
// index speaker_attributions_validated: concurrent callers race on one INSERT and the
// losers roll back everything, including a Person they created in the same call.
// The clip row is share-locked so a re-diarization (which takes it FOR UPDATE and
// replaces the local speakers) cannot interleave with a validation.
func (s *Server) changeAttribution(w http.ResponseWriter, r *http.Request, correct bool) {
	sp, ok := s.authorizedSpeaker(w, r)
	if !ok {
		return
	}
	var input attributionInput
	if !decode(w, r, &input) {
		return
	}
	name := ""
	if input.NewPersonName != nil {
		var valid bool
		if name, valid = cleanPersonName(*input.NewPersonName); !valid {
			fail(w, http.StatusBadRequest, "invalid_person")
			return
		}
	}
	if (input.PersonID == nil) == (input.NewPersonName == nil) || (input.PersonID != nil && *input.PersonID < 1) || (correct && (input.ExpectedPersonID == nil || *input.ExpectedPersonID < 1)) || (!correct && input.ExpectedPersonID != nil) {
		fail(w, http.StatusBadRequest, "invalid_request")
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
	if _, err = tx.Exec(ctx, `SELECT id FROM clips WHERE id=$1 FOR SHARE`, sp.ClipID); err != nil {
		s.internal(w, err)
		return
	}
	var lockedID int64
	if err = tx.QueryRow(ctx, `SELECT id FROM local_speakers WHERE id=$1 FOR KEY SHARE`, sp.ID).Scan(&lockedID); errors.Is(err, pgx.ErrNoRows) {
		fail(w, http.StatusNotFound, "local_speaker_not_found")
		return
	} else if err != nil {
		s.internal(w, err)
		return
	}
	var oldPersonID *int64
	var oldName string
	var oldLevel DecisionLevel
	var oldSampleID *int64
	var attributionID int64
	if correct {
		err = tx.QueryRow(ctx, `SELECT a.id,a.person_id,p.display_name FROM speaker_attributions a JOIN persons p ON p.id=a.person_id WHERE a.local_speaker_id=$1 AND a.validation_state='validated' FOR UPDATE OF a`, sp.ID).Scan(&attributionID, &oldPersonID, &oldName)
		if errors.Is(err, pgx.ErrNoRows) {
			fail(w, http.StatusConflict, "not_validated")
			return
		}
		if err != nil {
			s.internal(w, err)
			return
		}
		oldLevel = LevelHuman
		if *oldPersonID != *input.ExpectedPersonID {
			tx.Rollback(ctx)
			s.conflict(w, r, sp.ID, "attribution_changed")
			return
		}
		if input.PersonID != nil && *input.PersonID == *oldPersonID {
			fail(w, http.StatusConflict, "same_person")
			return
		}
	} else {
		// What the machine said before the human decided stays in the history.
		err = tx.QueryRow(ctx, `SELECT a.person_id,COALESCE(p.display_name,''),a.decision_level FROM speaker_attributions a LEFT JOIN persons p ON p.id=a.person_id WHERE a.local_speaker_id=$1 AND a.validation_state='unvalidated'`, sp.ID).Scan(&oldPersonID, &oldName, &oldLevel)
		if errors.Is(err, pgx.ErrNoRows) {
			oldLevel = LevelUnknown
		} else if err != nil {
			s.internal(w, err)
			return
		}
	}
	var personID int64
	var optedOut bool
	if input.NewPersonName != nil {
		if err = tx.QueryRow(ctx, `INSERT INTO persons(display_name,created_by) VALUES($1,$2) RETURNING id`, name, ss.User.ID).Scan(&personID); err != nil {
			s.internal(w, err)
			return
		}
	} else {
		personID = *input.PersonID
		if err = tx.QueryRow(ctx, `SELECT display_name,voice_profile_deleted_at IS NOT NULL FROM persons WHERE id=$1 FOR SHARE`, personID).Scan(&name, &optedOut); errors.Is(err, pgx.ErrNoRows) {
			fail(w, http.StatusNotFound, "person_not_found")
			return
		} else if err != nil {
			s.internal(w, err)
			return
		}
		if optedOut {
			// The Person deleted their voice profile: no new reference may be created.
			fail(w, http.StatusConflict, "voice_profile_deleted")
			return
		}
	}
	action := "validate"
	if correct {
		action = "correct"
		if _, err = tx.Exec(ctx, `UPDATE speaker_attributions SET person_id=$2,author_id=$3,author_username=$4,revision=revision+1,decided_at=now() WHERE id=$1`, attributionID, personID, ss.User.ID, ss.User.Username); err != nil {
			s.internal(w, err)
			return
		}
		if err = tx.QueryRow(ctx, `UPDATE voice_samples SET retired_at=now(),retired_by=$2,retired_reason='corrected' WHERE local_speaker_id=$1 AND retired_at IS NULL RETURNING id`, sp.ID, ss.User.ID).Scan(&oldSampleID); err != nil && !errors.Is(err, pgx.ErrNoRows) {
			s.internal(w, err)
			return
		}
	} else {
		err = tx.QueryRow(ctx, `INSERT INTO speaker_attributions(local_speaker_id,person_id,decision_level,validation_state,origin,author_id,author_username) VALUES($1,$2,'human','validated','human',$3,$4) ON CONFLICT (local_speaker_id) WHERE validation_state='validated' DO NOTHING RETURNING id`, sp.ID, personID, ss.User.ID, ss.User.Username).Scan(&attributionID)
		if errors.Is(err, pgx.ErrNoRows) {
			tx.Rollback(ctx)
			s.conflict(w, r, sp.ID, "already_validated")
			return
		}
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23503" {
				fail(w, http.StatusNotFound, "person_not_found")
				return
			}
			s.internal(w, err)
			return
		}
	}
	var sampleID int64
	if err = tx.QueryRow(ctx, `INSERT INTO voice_samples(person_id,local_speaker_id,clip_id,attribution_id,created_by) VALUES($1,$2,$3,$4,$5) RETURNING id`, personID, sp.ID, sp.ClipID, attributionID, ss.User.ID).Scan(&sampleID); err != nil {
		s.internal(w, err)
		return
	}
	if _, err = tx.Exec(ctx, `INSERT INTO attribution_events(local_speaker_id,clip_id,action,old_person_id,old_person_name,old_decision_level,old_sample_id,new_person_id,new_person_name,new_decision_level,new_sample_id,author_id,author_username) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,'human',$10,$11,$12)`, sp.ID, sp.ClipID, action, oldPersonID, oldName, oldLevel, oldSampleID, personID, name, sampleID, ss.User.ID, ss.User.Username); err != nil {
		s.internal(w, err)
		return
	}
	if err = tx.Commit(ctx); err != nil {
		s.internal(w, err)
		return
	}
	// Adding, correcting or retiring a reference re-examines the stored fingerprints.
	s.matchingChanged(ctx, 0)
	view, err := s.validatedAttribution(ctx, sp.ID)
	if err != nil {
		s.internal(w, err)
		return
	}
	status := http.StatusCreated
	if correct {
		status = http.StatusOK
	}
	reply(w, status, view)
}

// conflict answers 409 with the attribution that exists now and its author.
func (s *Server) conflict(w http.ResponseWriter, r *http.Request, speakerID int64, code string) {
	existing, err := s.validatedAttribution(r.Context(), speakerID)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusConflict, attributionConflict{Error: code, Attribution: existing})
}

type attributionEventView struct {
	ID               int64         `json:"id"`
	LocalSpeakerID   int64         `json:"local_speaker_id"`
	Action           string        `json:"action"`
	OldPerson        *personRef    `json:"old_person"`
	OldDecisionLevel DecisionLevel `json:"old_decision_level"`
	NewPerson        *personRef    `json:"new_person"`
	NewDecisionLevel DecisionLevel `json:"new_decision_level"`
	Author           *authorRef    `json:"author"`
	CreatedAt        time.Time     `json:"created_at"`
}

func (s *Server) localSpeakerHistory(w http.ResponseWriter, r *http.Request) {
	sp, ok := s.authorizedSpeaker(w, r)
	if !ok {
		return
	}
	rows, err := s.Pool.Query(r.Context(), `SELECT id,local_speaker_id,action,old_person_id,old_person_name,old_decision_level,new_person_id,new_person_name,new_decision_level,author_id,author_username,created_at FROM attribution_events WHERE local_speaker_id=$1 ORDER BY id DESC`, sp.ID)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer rows.Close()
	items := []attributionEventView{}
	for rows.Next() {
		var e attributionEventView
		var oldID, newID, authorID *int64
		var oldName, newName, username string
		if err = rows.Scan(&e.ID, &e.LocalSpeakerID, &e.Action, &oldID, &oldName, &e.OldDecisionLevel, &newID, &newName, &e.NewDecisionLevel, &authorID, &username, &e.CreatedAt); err != nil {
			s.internal(w, err)
			return
		}
		if oldID != nil {
			e.OldPerson = &personRef{ID: *oldID, DisplayName: oldName}
		}
		if newID != nil {
			e.NewPerson = &personRef{ID: *newID, DisplayName: newName}
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

// effectiveAttributionPointers is effectiveAttributions with addressable values.
func (s *Server) effectiveAttributionPointers(ctx context.Context, clipID int64) (map[int64]*attributionView, error) {
	values, err := s.effectiveAttributions(ctx, clipID)
	out := make(map[int64]*attributionView, len(values))
	for id, v := range values {
		v := v
		out[id] = &v
	}
	return out, err
}

type personImpactView struct {
	MicrophoneRules int  `json:"microphone_rules"`
	HasHiddenVoices bool `json:"has_hidden_voices"`
	// What deleting the voice profile erases: same predicates as the deletion itself.
	VoiceSamples int `json:"voice_samples"`
	Fingerprints int `json:"fingerprints"`
	Attributions int `json:"attributions"`
}

// getPersonImpact counts what a merge or a deletion of the Person would touch beyond the
// counts of the Person itself, so the confirmation can name it and a refusal can be
// explained. A clip the caller cannot see is only reported as a boolean.
func (s *Server) getPersonImpact(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "invalid_person_id")
	if !ok {
		return
	}
	ss := r.Context().Value(sessionKey{}).(session)
	var v personImpactView
	err := s.Pool.QueryRow(r.Context(), `SELECT
		(SELECT count(*) FROM source_microphone_tracks WHERE person_id=p.id)+(SELECT count(*) FROM clip_microphone_overrides WHERE person_id=p.id),
		NOT $3 AND EXISTS (SELECT 1 FROM speaker_attributions a JOIN local_speakers ls ON ls.id=a.local_speaker_id JOIN clips c ON c.id=ls.clip_id WHERE a.person_id=p.id AND NOT (c.visibility='shared' OR c.owner_id=$2)),
		(SELECT count(*) FROM voice_samples WHERE person_id=p.id),
		(SELECT count(*) FROM voice_fingerprints f WHERE f.local_speaker_id IN (SELECT a.local_speaker_id FROM speaker_attributions a WHERE a.person_id=p.id AND (a.validation_state='validated' OR NOT EXISTS (SELECT 1 FROM speaker_attributions x WHERE x.local_speaker_id=a.local_speaker_id AND x.validation_state='validated')))),
		(SELECT count(*) FROM speaker_attributions WHERE person_id=p.id)
		FROM persons p WHERE p.id=$1`, id, ss.User.ID, isAdministrator(ss.User.Role)).Scan(&v.MicrophoneRules, &v.HasHiddenVoices, &v.VoiceSamples, &v.Fingerprints, &v.Attributions)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, http.StatusNotFound, "person_not_found")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, v)
}
