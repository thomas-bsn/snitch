package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"snitch/internal/jobs"
)

// AI model benchmarks, voice fingerprint stage (#287, ADR 0056). An administrator builds a dataset from voices
// that humans validated, then runs several catalogue fingerprint models over the same clips. Every run only
// schedules jobs that ask for a precise model (ADR 0038): their fingerprints are flagged benchmark and no
// identification, attribution or matching ever reads them. The run neither changes the active model nor any
// Person, attribution or validation: it reads them once, to snapshot the ground truth.
//
// Progress and results come from the data. A run is settled when it is read and its jobs have all ended, in
// one transaction that locks the run, so an interruption or a restart loses nothing and reading twice is safe.

const (
	benchMaxSpeakers = 2000
	benchMaxModels   = 4
)

func (s *Server) registerBenchmarkRoutes(mux *http.ServeMux) {
	const base = "/api/admin/benchmarks"
	mux.Handle("GET "+base+"/clips", s.protected(true, http.HandlerFunc(s.listBenchmarkClips)))
	mux.Handle("GET "+base+"/models", s.protected(true, http.HandlerFunc(s.listBenchmarkModels)))
	mux.Handle("GET "+base+"/datasets", s.protected(true, http.HandlerFunc(s.listBenchmarkDatasets)))
	mux.Handle("POST "+base+"/datasets", s.protected(true, http.HandlerFunc(s.createBenchmarkDataset)))
	mux.Handle("DELETE "+base+"/datasets/{id}", s.protected(true, http.HandlerFunc(s.deleteBenchmarkDataset)))
	mux.Handle("GET "+base+"/runs", s.protected(true, http.HandlerFunc(s.listBenchmarkRuns)))
	mux.Handle("POST "+base+"/runs", s.protected(true, http.HandlerFunc(s.startBenchmarkRun)))
	mux.Handle("GET "+base+"/runs/{id}", s.protected(true, http.HandlerFunc(s.getBenchmarkRun)))
	mux.Handle("POST "+base+"/runs/{id}/cancel", s.protected(true, http.HandlerFunc(s.cancelBenchmarkRun)))
}

func (s *Server) deleteBenchmarkDataset(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		fail(w, http.StatusBadRequest, "invalid_request")
		return
	}
	// A dataset a run was made on stays: the history of the results points at it (ON DELETE RESTRICT).
	tag, err := s.Pool.Exec(r.Context(), `DELETE FROM benchmark_datasets d WHERE d.id=$1 AND NOT EXISTS(SELECT 1 FROM benchmark_runs WHERE dataset_id=d.id)`, id)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23503" {
		// A run started between the check and the deletion: the foreign key holds the dataset.
		fail(w, http.StatusConflict, "benchmark_dataset_used")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		var used bool
		if err = s.Pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM benchmark_runs WHERE dataset_id=$1)`, id).Scan(&used); err != nil {
			s.internal(w, err)
			return
		}
		if used {
			fail(w, http.StatusConflict, "benchmark_dataset_used")
		} else {
			fail(w, http.StatusNotFound, "not_found")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type benchmarkRunView struct {
	ID          int64                 `json:"id"`
	DatasetID   int64                 `json:"dataset_id"`
	DatasetName string                `json:"dataset_name"`
	Stage       string                `json:"stage"`
	Status      string                `json:"status"`
	ModelIDs    []string              `json:"model_ids"`
	Chains      []benchmarkChain      `json:"chains"`
	AuthorName  string                `json:"author_name"`
	CreatedAt   time.Time             `json:"created_at"`
	FinishedAt  *time.Time            `json:"finished_at"`
	Progress    benchmarkProgress     `json:"progress"`
	Results     []benchmarkResultView `json:"results"`
}

type benchmarkProgress struct {
	Jobs    int                 `json:"jobs"`
	Done    int                 `json:"done"`
	Failed  int                 `json:"failed"`
	Active  int                 `json:"active"`
	Errored []benchmarkJobError `json:"errors"`
}

type benchmarkJobError struct {
	JobID   int64  `json:"job_id"`
	ModelID string `json:"model_id"`
	Status  string `json:"status"`
	Error   string `json:"error"`
}

type benchmarkResultView struct {
	ModelID        string          `json:"model_id"`
	Model          string          `json:"model"`
	ModelVersion   string          `json:"model_version"`
	Dimensions     int             `json:"dimensions"`
	State          string          `json:"state"`
	ItemsTotal     int             `json:"items_total"`
	ItemsScored    int             `json:"items_scored"`
	ElapsedSeconds *float64        `json:"elapsed_seconds"`
	Metrics        json.RawMessage `json:"metrics"`
	// RAM and VRAM are not reported by the Worker for a job yet: the field says so instead of showing zero.
	Resources *struct{} `json:"resources"`
}

const benchmarkRunColumns = `id,dataset_id,dataset_name,stage,status,model_ids,author_name,created_at,finished_at`

func scanBenchmarkRun(row pgx.Row) (benchmarkRunView, error) {
	v := benchmarkRunView{Results: []benchmarkResultView{}, Chains: []benchmarkChain{}}
	err := row.Scan(&v.ID, &v.DatasetID, &v.DatasetName, &v.Stage, &v.Status, &v.ModelIDs, &v.AuthorName, &v.CreatedAt, &v.FinishedAt)
	return v, err
}

func (s *Server) listBenchmarkRuns(w http.ResponseWriter, r *http.Request) {
	rows, err := s.Pool.Query(r.Context(), `SELECT `+benchmarkRunColumns+` FROM benchmark_runs ORDER BY id DESC LIMIT 100`)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer rows.Close()
	out := []benchmarkRunView{}
	for rows.Next() {
		v, err := scanBenchmarkRun(rows)
		if err != nil {
			s.internal(w, err)
			return
		}
		out = append(out, v)
	}
	if err = rows.Err(); err != nil {
		s.internal(w, err)
		return
	}
	rows.Close()
	for i := range out {
		if err = s.settleBenchmarkRun(r.Context(), out[i].ID); err != nil {
			s.internal(w, err)
			return
		}
		if out[i], err = s.loadBenchmarkRun(r.Context(), out[i].ID, false); err != nil {
			s.internal(w, err)
			return
		}
	}
	reply(w, http.StatusOK, map[string]any{"runs": out})
}

func (s *Server) getBenchmarkRun(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		fail(w, http.StatusBadRequest, "invalid_request")
		return
	}
	if err = s.settleBenchmarkRun(r.Context(), id); err != nil {
		s.internal(w, err)
		return
	}
	v, err := s.loadBenchmarkRun(r.Context(), id, true)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, http.StatusNotFound, "not_found")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, v)
}

// loadBenchmarkRun reads a run with its progress and results. Job errors are listed only on the detail view.
func (s *Server) loadBenchmarkRun(ctx context.Context, id int64, detail bool) (benchmarkRunView, error) {
	v, err := scanBenchmarkRun(s.Pool.QueryRow(ctx, `SELECT `+benchmarkRunColumns+` FROM benchmark_runs WHERE id=$1`, id))
	if err != nil {
		return v, err
	}
	v.Progress.Errored = []benchmarkJobError{}
	// The pairs of models of an end-to-end run, which the ids of its results stand for.
	var params []byte
	if err = s.Pool.QueryRow(ctx, `SELECT params FROM benchmark_runs WHERE id=$1`, id).Scan(&params); err != nil {
		return v, err
	}
	var parsed struct {
		Chains []benchmarkChain `json:"chains"`
	}
	if json.Unmarshal(params, &parsed) == nil && parsed.Chains != nil {
		v.Chains = parsed.Chains
	}
	rows, err := s.Pool.Query(ctx, `SELECT j.id,rj.model_id,j.status,j.last_error FROM benchmark_run_jobs rj JOIN jobs j ON j.id=rj.job_id WHERE rj.run_id=$1 ORDER BY j.id,rj.model_id`, id)
	if err != nil {
		return v, err
	}
	defer rows.Close()
	for rows.Next() {
		var e benchmarkJobError
		if err = rows.Scan(&e.JobID, &e.ModelID, &e.Status, &e.Error); err != nil {
			return v, err
		}
		v.Progress.Jobs++
		switch e.Status {
		case "done":
			v.Progress.Done++
		case "pending", "leased", "retry_wait":
			v.Progress.Active++
		default:
			v.Progress.Failed++
		}
		if detail && e.Error != "" && e.Status != "done" {
			v.Progress.Errored = append(v.Progress.Errored, e)
		}
	}
	if err = rows.Err(); err != nil {
		return v, err
	}
	rows.Close()
	res, err := s.Pool.Query(ctx, `SELECT model_id,model,model_version,dimensions,state,items_total,items_scored,elapsed_seconds,metrics FROM benchmark_results WHERE run_id=$1 ORDER BY model_id`, id)
	if err != nil {
		return v, err
	}
	defer res.Close()
	for res.Next() {
		var x benchmarkResultView
		if err = res.Scan(&x.ModelID, &x.Model, &x.ModelVersion, &x.Dimensions, &x.State, &x.ItemsTotal, &x.ItemsScored, &x.ElapsedSeconds, &x.Metrics); err != nil {
			return v, err
		}
		v.Results = append(v.Results, x)
	}
	return v, res.Err()
}

func (s *Server) startBenchmarkRun(w http.ResponseWriter, r *http.Request) {
	ss := r.Context().Value(sessionKey{}).(session)
	var input struct {
		DatasetID int64    `json:"dataset_id"`
		Stage     string   `json:"stage"`
		ModelIDs  []string `json:"model_ids"`
		Chains    []struct {
			DiarizationModelID string `json:"diarization_model_id"`
			TranscriptionModel string `json:"transcription_model"`
		} `json:"chains"`
	}
	if !decode(w, r, &input) {
		return
	}
	var chains []benchmarkChain
	if input.Stage == "e2e" {
		// A chain is a pair of models; its id stands for it in the results.
		if len(input.ModelIDs) != 0 || len(input.Chains) < 1 || len(input.Chains) > benchMaxModels {
			fail(w, http.StatusBadRequest, "invalid_request")
			return
		}
		pairs := map[[2]string]bool{}
		for i, c := range input.Chains {
			key := [2]string{c.DiarizationModelID, c.TranscriptionModel}
			if pairs[key] {
				fail(w, http.StatusBadRequest, "invalid_request")
				return
			}
			pairs[key] = true
			id := fmt.Sprintf("c%d", i+1)
			chains = append(chains, benchmarkChain{id, c.DiarizationModelID, c.TranscriptionModel})
			input.ModelIDs = append(input.ModelIDs, id)
		}
	}
	if input.DatasetID < 1 || (input.Stage != "fingerprint" && input.Stage != "transcription" && input.Stage != "diarization" && input.Stage != "e2e") || len(input.ModelIDs) < 1 || len(input.ModelIDs) > benchMaxModels {
		fail(w, http.StatusBadRequest, "invalid_request")
		return
	}
	seen := map[string]bool{}
	for _, m := range input.ModelIDs {
		if m == "" || seen[m] {
			fail(w, http.StatusBadRequest, "invalid_request")
			return
		}
		seen[m] = true
	}
	if input.Stage == "e2e" {
		diarizers := []string{}
		for _, c := range chains {
			if !jobs.ValidTranscriptionModelName(c.TranscriptionModel) {
				fail(w, http.StatusUnprocessableEntity, "benchmark_model_invalid")
				return
			}
			diarizers = append(diarizers, c.DiarizationModelID)
		}
		var valid int
		if err := s.Pool.QueryRow(r.Context(), `SELECT count(*) FROM voice_model_catalog WHERE id=ANY($1) AND kind='diarization' AND enabled AND supported`, diarizers).Scan(&valid); err != nil {
			s.internal(w, err)
			return
		}
		distinct := map[string]bool{}
		for _, d := range diarizers {
			distinct[d] = true
		}
		if valid != len(distinct) {
			fail(w, http.StatusUnprocessableEntity, "benchmark_model_invalid")
			return
		}
	} else if input.Stage == "transcription" {
		// A transcription model is a name of the external service: the Worker finds out if it exists (ADR 0058).
		for _, m := range input.ModelIDs {
			if !jobs.ValidTranscriptionModelName(m) {
				fail(w, http.StatusUnprocessableEntity, "benchmark_model_invalid")
				return
			}
		}
	} else {
		// Only enabled, supported models of the stage's kind in the catalogue: another kind would replace a production result.
		var valid int
		if err := s.Pool.QueryRow(r.Context(), `SELECT count(*) FROM voice_model_catalog WHERE id=ANY($1) AND kind=$2 AND enabled AND supported`, input.ModelIDs, input.Stage).Scan(&valid); err != nil {
			s.internal(w, err)
			return
		}
		if valid != len(input.ModelIDs) {
			fail(w, http.StatusUnprocessableEntity, "benchmark_model_invalid")
			return
		}
	}
	// A run nobody read since its jobs ended would block the next one: settle it first.
	var running int64
	switch err := s.Pool.QueryRow(r.Context(), `SELECT id FROM benchmark_runs WHERE status='running'`).Scan(&running); {
	case err == nil:
		if err = s.settleBenchmarkRun(r.Context(), running); err != nil {
			s.internal(w, err)
			return
		}
	case !errors.Is(err, pgx.ErrNoRows):
		s.internal(w, err)
		return
	}
	// A fingerprint transition recomputes the same model groups: the two must not run together.
	if _, open, err := openTransition(r.Context(), s.Pool); err != nil {
		s.internal(w, err)
		return
	} else if open && input.Stage == "fingerprint" {
		fail(w, http.StatusConflict, "benchmark_transition_open")
		return
	}
	var name string
	var stages []string
	err := s.Pool.QueryRow(r.Context(), `SELECT name,stages FROM benchmark_datasets WHERE id=$1`, input.DatasetID).Scan(&name, &stages)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, http.StatusNotFound, "not_found")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	if !slices.Contains(stages, input.Stage) {
		fail(w, http.StatusUnprocessableEntity, "benchmark_stage_unsupported_by_dataset")
		return
	}
	var tasks []benchTask
	var params []byte
	if input.Stage == "e2e" {
		// The tasks of the two stages, shared by the chains that use the same model.
		diarWorks, _, ok := s.benchmarkWorks(w, r, "diarization", input.DatasetID)
		if !ok {
			return
		}
		asrWorks, _, ok := s.benchmarkWorks(w, r, "transcription", input.DatasetID)
		if !ok {
			return
		}
		seenDiar, seenAsr := map[string]bool{}, map[string]bool{}
		for _, c := range chains {
			if !seenDiar[c.DiarizationModelID] {
				seenDiar[c.DiarizationModelID] = true
				for _, wk := range diarWorks {
					tasks = append(tasks, benchTask{c.DiarizationModelID, wk})
				}
			}
			if !seenAsr[c.TranscriptionModel] {
				seenAsr[c.TranscriptionModel] = true
				for _, wk := range asrWorks {
					tasks = append(tasks, benchTask{c.TranscriptionModel, wk})
				}
			}
		}
		params, _ = json.Marshal(map[string]any{"chains": chains, "composition": "transcript segments take the speaker with the most overlap; cpWER", "profile": "production"})
	} else {
		works, p, ok := s.benchmarkWorks(w, r, input.Stage, input.DatasetID)
		if !ok {
			return
		}
		params = p
		for _, model := range input.ModelIDs {
			for _, wk := range works {
				tasks = append(tasks, benchTask{model, wk})
			}
		}
	}
	var runID int64
	err = s.Pool.QueryRow(r.Context(), `INSERT INTO benchmark_runs(dataset_id,dataset_name,stage,status,model_ids,params,author_id,author_name) VALUES($1,$2,$3,'running',$4,$5,$6,$7) RETURNING id`,
		input.DatasetID, name, input.Stage, input.ModelIDs, params, ss.User.ID, ss.User.Username).Scan(&runID)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		fail(w, http.StatusConflict, "benchmark_run_active")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	for _, task := range tasks {
		wk := task.work
		job, _, err := s.Jobs.Schedule(r.Context(), jobs.ScheduleInput{ClipID: wk.clip, Step: wk.step, InputVersion: wk.version, RequestedModelID: task.model})
		if err == nil {
			_, err = s.Pool.Exec(r.Context(), `INSERT INTO benchmark_run_jobs(run_id,job_id,model_id) VALUES($1,$2,$3) ON CONFLICT DO NOTHING`, runID, job.ID, task.model)
		}
		if err != nil {
			s.abortBenchmarkRun(context.WithoutCancel(r.Context()), runID, ss.User.ID)
			if errors.Is(err, jobs.ErrInvalidInput) || errors.Is(err, jobs.ErrClipNotFound) {
				fail(w, http.StatusUnprocessableEntity, "benchmark_dataset_stale")
			} else {
				s.internal(w, err)
			}
			return
		}
	}
	reply(w, http.StatusCreated, map[string]any{"id": runID})
}

// abortBenchmarkRun ends a run that could not be fully scheduled, so no half-started run blocks the next one.
func (s *Server) abortBenchmarkRun(ctx context.Context, runID, adminID int64) {
	s.cancelBenchmarkJobs(ctx, runID, adminID)
	if _, err := s.Pool.Exec(ctx, `UPDATE benchmark_runs SET status='failed',finished_at=now() WHERE id=$1 AND status='running'`, runID); err != nil {
		s.Log.Error("benchmark run abort failed", "kind", fmtError(err))
	}
}

func (s *Server) cancelBenchmarkJobs(ctx context.Context, runID, adminID int64) {
	rows, err := s.Pool.Query(ctx, `SELECT DISTINCT j.id FROM benchmark_run_jobs rj JOIN jobs j ON j.id=rj.job_id WHERE rj.run_id=$1 AND j.status IN ('pending','leased','retry_wait')`, runID)
	if err != nil {
		s.Log.Error("benchmark jobs read failed", "kind", fmtError(err))
		return
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()
	for _, id := range ids {
		if err = s.Jobs.Cancel(ctx, id, adminID); err != nil && !errors.Is(err, jobs.ErrNotCancellable) && !errors.Is(err, jobs.ErrJobNotFound) {
			s.Log.Error("benchmark job cancel failed", "kind", fmtError(err))
		}
	}
}

func (s *Server) cancelBenchmarkRun(w http.ResponseWriter, r *http.Request) {
	ss := r.Context().Value(sessionKey{}).(session)
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		fail(w, http.StatusBadRequest, "invalid_request")
		return
	}
	tag, err := s.Pool.Exec(r.Context(), `UPDATE benchmark_runs SET status='cancelled',finished_at=now() WHERE id=$1 AND status='running'`, id)
	if err != nil {
		s.internal(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		var exists bool
		if err = s.Pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM benchmark_runs WHERE id=$1)`, id).Scan(&exists); err != nil {
			s.internal(w, err)
			return
		}
		if exists {
			fail(w, http.StatusConflict, "benchmark_not_running")
		} else {
			fail(w, http.StatusNotFound, "not_found")
		}
		return
	}
	// Cancelling never touches a production job: only the jobs this run scheduled for its own models.
	s.cancelBenchmarkJobs(r.Context(), id, ss.User.ID)
	v, err := s.loadBenchmarkRun(r.Context(), id, true)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, v)
}

// settleBenchmarkRun computes and stores the results of a running run whose jobs have all ended. It does nothing
// otherwise, and is idempotent: the run row is locked, so two readers never write the results twice.
func (s *Server) settleBenchmarkRun(ctx context.Context, runID int64) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var status, stage string
	var datasetID int64
	var models []string
	var paramsRaw []byte
	err = tx.QueryRow(ctx, `SELECT status,dataset_id,model_ids,stage,params FROM benchmark_runs WHERE id=$1 FOR UPDATE`, runID).Scan(&status, &datasetID, &models, &stage, &paramsRaw)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && status != "running") {
		return nil
	}
	if err != nil {
		return err
	}
	var active int
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM benchmark_run_jobs rj JOIN jobs j ON j.id=rj.job_id WHERE rj.run_id=$1 AND j.status IN ('pending','leased','retry_wait')`, runID).Scan(&active); err != nil {
		return err
	}
	if active > 0 {
		return nil
	}
	scored := 0
	for _, model := range models {
		var res benchModelResult
		var err error
		switch stage {
		case "e2e":
			var p struct {
				Chains []benchmarkChain `json:"chains"`
			}
			if err = json.Unmarshal(paramsRaw, &p); err != nil {
				return err
			}
			for _, c := range p.Chains {
				if c.ID == model {
					res, err = computeE2EResult(ctx, tx, runID, datasetID, c)
				}
			}
		case "transcription":
			res, err = computeTranscriptionResult(ctx, tx, runID, datasetID, model)
		case "diarization":
			res, err = computeDiarizationResult(ctx, tx, runID, datasetID, model)
		default:
			res, err = computeFingerprintResult(ctx, tx, runID, datasetID, model)
		}
		if err != nil {
			return err
		}
		if res.state != "failed" {
			scored++
		}
		if _, err = tx.Exec(ctx, `INSERT INTO benchmark_results(run_id,model_id,model,model_version,dimensions,state,items_total,items_scored,elapsed_seconds,metrics) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
			runID, model, res.model, res.version, res.dimensions, res.state, res.total, res.scored, res.elapsed, res.metrics); err != nil {
			return err
		}
	}
	final := "completed"
	if scored == 0 {
		final = "failed"
	}
	if _, err = tx.Exec(ctx, `UPDATE benchmark_runs SET status=$2,finished_at=now() WHERE id=$1`, runID, final); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

type benchModelResult struct {
	model, version string
	dimensions     int
	state          string
	total, scored  int
	elapsed        *float64
	metrics        []byte
}

// computeFingerprintResult measures one model on the dataset from the benchmark fingerprints its own jobs
// produced in this run (fingerprint_job_id), never from an older run's rows nor from production.
func computeFingerprintResult(ctx context.Context, q pgQuerier, runID, datasetID int64, modelID string) (benchModelResult, error) {
	res := benchModelResult{model: modelID}
	var repo string
	err := q.QueryRow(ctx, `SELECT repo FROM voice_model_catalog WHERE id=$1`, modelID).Scan(&repo)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return res, err
	}
	if repo == "" {
		repo = modelID
	}
	res.model = repo
	rows, err := q.Query(ctx, `SELECT i.person_id,i.split,f.model_version,f.dimensions,f.embedding::real[]
		FROM benchmark_dataset_items i
		LEFT JOIN LATERAL (SELECT model_version,dimensions,embedding FROM voice_fingerprints
			WHERE local_speaker_id=i.local_speaker_id AND benchmark AND usable AND scope='speaker' AND model=$3
			  AND fingerprint_job_id IN (SELECT job_id FROM benchmark_run_jobs WHERE run_id=$1 AND model_id=$2)
			ORDER BY id DESC LIMIT 1) f ON true
		WHERE i.dataset_id=$4`, runID, modelID, repo, datasetID)
	if err != nil {
		return res, err
	}
	defer rows.Close()
	var refs, tests []benchSample
	for rows.Next() {
		var person int64
		var split string
		var version *string
		var dims *int
		var vector []float32
		if err = rows.Scan(&person, &split, &version, &dims, &vector); err != nil {
			return res, err
		}
		res.total++
		if vector == nil {
			continue
		}
		res.scored++
		if res.version == "" && version != nil {
			res.version = *version
		}
		if dims != nil && res.dimensions == 0 {
			res.dimensions = *dims
		}
		if split == "reference" {
			refs = append(refs, benchSample{person, vector})
		} else {
			tests = append(tests, benchSample{person, vector})
		}
	}
	if err = rows.Err(); err != nil {
		return res, err
	}
	rows.Close()
	if err = q.QueryRow(ctx, `SELECT extract(epoch FROM max(j.updated_at)-min(j.created_at)) FROM benchmark_run_jobs rj JOIN jobs j ON j.id=rj.job_id WHERE rj.run_id=$1 AND rj.model_id=$2 AND j.status='done'`, runID, modelID).Scan(&res.elapsed); err != nil {
		return res, err
	}
	switch {
	case res.scored == 0 || len(refs) == 0:
		res.state = "failed"
	case res.scored < res.total:
		res.state = "partial"
	default:
		res.state = "complete"
	}
	metrics := evaluateFingerprints(refs, tests, res.dimensions)
	if res.metrics, err = json.Marshal(metrics); err != nil {
		return res, err
	}
	return res, nil
}
