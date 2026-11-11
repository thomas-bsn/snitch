package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"snitch/internal/jobs"
	"snitch/internal/logging"
)

// Recalculation of the stored fingerprints on the net speech (#395, spec voix-breves section 6 step 3, ADR 0064).
// The fingerprints whose vector predates the net speech (no Worker measures) are recomputed by fingerprint jobs on
// the diarizations already made: nothing is rediarized, no local speaker, validation or sample is touched. The old
// fingerprint stays in service until the job's result replaces it in the transaction of its completion
// (projectFingerprints); a failure, or a result for a diarization that is no longer current (ErrStaleResult),
// leaves it in place.
//
// The run follows the benchmark runs (jobs attached explicitly, cancellation limited to them, idempotent settling)
// and the fingerprint model transition (a preview whose figure is echoed back, progress read from the data, refused
// while a transition is open). Its jobs never carry a requested model: they are production jobs of the active model,
// whose results go to the active group only. A fingerprint of a catalogue model that is not the active one is never
// taken: its job would be refused (stale_voice_result) and would move the voice to another group.
//
// The job's input version is the production one with a marker before the pipeline, f5-a<track>-net-p<pipeline>-d<id>:
// jobs.VoiceInputVersion still reads the pipeline and the diarization, the claim still reads the pinned pipeline,
// and the readers of the clip's analysis and audio plan, which look for the exact production version, never take a
// recalculation job for the clip's own fingerprint step. A different version means jobs.Schedule would supersede it:
// the run therefore schedules its jobs itself, never for a clip whose production fingerprint job is active, and
// attaches only the jobs it created.

const recomputePriority = -10

var (
	recomputeMu         sync.Mutex
	recomputeRematching atomic.Bool
)

func (s *Server) registerFingerprintRecomputeRoutes(mux *http.ServeMux) {
	const base = "/api/admin/voice-fingerprint-recompute"
	mux.Handle("GET "+base+"/preview", s.protected(true, http.HandlerFunc(s.previewFingerprintRecompute)))
	mux.Handle("GET "+base, s.protected(true, http.HandlerFunc(s.getFingerprintRecompute)))
	mux.Handle("POST "+base, s.protected(true, http.HandlerFunc(s.startFingerprintRecompute)))
	mux.Handle("POST "+base+"/resume", s.protected(true, http.HandlerFunc(s.resumeFingerprintRecompute)))
	mux.Handle("DELETE "+base, s.protected(true, http.HandlerFunc(s.cancelFingerprintRecompute)))
}

// recomputeTargetsSQL lists the production speaker fingerprints with an original vector, each with its kind: a
// voice without a current diarization (archived after a validation, without turns) or whose diarization version is
// unknown cannot be recomputed; a clip whose audio source the database does not know is reported as missing.
var recomputeTargetsSQL = `SELECT f.id AS fingerprint_id,f.local_speaker_id,f.clip_id,ls.audio_stream_index,ls.diarization_job_id,f.usable,f.contributive,
  CASE WHEN cur.pipeline IS NULL THEN 'archived' WHEN ` + clipSourceAbsentSQL("f.clip_id") + ` THEN 'missing_source' ELSE 'recomputable' END AS kind,
  CASE WHEN cur.pipeline IS NOT NULL THEN 'voice.fingerprint.'||ls.audio_stream_index END AS step,
  CASE WHEN cur.pipeline IS NOT NULL THEN 'f5-a'||ls.audio_stream_index||'-net-p'||cur.pipeline||'-d'||cur.id END AS input_version,
  EXISTS (SELECT 1 FROM (` + activeSamplesSQL + `) s WHERE s.local_speaker_id=f.local_speaker_id) AS reference
FROM voice_fingerprints f JOIN local_speakers ls ON ls.id=f.local_speaker_id
LEFT JOIN LATERAL (SELECT d.id,substring(d.input_version FROM '^d5-a[0-9]+-p(.+)$') AS pipeline FROM jobs d
  WHERE d.clip_id=f.clip_id AND d.step='voice.diarization.'||ls.audio_stream_index AND d.status='done' ORDER BY d.updated_at DESC,d.id DESC LIMIT 1) cur
  ON ls.kind='diarized' AND cur.id=ls.diarization_job_id
WHERE NOT f.benchmark AND f.scope='speaker' AND f.measures_version IS NULL
  AND NOT EXISTS (SELECT 1 FROM voice_model_catalog c WHERE c.kind='fingerprint' AND c.repo=f.model
    AND NOT EXISTS (SELECT 1 FROM voice_active_model a WHERE a.kind='fingerprint' AND a.model_id=c.id))`

// recomputeCurrentSQL is true while the item's diarization is still the current one of its track and its voice is
// still diarized: the only case where a job can replace its fingerprint (projectFingerprints refuses the others).
const recomputeCurrentSQL = `(EXISTS (SELECT 1 FROM local_speakers ls WHERE ls.id=i.local_speaker_id AND ls.kind='diarized')
  AND i.diarization_job_id=(SELECT d.id FROM jobs d WHERE d.clip_id=i.clip_id AND d.step='voice.diarization.'||i.audio_stream_index AND d.status='done' ORDER BY d.updated_at DESC,d.id DESC LIMIT 1))`

const recomputeOriginalSQL = `EXISTS (SELECT 1 FROM voice_fingerprints o WHERE o.id=i.fingerprint_id)`

// recomputeScheduleSQL creates the jobs of the run's recomputable fingerprints that still have their original
// vector and a current diarization, one per diarization, and attaches them: $1 the run, $2 also retry the
// fingerprints whose job ended without replacing them, $3 the priority. A clip whose production fingerprint job is
// active is left for a later step: that job is never superseded, nor merged into the run.
const recomputeScheduleSQL = `WITH todo AS (
  SELECT DISTINCT i.clip_id,i.step,i.input_version FROM voice_recompute_items i LEFT JOIN jobs j ON j.id=i.job_id
  WHERE i.run_id=$1::bigint AND i.kind='recomputable' AND (i.job_id IS NULL OR ($2::boolean AND j.status IN ('failed','dead','cancelled')))
    AND ` + recomputeOriginalSQL + ` AND ` + recomputeCurrentSQL + `
    AND NOT EXISTS (SELECT 1 FROM jobs a WHERE a.clip_id=i.clip_id AND a.step=i.step AND a.requested_model_id IS NULL AND a.status IN ('pending','leased','retry_wait'))
), created AS (
  INSERT INTO jobs(clip_id,step,input_version,priority,max_attempts) SELECT clip_id,step,input_version,$3::int,3 FROM todo
  ON CONFLICT (clip_id,step,input_version,(COALESCE(requested_model_id,''))) WHERE status IN ('pending','leased','retry_wait') DO NOTHING
  RETURNING id,clip_id,step,input_version
), attached AS (
  INSERT INTO voice_recompute_jobs(job_id,run_id) SELECT id,$1::bigint FROM created
)
UPDATE voice_recompute_items i SET job_id=c.id FROM created c
WHERE i.run_id=$1 AND i.kind='recomputable' AND i.clip_id=c.clip_id AND i.step=c.step AND i.input_version=c.input_version AND ` + recomputeOriginalSQL

// recomputeReportSQL derives the state of each fingerprint of a run from the jobs and the fingerprints.
const recomputeReportSQL = `WITH st AS (
  SELECT i.before_usable,i.before_contributive,nf.usable AS now_usable,nf.contributive AS now_contributive,
    CASE WHEN i.kind<>'recomputable' THEN i.kind
         WHEN nf.usable IS NOT NULL THEN 'recomputed'
         WHEN NOT ` + recomputeOriginalSQL + ` THEN 'obsolete'
         WHEN j.status IN ('pending','leased','retry_wait') THEN 'pending'
         WHEN NOT ` + recomputeCurrentSQL + ` THEN 'archived'
         WHEN i.job_id IS NULL THEN 'pending'
         WHEN j.status='cancelled' THEN 'cancelled'
         ELSE 'error' END AS state
  FROM voice_recompute_items i LEFT JOIN jobs j ON j.id=i.job_id
  LEFT JOIN LATERAL (SELECT f.usable,f.contributive FROM voice_fingerprints f WHERE f.local_speaker_id=i.local_speaker_id AND f.fingerprint_job_id=i.job_id AND f.scope='speaker' AND NOT f.benchmark ORDER BY f.id DESC LIMIT 1) nf ON true
  WHERE i.run_id=$1)
SELECT count(*),count(*) FILTER (WHERE state='pending'),count(*) FILTER (WHERE state='recomputed'),
  count(*) FILTER (WHERE state='recomputed' AND NOT before_usable AND now_usable),
  count(*) FILTER (WHERE state='recomputed' AND NOT before_contributive AND now_contributive),
  count(*) FILTER (WHERE state='recomputed' AND NOT before_usable AND NOT now_usable),
  count(*) FILTER (WHERE state='recomputed' AND before_usable AND NOT now_usable),
  count(*) FILTER (WHERE state='error'),count(*) FILTER (WHERE state='cancelled'),count(*) FILTER (WHERE state='obsolete'),
  count(*) FILTER (WHERE state='missing_source'),count(*) FILTER (WHERE state='archived')
FROM st`

type recomputePreview struct {
	Fingerprints   int  `json:"fingerprints"`
	Recomputable   int  `json:"recomputable"`
	Jobs           int  `json:"jobs"`
	Clips          int  `json:"clips"`
	MissingSources int  `json:"missing_sources"`
	Archived       int  `json:"archived"`
	References     int  `json:"references"`
	TransitionOpen bool `json:"transition_open"`
	Running        bool `json:"running"`
}

// recomputeReport counts the fingerprints of a run by state. Recovered: rejected before, comparable now; still
// rejected: rejected before and after; lost: comparable before, not any more. Errors: the job ended without a new
// fingerprint, the old one stays in service. Obsolete: the original was replaced or removed outside the run (a new
// diarization, a deletion). Missing sources and archived voices keep their fingerprint, requalified only.
type recomputeReport struct {
	Fingerprints          int `json:"fingerprints"`
	Pending               int `json:"pending"`
	Recomputed            int `json:"recomputed"`
	Recovered             int `json:"recovered"`
	RecoveredContributive int `json:"recovered_contributive"`
	StillRejected         int `json:"still_rejected"`
	Lost                  int `json:"lost"`
	Errors                int `json:"errors"`
	Cancelled             int `json:"cancelled"`
	Obsolete              int `json:"obsolete"`
	MissingSources        int `json:"missing_sources"`
	Archived              int `json:"archived"`
}

type recomputeRunView struct {
	ID          int64           `json:"id"`
	Status      string          `json:"status"`
	Announced   int             `json:"announced"`
	Author      *string         `json:"author"`
	CreatedAt   time.Time       `json:"created_at"`
	FinishedAt  *time.Time      `json:"finished_at"`
	RematchedAt *time.Time      `json:"rematched_at"`
	Report      recomputeReport `json:"report"`
}

type recomputeStateView struct {
	TransitionOpen bool              `json:"transition_open"`
	Run            *recomputeRunView `json:"run,omitempty"`
}

func recomputeReportOf(ctx context.Context, q querier, runID int64) (recomputeReport, error) {
	var r recomputeReport
	err := q.QueryRow(ctx, recomputeReportSQL, runID).Scan(&r.Fingerprints, &r.Pending, &r.Recomputed, &r.Recovered, &r.RecoveredContributive,
		&r.StillRejected, &r.Lost, &r.Errors, &r.Cancelled, &r.Obsolete, &r.MissingSources, &r.Archived)
	return r, err
}

func previewOfRecompute(ctx context.Context, q pgQuerier) (recomputePreview, error) {
	var p recomputePreview
	err := q.QueryRow(ctx, `WITH t AS (`+recomputeTargetsSQL+`)
SELECT count(*),count(*) FILTER (WHERE kind='recomputable'),count(DISTINCT input_version) FILTER (WHERE kind='recomputable'),
  count(DISTINCT clip_id) FILTER (WHERE kind='recomputable'),count(*) FILTER (WHERE kind='missing_source'),count(*) FILTER (WHERE kind='archived'),
  count(*) FILTER (WHERE reference),EXISTS (SELECT 1 FROM voice_recompute_runs WHERE status='running')
FROM t`).Scan(&p.Fingerprints, &p.Recomputable, &p.Jobs, &p.Clips, &p.MissingSources, &p.Archived, &p.References, &p.Running)
	if err != nil {
		return p, err
	}
	_, p.TransitionOpen, err = openTransition(ctx, q)
	return p, err
}

func (s *Server) previewFingerprintRecompute(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tx, err := s.Pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		s.internal(w, err)
		return
	}
	defer tx.Rollback(ctx)
	p, err := previewOfRecompute(ctx, tx)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, p)
}

// recomputeState is the running run or, failing one, the last that ended. A run that ended has its frozen report.
func (s *Server) recomputeState(ctx context.Context) (recomputeStateView, error) {
	var out recomputeStateView
	var err error
	if _, out.TransitionOpen, err = openTransition(ctx, s.Pool); err != nil {
		return out, err
	}
	var run recomputeRunView
	var report []byte
	err = s.Pool.QueryRow(ctx, `SELECT id,status,announced,author_username,created_at,finished_at,rematched_at,report FROM voice_recompute_runs ORDER BY (status='running') DESC,id DESC LIMIT 1`).
		Scan(&run.ID, &run.Status, &run.Announced, &run.Author, &run.CreatedAt, &run.FinishedAt, &run.RematchedAt, &report)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, nil
	}
	if err != nil {
		return out, err
	}
	if report != nil {
		err = json.Unmarshal(report, &run.Report)
	} else {
		run.Report, err = recomputeReportOf(ctx, s.Pool, run.ID)
	}
	out.Run = &run
	return out, err
}

func (s *Server) respondRecompute(w http.ResponseWriter, ctx context.Context, status int) {
	view, err := s.recomputeState(ctx)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, status, view)
}

func (s *Server) getFingerprintRecompute(w http.ResponseWriter, r *http.Request) {
	// Reading the state also moves a run that nobody advanced since an interruption.
	s.stepFingerprintRecompute(r.Context())
	s.respondRecompute(w, r.Context(), http.StatusOK)
}

func (s *Server) startFingerprintRecompute(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ConfirmFingerprints *int `json:"confirm_fingerprints"`
	}
	if !decode(w, r, &input) {
		return
	}
	if input.ConfirmFingerprints == nil || *input.ConfirmFingerprints < 0 {
		fail(w, http.StatusBadRequest, "invalid_voice_recompute")
		return
	}
	author := s.authorOf(r)
	recomputeMu.Lock()
	err := s.withVoiceModelLock(r.Context(), func(tx pgx.Tx) error {
		ctx := r.Context()
		if _, open, err := openTransition(ctx, tx); err != nil || open {
			if err != nil {
				return err
			}
			return orderFailure{http.StatusConflict, "voice_transition_in_progress"}
		}
		var running bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM voice_recompute_runs WHERE status='running')`).Scan(&running); err != nil || running {
			if err != nil {
				return err
			}
			return orderFailure{http.StatusConflict, "voice_recompute_in_progress"}
		}
		var id int64
		if err := tx.QueryRow(ctx, `INSERT INTO voice_recompute_runs(status,announced,author_id,author_username) VALUES('running',$1,$2,$3) RETURNING id`, *input.ConfirmFingerprints, author.ID, author.Username).Scan(&id); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `INSERT INTO voice_recompute_items(run_id,fingerprint_id,local_speaker_id,clip_id,kind,audio_stream_index,diarization_job_id,step,input_version,before_usable,before_contributive)
SELECT $1::bigint,fingerprint_id,local_speaker_id,clip_id,kind,audio_stream_index,diarization_job_id,
  CASE WHEN kind='recomputable' THEN step END,CASE WHEN kind='recomputable' THEN input_version END,usable,contributive FROM (`+recomputeTargetsSQL+`) t`, id)
		if err != nil {
			return err
		}
		// The administrator confirms the number he saw: a library that changed since is shown again.
		if int(tag.RowsAffected()) != *input.ConfirmFingerprints {
			return orderFailure{http.StatusConflict, "voice_recompute_preview_stale"}
		}
		_, err = tx.Exec(ctx, recomputeScheduleSQL, id, false, recomputePriority)
		return err
	})
	recomputeMu.Unlock()
	if err != nil {
		if !s.writeOrderFailure(w, err) {
			s.internal(w, err)
		}
		return
	}
	s.Log.Info("fingerprint recompute started", "fingerprints", *input.ConfirmFingerprints, "author_id", author.ID)
	s.stepFingerprintRecompute(r.Context())
	s.respondRecompute(w, r.Context(), http.StatusCreated)
}

// resumeFingerprintRecompute reopens the last run and schedules again the fingerprints whose job ended without
// replacing them (failed, dead or cancelled). A job that succeeded is never created again.
func (s *Server) resumeFingerprintRecompute(w http.ResponseWriter, r *http.Request) {
	recomputeMu.Lock()
	err := s.withVoiceModelLock(r.Context(), func(tx pgx.Tx) error {
		ctx := r.Context()
		if _, open, err := openTransition(ctx, tx); err != nil || open {
			if err != nil {
				return err
			}
			return orderFailure{http.StatusConflict, "voice_transition_in_progress"}
		}
		var id int64
		err := tx.QueryRow(ctx, `SELECT id FROM voice_recompute_runs ORDER BY (status='running') DESC,id DESC LIMIT 1 FOR UPDATE`).Scan(&id)
		if errors.Is(err, pgx.ErrNoRows) {
			return orderFailure{http.StatusNotFound, "voice_recompute_not_found"}
		}
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `UPDATE voice_recompute_runs SET status='running',finished_at=NULL,report=NULL,rematched_at=NULL,updated_at=now() WHERE id=$1 AND status<>'running'`, id); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, recomputeScheduleSQL, id, true, recomputePriority)
		return err
	})
	recomputeMu.Unlock()
	if err != nil {
		if !s.writeOrderFailure(w, err) {
			s.internal(w, err)
		}
		return
	}
	s.stepFingerprintRecompute(r.Context())
	s.respondRecompute(w, r.Context(), http.StatusOK)
}

// cancelFingerprintRecompute ends the running run and cancels its own jobs, never another one. The fingerprints
// already recomputed stay; the report is frozen and the library matched again.
func (s *Server) cancelFingerprintRecompute(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ss := r.Context().Value(sessionKey{}).(session)
	recomputeMu.Lock()
	defer recomputeMu.Unlock()
	var id int64
	err := s.Pool.QueryRow(ctx, `UPDATE voice_recompute_runs SET status='cancelled',finished_at=now(),updated_at=now() WHERE status='running' RETURNING id`).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		var exists bool
		if err = s.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM voice_recompute_runs)`).Scan(&exists); err != nil {
			s.internal(w, err)
		} else if exists {
			fail(w, http.StatusConflict, "voice_recompute_not_running")
		} else {
			fail(w, http.StatusNotFound, "voice_recompute_not_found")
		}
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	rows, err := s.Pool.Query(ctx, `SELECT j.id FROM voice_recompute_jobs rj JOIN jobs j ON j.id=rj.job_id WHERE rj.run_id=$1 AND j.status IN ('pending','leased','retry_wait')`, id)
	if err != nil {
		s.internal(w, err)
		return
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[int64])
	if err != nil {
		s.internal(w, err)
		return
	}
	for _, job := range ids {
		if err = s.Jobs.Cancel(ctx, job, ss.User.ID); err != nil && !errors.Is(err, jobs.ErrNotCancellable) && !errors.Is(err, jobs.ErrJobNotFound) {
			s.internal(w, err)
			return
		}
	}
	report, err := recomputeReportOf(ctx, s.Pool, id)
	if err == nil {
		err = freezeRecompute(ctx, s.Pool, id, report)
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	s.rematchEndedRecompute()
	s.respondRecompute(w, ctx, http.StatusOK)
}

func freezeRecompute(ctx context.Context, q pgQuerier, id int64, report recomputeReport) error {
	raw, err := json.Marshal(report)
	if err != nil {
		return err
	}
	_, err = q.Exec(ctx, `UPDATE voice_recompute_runs SET report=$2,updated_at=now() WHERE id=$1`, id, raw)
	return err
}

// advanceFingerprintRecompute schedules what the running run can schedule now (never during an open transition),
// then completes it once no fingerprint is pending. It is idempotent: it runs when the run starts or resumes, after
// each of its results, when its state is read and periodically, so a run interrupted anywhere resumes by itself.
func (s *Server) advanceFingerprintRecompute(ctx context.Context) error {
	recomputeMu.Lock()
	defer recomputeMu.Unlock()
	var running bool
	if err := s.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM voice_recompute_runs WHERE status='running')`).Scan(&running); err != nil || !running {
		return err
	}
	err := s.withVoiceModelLock(ctx, func(tx pgx.Tx) error {
		var id int64
		err := tx.QueryRow(ctx, `SELECT id FROM voice_recompute_runs WHERE status='running' FOR UPDATE`).Scan(&id)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if _, open, err := openTransition(ctx, tx); err != nil || open {
			return err
		}
		if _, err = tx.Exec(ctx, recomputeScheduleSQL, id, false, recomputePriority); err != nil {
			return err
		}
		report, err := recomputeReportOf(ctx, tx, id)
		if err != nil || report.Pending > 0 {
			return err
		}
		if _, err = tx.Exec(ctx, `UPDATE voice_recompute_runs SET status='completed',finished_at=now() WHERE id=$1`, id); err != nil {
			return err
		}
		return freezeRecompute(ctx, tx, id, report)
	})
	if err != nil {
		return err
	}
	s.rematchEndedRecompute()
	return nil
}

func (s *Server) stepFingerprintRecompute(ctx context.Context) {
	if err := s.advanceFingerprintRecompute(ctx); err != nil {
		s.Log.Error("fingerprint recompute step failed", logging.Failure(err)...)
	}
}

// rematchEndedRecompute matches the whole library again after a run that recomputed fingerprints ended, once:
// the recomputed references may now find voices of other clips. It never validates anything (ADR 0008). It runs in
// the background and is marked done only once finished, so an interruption is caught up by the periodic step.
func (s *Server) rematchEndedRecompute() {
	var id int64
	err := s.Pool.QueryRow(context.Background(), `SELECT id FROM voice_recompute_runs WHERE status<>'running' AND rematched_at IS NULL AND (report->>'recomputed')::int>0 ORDER BY id DESC LIMIT 1`).Scan(&id)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			s.Log.Error("fingerprint recompute rematch lookup failed", logging.Failure(err)...)
		}
		return
	}
	if !recomputeRematching.CompareAndSwap(false, true) {
		return
	}
	run := func(ctx context.Context) {
		defer recomputeRematching.Store(false)
		if _, err := s.rematchVoices(ctx, 0); err != nil {
			s.Log.Error("rematch after the fingerprint recompute failed", logging.Failure(err)...)
			return
		}
		if _, err := s.Pool.Exec(ctx, `UPDATE voice_recompute_runs SET rematched_at=now() WHERE id=$1 AND status<>'running'`, id); err != nil {
			s.Log.Error("fingerprint recompute rematch record failed", logging.Failure(err)...)
		}
	}
	if s.backgroundContext == nil {
		run(context.Background())
		return
	}
	s.backgroundWG.Add(1)
	go func() {
		defer s.backgroundWG.Done()
		run(s.backgroundContext)
	}()
}

// serveFingerprintRecompute advances a running run periodically: a job that died is noticed without anybody
// reading the state.
func (s *Server) serveFingerprintRecompute(ctx context.Context, every time.Duration) {
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.stepFingerprintRecompute(ctx)
			s.rematchEndedRecompute()
		}
	}
}

// isRecomputeJob tells whether a job belongs to a recalculation run.
func isRecomputeJob(ctx context.Context, q querier, jobID int64) (bool, error) {
	var found bool
	err := q.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM voice_recompute_jobs WHERE job_id=$1)`, jobID).Scan(&found)
	return found, err
}
