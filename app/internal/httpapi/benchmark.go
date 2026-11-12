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
	ID          int64            `json:"id"`
	DatasetID   int64            `json:"dataset_id"`
	DatasetName string           `json:"dataset_name"`
	Stage       string           `json:"stage"`
	Status      string           `json:"status"`
	ModelIDs    []string         `json:"model_ids"`
	Chains      []benchmarkChain `json:"chains"`
	// ReferenceSources: where the references of the corpus come from (production and/or import); the bias warning
	// applies only when production is among them.
	ReferenceSources []string `json:"reference_sources"`
	// Scoring: for a fingerprint run, the score mode and eligibility rule it is scored under (#397); null otherwise.
	Scoring    *benchmarkScoring     `json:"scoring"`
	AuthorName string                `json:"author_name"`
	CreatedAt  time.Time             `json:"created_at"`
	FinishedAt *time.Time            `json:"finished_at"`
	Progress   benchmarkProgress     `json:"progress"`
	Results    []benchmarkResultView `json:"results"`
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
	ModelID        string           `json:"model_id"`
	Model          string           `json:"model"`
	ModelVersion   string           `json:"model_version"`
	Dimensions     int              `json:"dimensions"`
	State          string           `json:"state"`
	ItemsTotal     int              `json:"items_total"`
	ItemsScored    int              `json:"items_scored"`
	ElapsedSeconds *float64         `json:"elapsed_seconds"`
	Metrics        json.RawMessage  `json:"metrics"`
	Resources      []modelResources `json:"resources"`
}

// modelResources is the cost a Worker reported for the done tasks of one model of a run (#403). A value is null as
// soon as one task did not report it (an older Worker, no CUDA, an external transcription): never a made-up zero.
type modelResources struct {
	ModelID          string   `json:"model_id"`
	Tasks            int      `json:"tasks"`
	InferenceSeconds *float64 `json:"inference_seconds"`
	GPUPeakBytes     *int64   `json:"gpu_peak_bytes"`
}

const benchmarkRunColumns = `id,dataset_id,dataset_name,stage,status,model_ids,author_name,created_at,finished_at`

func scanBenchmarkRun(row pgx.Row) (benchmarkRunView, error) {
	v := benchmarkRunView{Results: []benchmarkResultView{}, Chains: []benchmarkChain{}, ReferenceSources: []string{}}
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
		// A read writes nothing for a run that is over; its results are still loaded.
		if out[i].Status == "running" {
			if err = s.settleBenchmarkRun(r.Context(), out[i].ID); err != nil {
				s.internal(w, err)
				return
			}
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
	if err = s.Pool.QueryRow(ctx, `SELECT COALESCE(array_agg(DISTINCT reference_source ORDER BY reference_source) FILTER (WHERE reference_source IS NOT NULL),'{}') FROM benchmark_dataset_clips WHERE dataset_id=$1`, v.DatasetID).Scan(&v.ReferenceSources); err != nil {
		return v, err
	}
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
	v.Scoring = recordedScoring(v.Stage, params)
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
	if err = res.Err(); err != nil {
		return v, err
	}
	res.Close()
	costs := map[string]modelResources{}
	// A value outside a sane range reads as not reported, never as an error of the whole run view.
	cr, err := s.Pool.Query(ctx, `SELECT rj.model_id,count(*),
		CASE WHEN bool_and(CASE WHEN jsonb_typeof(j.result->'inference_seconds')='number' THEN (j.result->>'inference_seconds')::float8 BETWEEN 0 AND 1e7 ELSE false END)
			THEN sum(CASE WHEN jsonb_typeof(j.result->'inference_seconds')='number' THEN (j.result->>'inference_seconds')::float8 END) END,
		CASE WHEN bool_and(CASE WHEN jsonb_typeof(j.result->'gpu_peak_bytes')='number' THEN (j.result->>'gpu_peak_bytes')::float8 BETWEEN 0 AND 1e15 ELSE false END)
			THEN max(CASE WHEN jsonb_typeof(j.result->'gpu_peak_bytes')='number' THEN (j.result->>'gpu_peak_bytes')::float8 END)::bigint END
		FROM benchmark_run_jobs rj JOIN jobs j ON j.id=rj.job_id WHERE rj.run_id=$1 AND j.status='done' AND j.result IS NOT NULL GROUP BY 1`, id)
	if err != nil {
		return v, err
	}
	defer cr.Close()
	for cr.Next() {
		var c modelResources
		if err = cr.Scan(&c.ModelID, &c.Tasks, &c.InferenceSeconds, &c.GPUPeakBytes); err != nil {
			return v, err
		}
		costs[c.ModelID] = c
	}
	if err = cr.Err(); err != nil {
		return v, err
	}
	for i, x := range v.Results {
		models := []string{x.ModelID}
		for _, c := range v.Chains {
			if c.ID == x.ModelID {
				models = []string{c.DiarizationModelID, c.TranscriptionModel, c.SeparationModelID}
			}
		}
		v.Results[i].Resources = []modelResources{}
		for _, m := range models {
			if c, ok := costs[m]; ok {
				v.Results[i].Resources = append(v.Results[i].Resources, c)
			}
		}
	}
	return v, nil
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
			SeparationModelID  string `json:"separation_model_id"`
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
		pairs := map[[3]string]bool{}
		for i, c := range input.Chains {
			key := [3]string{c.DiarizationModelID, c.TranscriptionModel, c.SeparationModelID}
			if pairs[key] {
				fail(w, http.StatusBadRequest, "invalid_request")
				return
			}
			pairs[key] = true
			id := fmt.Sprintf("c%d", i+1)
			chains = append(chains, benchmarkChain{id, c.DiarizationModelID, c.TranscriptionModel, c.SeparationModelID})
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
		for _, c := range chains {
			if c.SeparationModelID == "" {
				continue
			}
			var separation bool
			if err := s.Pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM voice_model_catalog WHERE id=$1 AND kind='separation' AND enabled AND supported)`, c.SeparationModelID).Scan(&separation); err != nil {
				s.internal(w, err)
				return
			}
			if !separation {
				fail(w, http.StatusUnprocessableEntity, "benchmark_model_invalid")
				return
			}
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
		params, _ = json.Marshal(map[string]any{"chains": chains, "composition": "transcript attributed by the segment rule and by the word rule (ADR 0065); cpWER of each", "profile": "production"})
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
	params = s.withProfiles(r.Context(), params, input.Stage)
	if input.Stage == benchmarkStageFingerprint {
		if params, err = withScoring(r.Context(), s.Pool, params); err != nil {
			s.internal(w, err)
			return
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

func (s *Server) cancelBenchmarkJobs(ctx context.Context, runID, adminID int64) error {
	rows, err := s.Pool.Query(ctx, `SELECT DISTINCT j.id FROM benchmark_run_jobs rj JOIN jobs j ON j.id=rj.job_id WHERE rj.run_id=$1 AND j.status IN ('pending','leased','retry_wait')`, runID)
	if err != nil {
		s.Log.Error("benchmark jobs read failed", "kind", fmtError(err))
		return err
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()
	var failed error
	for _, id := range ids {
		if err = s.Jobs.Cancel(ctx, id, adminID); err != nil && !errors.Is(err, jobs.ErrNotCancellable) && !errors.Is(err, jobs.ErrJobNotFound) {
			s.Log.Error("benchmark job cancel failed", "kind", fmtError(err))
			failed = err
		}
	}
	return failed
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
	// Cancelling never touches a production job: only the jobs this run scheduled for its own models. A task that could
	// not be cancelled is told, not hidden: it may still be running on a Worker.
	if err = s.cancelBenchmarkJobs(r.Context(), id, ss.User.ID); err != nil {
		s.internal(w, err)
		return
	}
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
	if stage == "e2e" {
		// A chain with a separation model has a second round: its separation tasks, once the others are over.
		scheduled, err := s.scheduleSeparationTasks(ctx, tx, runID, paramsRaw)
		if err != nil {
			return err
		}
		if scheduled {
			return tx.Commit(ctx)
		}
	}
	// A fingerprint run is scored under the mode and rule recorded when it started. One started before they were
	// recorded is scored under those active now, which it records first: what is shown is what was used.
	scoring := recordedScoring(stage, paramsRaw)
	if scoring != nil && scoring.ScoreMode == scoreCentroidLegacy {
		if paramsRaw, err = withScoring(ctx, tx, paramsRaw); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `UPDATE benchmark_runs SET params=$2 WHERE id=$1`, runID, paramsRaw); err != nil {
			return err
		}
		scoring = recordedScoring(stage, paramsRaw)
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
			res = benchModelResult{model: model, state: "failed", metrics: []byte(`{}`)}
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
			res, err = computeFingerprintResult(ctx, tx, runID, datasetID, model, *scoring)
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

// benchmarkFingerprintsSQL lists the items of a dataset ($4) with the latest benchmark speaker fingerprint the
// jobs of a run ($1) produced for a model ($2, repository $3), decided by the eligibility statement of production
// under the rule of the run ($5-$8), read-only: no fingerprint is requalified, nothing is written.
var benchmarkFingerprintsSQL = `WITH rule AS (
  SELECT $5::int AS version,$6::double precision AS comparable_min_seconds,$7::double precision AS contributive_min_seconds,$8::double precision AS max_clipped_ratio
), picked AS (
  SELECT i.person_id,i.split,i.local_speaker_id,f.id AS fingerprint_id
  FROM benchmark_dataset_items i
  LEFT JOIN LATERAL (SELECT id FROM voice_fingerprints
    WHERE local_speaker_id=i.local_speaker_id AND benchmark AND scope='speaker' AND model=$3
      AND fingerprint_job_id IN (SELECT job_id FROM benchmark_run_jobs WHERE run_id=$1 AND model_id=$2)
    ORDER BY id DESC LIMIT 1) f ON true
  WHERE i.dataset_id=$4
), ` + eligibilityDecidedOverSQL(`f.benchmark`, `f.id IN (SELECT fingerprint_id FROM picked)`) + `
SELECT p.person_id,p.split,d.clip_id,d.model_version,d.dimensions,f.embedding::real[],d.net,d.reason='',d.now_contributive
FROM picked p LEFT JOIN decided d ON d.id=p.fingerprint_id LEFT JOIN voice_fingerprints f ON f.id=d.id
ORDER BY p.person_id,p.local_speaker_id`

// computeFingerprintResult measures one model on the dataset from the benchmark fingerprints its own jobs
// produced in this run (fingerprint_job_id), never from an older run's rows nor from production. They are decided
// by the eligibility rule of the run and scored under its score mode, as production would decide and score them.
func computeFingerprintResult(ctx context.Context, q pgQuerier, runID, datasetID int64, modelID string, scoring benchmarkScoring) (benchModelResult, error) {
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
	rule := scoring.EligibilityRule
	rows, err := q.Query(ctx, benchmarkFingerprintsSQL, runID, modelID, repo, datasetID, rule.Version, rule.ComparableMinSeconds, rule.ContributiveMinSeconds, rule.MaxClippedRatio)
	if err != nil {
		return res, err
	}
	defer rows.Close()
	var refs, tests []benchSample
	withVector := 0
	for rows.Next() {
		var person int64
		var split string
		var clip *int64
		var version *string
		var dims *int
		var vector []float32
		var net *float64
		var comparable, contributive *bool
		if err = rows.Scan(&person, &split, &clip, &version, &dims, &vector, &net, &comparable, &contributive); err != nil {
			return res, err
		}
		res.total++
		if vector == nil {
			continue
		}
		withVector++
		if res.version == "" && version != nil {
			res.version = *version
		}
		if dims != nil && res.dimensions == 0 {
			res.dimensions = *dims
		}
		s := benchSample{Person: person, Vector: vector}
		if clip != nil {
			s.Clip = *clip
		}
		if net != nil {
			s.NetSpeech = *net
		}
		s.Comparable = comparable != nil && *comparable
		s.Contributive = contributive != nil && *contributive
		if split == "reference" {
			refs = append(refs, s)
		} else {
			tests = append(tests, s)
		}
	}
	if err = rows.Err(); err != nil {
		return res, err
	}
	rows.Close()
	if err = q.QueryRow(ctx, `SELECT extract(epoch FROM max(j.updated_at)-min(j.created_at)) FROM benchmark_run_jobs rj JOIN jobs j ON j.id=rj.job_id WHERE rj.run_id=$1 AND rj.model_id=$2 AND j.status='done'`, runID, modelID).Scan(&res.elapsed); err != nil {
		return res, err
	}
	metrics := evaluateFingerprints(scoring.ScoreMode, refs, tests, res.dimensions)
	// An item is measured when its fingerprint exists and the rule lets it play its part, as in production.
	res.scored = withVector - metrics.Ineligible
	switch {
	case res.scored == 0 || metrics.References == 0:
		res.state = "failed"
	case res.scored < res.total:
		res.state = "partial"
	default:
		res.state = "complete"
	}
	if res.metrics, err = json.Marshal(metrics); err != nil {
		return res, err
	}
	return res, nil
}
