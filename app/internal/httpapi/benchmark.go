package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
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
	mux.Handle("GET "+base+"/datasets", s.protected(true, http.HandlerFunc(s.listBenchmarkDatasets)))
	mux.Handle("POST "+base+"/datasets", s.protected(true, http.HandlerFunc(s.createBenchmarkDataset)))
	mux.Handle("DELETE "+base+"/datasets/{id}", s.protected(true, http.HandlerFunc(s.deleteBenchmarkDataset)))
	mux.Handle("GET "+base+"/runs", s.protected(true, http.HandlerFunc(s.listBenchmarkRuns)))
	mux.Handle("POST "+base+"/runs", s.protected(true, http.HandlerFunc(s.startBenchmarkRun)))
	mux.Handle("GET "+base+"/runs/{id}", s.protected(true, http.HandlerFunc(s.getBenchmarkRun)))
	mux.Handle("POST "+base+"/runs/{id}/cancel", s.protected(true, http.HandlerFunc(s.cancelBenchmarkRun)))
}

type benchmarkDataset struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name"`
	AuthorName string    `json:"author_name"`
	CreatedAt  time.Time `json:"created_at"`
	References int       `json:"references"`
	Tests      int       `json:"tests"`
	Persons    int       `json:"persons"`
	Clips      int       `json:"clips"`
}

func (s *Server) listBenchmarkDatasets(w http.ResponseWriter, r *http.Request) {
	rows, err := s.Pool.Query(r.Context(), `SELECT d.id,d.name,d.author_name,d.created_at,
		count(*) FILTER (WHERE i.split='reference'),count(*) FILTER (WHERE i.split='test'),count(DISTINCT i.person_id),count(DISTINCT ls.clip_id)
		FROM benchmark_datasets d LEFT JOIN benchmark_dataset_items i ON i.dataset_id=d.id LEFT JOIN local_speakers ls ON ls.id=i.local_speaker_id
		GROUP BY d.id ORDER BY d.id DESC`)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer rows.Close()
	out := []benchmarkDataset{}
	for rows.Next() {
		var d benchmarkDataset
		if err = rows.Scan(&d.ID, &d.Name, &d.AuthorName, &d.CreatedAt, &d.References, &d.Tests, &d.Persons, &d.Clips); err != nil {
			s.internal(w, err)
			return
		}
		out = append(out, d)
	}
	if err = rows.Err(); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{"datasets": out})
}

func (s *Server) createBenchmarkDataset(w http.ResponseWriter, r *http.Request) {
	ss := r.Context().Value(sessionKey{}).(session)
	var input struct {
		Name                string  `json:"name"`
		ReferenceSpeakerIDs []int64 `json:"reference_speaker_ids"`
		TestSpeakerIDs      []int64 `json:"test_speaker_ids"`
	}
	if !decode(w, r, &input) {
		return
	}
	name := strings.TrimSpace(input.Name)
	if name == "" || len([]rune(name)) > 100 || len(input.ReferenceSpeakerIDs)+len(input.TestSpeakerIDs) > benchMaxSpeakers {
		fail(w, http.StatusBadRequest, "invalid_request")
		return
	}
	split := map[int64]string{}
	for _, group := range []struct {
		ids   []int64
		split string
	}{{input.ReferenceSpeakerIDs, "reference"}, {input.TestSpeakerIDs, "test"}} {
		for _, id := range group.ids {
			if _, dup := split[id]; dup || id < 1 {
				// A voice cannot be its own reference: the same speaker on both sides would inflate every score.
				fail(w, http.StatusUnprocessableEntity, "benchmark_speaker_duplicate")
				return
			}
			split[id] = group.split
		}
	}
	ids := make([]int64, 0, len(split))
	for id := range split {
		ids = append(ids, id)
	}
	// The ground truth is the human validation, never a machine attribution. A microphone voice is designated,
	// not recognised, so it is no test of a model.
	rows, err := s.Pool.Query(r.Context(), `SELECT ls.id,a.person_id FROM local_speakers ls JOIN speaker_attributions a ON a.local_speaker_id=ls.id AND a.validation_state='validated' WHERE ls.id=ANY($1) AND ls.kind='diarized' AND a.person_id IS NOT NULL`, ids)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer rows.Close()
	person := map[int64]int64{}
	for rows.Next() {
		var id, p int64
		if err = rows.Scan(&id, &p); err != nil {
			s.internal(w, err)
			return
		}
		person[id] = p
	}
	if err = rows.Err(); err != nil {
		s.internal(w, err)
		return
	}
	rows.Close()
	if len(person) != len(ids) {
		fail(w, http.StatusUnprocessableEntity, "benchmark_speaker_not_validated")
		return
	}
	refPersons, testPersons := map[int64]bool{}, map[int64]bool{}
	for id, p := range person {
		if split[id] == "reference" {
			refPersons[p] = true
		} else {
			testPersons[p] = true
		}
	}
	for p := range testPersons {
		if !refPersons[p] {
			fail(w, http.StatusUnprocessableEntity, "benchmark_test_without_reference")
			return
		}
	}
	if len(refPersons) < 2 || len(testPersons) == 0 {
		// One Person gives no impostor trial: no error rate can be measured.
		fail(w, http.StatusUnprocessableEntity, "benchmark_dataset_too_small")
		return
	}
	tx, err := s.Pool.Begin(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	var datasetID int64
	err = tx.QueryRow(r.Context(), `INSERT INTO benchmark_datasets(name,author_id,author_name) VALUES($1,$2,$3) RETURNING id`, name, ss.User.ID, ss.User.Username).Scan(&datasetID)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		fail(w, http.StatusConflict, "benchmark_dataset_name_taken")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for _, id := range ids {
		if _, err = tx.Exec(r.Context(), `INSERT INTO benchmark_dataset_items(dataset_id,local_speaker_id,person_id,split) VALUES($1,$2,$3,$4)`, datasetID, id, person[id], split[id]); err != nil {
			s.internal(w, err)
			return
		}
	}
	if err = tx.Commit(r.Context()); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusCreated, map[string]any{"id": datasetID})
}

func (s *Server) deleteBenchmarkDataset(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		fail(w, http.StatusBadRequest, "invalid_request")
		return
	}
	// A dataset a run was made on stays: the history of the results points at it (ON DELETE RESTRICT).
	tag, err := s.Pool.Exec(r.Context(), `DELETE FROM benchmark_datasets d WHERE d.id=$1 AND NOT EXISTS(SELECT 1 FROM benchmark_runs WHERE dataset_id=d.id)`, id)
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
	v := benchmarkRunView{Results: []benchmarkResultView{}}
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
	}
	if !decode(w, r, &input) {
		return
	}
	if input.DatasetID < 1 || input.Stage != "fingerprint" || len(input.ModelIDs) < 1 || len(input.ModelIDs) > benchMaxModels {
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
	// Only enabled, supported fingerprint models of the catalogue: another kind would replace a production result.
	var valid int
	if err := s.Pool.QueryRow(r.Context(), `SELECT count(*) FROM voice_model_catalog WHERE id=ANY($1) AND kind='fingerprint' AND enabled AND supported`, input.ModelIDs).Scan(&valid); err != nil {
		s.internal(w, err)
		return
	}
	if valid != len(input.ModelIDs) {
		fail(w, http.StatusUnprocessableEntity, "benchmark_model_invalid")
		return
	}
	var name string
	err := s.Pool.QueryRow(r.Context(), `SELECT name FROM benchmark_datasets WHERE id=$1`, input.DatasetID).Scan(&name)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, http.StatusNotFound, "not_found")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	// The distinct fingerprint works of the dataset: one per clip, audio track and diarization.
	rows, err := s.Pool.Query(r.Context(), `SELECT DISTINCT ls.clip_id,ls.audio_stream_index,ls.diarization_job_id,dj.input_version
		FROM benchmark_dataset_items i JOIN local_speakers ls ON ls.id=i.local_speaker_id JOIN jobs dj ON dj.id=ls.diarization_job_id
		WHERE i.dataset_id=$1 ORDER BY 1,2,3`, input.DatasetID)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer rows.Close()
	type work struct {
		clip    int64
		step    string
		version string
	}
	var works []work
	for rows.Next() {
		var clip, diarization int64
		var track int
		var diarizationVersion string
		if err = rows.Scan(&clip, &track, &diarization, &diarizationVersion); err != nil {
			s.internal(w, err)
			return
		}
		pipeline, _, ok := jobs.VoiceInputVersion(diarizationVersion)
		if !ok {
			fail(w, http.StatusUnprocessableEntity, "benchmark_dataset_stale")
			return
		}
		works = append(works, work{clip, fmt.Sprintf("voice.fingerprint.%d", track), fmt.Sprintf("f5-a%d-p%s-d%d", track, pipeline, diarization)})
	}
	if err = rows.Err(); err != nil {
		s.internal(w, err)
		return
	}
	rows.Close()
	if len(works) == 0 {
		fail(w, http.StatusUnprocessableEntity, "benchmark_dataset_stale")
		return
	}
	params, _ := json.Marshal(map[string]any{"split": "reference/test", "scope": "speaker", "preprocessing": "worker pipeline", "clips": len(works)})
	var runID int64
	err = s.Pool.QueryRow(r.Context(), `INSERT INTO benchmark_runs(dataset_id,dataset_name,stage,status,model_ids,params,author_id,author_name) VALUES($1,$2,'fingerprint','running',$3,$4,$5,$6) RETURNING id`,
		input.DatasetID, name, input.ModelIDs, params, ss.User.ID, ss.User.Username).Scan(&runID)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		fail(w, http.StatusConflict, "benchmark_run_active")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	for _, model := range input.ModelIDs {
		for _, wk := range works {
			job, _, err := s.Jobs.Schedule(r.Context(), jobs.ScheduleInput{ClipID: wk.clip, Step: wk.step, InputVersion: wk.version, RequestedModelID: model})
			if err == nil {
				_, err = s.Pool.Exec(r.Context(), `INSERT INTO benchmark_run_jobs(run_id,job_id,model_id) VALUES($1,$2,$3) ON CONFLICT DO NOTHING`, runID, job.ID, model)
			}
			if err != nil {
				s.abortBenchmarkRun(r.Context(), runID, ss.User.ID)
				if errors.Is(err, jobs.ErrInvalidInput) || errors.Is(err, jobs.ErrClipNotFound) {
					fail(w, http.StatusUnprocessableEntity, "benchmark_dataset_stale")
				} else {
					s.internal(w, err)
				}
				return
			}
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
	var status string
	var datasetID int64
	var models []string
	err = tx.QueryRow(ctx, `SELECT status,dataset_id,model_ids FROM benchmark_runs WHERE id=$1 FOR UPDATE`, runID).Scan(&status, &datasetID, &models)
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
		res, err := computeFingerprintResult(ctx, tx, runID, datasetID, model)
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
