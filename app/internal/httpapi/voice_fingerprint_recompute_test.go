//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"snitch/internal/jobs"
)

// Recalculation of the stored fingerprints on the net speech (#395, spec voix-breves section 6 step 3). The library
// is synthetic: diarizations, turns and fingerprints are stored the way the Worker results of the phase 5 projected
// them (vectors without measures), then the recalculation is driven through the admin API and a simulated Worker.

type recomputePreviewView struct {
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

type recomputeReportView struct {
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

type recomputeStateBody struct {
	TransitionOpen bool `json:"transition_open"`
	Run            *struct {
		ID          int64               `json:"id"`
		Status      string              `json:"status"`
		Announced   int                 `json:"announced"`
		Author      *string             `json:"author"`
		RematchedAt *time.Time          `json:"rematched_at"`
		FinishedAt  *time.Time          `json:"finished_at"`
		Report      recomputeReportView `json:"report"`
	} `json:"run"`
}

const recomputePath = "/api/admin/voice-fingerprint-recompute"

func (h *voiceHarness) recomputePreview() recomputePreviewView {
	h.t.Helper()
	var out recomputePreviewView
	if err := json.Unmarshal(h.adminCall(http.MethodGet, recomputePath+"/preview", nil, http.StatusOK), &out); err != nil {
		h.t.Fatal(err)
	}
	return out
}

func (h *voiceHarness) recomputeCall(method, path string, body any, want int) recomputeStateBody {
	h.t.Helper()
	var out recomputeStateBody
	data := h.adminCall(method, recomputePath+path, body, want)
	if want < 300 {
		if err := json.Unmarshal(data, &out); err != nil {
			h.t.Fatal(err)
		}
	}
	return out
}

// legacySpeaker is one voice of a stored diarization: its turns, its fingerprint as the phase 5 Worker projected it
// (no measures) and whether that fingerprint was comparable.
type legacySpeaker struct {
	label   string
	turns   [][2]float64
	degrees float64
	usable  bool
}

type legacyDiarization struct {
	diarization, fingerprint int64
	speakers                 map[string]fixtureSpeaker
}

// legacyVoices stores a done diarization of track 2 of a clip, pinned to a pipeline, with its done fingerprint job.
func (h *voiceHarness) legacyVoices(clipID int64, pipeline string, speakers ...legacySpeaker) legacyDiarization {
	h.t.Helper()
	ctx := context.Background()
	out := legacyDiarization{speakers: map[string]fixtureSpeaker{}}
	if err := h.pool.QueryRow(ctx, `INSERT INTO jobs(clip_id,step,input_version,status) VALUES($1,'voice.diarization.2',$2,'done') RETURNING id`, clipID, "d5-a2-p"+pipeline).Scan(&out.diarization); err != nil {
		h.t.Fatal(err)
	}
	if err := h.pool.QueryRow(ctx, `INSERT INTO jobs(clip_id,step,input_version,status) VALUES($1,'voice.fingerprint.2',$2,'done') RETURNING id`, clipID, fmt.Sprintf("f5-a2-p%s-d%d", pipeline, out.diarization)).Scan(&out.fingerprint); err != nil {
		h.t.Fatal(err)
	}
	for ordinal, sp := range speakers {
		var id int64
		speaking := 0.0
		for _, turn := range sp.turns {
			speaking += turn[1] - turn[0]
		}
		if err := h.pool.QueryRow(ctx, `INSERT INTO local_speakers(clip_id,audio_stream_index,diarization_job_id,label,ordinal,speaking_seconds,turn_count) VALUES($1,2,$2,$3,$4,$5,$6) RETURNING id`, clipID, out.diarization, sp.label, ordinal, speaking, len(sp.turns)).Scan(&id); err != nil {
			h.t.Fatal(err)
		}
		for i, turn := range sp.turns {
			if _, err := h.pool.Exec(ctx, `INSERT INTO speaker_turns(local_speaker_id,clip_id,turn_index,start_seconds,end_seconds) VALUES($1,$2,$3,$4,$5)`, id, clipID, i, turn[0], turn[1]); err != nil {
				h.t.Fatal(err)
			}
		}
		reason := "too_short"
		if sp.usable {
			reason = ""
		}
		vector, _ := json.Marshal(angle(sp.degrees))
		if _, err := h.pool.Exec(ctx, `INSERT INTO voice_fingerprints(local_speaker_id,clip_id,fingerprint_job_id,scope,start_seconds,end_seconds,duration_seconds,quality,usable,unusable_reason,contributive,model,model_version,dimensions,embedding)
 VALUES($1,$2,$3,'speaker',0,$4,$4,0.3,$5,$6,$5,'fixture-embedding','emb-v1',4,$7::vector)`, id, clipID, out.fingerprint, speaking, sp.usable, reason, string(vector)); err != nil {
			h.t.Fatal(err)
		}
		out.speakers[sp.label] = fixtureSpeaker{ID: id, ClipID: clipID}
	}
	return out
}

// netFingerprints is a fingerprint result of the simulated Worker, computed on the net speech (measures v1).
func netFingerprints(diarization int64, items ...map[string]any) map[string]any {
	return map[string]any{"embedding_model": "fixture-embedding", "embedding_model_version": "emb-v1", "audio_stream_index": 2, "diarization_job_id": diarization, "dimensions": 4, "fingerprints": items}
}

func netItem(label string, degrees, net float64) map[string]any {
	return map[string]any{"speaker_label": label, "scope": "speaker", "start": 0, "end": net, "quality": 0.4, "embedding": angle(degrees),
		"measures": map[string]any{"version": 1, "net_speech_seconds": net, "overlap_ratio": 0, "overlapped": false}}
}

func (h *voiceHarness) speakerFingerprint(sp fixtureSpeaker) (id int64, usable bool, measured bool) {
	h.t.Helper()
	if err := h.pool.QueryRow(context.Background(), `SELECT id,usable,measures_version IS NOT NULL FROM voice_fingerprints WHERE local_speaker_id=$1 AND scope='speaker' AND NOT benchmark`, sp.ID).Scan(&id, &usable, &measured); err != nil {
		h.t.Fatal(err)
	}
	return id, usable, measured
}

func testVoiceFingerprintRecompute(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarnessClips(t, pool, 4)
	ctx := context.Background()
	clipA, clipB, clipC, clipD := h.clipIDs[0], h.clipIDs[1], h.clipIDs[2], h.clipIDs[3]
	pipeline := h.putPipeline(fakePipeline(2))

	// Clip A: Alice (validated, rejected for lack of speech before the net speech) and a short voice. Before, an
	// older diarization of clip A left Eve, validated then archived without turns.
	old := h.legacyVoices(clipA, pipeline, legacySpeaker{label: "EVE", turns: [][2]float64{{0, 3}}, degrees: 0})
	eve := old.speakers["EVE"]
	h.validate(eve, "Eve")
	for _, q := range []string{`UPDATE local_speakers SET kind='archived',archived_at=now() WHERE id=$1 AND $2::bigint>0`, `DELETE FROM speaker_turns WHERE local_speaker_id=$1 AND $2::bigint>0`, `UPDATE jobs SET updated_at=now()-interval '1 hour' WHERE id=$2 AND $1::bigint>0`} {
		if _, err := pool.Exec(ctx, q, eve.ID, old.diarization); err != nil {
			t.Fatal(err)
		}
	}
	a := h.legacyVoices(clipA, pipeline,
		legacySpeaker{label: "ALICE", turns: [][2]float64{{0, 1.4}, {2, 3.4}}, degrees: 0},
		legacySpeaker{label: "SHORT", turns: [][2]float64{{1.2, 2.2}}, degrees: 90},
		legacySpeaker{label: "SILENT", turns: [][2]float64{{5, 5.3}}, degrees: 150})
	alice, silent := a.speakers["ALICE"], a.speakers["SILENT"]
	h.validate(alice, "Alice")
	// Clip B: Bob, comparable before; a production fingerprint job of the clip is already queued.
	b := h.legacyVoices(clipB, pipeline, legacySpeaker{label: "BOB", turns: [][2]float64{{0, 4}}, degrees: 60, usable: true})
	bob := b.speakers["BOB"]
	var production int64
	if err := pool.QueryRow(ctx, `INSERT INTO jobs(clip_id,step,input_version,status) VALUES($1,'voice.fingerprint.2',$2,'pending') RETURNING id`, clipB, fmt.Sprintf("f5-a2-p%s-d%d", pipeline, b.diarization)).Scan(&production); err != nil {
		t.Fatal(err)
	}
	// Clip C has lost its source file.
	c := h.legacyVoices(clipC, pipeline, legacySpeaker{label: "CAROL", turns: [][2]float64{{0, 3}}, degrees: 120})
	if _, err := pool.Exec(ctx, `UPDATE source_files SET status='missing' WHERE clip_id=$1`, clipC); err != nil {
		t.Fatal(err)
	}
	// Clip D: an unknown voice close to Alice, rejected before the net speech.
	d := h.legacyVoices(clipD, pipeline, legacySpeaker{label: "DAVE", turns: [][2]float64{{0, 3}}, degrees: 10})
	dave := d.speakers["DAVE"]
	// Never recomputed: a fingerprint that already has its measures, a benchmark fingerprint, and one of a catalogue
	// fingerprint model that is not the active one.
	h.measuredSpeaker(clipC, "MEASURED", 30, measured{net: ptr(2.5), duration: 2.5})
	h.measuredSpeaker(clipC, "BENCH", 30, measured{duration: 2.5, benchmark: true})
	foreign := h.measuredSpeaker(clipC, "FOREIGN", 30, measured{duration: 2.5})
	if _, err := pool.Exec(ctx, `INSERT INTO voice_model_catalog(id,kind,engine,repo,revision) VALUES('recompute-foreign','fingerprint','pyannote','fixture/recompute-foreign','0123456789abcdef0123456789abcdef01234567') ON CONFLICT DO NOTHING`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE voice_fingerprints SET model='fixture/recompute-foreign' WHERE local_speaker_id=$1`, foreign.ID); err != nil {
		t.Fatal(err)
	}

	// The preview counts what the recalculation would do and changes nothing.
	protected := []string{"voice_fingerprints", "jobs", "persons", "voice_samples", "speaker_attributions", "local_speakers", "speaker_turns"}
	before := h.digests(protected...)
	preview := h.recomputePreview()
	if after := h.digests(protected...); fmt.Sprint(after) != fmt.Sprint(before) {
		t.Fatal("the preview wrote to the database")
	}
	want := recomputePreviewView{Fingerprints: 7, Recomputable: 5, Jobs: 3, Clips: 3, MissingSources: 1, Archived: 1, References: 2}
	if preview != want {
		t.Fatalf("preview: %+v, want %+v", preview, want)
	}
	peer := h.newUser("peer")
	h.userCall(peer, http.MethodGet, recomputePath+"/preview", nil, http.StatusForbidden)
	h.userCall(peer, http.MethodPost, recomputePath, map[string]any{"confirm_fingerprints": 7}, http.StatusForbidden)

	// Refused during an open fingerprint model transition (ADR 0044), and on a figure the administrator did not see.
	if _, err := pool.Exec(ctx, `INSERT INTO voice_fingerprint_transitions(from_model_id,to_model_id,announced_references,author_name) VALUES('a','b',0,'fixture')`); err != nil {
		t.Fatal(err)
	}
	if !h.recomputePreview().TransitionOpen {
		t.Fatal("the preview tells the open transition")
	}
	h.adminCall(http.MethodPost, recomputePath, map[string]any{"confirm_fingerprints": 7}, http.StatusConflict)
	if _, err := pool.Exec(ctx, `DELETE FROM voice_fingerprint_transitions`); err != nil {
		t.Fatal(err)
	}
	h.adminCall(http.MethodPost, recomputePath, map[string]any{"confirm_fingerprints": 6}, http.StatusConflict)
	h.adminCall(http.MethodPost, recomputePath, map[string]any{}, http.StatusBadRequest)
	humans := []string{"persons", "voice_samples", "local_speakers", "(SELECT * FROM speaker_attributions WHERE validation_state='validated') AS"}
	humanBefore := h.digests(humans...)

	state := h.recomputeCall(http.MethodPost, "", map[string]any{"confirm_fingerprints": 7}, http.StatusCreated)
	if state.Run == nil || state.Run.Status != "running" || state.Run.Announced != 7 || state.Run.Author == nil || *state.Run.Author != "administrator" {
		t.Fatalf("started run: %+v", state.Run)
	}
	h.adminCall(http.MethodPost, recomputePath, map[string]any{"confirm_fingerprints": 7}, http.StatusConflict)
	// A fingerprint model transition cannot start during the recalculation.
	h.adminCall(http.MethodPost, "/api/admin/voice-models/fingerprint-transition", map[string]any{"to_model_id": "x", "confirm_references": 0}, http.StatusConflict)

	// One job per diarization to recompute, at a low priority, never the clip whose production job is queued: that job
	// is neither replaced nor attached to the recalculation. Nothing is rediarized.
	recomputeJobs := func(clip int64) []jobs.Job {
		t.Helper()
		rows, err := pool.Query(ctx, `SELECT id,input_version,status,priority FROM jobs WHERE clip_id=$1 AND id IN (SELECT job_id FROM voice_recompute_jobs) ORDER BY id`, clip)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		var out []jobs.Job
		for rows.Next() {
			var j jobs.Job
			if err = rows.Scan(&j.ID, &j.InputVersion, &j.Status, &j.Priority); err != nil {
				t.Fatal(err)
			}
			out = append(out, j)
		}
		return out
	}
	jobA, jobD := recomputeJobs(clipA), recomputeJobs(clipD)
	if len(jobA) != 1 || len(jobD) != 1 || len(recomputeJobs(clipB)) != 0 || len(recomputeJobs(clipC)) != 0 {
		t.Fatalf("recalculation jobs: A %+v, D %+v", jobA, jobD)
	}
	if v := jobA[0].InputVersion; v != fmt.Sprintf("f5-a2-net-p%s-d%d", pipeline, a.diarization) || jobA[0].Priority >= 0 {
		t.Fatalf("recalculation job identity: %+v", jobA[0])
	}
	if p, dz, ok := jobs.VoiceInputVersion(jobA[0].InputVersion); !ok || p != pipeline || dz != a.diarization {
		t.Fatalf("the recalculation version keeps its pipeline and diarization: %q %d %v", p, dz, ok)
	}
	if h.count(`SELECT count(*) FROM jobs WHERE id=$1 AND status='pending'`, production) != 1 || h.count(`SELECT count(*) FROM jobs WHERE step LIKE 'voice.diarization.%' AND status<>'done'`) != 0 {
		t.Fatal("the recalculation replaced a production job or rediarized")
	}
	// The clip still shows its production fingerprint step, not the recalculation's.
	var clipView struct {
		Steps []struct {
			Step   string `json:"step"`
			Status string `json:"status"`
		} `json:"steps"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, fmt.Sprintf("/api/clips/%d/voice", clipA), nil, http.StatusOK), &clipView); err != nil {
		t.Fatal(err)
	}
	for _, s := range clipView.Steps {
		if s.Step == "voice.fingerprint.2" && s.Status != "done" {
			t.Fatalf("the clip shows the recalculation job as its fingerprint step: %+v", clipView.Steps)
		}
	}

	// The production job of clip B ends; the recalculation then schedules the clip.
	h.adminCall(http.MethodDelete, fmt.Sprintf("/api/admin/jobs/%d", production), nil, http.StatusNoContent)
	if state = h.recomputeCall(http.MethodGet, "", nil, http.StatusOK); state.Run.Report.Pending != 5 {
		t.Fatalf("progress: %+v", state.Run.Report)
	}
	jobB := recomputeJobs(clipB)
	if len(jobB) != 1 {
		t.Fatalf("clip B was not scheduled once free: %+v", jobB)
	}

	// The Worker receives the stored turns of the diarization, and nothing changes until it answers.
	aliceBefore, _, _ := h.speakerFingerprint(alice)
	h.adminCall(http.MethodPut, "/api/admin/job-controls", map[string]any{"paused": false, "max_concurrency": 2}, http.StatusOK)
	claimed := map[int64]claimedJob{}
	claimNext := func() {
		t.Helper()
		job, ok := h.claim()
		if !ok || job.Step != "voice.fingerprint.2" {
			t.Fatalf("expected a recalculation job: %+v", job)
		}
		claimed[job.ID] = job
	}
	claimNext()
	claimNext()
	if _, ok := claimed[jobA[0].ID]; !ok {
		t.Fatalf("the first scheduled jobs are claimed first: %+v", claimed)
	}
	var input struct {
		Voice struct {
			PipelineVersion string `json:"pipeline_version"`
			Diarization     struct {
				JobID    int64 `json:"job_id"`
				Speakers []struct {
					Label string `json:"label"`
					Turns []struct {
						Start float64 `json:"start"`
						End   float64 `json:"end"`
					} `json:"turns"`
				} `json:"speakers"`
			} `json:"diarization"`
		} `json:"voice"`
	}
	if err := json.Unmarshal(h.workerCall(http.MethodGet, fmt.Sprintf("/api/worker/jobs/%d/input", jobA[0].ID), nil, claimed[jobA[0].ID].Lease, http.StatusOK), &input); err != nil {
		t.Fatal(err)
	}
	if input.Voice.PipelineVersion != pipeline || input.Voice.Diarization.JobID != a.diarization || len(input.Voice.Diarization.Speakers) != 3 || len(input.Voice.Diarization.Speakers[0].Turns) != 2 || input.Voice.Diarization.Speakers[0].Turns[1].End != 3.4 {
		t.Fatalf("recalculation input: %+v", input.Voice)
	}
	if id, _, measured := h.speakerFingerprint(alice); id != aliceBefore || measured {
		t.Fatal("the old fingerprint left before the new one succeeded")
	}

	// Dave is recomputed first: no Reference matches him yet. Then Alice, whose net speech makes her comparable; the
	// short voice stays rejected; the Worker does not embed the silent one again, whose old fingerprint stays. Bob's
	// job fails: his old fingerprint stays in service.
	h.complete(claimed[jobD[0].ID], netFingerprints(d.diarization, netItem("DAVE", 10, 2.6)), http.StatusOK)
	if m := h.matched(dave); m.Attribution != nil && m.Attribution.Origin == "matching" {
		t.Fatalf("Dave matched before Alice's reference was recomputed: %+v", m)
	}
	silentBefore, _, _ := h.speakerFingerprint(silent)
	h.complete(claimed[jobA[0].ID], netFingerprints(a.diarization, netItem("ALICE", 0, 2.4), netItem("SHORT", 90, 0.6)), http.StatusOK)
	if id, _, measured := h.speakerFingerprint(silent); id != silentBefore || measured {
		t.Fatal("a voice missing from the result lost its old fingerprint")
	}
	bobBefore, _, _ := h.speakerFingerprint(bob)
	claimNext()
	h.workerCall(http.MethodPost, fmt.Sprintf("/api/worker/jobs/%d/fail", jobB[0].ID), map[string]string{"error": "invalid_input"}, claimed[jobB[0].ID].Lease, http.StatusOK)
	if id, usable, measured := h.speakerFingerprint(bob); id != bobBefore || !usable || measured {
		t.Fatal("a failed recalculation removed or changed the old fingerprint")
	}
	if _, usable, measured := h.speakerFingerprint(alice); !usable || !measured {
		t.Fatal("Alice's fingerprint was not replaced by the net speech one")
	}

	// The report, once every job ended; then the whole library is matched again, which finds Dave without ever
	// validating him.
	state = h.recomputeCall(http.MethodGet, "", nil, http.StatusOK)
	report := recomputeReportView{Fingerprints: 7, Recomputed: 3, Recovered: 2, RecoveredContributive: 2, StillRejected: 1, Errors: 2, MissingSources: 1, Archived: 1}
	if state.Run.Status != "completed" || state.Run.FinishedAt == nil || state.Run.Report != report {
		t.Fatalf("final report: %s %+v, want %+v", state.Run.Status, state.Run.Report, report)
	}
	deadline := time.Now().Add(10 * time.Second)
	for state.Run.RematchedAt == nil && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
		state = h.recomputeCall(http.MethodGet, "", nil, http.StatusOK)
	}
	if state.Run.RematchedAt == nil {
		t.Fatal("the library was not matched again at the end")
	}
	if m := h.matched(dave); m.Attribution == nil || m.Attribution.ValidationState != "unvalidated" || m.Person == nil || m.Person.ID != h.personOf(alice) || m.DecisionLevel != "suggestion" {
		t.Fatalf("Dave after the final matching: %+v", m)
	}
	if after := h.digests(humans...); fmt.Sprint(after) != fmt.Sprint(humanBefore) {
		t.Fatal("the recalculation changed a local speaker, a Person, a sample or a validation")
	}
	if _, _, measured := h.speakerFingerprint(eve); measured {
		t.Fatal("the archived speaker was recomputed")
	}
	if _, _, measured := h.speakerFingerprint(c.speakers["CAROL"]); measured {
		t.Fatal("the fingerprint without source was recomputed")
	}

	// Resuming retries the failed job only: the succeeded ones are not created again, even for the silent voice. Cancelling stops the
	// recalculation's jobs only.
	var other int64
	if err := pool.QueryRow(ctx, `INSERT INTO jobs(clip_id,step,input_version,status) VALUES($1,'voice.fingerprint.7','f5-a7-pother-d1','pending') RETURNING id`, clipC).Scan(&other); err != nil {
		t.Fatal(err)
	}
	jobsBefore := h.count(`SELECT count(*) FROM voice_recompute_jobs`)
	state = h.recomputeCall(http.MethodPost, "/resume", nil, http.StatusOK)
	if state.Run.Status != "running" || h.count(`SELECT count(*) FROM voice_recompute_jobs`) != jobsBefore+1 || len(recomputeJobs(clipB)) != 2 || len(recomputeJobs(clipA)) != 1 || len(recomputeJobs(clipD)) != 1 {
		t.Fatalf("resume: %+v", state.Run)
	}
	state = h.recomputeCall(http.MethodDelete, "", nil, http.StatusOK)
	if state.Run.Status != "cancelled" || state.Run.Report.Cancelled != 1 || state.Run.Report.Recomputed != 3 {
		t.Fatalf("cancelled run: %+v", state.Run)
	}
	if h.count(`SELECT count(*) FROM jobs WHERE id=$1 AND status='pending'`, other) != 1 {
		t.Fatal("cancelling the recalculation cancelled another job")
	}
	h.adminCall(http.MethodDelete, recomputePath, nil, http.StatusConflict)

	// What is left to recompute: only the fingerprints that were not.
	if p := h.recomputePreview(); p.Fingerprints != 4 || p.Recomputable != 2 || p.Running {
		t.Fatalf("preview after the recalculation: %+v", p)
	}
}
