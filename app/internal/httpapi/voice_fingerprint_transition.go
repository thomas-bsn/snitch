package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"snitch/internal/jobs"
	"snitch/internal/workers"
)

// Changing the fingerprint model (ADR 0044). The references of the Persons are the fingerprints, in the active
// model, of the local speakers of their validated samples. They are recomputed with the target model by jobs
// that ask for it (benchmark jobs, ADR 0038): the results land in a staging space, outside the production space
// that identification reads, so the production keeps working on the current model during the whole transition.
// When every reference has its target fingerprint, Snitch switches the active model and swaps the two spaces in
// one transaction. The fingerprints of the previous model are kept, outside the production space.
//
// The fingerprints of a benchmark run (benchmark_run_jobs, ADR 0056) share the benchmark flag but are never part of
// this staging space: every query below that reads or promotes it leaves them out.
//
// The progress is always read from the data (fingerprints and jobs), never from a counter, so an interrupted
// transition resumes by running the same idempotent step again.

var transitionMu sync.Mutex

func (s *Server) registerFingerprintTransitionRoutes(mux *http.ServeMux) {
	const base = "/api/admin/voice-models/fingerprint-transition"
	mux.Handle("GET "+base+"/preview", s.protected(true, http.HandlerFunc(s.previewFingerprintTransition)))
	mux.Handle("GET "+base, s.protected(true, http.HandlerFunc(s.getFingerprintTransition)))
	mux.Handle("POST "+base, s.protected(true, http.HandlerFunc(s.startFingerprintTransition)))
	mux.Handle("POST "+base+"/resume", s.protected(true, http.HandlerFunc(s.resumeFingerprintTransition)))
	mux.Handle("DELETE "+base, s.protected(true, http.HandlerFunc(s.cancelFingerprintTransition)))
}

type fingerprintTransition struct {
	ID              int64
	From, To, Phase string
	Announced       int
	AuthorName      string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	SwitchedAt      *time.Time
}

const transitionColumns = `id,from_model_id,to_model_id,phase,announced_references,author_name,created_at,updated_at,switched_at`

func scanTransition(row pgx.Row) (fingerprintTransition, error) {
	var t fingerprintTransition
	err := row.Scan(&t.ID, &t.From, &t.To, &t.Phase, &t.Announced, &t.AuthorName, &t.CreatedAt, &t.UpdatedAt, &t.SwitchedAt)
	return t, err
}

// openTransition is the transition that is not finished, if any.
func openTransition(ctx context.Context, db pgQuerier) (fingerprintTransition, bool, error) {
	t, err := scanTransition(db.QueryRow(ctx, `SELECT `+transitionColumns+` FROM voice_fingerprint_transitions WHERE phase IN ('preparing','recomputing')`))
	if errors.Is(err, pgx.ErrNoRows) {
		return t, false, nil
	}
	return t, err == nil, err
}

// latestTransition is the open transition, or failing one the last that ended.
func latestTransition(ctx context.Context, db pgQuerier) (fingerprintTransition, bool, error) {
	t, err := scanTransition(db.QueryRow(ctx, `SELECT `+transitionColumns+` FROM voice_fingerprint_transitions ORDER BY (phase IN ('preparing','recomputing')) DESC, id DESC LIMIT 1`))
	if errors.Is(err, pgx.ErrNoRows) {
		return t, false, nil
	}
	return t, err == nil, err
}

// transitionBlocks tells whether the model is the source or the target of an open transition: it can then be
// neither uninstalled nor disabled, the old model must stay until the essential migration is complete.
func transitionBlocks(ctx context.Context, db pgQuerier, modelID string) (bool, error) {
	var blocked bool
	err := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM voice_fingerprint_transitions WHERE phase IN ('preparing','recomputing') AND $1 IN (from_model_id,to_model_id))`, modelID).Scan(&blocked)
	return blocked, err
}

// reference is the fingerprint of a Person's validated sample in the production space, and whether its
// counterpart in the target model already exists.
type reference struct {
	SpeakerID, ClipID, PersonID int64
	Step, InputVersion          string
	Done                        bool
}

// loadReferences lists the references of the production space of the model fromRepo, with their state in the
// staging space of toRepo: every contributive sample, since all of them enter a cumulative Reference (ADR 0064).
// A Person whose voice profile is deleted has no reference.
func loadReferences(ctx context.Context, db pgQuerier, fromRepo, toRepo string) ([]reference, error) {
	rows, err := db.Query(ctx, `SELECT DISTINCT ON (f.local_speaker_id) f.local_speaker_id,f.clip_id,vs.person_id,j.step,j.input_version,
		EXISTS (SELECT 1 FROM voice_fingerprints g WHERE g.local_speaker_id=f.local_speaker_id AND g.benchmark AND g.scope='speaker' AND g.model=$2 AND g.fingerprint_job_id NOT IN (SELECT job_id FROM benchmark_run_jobs))
		FROM voice_samples vs JOIN voice_fingerprints f ON f.local_speaker_id=vs.local_speaker_id JOIN jobs j ON j.id=f.fingerprint_job_id
		WHERE vs.retired_at IS NULL AND NOT EXISTS (SELECT 1 FROM persons op WHERE op.id=vs.person_id AND op.voice_profile_deleted_at IS NOT NULL)
		  AND f.contributive AND NOT f.benchmark AND f.scope='speaker' AND f.model=$1
		ORDER BY f.local_speaker_id`, fromRepo, toRepo)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []reference{}
	for rows.Next() {
		var r reference
		if err = rows.Scan(&r.SpeakerID, &r.ClipID, &r.PersonID, &r.Step, &r.InputVersion, &r.Done); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// workersHolding counts the connected Workers that hold the model ready, with its runtime ready on the same
// Worker (the same rule as dispatch, #350).
func workersHolding(ctx context.Context, db pgQuerier, modelID string) (int, error) {
	var n int
	err := db.QueryRow(ctx, `SELECT count(*) FROM worker_voice_models m JOIN voice_model_catalog c ON c.id=m.model_id JOIN workers w ON w.id=m.worker_id WHERE m.model_id=$1 AND m.state='ready' AND m.removal_requested=false AND w.revoked_at IS NULL AND w.last_seen_at >= now() - ($2 * interval '1 second') AND `+jobs.RuntimeReadyClause("m", "c"),
		modelID, int64(workers.ConnectedWindow.Seconds())).Scan(&n)
	return n, err
}

func activeFingerprintModelID(ctx context.Context, db pgQuerier) (string, error) {
	var id string
	err := db.QueryRow(ctx, `SELECT model_id FROM voice_active_model WHERE kind='fingerprint'`).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return id, err
}

// transitionTarget validates the target model of a transition and returns it with the active model.
func transitionTarget(ctx context.Context, db pgQuerier, toID string) (from, to voiceCatalogEntry, err error) {
	fromID, err := activeFingerprintModelID(ctx, db)
	if err != nil {
		return from, to, err
	}
	to, ok, err := voiceCatalogByID(ctx, db, toID)
	if err != nil {
		return from, to, err
	}
	from, fromOK, err := voiceCatalogByID(ctx, db, fromID)
	if err != nil {
		return from, to, err
	}
	if !ok || !fromOK || to.Kind != "fingerprint" || !to.Enabled || !to.Supported || to.ID == from.ID {
		return from, to, orderFailure{http.StatusBadRequest, "invalid_voice_model"}
	}
	return from, to, nil
}

type transitionPreviewView struct {
	FromModelID       string `json:"from_model_id"`
	ToModelID         string `json:"to_model_id"`
	References        int    `json:"references"`
	Clips             int    `json:"clips"`
	Persons           int    `json:"persons"`
	OtherFingerprints int    `json:"other_fingerprints"`
	Validated         int    `json:"validated_attributions"`
	WorkersReady      int    `json:"workers_ready"`
}

func previewOfTransition(ctx context.Context, db pgQuerier, toID string) (transitionPreviewView, error) {
	from, to, err := transitionTarget(ctx, db, toID)
	if err != nil {
		return transitionPreviewView{}, err
	}
	refs, err := loadReferences(ctx, db, from.Repo, to.Repo)
	if err != nil {
		return transitionPreviewView{}, err
	}
	out := transitionPreviewView{FromModelID: from.ID, ToModelID: to.ID, References: len(refs)}
	clips, persons := map[int64]bool{}, map[int64]bool{}
	for _, r := range refs {
		clips[r.ClipID], persons[r.PersonID] = true, true
	}
	out.Clips, out.Persons = len(clips), len(persons)
	speakers := make([]int64, len(refs))
	for i, r := range refs {
		speakers[i] = r.SpeakerID
	}
	if err = db.QueryRow(ctx, `SELECT count(*) FROM voice_fingerprints WHERE usable AND NOT benchmark AND scope='speaker' AND model=$1 AND local_speaker_id<>ALL($2)`, from.Repo, speakers).Scan(&out.OtherFingerprints); err != nil {
		return out, err
	}
	if err = db.QueryRow(ctx, `SELECT count(*) FROM speaker_attributions a WHERE a.validation_state='validated' AND a.local_speaker_id IN (SELECT local_speaker_id FROM voice_fingerprints WHERE NOT benchmark AND model=$1)`, from.Repo).Scan(&out.Validated); err != nil {
		return out, err
	}
	out.WorkersReady, err = workersHolding(ctx, db, to.ID)
	return out, err
}

func (s *Server) previewFingerprintTransition(w http.ResponseWriter, r *http.Request) {
	view, err := previewOfTransition(r.Context(), s.Pool, r.URL.Query().Get("to"))
	if err != nil {
		if !s.writeOrderFailure(w, err) {
			s.internal(w, err)
		}
		return
	}
	reply(w, http.StatusOK, view)
}

type transitionView struct {
	ID                  int64      `json:"id"`
	FromModelID         string     `json:"from_model_id"`
	ToModelID           string     `json:"to_model_id"`
	Phase               string     `json:"phase"`
	AnnouncedReferences int        `json:"announced_references"`
	ReferencesTotal     int        `json:"references_total"`
	ReferencesDone      int        `json:"references_done"`
	ReferencesRemaining int        `json:"references_remaining"`
	JobsActive          int        `json:"jobs_active"`
	JobsFailed          int        `json:"jobs_failed"`
	CatchUpRemaining    int        `json:"catch_up_remaining"`
	WorkersReady        int        `json:"workers_ready"`
	AuthorName          string     `json:"author_name"`
	CreatedAt           time.Time  `json:"created_at"`
	SwitchedAt          *time.Time `json:"switched_at,omitempty"`
}

type transitionStateView struct {
	Open       bool            `json:"open"`
	Transition *transitionView `json:"transition,omitempty"`
}

// jobsByStep is what the requested-model jobs of a transition say, per clip and step.
type transitionJobs struct{ active, finished bool }

func transitionJobStates(ctx context.Context, db pgQuerier, toID string, since time.Time) (map[string]transitionJobs, error) {
	rows, err := db.Query(ctx, `SELECT clip_id::text||'/'||step,bool_or(status IN ('pending','leased','retry_wait')),bool_or(status IN ('done','failed','dead'))
		FROM jobs WHERE requested_model_id=$1 AND step LIKE 'voice.fingerprint.%' AND created_at >= $2 AND status<>'cancelled' GROUP BY 1`, toID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]transitionJobs{}
	for rows.Next() {
		var key string
		var j transitionJobs
		if err = rows.Scan(&key, &j.active, &j.finished); err != nil {
			return nil, err
		}
		out[key] = j
	}
	return out, rows.Err()
}

func jobKey(r reference) string { return strconv.FormatInt(r.ClipID, 10) + "/" + r.Step }

func (s *Server) transitionStateView(ctx context.Context, t fingerprintTransition) (transitionView, error) {
	view := transitionView{ID: t.ID, FromModelID: t.From, ToModelID: t.To, Phase: t.Phase, AnnouncedReferences: t.Announced, AuthorName: t.AuthorName, CreatedAt: t.CreatedAt, SwitchedAt: t.SwitchedAt}
	from, _, err := voiceCatalogByID(ctx, s.Pool, t.From)
	if err != nil {
		return view, err
	}
	to, _, err := voiceCatalogByID(ctx, s.Pool, t.To)
	if err != nil {
		return view, err
	}
	if view.WorkersReady, err = workersHolding(ctx, s.Pool, t.To); err != nil {
		return view, err
	}
	if t.Phase == "switched" {
		view.ReferencesTotal, view.ReferencesDone = t.Announced, t.Announced
		err = s.Pool.QueryRow(ctx, `SELECT count(DISTINCT f.local_speaker_id) FROM voice_fingerprints f WHERE f.benchmark AND f.usable AND f.scope='speaker' AND f.model=$1 AND f.fingerprint_job_id NOT IN (SELECT job_id FROM benchmark_run_jobs)
			AND NOT EXISTS (SELECT 1 FROM voice_fingerprints g WHERE g.local_speaker_id=f.local_speaker_id AND NOT g.benchmark AND g.scope='speaker')`, from.Repo).Scan(&view.CatchUpRemaining)
		return view, err
	}
	refs, err := loadReferences(ctx, s.Pool, from.Repo, to.Repo)
	if err != nil {
		return view, err
	}
	states, err := transitionJobStates(ctx, s.Pool, to.ID, t.CreatedAt)
	if err != nil {
		return view, err
	}
	view.ReferencesTotal = len(refs)
	for _, r := range refs {
		switch j, seen := states[jobKey(r)]; {
		case r.Done:
			view.ReferencesDone++
		case seen && j.active:
			view.JobsActive++
		case seen && j.finished:
			view.JobsFailed++
		}
	}
	view.ReferencesRemaining = view.ReferencesTotal - view.ReferencesDone
	return view, nil
}

func (s *Server) respondTransition(w http.ResponseWriter, status int, ctx context.Context) {
	t, ok, err := latestTransition(ctx, s.Pool)
	if err != nil {
		s.internal(w, err)
		return
	}
	if !ok {
		reply(w, status, transitionStateView{})
		return
	}
	view, err := s.transitionStateView(ctx, t)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, status, transitionStateView{Open: t.Phase == "preparing" || t.Phase == "recomputing", Transition: &view})
}

func (s *Server) getFingerprintTransition(w http.ResponseWriter, r *http.Request) {
	// Reading the state also moves a transition that nobody advanced since an interruption.
	if err := s.advanceFingerprintTransition(r.Context(), false); err != nil {
		s.Log.Error("fingerprint transition step failed", "error", err.Error())
	}
	s.respondTransition(w, http.StatusOK, r.Context())
}

func (s *Server) startFingerprintTransition(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ToModelID         string `json:"to_model_id"`
		ConfirmReferences *int   `json:"confirm_references"`
	}
	if !decode(w, r, &input) {
		return
	}
	if input.ConfirmReferences == nil {
		fail(w, http.StatusBadRequest, "invalid_voice_transition")
		return
	}
	ss := r.Context().Value(sessionKey{}).(session)
	err := s.withVoiceModelLock(r.Context(), func(tx pgx.Tx) error {
		if _, open, err := openTransition(r.Context(), tx); err != nil || open {
			if err != nil {
				return err
			}
			return orderFailure{http.StatusConflict, "voice_transition_in_progress"}
		}
		// A recalculation of the fingerprints writes in the active model's group: the model waits for its end (#395).
		var recomputing bool
		if err := tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM voice_recompute_runs WHERE status='running')`).Scan(&recomputing); err != nil || recomputing {
			if err != nil {
				return err
			}
			return orderFailure{http.StatusConflict, "voice_recompute_in_progress"}
		}
		view, err := previewOfTransition(r.Context(), tx, input.ToModelID)
		if err != nil {
			return err
		}
		// The administrator confirms the number he saw: a library that changed since is shown again.
		if view.References != *input.ConfirmReferences {
			return orderFailure{http.StatusConflict, "voice_transition_preview_stale"}
		}
		_, err = tx.Exec(r.Context(), `INSERT INTO voice_fingerprint_transitions(from_model_id,to_model_id,announced_references,author_id,author_name) VALUES($1,$2,$3,$4,$5)`,
			view.FromModelID, view.ToModelID, view.References, ss.User.ID, ss.User.Username)
		return err
	})
	if err != nil {
		if !s.writeOrderFailure(w, err) {
			s.internal(w, err)
		}
		return
	}
	s.Log.Info("fingerprint transition started", "to", input.ToModelID, "references", *input.ConfirmReferences, "author_id", ss.User.ID)
	if err = s.advanceFingerprintTransition(r.Context(), false); err != nil {
		s.Log.Error("fingerprint transition step failed", "error", err.Error())
	}
	s.respondTransition(w, http.StatusCreated, r.Context())
}

func (s *Server) resumeFingerprintTransition(w http.ResponseWriter, r *http.Request) {
	t, ok, err := latestTransition(r.Context(), s.Pool)
	if err != nil {
		s.internal(w, err)
		return
	}
	if !ok || t.Phase == "cancelled" {
		fail(w, http.StatusNotFound, "voice_transition_not_found")
		return
	}
	if err = s.advanceFingerprintTransition(r.Context(), true); err != nil {
		s.internal(w, err)
		return
	}
	if t.Phase == "switched" {
		if err = s.scheduleCatchUp(r.Context(), t); err != nil {
			s.internal(w, err)
			return
		}
	}
	s.respondTransition(w, http.StatusOK, r.Context())
}

func (s *Server) cancelFingerprintTransition(w http.ResponseWriter, r *http.Request) {
	err := s.withVoiceModelLock(r.Context(), func(tx pgx.Tx) error {
		t, open, err := openTransition(r.Context(), tx)
		if err != nil {
			return err
		}
		if !open {
			if _, found, err := latestTransition(r.Context(), tx); err != nil || !found {
				if err != nil {
					return err
				}
				return orderFailure{http.StatusNotFound, "voice_transition_not_found"}
			}
			return orderFailure{http.StatusConflict, "voice_transition_switched"}
		}
		if _, err = tx.Exec(r.Context(), `UPDATE voice_fingerprint_transitions SET phase='cancelled',updated_at=now() WHERE id=$1`, t.ID); err != nil {
			return err
		}
		// The target models installed next to the source are plain models again; the administrator removes them.
		_, err = tx.Exec(r.Context(), `DELETE FROM worker_model_replacements WHERE kind='fingerprint' AND old_model_id=$1 AND new_model_id=$2 AND remove_order_id IS NULL`, t.From, t.To)
		return err
	})
	if err != nil {
		if !s.writeOrderFailure(w, err) {
			s.internal(w, err)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// advanceFingerprintTransition runs the next step of the open transition as far as the facts allow. It is
// idempotent: it is called when the transition starts, after each requested fingerprint result, at each pull of
// a Worker and when the state is read, so a transition interrupted anywhere resumes by itself. retryFailed also
// schedules again the references whose job ended without a fingerprint.
func (s *Server) advanceFingerprintTransition(ctx context.Context, retryFailed bool) error {
	transitionMu.Lock()
	defer transitionMu.Unlock()
	t, open, err := openTransition(ctx, s.Pool)
	if err != nil || !open {
		return err
	}
	from, _, err := voiceCatalogByID(ctx, s.Pool, t.From)
	if err != nil {
		return err
	}
	to, _, err := voiceCatalogByID(ctx, s.Pool, t.To)
	if err != nil {
		return err
	}
	if t.Phase == "preparing" {
		ready, err := workersHolding(ctx, s.Pool, to.ID)
		if err != nil || ready == 0 {
			return err
		}
		if _, err = s.Pool.Exec(ctx, `UPDATE voice_fingerprint_transitions SET phase='recomputing',updated_at=now() WHERE id=$1 AND phase='preparing'`, t.ID); err != nil {
			return err
		}
	}
	refs, err := loadReferences(ctx, s.Pool, from.Repo, to.Repo)
	if err != nil {
		return err
	}
	states, err := transitionJobStates(ctx, s.Pool, to.ID, t.CreatedAt)
	if err != nil {
		return err
	}
	remaining := 0
	for _, r := range refs {
		if r.Done {
			continue
		}
		remaining++
		if j, seen := states[jobKey(r)]; seen && (j.active || !retryFailed) {
			continue
		}
		if _, _, err = s.Jobs.Schedule(ctx, jobs.ScheduleInput{ClipID: r.ClipID, Step: r.Step, InputVersion: r.InputVersion, RequestedModelID: to.ID}); err != nil {
			return err
		}
	}
	if remaining > 0 {
		return nil
	}
	return s.switchFingerprintModel(ctx, t, from, to)
}

// switchFingerprintModel is the atomic end of the essential migration: the target becomes the active model, its
// staged fingerprints enter the production space and those of the previous model leave it (kept, as a benchmark
// group). The matching lock is held, so no identification reads a mix of the two spaces.
func (s *Server) switchFingerprintModel(ctx context.Context, t fingerprintTransition, from, to voiceCatalogEntry) error {
	switched := false
	voiceMatchingLock.Lock()
	err := s.withVoiceModelLock(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, lockVoiceMatching); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `UPDATE voice_fingerprint_transitions SET phase='switched',switched_at=now(),updated_at=now() WHERE id=$1 AND phase='recomputing'`, t.ID)
		if err != nil || tag.RowsAffected() == 0 {
			return err
		}
		if _, err = tx.Exec(ctx, `UPDATE voice_fingerprints SET benchmark=true WHERE NOT benchmark AND model=$1`, from.Repo); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `UPDATE voice_fingerprints SET benchmark=false WHERE benchmark AND model=$1 AND fingerprint_job_id NOT IN (SELECT job_id FROM benchmark_run_jobs)`, to.Repo); err != nil {
			return err
		}
		// Staged, they kept the decision of their pinned pipeline; in production, the eligibility rule decides (#390).
		if err = lockEligibilityShared(ctx, tx); err != nil {
			return err
		}
		if _, err = qualifyFingerprints(ctx, tx, nil); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `UPDATE voice_active_model SET model_id=$1 WHERE kind='fingerprint'`, to.ID); err != nil {
			return err
		}
		// The library is matched again in the new space; the request commits with the switch (#392).
		if err = requestMatching(ctx, tx, wholeLibrary); err != nil {
			return err
		}
		switched = true
		return nil
	})
	voiceMatchingLock.Unlock()
	if err != nil || !switched {
		return err
	}
	s.Log.Info("fingerprint model switched", "from", from.ID, "to", to.ID, "references", t.Announced)
	t.Phase = "switched"
	if err = s.scheduleCatchUp(ctx, t); err != nil {
		s.Log.Error("fingerprint catch-up failed", "error", err.Error())
	}
	s.wakeMatching(ctx)
	return nil
}

// scheduleCatchUp schedules production fingerprint jobs, with the new active model, for the voices that only
// have a fingerprint of the previous model: they are identified again as the results arrive.
func (s *Server) scheduleCatchUp(ctx context.Context, t fingerprintTransition) error {
	from, ok, err := voiceCatalogByID(ctx, s.Pool, t.From)
	if err != nil || !ok {
		return err
	}
	rows, err := s.Pool.Query(ctx, `SELECT DISTINCT j.clip_id,j.step,j.input_version FROM voice_fingerprints f JOIN jobs j ON j.id=f.fingerprint_job_id
		WHERE f.benchmark AND f.scope='speaker' AND f.model=$1 AND f.fingerprint_job_id NOT IN (SELECT job_id FROM benchmark_run_jobs)
		  AND NOT EXISTS (SELECT 1 FROM voice_fingerprints g WHERE g.local_speaker_id=f.local_speaker_id AND NOT g.benchmark AND g.scope='speaker')`, from.Repo)
	if err != nil {
		return err
	}
	var todo []jobs.ScheduleInput
	for rows.Next() {
		var in jobs.ScheduleInput
		if err = rows.Scan(&in.ClipID, &in.Step, &in.InputVersion); err != nil {
			rows.Close()
			return err
		}
		todo = append(todo, in)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return err
	}
	for _, in := range todo {
		if _, _, err = s.Jobs.Schedule(ctx, in); err != nil {
			return err
		}
	}
	return nil
}

// stepFingerprintTransition advances the open transition after an event; a failure is logged and never
// undoes the event that triggered it, the next event or a reading of the state runs the step again.
func (s *Server) stepFingerprintTransition(ctx context.Context) {
	if err := s.advanceFingerprintTransition(ctx, false); err != nil {
		s.Log.Error("fingerprint transition step failed", "error", err.Error())
	}
}
