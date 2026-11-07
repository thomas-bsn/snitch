//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"snitch/internal/analysis"
	"snitch/internal/httpapi/contract"
	"snitch/internal/multitrack"
)

// End-to-end activation of #274 (#344, ADR 0054): the real diagnostic dispatch, durable children, acoustic
// comparison job, publisher, confirmation guard and legacy entry points, over HTTP and a real PostgreSQL. The only
// double is the Worker itself, played through the real Worker API (claim, input, complete, fail): its detector and
// recognizer answers are scripted, because their acoustic quality is not what is tested here (the detector and the
// dedup adapter have their own suites on synthetic audio, worker/tests/test_audio_acoustic.py and
// test_acoustic_dedup.py).

type scriptSegment struct {
	Start, End float64
	Text       string
}

type audioScript struct {
	// diag is the observation of each track (status, cause, intervals); a missing track answers speech.
	diag map[int]map[string]any
	// words is what the recognizer returns for each track.
	words map[int][]scriptSegment
	// evidence builds the acoustic report from the task the Worker received; nil answers no evidence.
	evidence func(tracks []dedupTrackTask) map[string]any
	// failTranscription makes the recognizer job of a track fail (the Worker reports an error).
	failTranscription map[int]bool
	// only restricts the driver to these step prefixes ("" = all), so a test can stop half way.
	only []string
	// handled records every step the Worker ran, in order.
	handled []string
}

type dedupTrackTask struct {
	TrackIndex   int     `json:"track_index"`
	JobID        int64   `json:"job_id"`
	StartSeconds float64 `json:"start_seconds"`
	Segments     []struct {
		Index int     `json:"index"`
		Start float64 `json:"start"`
		End   float64 `json:"end"`
	} `json:"segments"`
}

var speechIntervals = []map[string]any{{"start": 0.5, "end": 4.0}}

func (h *voiceHarness) readyAudioWorker() {
	h.t.Helper()
	h.workerCall("POST", "/api/worker/pulse", map[string]any{"capacity": 2, "supported_steps": []string{"audio.diagnostic.*"}}, "", 200)
	order := h.order(h.fixtureWorkerID(), "silero-vad-v6.2.1", "audio-activation-install", 201)
	h.workerCall("POST", "/api/worker/voice-model-orders/"+order.OrderID+"/ack", map[string]any{"accepted": true}, "", 204)
	h.workerCall("POST", "/api/worker/voice-models/status", map[string]any{"order_id": order.OrderID, "model_id": order.ModelID, "revision": order.Revision, "state": "ready", "cause_code": "", "bytes_received": 2272526, "bytes_total": 2272526, "occupied_bytes": 2272526}, "", 204)
}

// driveAudio plays the Worker until nothing is left to claim and returns the steps it ran.
func (h *voiceHarness) driveAudio(s *audioScript) []string {
	h.t.Helper()
	caps := []string{"audio.diagnostic.*", "transcription.audio.*", "audio.dedup.*", multitrack.Capability}
	if len(s.only) > 0 {
		// A Worker that only declares these families leaves the rest queued.
		caps = []string{multitrack.Capability}
		for _, prefix := range s.only {
			caps = append(caps, prefix+"*")
		}
	}
	for i := 0; i < 40; i++ {
		h.workerCall("POST", "/api/worker/pulse", map[string]any{"capacity": 2, "supported_steps": caps}, "", 200)
		var claim struct {
			Job        *claimedJob `json:"job"`
			LeaseToken string      `json:"lease_token"`
		}
		if err := json.Unmarshal(h.workerCall("POST", "/api/worker/jobs/claim", nil, "", 200), &claim); err != nil {
			h.t.Fatal(err)
		}
		if claim.Job == nil {
			return s.handled
		}
		job := *claim.Job
		job.Lease = claim.LeaseToken
		s.handled = append(s.handled, job.Step)
		raw := h.workerCall("GET", fmt.Sprintf("/api/worker/jobs/%d/input", job.ID), nil, job.Lease, 200)
		switch {
		case strings.HasPrefix(job.Step, "audio.diagnostic."):
			var track int
			fmt.Sscanf(job.Step, "audio.diagnostic.%d", &track)
			payload := map[string]any{"status": "speech", "cause": "neural_speech", "intervals": speechIntervals}
			if d, ok := s.diag[track]; ok {
				payload = map[string]any{}
				for k, v := range d {
					payload[k] = v
				}
			}
			payload["detector_version"], payload["detector_config_version"] = audioDetectorVersion, audioDetectorConfigVersion
			h.complete(job, payload, 200)
		case strings.HasPrefix(job.Step, "transcription.audio."):
			var track int
			fmt.Sscanf(job.Step, "transcription.audio.%d", &track)
			if s.failTranscription[track] {
				h.workerCall("POST", fmt.Sprintf("/api/worker/jobs/%d/fail", job.ID), map[string]any{"error": "recognizer_unavailable"}, job.Lease, 200)
				continue
			}
			var task struct {
				AudioPlan struct {
					PlanID       int64  `json:"plan_id"`
					InputVersion string `json:"input_version"`
				} `json:"audio_plan"`
			}
			if err := json.Unmarshal(raw, &task); err != nil || task.AudioPlan.PlanID == 0 {
				h.t.Fatalf("a plan child must receive its pinned task: %s", raw)
			}
			segments := []any{}
			for _, seg := range s.words[track] {
				words := []any{}
				for i, w := range strings.Fields(seg.Text) {
					step := (seg.End - seg.Start) / float64(len(strings.Fields(seg.Text)))
					words = append(words, map[string]any{"word": w, "start": seg.Start + float64(i)*step, "end": seg.Start + float64(i+1)*step})
				}
				segments = append(segments, map[string]any{"start": seg.Start, "end": seg.End, "text": seg.Text, "words": words})
			}
			h.complete(job, map[string]any{"provider": "synthetic", "model": "synthetic-model", "version": "1", "audio_stream_index": track, "time_base": "clip", "audio_plan_id": task.AudioPlan.PlanID, "audio_plan_version": task.AudioPlan.InputVersion, "segments": segments}, 200)
		case strings.HasPrefix(job.Step, "audio.dedup."):
			var task struct {
				Dedup struct {
					Tracks []dedupTrackTask `json:"tracks"`
				} `json:"audio_dedup"`
			}
			if err := json.Unmarshal(raw, &task); err != nil || len(task.Dedup.Tracks) < 2 {
				h.t.Fatalf("the comparison must receive the kept tracks: %s", raw)
			}
			report := map[string]any{"schema_version": 1, "comparisons": []any{}}
			if s.evidence != nil {
				report = s.evidence(task.Dedup.Tracks)
			}
			h.complete(job, report, 200)
		default:
			h.t.Fatalf("unexpected step %s", job.Step)
		}
	}
	h.t.Fatal("the Worker never ran out of work")
	return nil
}

func hasStep(steps []string, prefix string) bool {
	for _, s := range steps {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return false
}

func (h *voiceHarness) startMultitrackAnalysis() {
	h.t.Helper()
	// The legacy "transcribe this clip" action: it must start an audio plan, never a single-track job.
	raw := h.adminCall("POST", fmt.Sprintf("/api/admin/clips/%d/transcription", h.clipID), map[string]any{}, 201)
	if !strings.Contains(string(raw), `"audio.plan"`) {
		h.t.Fatalf("not an audio plan: %s", raw)
	}
}

type visibleSegment struct {
	Text     string `json:"text"`
	Start    float64
	Sources  []contract.TranscriptSource `json:"sources"`
	Grouping []contract.TranscriptGrouping
}

func (h *voiceHarness) visibleTexts() []string {
	h.t.Helper()
	out := []string{}
	for _, s := range h.clipTranscript(h.adminUser()).Transcription.Segments {
		out = append(out, s.Text)
	}
	return out
}

// revisionID is the visible revision of the clip, which every correction of a plan revision must name.
func (h *voiceHarness) revisionID() string {
	h.t.Helper()
	var id string
	if err := h.pool.QueryRow(context.Background(), `SELECT revision_id FROM audio_publications WHERE clip_id=$1`, h.clipID).Scan(&id); err != nil {
		h.t.Fatal(err)
	}
	return id
}

func (h *voiceHarness) exactCount(q string) int {
	h.t.Helper()
	var search struct {
		Total int `json:"total"`
	}
	if err := json.Unmarshal(h.adminCall("GET", "/api/search?mode=exact&q="+url.QueryEscape(q), nil, 200), &search); err != nil {
		h.t.Fatal(err)
	}
	return search.Total
}

func (h *voiceHarness) stepOf(name string) analysisStepView {
	h.t.Helper()
	run := h.analysis()
	if run == nil {
		h.t.Fatal("no run")
	}
	return run.step(name)
}

// Voice and pure game, in both orders: every track is inspected, only the speech reaches the recognizer and no
// choice is asked when the diagnosis is reliable. The titles lie on purpose; only the diagnosis counts.
func testAudioActivationVoiceGame(t *testing.T, pool *pgxpool.Pool, voiceTrack int) {
	h := newVoiceHarness(t, pool)
	h.readyAudioWorker()
	game := 3 - voiceTrack
	h.startMultitrackAnalysis()
	if n := h.count(`SELECT count(*) FROM jobs WHERE clip_id=$1 AND step LIKE 'transcription.audio.%'`, h.clipID); n != 0 {
		t.Fatalf("a recognizer job exists before any diagnosis: %d", n)
	}
	script := &audioScript{
		diag:  map[int]map[string]any{voiceTrack: {"status": "speech", "cause": "neural_speech", "intervals": speechIntervals}, game: {"status": "no_speech", "cause": "no_speech_detected", "intervals": []any{}}},
		words: map[int][]scriptSegment{voiceTrack: {{1.0, 2.0, "bonjour tout le monde"}, {3.0, 3.8, "on commence"}}},
	}
	steps := h.driveAudio(script)
	if !hasStep(steps, "audio.diagnostic.1") || !hasStep(steps, "audio.diagnostic.2") {
		t.Fatalf("every audio track must be inspected: %v", steps)
	}
	if !hasStep(steps, fmt.Sprintf("transcription.audio.%d", voiceTrack)) || hasStep(steps, fmt.Sprintf("transcription.audio.%d", game)) {
		t.Fatalf("only the track with speech may reach the recognizer: %v", steps)
	}
	if hasStep(steps, "audio.dedup.") {
		t.Fatalf("one kept track has nothing to compare: %v", steps)
	}
	run := h.analysis()
	if run == nil || run.Status != "completed" || h.stepOf("transcription").Status != "done" {
		t.Fatalf("the run must complete without any interaction: %+v", run)
	}
	if got := h.visibleTexts(); len(got) != 2 || got[0] != "bonjour tout le monde" || got[1] != "on commence" {
		t.Fatalf("published transcript: %v", got)
	}
	if h.exactCount("bonjour tout le monde") != 1 {
		t.Fatal("the published transcript is not searchable")
	}
	if n := h.count(`SELECT count(*) FROM jobs WHERE clip_id=$1 AND step LIKE 'transcription.audio.%' AND input_version NOT LIKE 'mt1:%'`, h.clipID); n != 0 {
		t.Fatalf("a single-track job was created: %d", n)
	}
	// The embeddings follow the publication (the same task rows as a legacy completion).
	if h.count(`SELECT count(*) FROM audio_publications WHERE clip_id=$1`, h.clipID) != 1 {
		t.Fatal("no publication")
	}
}

// Two distinct voices on two tracks, a common capture proven by the acoustic comparison, and two voices saying the
// same words together (distinct evidence): the copy is one passage with both sources, the two voices stay two.
func testAudioActivationDedup(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	h.readyAudioWorker()
	h.startMultitrackAnalysis()
	event := func(track int, job int64, seg int, start, end float64) map[string]any {
		return map[string]any{"id": fmt.Sprintf("%d:%d:%d", track, job, seg), "track_index": track, "job_id": job, "start": start, "end": end}
	}
	script := &audioScript{
		words: map[int][]scriptSegment{
			1: {{1.0, 2.0, "bonjour tout le monde"}, {5.0, 6.0, "oui exactement"}},
			2: {{1.0, 2.0, "bonjour tout le monde"}, {3.0, 4.0, "une autre voix"}, {5.0, 6.0, "oui exactement"}},
		},
		evidence: func(tracks []dedupTrackTask) map[string]any {
			jobs := map[int]int64{}
			for _, track := range tracks {
				jobs[track.TrackIndex] = track.JobID
			}
			exact := "exact_pcm"
			version := "1"
			zero := 0.0
			return map[string]any{"schema_version": 1, "comparisons": []any{
				map[string]any{"left": event(1, jobs[1], 0, 1, 2), "right": event(2, jobs[2], 0, 1, 2), "decision": "duplicate", "evidence_kind": "exact_pcm", "strategy": exact, "strategy_version": version, "reason_codes": []string{"identical_pcm"}, "lag_seconds": zero, "gain": nil,
					"matched": []any{map[string]any{"left_start": 1.0, "left_end": 2.0, "right_start": 1.0, "right_end": 2.0}}},
				map[string]any{"left": event(1, jobs[1], 1, 5, 6), "right": event(2, jobs[2], 2, 5, 6), "decision": "distinct", "evidence_kind": "none", "strategy": nil, "strategy_version": nil, "reason_codes": []string{"uncorrelated"}, "lag_seconds": nil, "gain": nil, "matched": []any{}},
			}}
		},
	}
	steps := h.driveAudio(script)
	if !hasStep(steps, "audio.dedup.") {
		t.Fatalf("two kept tracks must be compared acoustically: %v", steps)
	}
	var clip struct {
		Transcription struct {
			Segments []struct {
				Text     string `json:"text"`
				Sources  []contract.TranscriptSource
				Grouping []contract.TranscriptGrouping
			} `json:"segments"`
		} `json:"transcription"`
	}
	if err := json.Unmarshal(h.adminCall("GET", fmt.Sprintf("/api/clips/%d", h.clipID), nil, 200), &clip); err != nil {
		t.Fatal(err)
	}
	texts := []string{}
	grouped := 0
	for _, s := range clip.Transcription.Segments {
		texts = append(texts, s.Text)
		if len(s.Grouping) > 0 {
			grouped++
			if len(s.Sources) != 2 || s.Sources[0].TrackIndex != 1 || s.Sources[1].TrackIndex != 2 {
				t.Fatalf("the common capture keeps both sources, the lowest track represents it: %+v", s.Sources)
			}
		}
	}
	// 1 shared passage + the other voice + the same words said twice by two voices.
	if len(texts) != 4 || grouped != 1 {
		t.Fatalf("segments %v (grouped %d)", texts, grouped)
	}
	if h.exactCount("bonjour tout le monde") != 1 {
		t.Fatal("a proven copy must be one occurrence")
	}
	if h.exactCount("oui exactement") != 2 {
		t.Fatal("two voices saying the same words together must both stay")
	}
	if h.exactCount("une autre voix") != 1 {
		t.Fatal("the second voice was lost")
	}
}

// A track the detector cannot decide blocks the decision, not the others' diagnoses; the administrator decides with
// the selection route (a forced inclusion, a forced no-speech track, invalid choices refused) and the controlled
// plan is stored with its decisions.
func testAudioActivationAmbiguity(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	h.readyAudioWorker()
	h.startMultitrackAnalysis()
	script := &audioScript{
		diag:  map[int]map[string]any{1: {"status": "ambiguous", "cause": "insufficient_evidence", "intervals": []any{}}, 2: {"status": "speech", "cause": "neural_speech", "intervals": speechIntervals}},
		words: map[int][]scriptSegment{1: {{1.0, 2.0, "chat vocal du jeu"}}, 2: {{3.0, 4.0, "ma voix"}}},
	}
	script.only = []string{"audio.diagnostic."}
	h.driveAudio(script)
	script.only = nil
	if st := h.stepOf("transcription"); st.Status != "blocked" || st.Reason != "diagnostic_ambiguous" {
		t.Fatalf("an ambiguous track must block the decision with its cause: %+v", st)
	}
	if n := h.count(`SELECT count(*) FROM jobs WHERE clip_id=$1 AND step LIKE 'transcription.audio.%'`, h.clipID); n != 0 {
		t.Fatalf("nothing is transcribed before the decision: %d", n)
	}
	h.analysisAction("/cancel", 200)

	path := fmt.Sprintf("/api/admin/clips/%d/audio-selection", h.clipID)
	var current contract.AudioSelectionContext
	load := func() {
		t.Helper()
		current = contract.AudioSelectionContext{}
		if err := json.Unmarshal(h.adminCall("GET", path, nil, 200), &current); err != nil {
			t.Fatal(err)
		}
	}
	load()
	if !current.Enabled || len(current.Inventory) != 2 {
		t.Fatalf("the real engine must enable the selection: %+v", current)
	}
	seen := map[int]string{}
	for _, d := range current.Diagnostics {
		if d.Freshness == "current" {
			seen[d.TrackIndex] = string(d.Status)
		}
	}
	if seen[1] != "ambiguous" || seen[2] != "speech" {
		t.Fatalf("the previous diagnoses are offered with their freshness: %+v", current.Diagnostics)
	}
	auto := contract.AudioSelectionRequest{Version: current.Version, Mode: "auto", TrackIndices: []int{}, Decisions: []contract.AudioSelectionDecision{}, Analysis: contract.AudioAnalysisChoice{Scope: "transcription", Mode: "missing", Steps: []string{}}}
	h.adminCall("POST", path+"/preview", auto, 400) // the ambiguous track needs a decision
	bad := auto
	bad.Mode, bad.TrackIndices = "manual", []int{}
	h.adminCall("POST", path+"/preview", bad, 400) // empty selection
	bad.TrackIndices = []int{0}
	h.adminCall("POST", path+"/preview", bad, 400) // a video stream
	bad.TrackIndices = []int{42}
	h.adminCall("POST", path+"/preview", bad, 400) // unknown index
	stale := auto
	stale.Version = "stale"
	h.adminCall("POST", path+"/preview", stale, 409)

	auto.Decisions = []contract.AudioSelectionDecision{{TrackIndex: 1, Action: "include", Forced: true, Reason: "voice chat heard in the game track"}}
	var preview contract.AudioSelectionPreview
	if err := json.Unmarshal(h.adminCall("POST", path+"/preview", auto, 200), &preview); err != nil {
		t.Fatal(err)
	}
	if len(preview.Input.TrackIndices) != 2 {
		t.Fatalf("preview: %+v", preview)
	}
	h.adminCall("POST", path, contract.AudioSelectionStart{Selection: auto, PreviewToken: preview.PreviewToken}, 200)
	steps := h.driveAudio(script)
	if !hasStep(steps, "transcription.audio.1") || !hasStep(steps, "transcription.audio.2") || !hasStep(steps, "audio.dedup.") {
		t.Fatalf("the forced ambiguous track must be transcribed with the other: %v", steps)
	}
	if got := h.visibleTexts(); len(got) != 2 || got[0] != "chat vocal du jeu" || got[1] != "ma voix" {
		t.Fatalf("published transcript: %v", got)
	}
	var stored string
	if err := pool.QueryRow(context.Background(), `SELECT input::text FROM audio_plans WHERE clip_id=$1 ORDER BY id DESC LIMIT 1`, h.clipID).Scan(&stored); err == nil {
		if !strings.Contains(stored, `"forced": true`) {
			t.Fatalf("the forced decision must stay in the stored plan: %s", stored)
		}
	} else {
		t.Fatal(err)
	}

}

// One recognizer fails after its sibling succeeded: the published transcript is kept, only that child restarts, and
// a restart of the server in the middle duplicates nothing.
func testAudioActivationPartialFailure(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	h.readyAudioWorker()
	h.startMultitrackAnalysis()
	ctx := context.Background()
	script := &audioScript{
		words:             map[int][]scriptSegment{1: {{1.0, 2.0, "premiere piste"}}, 2: {{3.0, 4.0, "seconde piste"}}},
		failTranscription: map[int]bool{2: true},
	}
	// Run the diagnoses, then lower the retry budget of the recognizers so a failure is final.
	script.only = []string{"audio.diagnostic."}
	h.driveAudio(script)
	script.only = nil
	if _, err := pool.Exec(ctx, `UPDATE jobs SET max_attempts=1 WHERE clip_id=$1 AND step LIKE 'transcription.audio.%'`, h.clipID); err != nil {
		t.Fatal(err)
	}
	// A restart: a second orchestrator on the same database follows the same children.
	restarted := analysis.NewAudio(pool, analysis.WorkerAudioDiagnostics{}, audioPublisher{h.api})
	var planID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM audio_plans WHERE clip_id=$1`, h.clipID).Scan(&planID); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if err := restarted.Reconcile(ctx, planID); err != nil {
			t.Fatal(err)
		}
	}
	h.driveAudio(script)
	if st := h.stepOf("transcription"); st.Status != "failed" || st.Reason != "audio_child_failed" {
		t.Fatalf("a failed child must be visible: %+v", st)
	}
	if len(h.visibleTexts()) != 0 {
		t.Fatal("a partial plan was published")
	}
	var okJob, badJob int64
	if err := pool.QueryRow(ctx, `SELECT (SELECT id FROM jobs WHERE clip_id=$1 AND step='transcription.audio.1'),(SELECT id FROM jobs WHERE clip_id=$1 AND step='transcription.audio.2')`, h.clipID).Scan(&okJob, &badJob); err != nil {
		t.Fatal(err)
	}
	h.analysisAction("/steps/transcription/retry", 200)
	delete(script.failTranscription, 2)
	steps := h.driveAudio(script)
	if !hasStep(steps, "transcription.audio.2") {
		t.Fatalf("steps %v", steps)
	}
	if n := h.count(`SELECT count(*) FROM jobs WHERE clip_id=$1 AND step LIKE 'transcription.audio.%'`, h.clipID); n != 2 {
		t.Fatalf("no duplicate recognizer jobs: %d", n)
	}
	if n := h.count(`SELECT attempts FROM jobs WHERE id=$1`, okJob); n != 1 {
		t.Fatalf("the successful sibling was recomputed: %d attempts", n)
	}
	var retriedID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM jobs WHERE clip_id=$1 AND step='transcription.audio.2'`, h.clipID).Scan(&retriedID); err != nil || retriedID != badJob {
		t.Fatalf("only the failed child is retried in place: %d vs %d (%v)", retriedID, badJob, err)
	}
	if h.stepOf("transcription").Status != "done" || len(h.visibleTexts()) != 2 {
		t.Fatalf("the retry completes the plan: %v", h.visibleTexts())
	}
}

// Human work: the confirmation is the one of the preview; an edit after the preview, or during the run, never
// goes unnoticed; the old revision and its corrections stay visible until the administrator confirms again.
func testAudioActivationHumanWork(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	h.readyAudioWorker()
	h.startMultitrackAnalysis()
	script := &audioScript{words: map[int][]scriptSegment{1: {{1.0, 2.0, "premier texte"}}, 2: {{3.0, 4.0, "second texte"}}}}
	h.driveAudio(script)
	if got := h.visibleTexts(); len(got) != 2 {
		t.Fatalf("first publication: %v", got)
	}
	h.adminCall("PUT", segmentTextPath(h.clipID, 0), map[string]any{"text": "texte corrigé à la main", "expected_version": 0, "expected_revision_id": h.revisionID()}, 200)

	path := fmt.Sprintf("/api/admin/clips/%d/audio-selection", h.clipID)
	selection := func() contract.AudioSelectionRequest {
		var current contract.AudioSelectionContext
		if err := json.Unmarshal(h.adminCall("GET", path, nil, 200), &current); err != nil {
			t.Fatal(err)
		}
		return contract.AudioSelectionRequest{Version: current.Version, Mode: "auto", TrackIndices: []int{}, Decisions: []contract.AudioSelectionDecision{}, Analysis: contract.AudioAnalysisChoice{Scope: "transcription", Mode: "full", Steps: []string{}}}
	}
	choice := selection()
	var preview contract.AudioSelectionPreview
	if err := json.Unmarshal(h.adminCall("POST", path+"/preview", choice, 200), &preview); err != nil {
		t.Fatal(err)
	}
	if !preview.Effects.Losses.ConfirmationRequired || preview.Effects.Losses.CorrectionCount != 1 || preview.Effects.StakeVersion == "" {
		t.Fatalf("the losses must be shown: %+v", preview.Effects)
	}
	h.adminCall("POST", path, contract.AudioSelectionStart{Selection: choice, PreviewToken: preview.PreviewToken}, 409) // not confirmed
	// An edit after the preview invalidates it: the confirmation covered another history.
	h.adminCall("PUT", segmentTextPath(h.clipID, 1), map[string]any{"text": "seconde correction", "expected_version": 0, "expected_revision_id": h.revisionID()}, 200)
	h.adminCall("POST", path, contract.AudioSelectionStart{Selection: choice, PreviewToken: preview.PreviewToken, ConfirmLosses: true}, 409)

	choice = selection()
	if err := json.Unmarshal(h.adminCall("POST", path+"/preview", choice, 200), &preview); err != nil {
		t.Fatal(err)
	}
	if preview.Effects.Losses.CorrectionCount != 2 {
		t.Fatalf("both corrections are at stake: %+v", preview.Effects.Losses)
	}
	h.adminCall("POST", path, contract.AudioSelectionStart{Selection: choice, PreviewToken: preview.PreviewToken, ConfirmLosses: true}, 200)
	// A third edit lands while the plan runs: publication must stop, nothing is lost.
	h.adminCall("PUT", segmentTextPath(h.clipID, 0), map[string]any{"text": "texte corrigé encore", "expected_version": 1, "expected_revision_id": h.revisionID()}, 200)
	script = &audioScript{words: map[int][]scriptSegment{1: {{1.0, 2.0, "nouveau premier"}}, 2: {{3.0, 4.0, "nouveau second"}}}}
	h.driveAudio(script)
	if st := h.stepOf("transcription"); st.Status != "blocked" || st.Reason != "audio_publication_changed" {
		t.Fatalf("an edit during the run must hold the publication: %+v", st)
	}
	if got := h.visibleTexts(); len(got) != 2 || got[0] != "texte corrigé encore" || got[1] != "seconde correction" {
		t.Fatalf("the old revision and the corrections stay visible: %v", got)
	}
	var planID int64
	if err := pool.QueryRow(context.Background(), `SELECT max(id) FROM audio_plans WHERE clip_id=$1`, h.clipID).Scan(&planID); err != nil {
		t.Fatal(err)
	}
	pub := fmt.Sprintf("/api/admin/clips/%d/audio-plans/%d/publication", h.clipID, planID)
	var view contract.AudioPublicationPreview
	if err := json.Unmarshal(h.adminCall("GET", pub, nil, 200), &view); err != nil {
		t.Fatal(err)
	}
	if view.ConfirmationCurrent || view.Confirmed || !view.Losses.ConfirmationRequired || view.Losses.CorrectionCount != 2 {
		t.Fatalf("the earlier confirmation is no longer current: %+v", view)
	}
	h.adminCall("POST", pub, contract.AudioPublicationConfirm{Guard: "stale", ConfirmLosses: true}, 409)
	h.adminCall("POST", pub, contract.AudioPublicationConfirm{Guard: view.Guard, ConfirmLosses: false}, 409)
	h.adminCall("POST", pub, contract.AudioPublicationConfirm{Guard: view.Guard, ConfirmLosses: true}, 200)
	if got := h.visibleTexts(); len(got) != 2 || got[0] != "nouveau premier" || got[1] != "nouveau second" {
		t.Fatalf("the confirmed replacement is published: %v", got)
	}
	if h.exactCount("corrigé") != 0 || h.exactCount("nouveau premier") != 1 {
		t.Fatal("search follows the published revision")
	}
	if run := h.analysis(); run == nil || run.Status != "completed" {
		t.Fatalf("the run completes after the confirmation: %+v", run)
	}
}

// Every track without speech: an explicit empty revision, no recognizer, no voice. An old Worker cannot take the
// work and the wait is named.
func testAudioActivationNoSpeechAndOldWorker(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	h.startMultitrackAnalysis()
	// A Worker that only knows the single-track format takes nothing.
	if job, ok := h.claim(); ok {
		t.Fatalf("an old Worker claimed %+v", job)
	}
	if st := h.stepOf("preparation"); st.Status != "waiting" || st.Reason != "awaiting_compatible_worker" {
		t.Fatalf("the wait for a compatible Worker must be named: %+v", st)
	}
	h.readyAudioWorker()
	script := &audioScript{diag: map[int]map[string]any{
		1: {"status": "no_speech", "cause": "no_speech_detected", "intervals": []any{}},
		2: {"status": "no_speech", "cause": "no_speech_detected", "intervals": []any{}},
	}}
	steps := h.driveAudio(script)
	if hasStep(steps, "transcription.audio.") {
		t.Fatalf("no recognizer for a clip without speech: %v", steps)
	}
	if run := h.analysis(); run == nil || run.Status != "completed" {
		t.Fatalf("a clip without speech completes: %+v", run)
	}
	if n := h.count(`SELECT count(*) FROM transcript_revisions r JOIN audio_publications p ON p.revision_id=r.id WHERE p.clip_id=$1 AND (r.manifest->>'empty')::boolean`, h.clipID); n != 1 {
		t.Fatal("the empty revision must be explicit and published")
	}
	if len(h.visibleTexts()) != 0 {
		t.Fatal("an invented transcript")
	}
	if n := h.count(`SELECT count(*) FROM jobs WHERE clip_id=$1 AND step LIKE 'voice.%'`, h.clipID); n != 0 {
		t.Fatal("voices were started for a clip without speech")
	}

	// A track diagnosed without speech is only transcribed when the administrator forces it; the forcing is
	// stored in the plan and the text is published.
	path := fmt.Sprintf("/api/admin/clips/%d/audio-selection", h.clipID)
	var current contract.AudioSelectionContext
	if err := json.Unmarshal(h.adminCall("GET", path, nil, 200), &current); err != nil {
		t.Fatal(err)
	}
	manual := contract.AudioSelectionRequest{Version: current.Version, Mode: "manual", TrackIndices: []int{1}, Decisions: []contract.AudioSelectionDecision{}, Analysis: contract.AudioAnalysisChoice{Scope: "transcription", Mode: "full", Steps: []string{}}}
	h.adminCall("POST", path+"/preview", manual, 400) // the diagnosis is known: forcing is explicit
	manual.Decisions = []contract.AudioSelectionDecision{{TrackIndex: 1, Action: "include", Forced: true, Reason: "faint speech heard in the mix"}}
	var preview contract.AudioSelectionPreview
	if err := json.Unmarshal(h.adminCall("POST", path+"/preview", manual, 200), &preview); err != nil {
		t.Fatal(err)
	}
	h.adminCall("POST", path, contract.AudioSelectionStart{Selection: manual, PreviewToken: preview.PreviewToken, ConfirmLosses: preview.Effects.Losses.ConfirmationRequired}, 200)
	script = &audioScript{
		diag:  script.diag,
		words: map[int][]scriptSegment{1: {{6.0, 7.0, "murmure forcé"}}},
	}
	steps = h.driveAudio(script)
	if !hasStep(steps, "transcription.audio.1") || hasStep(steps, "transcription.audio.2") {
		t.Fatalf("exactly the forced track: %v", steps)
	}
	if got := h.visibleTexts(); len(got) != 1 || got[0] != "murmure forcé" {
		t.Fatalf("a forced track is stored and published: %v", got)
	}
	var stored string
	if err := h.pool.QueryRow(context.Background(), `SELECT input::text FROM audio_plans WHERE clip_id=$1 ORDER BY id DESC LIMIT 1`, h.clipID).Scan(&stored); err != nil || !strings.Contains(stored, `"forced": true`) {
		t.Fatalf("the forced decision must stay in the stored plan: %s %v", stored, err)
	}
}

// Every legacy way to ask for a transcription goes through the plan: batch, source rule, clip override, analysis
// (missing and full), raw job creation. A clip with one audio track keeps its single-track job.
func testAudioActivationLegacyEntryPoints(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	h.readyAudioWorker()
	ctx := context.Background()
	jobsOf := func(like string) int {
		return h.count(`SELECT count(*) FROM jobs WHERE clip_id=$1 AND step LIKE $2`, h.clipID, like)
	}
	plans := func() int { return h.count(`SELECT count(*) FROM audio_plans WHERE clip_id=$1`, h.clipID) }
	var batch struct {
		Eligible          int `json:"eligible"`
		SelectionRequired int `json:"selection_required"`
	}
	if err := json.Unmarshal(h.adminCall("GET", "/api/admin/transcriptions/batch-preview", nil, 200), &batch); err != nil {
		t.Fatal(err)
	}
	if batch.Eligible != 1 || batch.SelectionRequired != 0 {
		t.Fatalf("a multitrack clip is planned, not blocked on a choice: %+v", batch)
	}
	h.adminCall("POST", "/api/admin/transcriptions/batch-schedule", map[string]bool{"confirm": true}, 200)
	if plans() != 1 || jobsOf("transcription.audio.%") != 0 || jobsOf("audio.diagnostic.%") != 2 {
		t.Fatalf("batch must create the plan: plans=%d", plans())
	}
	h.adminCall("POST", "/api/admin/transcriptions/batch-schedule", map[string]bool{"confirm": true}, 200)
	if plans() != 1 {
		t.Fatal("a second batch duplicated the active plan")
	}
	h.adminCall("POST", "/api/admin/jobs", map[string]any{"clip_id": h.clipID, "step": "transcription.audio.2", "input_version": "t4-a2-r0-vx-py"}, 409)
	h.analysisAction("/cancel", 200)
	h.adminCall("POST", "/api/admin/jobs", map[string]any{"clip_id": h.clipID, "step": "audio.diagnostic.2", "input_version": "x"}, 400)
	h.adminCall("POST", "/api/admin/jobs", map[string]any{"clip_id": h.clipID, "step": "voice.diarization.2", "input_version": "mt1:x"}, 400)

	// A clip override and a source rule name one track: a manual plan of that track, never a legacy job.
	h.adminCall("PUT", fmt.Sprintf("/api/admin/clips/%d/audio-track", h.clipID), map[string]any{"audio_stream_index": 2}, 200)
	var mode string
	var indices []int
	if err := pool.QueryRow(ctx, `SELECT input->>'mode',ARRAY(SELECT jsonb_array_elements_text(input->'track_indices')::int) FROM audio_plans WHERE clip_id=$1 ORDER BY id DESC LIMIT 1`, h.clipID).Scan(&mode, &indices); err != nil {
		t.Fatal(err)
	}
	if mode != "manual" || len(indices) != 1 || indices[0] != 2 || plans() != 2 {
		t.Fatalf("override: %s %v", mode, indices)
	}
	h.analysisAction("/cancel", 200)
	var sourceID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM clip_sources ORDER BY id LIMIT 1`).Scan(&sourceID); err != nil {
		t.Fatal(err)
	}
	h.adminCall("DELETE", fmt.Sprintf("/api/admin/clips/%d/audio-track", h.clipID), nil, 200)
	h.analysisAction("/cancel", 200)
	h.adminCall("PUT", fmt.Sprintf("/api/admin/sources/%d/audio-track-rule", sourceID), map[string]any{"title": "Micro"}, 200)
	if jobsOf("transcription.audio.%") != 0 {
		t.Fatal("a source rule created a single-track job")
	}
	h.analysisAction("/cancel", 200)

	// The analysis API: missing and full runs on a clip that needs a transcription get a plan.
	h.startAnalysis(map[string]any{"scope": "transcription", "mode": "missing"}, 201)
	if jobsOf("transcription.audio.%") != 0 || h.stepOf("transcription").Reason == "audio_track_selection_required" {
		t.Fatal("analysis bypassed the plan")
	}
	before := plans()
	h.analysisAction("/cancel", 200)
	h.startAnalysis(map[string]any{"scope": "full", "mode": "full"}, 201)
	if plans() != before+1 {
		t.Fatalf("a full run needs its own plan: %d vs %d", plans(), before)
	}
	h.analysisAction("/cancel", 200)

	// One audio track: the historical single-track job, unchanged.
	h.useSingleAudioTrack()
	h.adminCall("DELETE", fmt.Sprintf("/api/admin/clips/%d/audio-track", h.clipID), nil, 200)
	plansBefore := plans()
	h.adminCall("POST", fmt.Sprintf("/api/admin/clips/%d/transcription", h.clipID), map[string]any{}, 200)
	if jobsOf("transcription.audio.2") != 1 || plans() != plansBefore {
		t.Fatalf("a single-track clip keeps its legacy job: jobs=%d plans=%d", jobsOf("transcription.audio.2"), plans())
	}
}

// The full analysis publishes the plan, then starts the voice analysis of the plan's tracks, bound to the plan
// (ADR 0052): the diarization examines every track of the plan, including one the speech detector excluded, so a
// voice the detector missed stays visible with its diagnostic instead of being lost (#273).
func testAudioActivationVoices(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	h.readyAudioWorker()
	h.putPipeline(fakePipeline(2))
	run := h.startAnalysis(map[string]any{"scope": "full"}, 201)
	if run.Status != "active" {
		t.Fatalf("run: %+v", run)
	}
	script := &audioScript{
		diag:  map[int]map[string]any{1: {"status": "no_speech", "cause": "no_speech_detected", "intervals": []any{}}, 2: {"status": "speech", "cause": "neural_speech", "intervals": speechIntervals}},
		words: map[int][]scriptSegment{2: {{1.0, 2.0, "une voix"}}},
	}
	h.driveAudio(script)
	if n := h.count(`SELECT count(*) FROM audio_publications WHERE clip_id=$1`, h.clipID); n != 1 {
		t.Fatal("not published")
	}
	if n := h.count(`SELECT count(*) FROM jobs j JOIN audio_plan_children c ON c.job_id=j.id WHERE j.clip_id=$1 AND c.stage='diarization' AND j.step LIKE 'voice.diarization.%'`, h.clipID); n != 2 {
		t.Fatalf("the voice analysis must start after the publication, bound to the plan: %d", n)
	}
	if hasStep(script.handled, "transcription.audio.1") {
		t.Fatal("the excluded track reached the recognizer")
	}
	var planID int64
	if err := pool.QueryRow(context.Background(), `SELECT id FROM audio_plans WHERE clip_id=$1`, h.clipID).Scan(&planID); err != nil {
		t.Fatal(err)
	}
	var voice struct {
		Tracks []struct {
			AudioStreamIndex int    `json:"audio_stream_index"`
			Status           string `json:"status"`
		} `json:"tracks"`
	}
	if err := json.Unmarshal(h.adminCall("GET", fmt.Sprintf("/api/admin/clips/%d/audio-plans/%d/voice", h.clipID, planID), nil, 200), &voice); err != nil {
		t.Fatal(err)
	}
	if len(voice.Tracks) != 2 || voice.Tracks[0].Status != "queued" || voice.Tracks[1].Status != "queued" {
		t.Fatalf("per-track voice state: %+v", voice)
	}
	if run := h.analysis(); run == nil || run.Status != "completed" {
		t.Fatalf("run: %+v", run)
	}
}
