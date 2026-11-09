package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"snitch/internal/jobs"
)

// The corpus of a benchmark (#287, ADR 0057): clips whose voices a human validated and, for the test clips the
// administrator certifies, whose transcript and speakers were reviewed in full. The corpus is snapshotted at
// creation: the Person of each voice is copied, so a later correction never changes an earlier run.

const benchMaxClips = 500

type benchmarkPerson struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type benchmarkClipView struct {
	ID                int64             `json:"id"`
	Name              string            `json:"name"`
	DurationSeconds   *float64          `json:"duration_seconds"`
	SegmentsTotal     int               `json:"segments_total"`
	SegmentsCorrected int               `json:"segments_corrected"`
	ValidatedPersons  []benchmarkPerson `json:"validated_persons"`
}

// listBenchmarkClips lists the clips that can enter a corpus: shared clips with a transcript or a human-validated
// voice, outside the private folders. Neither a private clip nor a clip only found in a private folder is ever
// offered (ADR 0026): a benchmark would expose their names and Persons to every Administrator.
func (s *Server) listBenchmarkClips(w http.ResponseWriter, r *http.Request) {
	rows, err := s.Pool.Query(r.Context(), `
		WITH tj AS (
			SELECT DISTINCT ON (clip_id,step) id,clip_id,CASE WHEN jsonb_typeof(result->'segments')='array' THEN jsonb_array_length(result->'segments') ELSE 0 END AS segs
			FROM jobs WHERE step LIKE 'transcription.audio.%' AND status='done' AND requested_model_id IS NULL ORDER BY clip_id,step,id DESC),
		tstat AS (SELECT clip_id,sum(segs)::int AS total FROM tj GROUP BY clip_id),
		cstat AS (SELECT sc.clip_id,count(*)::int AS corrected FROM segment_corrections sc JOIN tj ON tj.id=sc.transcription_job_id WHERE sc.text IS NOT NULL AND sc.stale_at IS NULL GROUP BY sc.clip_id),
		vs AS (SELECT ls.clip_id,jsonb_agg(DISTINCT jsonb_build_object('id',p.id,'name',p.display_name)) AS persons
			FROM local_speakers ls JOIN speaker_attributions a ON a.local_speaker_id=ls.id AND a.validation_state='validated' JOIN persons p ON p.id=a.person_id
			WHERE ls.kind='diarized' GROUP BY ls.clip_id)
		SELECT c.id,c.original_name,c.duration_seconds,COALESCE(t.total,0),COALESCE(cs.corrected,0),COALESCE(vs.persons,'[]'::jsonb)
		FROM clips c LEFT JOIN tstat t ON t.clip_id=c.id LEFT JOIN cstat cs ON cs.clip_id=c.id LEFT JOIN vs ON vs.clip_id=c.id
		WHERE c.visibility='shared' AND clip_folder_open(c.id) AND (COALESCE(t.total,0)>0 OR vs.persons IS NOT NULL)
		ORDER BY c.id DESC LIMIT $1`, benchMaxClips)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer rows.Close()
	out := []benchmarkClipView{}
	for rows.Next() {
		var c benchmarkClipView
		var persons []byte
		if err = rows.Scan(&c.ID, &c.Name, &c.DurationSeconds, &c.SegmentsTotal, &c.SegmentsCorrected, &persons); err != nil {
			s.internal(w, err)
			return
		}
		if err = json.Unmarshal(persons, &c.ValidatedPersons); err != nil {
			s.internal(w, err)
			return
		}
		out = append(out, c)
	}
	if err = rows.Err(); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{"clips": out})
}

type benchmarkModelView struct {
	ID           string `json:"id"`
	Label        string `json:"label"`
	Repo         string `json:"repo"`
	Revision     string `json:"revision"`
	Engine       string `json:"engine"`
	Active       bool   `json:"active"`
	ReadyWorkers int    `json:"ready_workers"`
	TotalWorkers int    `json:"total_workers"`
}

// listBenchmarkModels lists the catalogue models a benchmark of the stage may compare, with the number of Workers
// that really hold each one: a model ready nowhere would leave the run waiting for ever, so the screen says so.
func (s *Server) listBenchmarkModels(w http.ResponseWriter, r *http.Request) {
	stage := r.URL.Query().Get("stage")
	switch stage {
	case "fingerprint", "diarization":
	case "transcription":
		s.listTranscriptionBenchmarkModels(w, r)
		return
	default:
		fail(w, http.StatusUnprocessableEntity, "benchmark_stage_unavailable")
		return
	}
	// The active model of a kind: the fingerprint one is stored apart, the diarization one lives in the pipeline.
	rows, err := s.Pool.Query(r.Context(), `SELECT c.id,CASE WHEN c.label<>'' THEN c.label ELSE c.repo END,c.repo,c.revision,c.engine,
		CASE WHEN c.kind='fingerprint' THEN EXISTS(SELECT 1 FROM voice_active_model a WHERE a.kind='fingerprint' AND a.model_id=c.id)
		     ELSE c.repo=COALESCE((SELECT v.config->>'model' FROM voice_pipeline_active pa JOIN voice_pipeline_versions v ON v.version=pa.version WHERE pa.id=true),'') END,
		(SELECT count(*) FROM worker_voice_models wm JOIN workers w ON w.id=wm.worker_id AND w.revoked_at IS NULL
			WHERE wm.model_id=c.id AND wm.state='ready' AND NOT wm.removal_requested AND `+jobs.RuntimeReadyClause("wm", "c")+`),
		(SELECT count(*) FROM workers WHERE revoked_at IS NULL)
		FROM voice_model_catalog c WHERE c.kind=$1 AND c.enabled AND c.supported ORDER BY c.id`, stage)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer rows.Close()
	out := []benchmarkModelView{}
	for rows.Next() {
		var m benchmarkModelView
		if err = rows.Scan(&m.ID, &m.Label, &m.Repo, &m.Revision, &m.Engine, &m.Active, &m.ReadyWorkers, &m.TotalWorkers); err != nil {
			s.internal(w, err)
			return
		}
		out = append(out, m)
	}
	if err = rows.Err(); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{"models": out})
}

type benchmarkDataset struct {
	ID             int64     `json:"id"`
	Name           string    `json:"name"`
	AuthorName     string    `json:"author_name"`
	CreatedAt      time.Time `json:"created_at"`
	Stages         []string  `json:"stages"`
	TestClips      int       `json:"test_clips"`
	ReferenceClips int       `json:"reference_clips"`
	VerifiedClips  int       `json:"verified_clips"`
	// ReferenceSources lists where the references of the certified clips come from: "production" (corrected output
	// of the production pipeline, biased in its favour) and/or "import" (an STM or RTTM written by hand).
	ReferenceSources []string `json:"reference_sources"`
	Persons          int      `json:"persons"`
	DurationSeconds  float64  `json:"duration_seconds"`
}

func (s *Server) listBenchmarkDatasets(w http.ResponseWriter, r *http.Request) {
	rows, err := s.Pool.Query(r.Context(), `SELECT d.id,d.name,d.author_name,d.created_at,d.stages,
		count(c.clip_id) FILTER (WHERE c.role='test'),count(c.clip_id) FILTER (WHERE c.role='reference'),count(c.clip_id) FILTER (WHERE c.verified),
		(SELECT count(DISTINCT person_id) FROM benchmark_dataset_items WHERE dataset_id=d.id),COALESCE(sum(cl.duration_seconds),0),
		COALESCE(array_agg(DISTINCT c.reference_source ORDER BY c.reference_source) FILTER (WHERE c.reference_source IS NOT NULL),'{}')
		FROM benchmark_datasets d LEFT JOIN benchmark_dataset_clips c ON c.dataset_id=d.id LEFT JOIN clips cl ON cl.id=c.clip_id
		GROUP BY d.id ORDER BY d.id DESC`)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer rows.Close()
	out := []benchmarkDataset{}
	for rows.Next() {
		var d benchmarkDataset
		if err = rows.Scan(&d.ID, &d.Name, &d.AuthorName, &d.CreatedAt, &d.Stages, &d.TestClips, &d.ReferenceClips, &d.VerifiedClips, &d.Persons, &d.DurationSeconds, &d.ReferenceSources); err != nil {
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
		Name             string  `json:"name"`
		TestClipIDs      []int64 `json:"test_clip_ids"`
		ReferenceClipIDs []int64 `json:"reference_clip_ids"`
		VerifiedClipIDs  []int64 `json:"verified_clip_ids"`
		// ReferenceImports replace the production reference of a test clip by an STM and/or RTTM (ADR 0061).
		ReferenceImports []benchmarkImportInput `json:"reference_imports"`
	}
	if !decode(w, r, &input) {
		return
	}
	name := strings.TrimSpace(input.Name)
	if name == "" || len([]rune(name)) > 100 || len(input.TestClipIDs) == 0 || len(input.TestClipIDs)+len(input.ReferenceClipIDs) > benchMaxClips {
		fail(w, http.StatusBadRequest, "invalid_request")
		return
	}
	role := map[int64]string{}
	for _, group := range []struct {
		ids  []int64
		role string
	}{{input.TestClipIDs, "test"}, {input.ReferenceClipIDs, "reference"}} {
		for _, id := range group.ids {
			if _, dup := role[id]; dup || id < 1 {
				// A clip is a test or a reference, once: a voice is never its own reference.
				fail(w, http.StatusUnprocessableEntity, "benchmark_clip_duplicate")
				return
			}
			role[id] = group.role
		}
	}
	verified := map[int64]bool{}
	for _, id := range input.VerifiedClipIDs {
		if role[id] != "test" {
			fail(w, http.StatusUnprocessableEntity, "benchmark_verified_not_test")
			return
		}
		verified[id] = true
	}
	imports, code := prepareReferenceImports(input.ReferenceImports, role)
	if code != "" {
		fail(w, http.StatusUnprocessableEntity, code)
		return
	}
	// An imported clip is certified by its import, and the import replaces the production reference of the clip:
	// only the clips that rely on production are read from it (prodVerified).
	imported := map[int64]referenceImport{}
	for _, item := range imports {
		imported[item.clip] = item
	}
	prodVerified := map[int64]bool{}
	for id := range verified {
		if _, replaced := imported[id]; !replaced {
			prodVerified[id] = true
		}
	}
	ids := make([]int64, 0, len(role))
	for id := range role {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	var shared int
	if err := s.Pool.QueryRow(r.Context(), `SELECT count(*) FROM clips WHERE id=ANY($1) AND visibility='shared' AND clip_folder_open(id)`, ids).Scan(&shared); err != nil {
		s.internal(w, err)
		return
	}
	if shared != len(ids) {
		fail(w, http.StatusUnprocessableEntity, "benchmark_clip_invalid")
		return
	}
	// The imports carry no track: it is the one of the production transcript of the clip, which must be unique.
	for i := range imports {
		track, found, err := importTrack(r.Context(), s.Pool, imports[i].clip)
		if err != nil {
			s.internal(w, err)
			return
		}
		if !found {
			fail(w, http.StatusUnprocessableEntity, "benchmark_import_track")
			return
		}
		imports[i].track = track
		imported[imports[i].clip] = imports[i]
	}
	// A certification vouches for a transcript or speakers: a clip with neither has nothing to certify.
	if len(prodVerified) > 0 {
		verifiedIDs := make([]int64, 0, len(prodVerified))
		for id := range prodVerified {
			verifiedIDs = append(verifiedIDs, id)
		}
		var withContent int
		if err := s.Pool.QueryRow(r.Context(), `SELECT count(*) FROM clips c WHERE c.id=ANY($1) AND (
			EXISTS(SELECT 1 FROM jobs j WHERE j.clip_id=c.id AND j.step LIKE 'transcription.audio.%' AND j.status='done' AND j.requested_model_id IS NULL AND jsonb_typeof(j.result->'segments')='array' AND jsonb_array_length(j.result->'segments')>0)
			OR EXISTS(SELECT 1 FROM local_speakers ls JOIN speaker_attributions a ON a.local_speaker_id=ls.id AND a.validation_state='validated' WHERE ls.clip_id=c.id AND ls.kind='diarized' AND a.person_id IS NOT NULL))`, verifiedIDs).Scan(&withContent); err != nil {
			s.internal(w, err)
			return
		}
		if withContent != len(verifiedIDs) {
			fail(w, http.StatusUnprocessableEntity, "benchmark_verified_empty")
			return
		}
	}
	// The ground truth of the voices is the human validation, never a machine attribution. A microphone voice is
	// designated, not recognised, so it is no test of a model.
	rows, err := s.Pool.Query(r.Context(), `SELECT ls.id,a.person_id,ls.clip_id FROM local_speakers ls JOIN speaker_attributions a ON a.local_speaker_id=ls.id AND a.validation_state='validated' WHERE ls.clip_id=ANY($1) AND ls.kind='diarized' AND a.person_id IS NOT NULL ORDER BY ls.id`, ids)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer rows.Close()
	type voice struct{ speaker, person, clip int64 }
	var voices []voice
	for rows.Next() {
		var v voice
		if err = rows.Scan(&v.speaker, &v.person, &v.clip); err != nil {
			s.internal(w, err)
			return
		}
		voices = append(voices, v)
	}
	if err = rows.Err(); err != nil {
		s.internal(w, err)
		return
	}
	rows.Close()
	refPersons, testPersons, productionVoices := map[int64]bool{}, map[int64]bool{}, false
	for _, v := range voices {
		if role[v.clip] == "reference" {
			refPersons[v.person] = true
		} else {
			testPersons[v.person] = true
			productionVoices = productionVoices || prodVerified[v.clip]
		}
	}
	// The stages the corpus can measure. The voice fingerprint needs two Persons with a reference and each tested
	// Person enrolled (else no impostor trial or no score); transcription and diarization need a certified clip.
	fingerprint := len(refPersons) >= 2 && len(testPersons) > 0
	for p := range testPersons {
		fingerprint = fingerprint && refPersons[p]
	}
	transcription, verifiedVoices := false, productionVoices
	for _, item := range imports {
		transcription = transcription || item.segments != nil
		verifiedVoices = verifiedVoices || item.turns != nil
	}
	if len(prodVerified) > 0 && !transcription {
		verifiedIDs := make([]int64, 0, len(prodVerified))
		for id := range prodVerified {
			verifiedIDs = append(verifiedIDs, id)
		}
		if err = s.Pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM jobs WHERE clip_id=ANY($1) AND step LIKE 'transcription.audio.%' AND status='done' AND requested_model_id IS NULL AND jsonb_typeof(result->'segments')='array' AND jsonb_array_length(result->'segments')>0)`, verifiedIDs).Scan(&transcription); err != nil {
			s.internal(w, err)
			return
		}
	}
	stages := []string{}
	if fingerprint {
		stages = append(stages, "fingerprint")
	}
	if transcription {
		stages = append(stages, "transcription")
	}
	if verifiedVoices {
		stages = append(stages, "diarization")
	}
	// The end-to-end stage composes the transcription and the diarization (ADR 0060); the identification of a Person
	// is measured by the fingerprint stage on its own.
	if transcription && verifiedVoices {
		stages = append(stages, "e2e")
	}
	if len(stages) == 0 {
		fail(w, http.StatusUnprocessableEntity, "benchmark_dataset_unusable")
		return
	}
	tx, err := s.Pool.Begin(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	var datasetID int64
	err = tx.QueryRow(r.Context(), `INSERT INTO benchmark_datasets(name,author_id,author_name,stages) VALUES($1,$2,$3,$4) RETURNING id`, name, ss.User.ID, ss.User.Username, stages).Scan(&datasetID)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		fail(w, http.StatusConflict, "benchmark_dataset_name_taken")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	for _, id := range ids {
		if _, err = tx.Exec(r.Context(), `INSERT INTO benchmark_dataset_clips(dataset_id,clip_id,role,verified,reference_source) VALUES($1,$2,$3,$4,$5)`, datasetID, id, role[id], verified[id] || imported[id].clip == id, referenceSource(id, verified, imported)); err != nil {
			s.internal(w, err)
			return
		}
	}
	// The reference text is frozen now: the latest production transcript of each certified clip, a human correction
	// replacing the text of its segment. Later corrections never reach an existing corpus.
	if len(prodVerified) > 0 {
		verifiedIDs := make([]int64, 0, len(prodVerified))
		for id := range prodVerified {
			verifiedIDs = append(verifiedIDs, id)
		}
		if _, err = tx.Exec(r.Context(), `INSERT INTO benchmark_reference_segments(dataset_id,clip_id,audio_stream_index,ordinal,start_seconds,end_seconds,text,corrected)
			SELECT $1,tj.clip_id,substring(tj.step FROM '[0-9]+$')::int,(seg.ord-1)::int,(seg.val->>'start')::float8,(seg.val->>'end')::float8,COALESCE(sc.text,seg.val->>'text'),sc.text IS NOT NULL
			FROM (SELECT DISTINCT ON (clip_id,step) id,clip_id,step,result FROM jobs
				WHERE clip_id=ANY($2) AND step LIKE 'transcription.audio.%' AND status='done' AND requested_model_id IS NULL AND jsonb_typeof(result->'segments')='array'
				ORDER BY clip_id,step,id DESC) tj
			CROSS JOIN LATERAL jsonb_array_elements(tj.result->'segments') WITH ORDINALITY AS seg(val,ord)
			LEFT JOIN segment_corrections sc ON sc.transcription_job_id=tj.id AND sc.segment_index=seg.ord-1 AND sc.text IS NOT NULL AND sc.stale_at IS NULL
			WHERE btrim(COALESCE(sc.text,seg.val->>'text'))<>''`, datasetID, verifiedIDs); err != nil {
			s.internal(w, err)
			return
		}
	}
	for _, item := range imports {
		if err = freezeImportedReference(r.Context(), tx, datasetID, item); err != nil {
			s.internal(w, err)
			return
		}
	}
	if verifiedVoices {
		shifted := false
		if productionVoices {
			verifiedIDs := make([]int64, 0, len(prodVerified))
			for id := range prodVerified {
				verifiedIDs = append(verifiedIDs, id)
			}
			var snapErr error
			if shifted, snapErr = s.snapshotReferenceTurns(r.Context(), tx, datasetID, verifiedIDs); snapErr != nil {
				s.internal(w, snapErr)
				return
			}
		}
		// Who spoke each reference segment: the speaker whose turns overlap it the most (an imported text keeps the
		// speaker of its file where no turn overlaps it).
		if _, err = tx.Exec(r.Context(), `UPDATE benchmark_reference_segments s SET speaker=COALESCE((
			SELECT t.speaker FROM benchmark_reference_turns t
			WHERE t.dataset_id=s.dataset_id AND t.clip_id=s.clip_id AND t.audio_stream_index=s.audio_stream_index
			  AND least(t.end_seconds,s.end_seconds)-greatest(t.start_seconds,s.start_seconds)>0
			GROUP BY t.speaker ORDER BY sum(least(t.end_seconds,s.end_seconds)-greatest(t.start_seconds,s.start_seconds)) DESC,t.speaker LIMIT 1),s.speaker)
			WHERE s.dataset_id=$1`, datasetID); err != nil {
			s.internal(w, err)
			return
		}
		// A multitrack stream whose clock is shifted against the transcript cannot be composed with it: the stage is
		// withdrawn rather than measured on misaligned times.
		if shifted && slices.Contains(stages, "e2e") {
			stages = slices.DeleteFunc(stages, func(x string) bool { return x == "e2e" })
			if _, err = tx.Exec(r.Context(), `UPDATE benchmark_datasets SET stages=$2 WHERE id=$1`, datasetID, stages); err != nil {
				s.internal(w, err)
				return
			}
		}
	}
	for _, v := range voices {
		if _, err = tx.Exec(r.Context(), `INSERT INTO benchmark_dataset_items(dataset_id,local_speaker_id,person_id,split) VALUES($1,$2,$3,$4)`, datasetID, v.speaker, v.person, role[v.clip]); err != nil {
			s.internal(w, err)
			return
		}
	}
	if err = tx.Commit(r.Context()); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusCreated, map[string]any{"id": datasetID, "stages": stages})
}

// listTranscriptionBenchmarkModels offers the production model of the transcription service. The other models are
// named by the administrator: the server does not reach the service itself (only the Workers do), so it cannot list
// them; a name the service does not know fails on the Worker with its own error.
func (s *Server) listTranscriptionBenchmarkModels(w http.ResponseWriter, r *http.Request) {
	var model string
	err := s.Pool.QueryRow(r.Context(), `SELECT COALESCE(v.config->>'model','') FROM transcription_pipeline_active a JOIN transcription_pipeline_versions v ON v.version=a.version WHERE a.id=true`).Scan(&model)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		s.internal(w, err)
		return
	}
	var ready, total int
	if err = s.Pool.QueryRow(r.Context(), `SELECT count(*) FILTER (WHERE COALESCE(c.active_version,'')<>'' AND c.active_version=(SELECT version FROM transcription_settings WHERE id=true)),count(*)
		FROM workers w LEFT JOIN worker_transcription_config c ON c.worker_id=w.id WHERE w.revoked_at IS NULL`).Scan(&ready, &total); err != nil {
		s.internal(w, err)
		return
	}
	out := []benchmarkModelView{}
	if model != "" {
		out = append(out, benchmarkModelView{ID: model, Label: model, Repo: model, Engine: "transcription service", Active: true, ReadyWorkers: ready, TotalWorkers: total})
	}
	reply(w, http.StatusOK, map[string]any{"models": out})
}

// referenceSource says where the reference of a clip comes from: its import, else the production pipeline when the
// administrator certified it, else nothing.
func referenceSource(id int64, verified map[int64]bool, imported map[int64]referenceImport) any {
	if _, ok := imported[id]; ok {
		return "import"
	}
	if verified[id] {
		return "production"
	}
	return nil
}

// freezeImportedReference stores the imported text and turns of a clip in the corpus. Speakers are free names, not
// Persons: they are never joined to the Persons of the library.
func freezeImportedReference(ctx context.Context, tx pgQuerier, datasetID int64, item referenceImport) error {
	for i, seg := range item.segments {
		if _, err := tx.Exec(ctx, `INSERT INTO benchmark_reference_segments(dataset_id,clip_id,audio_stream_index,ordinal,start_seconds,end_seconds,text,corrected,speaker) VALUES($1,$2,$3,$4,$5,$6,$7,false,$8)`,
			datasetID, item.clip, item.track, i, seg.start, seg.end, seg.text, seg.speaker); err != nil {
			return err
		}
	}
	for _, turn := range item.turns {
		if _, err := tx.Exec(ctx, `INSERT INTO benchmark_reference_turns(dataset_id,clip_id,audio_stream_index,speaker,start_seconds,end_seconds) VALUES($1,$2,$3,$4,$5,$6)`,
			datasetID, item.clip, item.track, turn.speaker, turn.start, turn.end); err != nil {
			return err
		}
	}
	return nil
}
