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
	"snitch/internal/audioplans"
	"snitch/internal/httpapi/contract"
)

// The real orchestrator is not needed: a published plan revision is staged from done transcription
// children, like the publication suite does, then the voice analysis of the plan is driven through
// its public routes and the Worker routes.

type votedPlan struct {
	ID       int64
	Revision string
}

// publishVoicePlan publishes a plan keeping the tracks; words maps a track to the text of its single
// segment (clip time 1..2), an empty text meaning a kept track whose transcription holds no word.
func (h *voiceHarness) publishVoicePlan(tracks []int, words map[int]string) votedPlan {
	h.t.Helper()
	ctx := context.Background()
	var run int64
	if err := h.pool.QueryRow(ctx, `INSERT INTO clip_analysis_runs(clip_id,scope,status) VALUES($1,'full','active') RETURNING id`, h.clipID).Scan(&run); err != nil {
		h.t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(audioplans.ContractFixture(), &raw); err != nil {
		h.t.Fatal(err)
	}
	raw["inventory"] = append(raw["inventory"].([]any), map[string]any{"index": 1, "title": "Micro", "codec": "aac", "language": "fra", "channels": 2, "channel_layout": "stereo", "duration_seconds": 4.25, "start_seconds": 0, "evidence": []string{}})
	raw["track_indices"] = tracks
	decisions := []map[string]any{}
	for _, track := range tracks {
		decisions = append(decisions, map[string]any{"track_index": track, "action": "include", "forced": true, "actor_id": 1, "reason": "fixture", "decided_at": time.Now().UTC().Format(time.RFC3339Nano)})
	}
	raw["decisions"] = decisions
	encoded, _ := json.Marshal(raw)
	var input audioplans.Input
	if err := json.Unmarshal(encoded, &input); err != nil {
		h.t.Fatal(err)
	}
	store := audioplans.New(h.pool)
	plan, err := store.CreatePlan(ctx, run, input)
	if err != nil {
		h.t.Fatal(err)
	}
	revision := audioplans.Revision{ClipId: h.clipID, PlanId: &plan.Id, TimeBase: "clip"}
	for _, track := range tracks {

		var job int64
		version := fmt.Sprintf("voice-plan-%d-%d", plan.Id, track)
		if err = h.pool.QueryRow(ctx, `INSERT INTO jobs(clip_id,step,input_version) VALUES($1,$2,$3) RETURNING id`, h.clipID, fmt.Sprintf("transcription.audio.%d", track), version).Scan(&job); err != nil {
			h.t.Fatal(err)
		}
		if err = store.BindChild(ctx, audioplans.Child{PlanId: plan.Id, TrackIndex: track, Stage: "transcription", JobId: job, InputVersion: version}); err != nil {
			h.t.Fatal(err)
		}
		segments := []map[string]any{{"start": 1, "end": 2, "text": words[track]}}
		if words[track] != "" {
			revision.Segments = append(revision.Segments, contract.TranscriptManifestSegment{Start: 1, End: 2, Sources: []contract.TranscriptSource{{TrackIndex: track, JobId: job, SegmentIndex: 0}}})
		}
		if words[track] == "" {
			segments = []map[string]any{}
		}
		encodedSegments, _ := json.Marshal(segments)
		if _, err = h.pool.Exec(ctx, `UPDATE jobs SET status='done',result=jsonb_build_object('audio_stream_index',$2::int,'segments',$3::jsonb) WHERE id=$1`, job, track, encodedSegments); err != nil {
			h.t.Fatal(err)
		}
	}
	staged, err := store.StageRevision(ctx, revision)
	if err != nil {
		h.t.Fatal(err)
	}
	preview, err := store.PreviewPublication(ctx, plan.Id)
	if err != nil {
		h.t.Fatal(err)
	}
	if err = store.PublishRevision(ctx, staged.Id, preview, true); err != nil {
		h.t.Fatal(err)
	}
	// These tests drive voice work explicitly, outside the analysis scheduler.
	if _, err = h.pool.Exec(ctx, `UPDATE clip_analysis_runs SET status='completed',finished_at=now() WHERE id=$1`, run); err != nil {
		h.t.Fatal(err)
	}
	return votedPlan{ID: plan.Id, Revision: staged.Id}
}

type planVoiceTrackView struct {
	Index      int    `json:"audio_stream_index"`
	Status     string `json:"status"`
	Cause      string `json:"cause"`
	Reused     bool   `json:"reused"`
	Microphone bool   `json:"microphone"`
	Speakers   int    `json:"speaker_count"`
	Diar       *struct {
		JobID  int64  `json:"job_id"`
		Status string `json:"status"`
		Error  string `json:"last_error"`
	} `json:"diarization"`
	Fingerprint *struct {
		JobID  int64  `json:"job_id"`
		Status string `json:"status"`
	} `json:"fingerprint"`
	Stake voicePlanStake `json:"stake"`
}

type planVoiceView struct {
	PlanID    int64                `json:"plan_id"`
	Published bool                 `json:"published"`
	Status    string               `json:"status"`
	Cause     string               `json:"cause"`
	Tracks    []planVoiceTrackView `json:"tracks"`
	Stake     voicePlanStake       `json:"stake"`
}

func (v planVoiceView) track(index int) planVoiceTrackView {
	for _, t := range v.Tracks {
		if t.Index == index {
			return t
		}
	}
	return planVoiceTrackView{Index: -1}
}

func (h *voiceHarness) planVoicePath(plan votedPlan) string {
	return fmt.Sprintf("/api/admin/clips/%d/audio-plans/%d/voice", h.clipID, plan.ID)
}

func (h *voiceHarness) planVoice(plan votedPlan) planVoiceView {
	h.t.Helper()
	var v planVoiceView
	if err := json.Unmarshal(h.adminCall(http.MethodGet, h.planVoicePath(plan), nil, http.StatusOK), &v); err != nil {
		h.t.Fatal(err)
	}
	assertAudioContract(h.t, "AudioPlanVoice", h.adminCall(http.MethodGet, h.planVoicePath(plan), nil, http.StatusOK))
	return v
}

func (h *voiceHarness) startPlanVoice(plan votedPlan, confirm bool, want int) []byte {
	h.t.Helper()
	return h.adminCall(http.MethodPost, h.planVoicePath(plan), map[string]any{"confirm_replace_validations": confirm}, want)
}

// claimVoice leases the next n voice tasks the Worker is offered, by step.
func (h *voiceHarness) claimVoice(n int) map[string]claimedJob {
	h.t.Helper()
	h.adminCall(http.MethodPut, "/api/admin/job-controls", map[string]any{"paused": false, "max_concurrency": 3}, http.StatusOK)
	out := map[string]claimedJob{}
	for i := 0; i < n; i++ {
		job, ok := h.claim()
		if !ok {
			h.t.Fatalf("expected %d voice tasks, got %v", n, out)
		}
		out[job.Step] = job
	}
	return out
}

func trackDiarization(index int, speakers map[string][][2]float64, order ...string) map[string]any {
	items := []map[string]any{}
	for _, label := range order {
		turns := []map[string]any{}
		for _, t := range speakers[label] {
			turns = append(turns, map[string]any{"start": t[0], "end": t[1]})
		}
		items = append(items, map[string]any{"label": label, "turns": turns})
	}
	return map[string]any{"provider": "fake", "model": "fixture-diarization", "model_version": "fixture-v1", "audio_stream_index": index, "speakers": items}
}

func trackFingerprints(index int, diarizationJob int64, spans map[string][2]float64, order ...string) map[string]any {
	items := []map[string]any{}
	for _, label := range order {
		items = append(items, map[string]any{"speaker_label": label, "scope": "speaker", "start": spans[label][0], "end": spans[label][1], "quality": 0.9, "embedding": []float64{0.5, 0.5, 0.5, 0.5}})
	}
	return map[string]any{"provider": "fake", "model": "fixture-diarization", "model_version": "fixture-v1", "embedding_model": "fixture-embedding", "embedding_model_version": "emb-v1", "audio_stream_index": index, "diarization_job_id": diarizationJob, "dimensions": 4, "fingerprints": items}
}

type planTranscript struct {
	Transcription struct {
		Segments []struct {
			Text      string `json:"text"`
			SpeakerID *int64 `json:"local_speaker_id"`
		} `json:"segments"`
		LocalSpeakers []struct {
			ID    int64 `json:"id"`
			Track int   `json:"audio_stream_index"`
		} `json:"local_speakers"`
		Coverage []struct {
			Track    int `json:"audio_stream_index"`
			Coverage []struct {
				SpeakerID int64  `json:"local_speaker_id"`
				Status    string `json:"coverage_status"`
			} `json:"coverage"`
		} `json:"speaker_coverage_by_track"`
	} `json:"transcription"`
}

func (h *voiceHarness) planTranscript() planTranscript {
	h.t.Helper()
	var out planTranscript
	if err := json.Unmarshal(h.adminCall(http.MethodGet, fmt.Sprintf("/api/clips/%d", h.clipID), nil, http.StatusOK), &out); err != nil {
		h.t.Fatal(err)
	}
	return out
}

func (t planTranscript) coverage(track int, speaker int64) string {
	for _, c := range t.Transcription.Coverage {
		if c.Track != track {
			continue
		}
		for _, entry := range c.Coverage {
			if entry.SpeakerID == speaker {
				return entry.Status
			}
		}
	}
	return ""
}

// Two kept tracks with their own voices: each voice stays on its stream and its provenance, times
// are converted once with the stream offset (track 7 starts 1.25 s into the clip), and a voice that
// speaks where nothing was transcribed is still listed with its coverage.
func testAudioPlanVoiceTwoTracks(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	h.putPipeline(fakePipeline(2))
	plan := h.publishVoicePlan([]int{2, 7}, map[int]string{2: "voix deux", 7: "voix sept"})

	before := h.planVoice(plan)
	if !before.Published || before.Status != "not_started" || len(before.Tracks) != 2 || before.Tracks[0].Index != 2 || before.Tracks[1].Index != 7 {
		t.Fatalf("a published plan lists its kept tracks, nothing started: %+v", before)
	}
	// The published transcript is untouched by the voice start.
	texts := func() int { return len(h.planTranscript().Transcription.Segments) }
	if texts() != 2 {
		t.Fatal("published transcript expected")
	}
	h.startPlanVoice(plan, false, http.StatusOK)
	started := h.planVoice(plan)
	if started.Status != "queued" || started.track(2).Diar == nil || started.track(7).Diar == nil || started.track(2).Diar.Status != "pending" {
		t.Fatalf("both tracks queue a diarization: %+v", started)
	}
	if h.count(`SELECT count(*) FROM audio_plan_children WHERE plan_id=$1 AND stage='diarization'`, plan.ID) != 2 {
		t.Fatal("each diarization is bound to the plan before any Worker can claim it")
	}
	// Starting again only returns the same work.
	h.startPlanVoice(plan, false, http.StatusOK)
	if h.count(`SELECT count(*) FROM jobs WHERE clip_id=$1 AND step LIKE 'voice.diarization.%'`, h.clipID) != 2 {
		t.Fatal("a second start must not duplicate jobs")
	}

	claimed := h.claimVoice(2)
	two, seven := claimed["voice.diarization.2"], claimed["voice.diarization.7"]
	if two.ID == 0 || seven.ID == 0 {
		t.Fatalf("one diarization per kept track: %v", claimed)
	}
	h.complete(two, trackDiarization(2, map[string][][2]float64{"SPEAKER_00": {{0.9, 2.1}}, "SPEAKER_01": {{3.0, 3.5}}}, "SPEAKER_00", "SPEAKER_01"), http.StatusOK)
	// The Worker measures track 7 from its own first sample.
	h.complete(seven, trackDiarization(7, map[string][][2]float64{"SPEAKER_00": {{0, 1}}}, "SPEAKER_00"), http.StatusOK)
	if h.count(`SELECT count(*) FROM speaker_turns st JOIN local_speakers ls ON ls.id=st.local_speaker_id WHERE ls.clip_id=$1 AND ls.audio_stream_index=7 AND st.start_seconds=1.25 AND st.end_seconds=2.25`, h.clipID) != 1 {
		t.Fatal("track 7 turns must be stored in clip time, offset added once")
	}
	if h.count(`SELECT count(*) FROM speaker_turns st JOIN local_speakers ls ON ls.id=st.local_speaker_id WHERE ls.clip_id=$1 AND ls.audio_stream_index=2 AND st.start_seconds=0.9`, h.clipID) != 1 {
		t.Fatal("track 2 has no offset")
	}

	fingerprints := h.claimVoice(2)
	fpTwo, fpSeven := fingerprints["voice.fingerprint.2"], fingerprints["voice.fingerprint.7"]
	if fpTwo.ID == 0 || fpSeven.ID == 0 {
		t.Fatalf("one fingerprint step per kept track: %v", fingerprints)
	}
	if h.count(`SELECT count(*) FROM audio_plan_children WHERE plan_id=$1 AND stage='fingerprint' AND job_id IN ($2,$3)`, plan.ID, fpTwo.ID, fpSeven.ID) != 2 {
		t.Fatal("fingerprint steps belong to the plan")
	}
	var input struct {
		Voice struct {
			Diarization struct {
				Speakers []struct {
					Turns []struct {
						Start float64 `json:"start"`
						End   float64 `json:"end"`
					} `json:"turns"`
				} `json:"speakers"`
			} `json:"diarization"`
		} `json:"voice"`
	}
	if err := json.Unmarshal(h.workerCall(http.MethodGet, fmt.Sprintf("/api/worker/jobs/%d/input", fpSeven.ID), nil, fpSeven.Lease, http.StatusOK), &input); err != nil {
		t.Fatal(err)
	}
	if turn := input.Voice.Diarization.Speakers[0].Turns[0]; turn.Start != 0 || turn.End != 1 {
		t.Fatalf("the Worker gets turns in its own stream time again: %+v", turn)
	}
	h.complete(fpTwo, trackFingerprints(2, two.ID, map[string][2]float64{"SPEAKER_00": {0.9, 2.1}, "SPEAKER_01": {3.0, 3.5}}, "SPEAKER_00", "SPEAKER_01"), http.StatusOK)
	// Track 7's Worker reports measures (#389): they carry no time, so the offset never touches them.
	seventh := trackFingerprints(7, seven.ID, map[string][2]float64{"SPEAKER_00": {0, 1}}, "SPEAKER_00")
	seventh["fingerprints"].([]map[string]any)[0]["measures"] = map[string]any{"version": 1, "net_speech_seconds": 1.0, "overlap_ratio": 0, "overlapped": false, "clipped_ratio": nil, "mean_level_dbfs": nil, "coherence": nil}
	h.complete(fpSeven, seventh, http.StatusOK)
	if h.count(`SELECT count(*) FROM voice_fingerprints vf JOIN local_speakers ls ON ls.id=vf.local_speaker_id WHERE ls.audio_stream_index=7 AND vf.start_seconds=1.25 AND vf.end_seconds=2.25 AND vf.duration_seconds=1 AND vf.net_speech_seconds=1`) != 1 {
		t.Fatal("fingerprint times follow the same single conversion, measures are kept as sent")
	}

	done := h.planVoice(plan)
	if done.Status != "done" || done.track(2).Status != "done" || done.track(7).Status != "done" || done.track(2).Speakers != 2 || done.track(7).Speakers != 1 {
		t.Fatalf("both tracks completed independently: %+v", done)
	}
	var view struct {
		PlanID   int64 `json:"plan_id"`
		Status   string
		Speakers []struct {
			ID    int64  `json:"id"`
			Label string `json:"label"`
			Track int    `json:"audio_stream_index"`
		} `json:"speakers"`
		Steps []struct{ Step string } `json:"steps"`
	}
	body := h.adminCall(http.MethodGet, fmt.Sprintf("/api/clips/%d/voice", h.clipID), nil, http.StatusOK)
	if err := json.Unmarshal(body, &view); err != nil {
		t.Fatal(err)
	}
	assertAudioContract(t, "ClipVoice", body)
	if view.PlanID != plan.ID || len(view.Speakers) != 3 || len(view.Steps) != 4 {
		t.Fatalf("the clip voice view covers every kept track: %s", body)
	}
	onTrack := map[int][]int64{}
	for _, s := range view.Speakers {
		onTrack[s.Track] = append(onTrack[s.Track], s.ID)
	}
	if len(onTrack[2]) != 2 || len(onTrack[7]) != 1 {
		t.Fatalf("each voice carries its own stream: %v", onTrack)
	}

	// Segments follow time AND track: both segments sit at 1..2, each takes the voice of its own stream.
	transcript := h.planTranscript()
	segments := transcript.Transcription.Segments
	if len(segments) != 2 || segments[0].SpeakerID == nil || *segments[0].SpeakerID != onTrack[2][0] || segments[1].SpeakerID == nil || *segments[1].SpeakerID != onTrack[7][0] {
		t.Fatalf("segments must take the voice of their own track only: %+v", segments)
	}
	if transcript.coverage(2, onTrack[2][0]) != "words" || transcript.coverage(7, onTrack[7][0]) != "words" {
		t.Fatalf("voices with words: %+v", transcript.Transcription.Coverage)
	}
	// SPEAKER_01 speaks only after the transcript: detected, visible, without words.
	if status := transcript.coverage(2, onTrack[2][1]); status != "outside_transcript" {
		t.Fatalf("a voice with no word must stay visible with its diagnostic, got %q", status)
	}
	// No identity came out of the merge of the texts.
	if h.count(`SELECT count(*) FROM persons`) != 0 || h.count(`SELECT count(*) FROM voice_samples`) != 0 || h.count(`SELECT count(*) FROM speaker_attributions WHERE validation_state='validated'`) != 0 {
		t.Fatal("no Person, Voice sample or validation may come from the voice analysis of a merged text")
	}
}

// A voice detected on a kept speech track whose recognizer found no words keeps a visible diagnostic, and a
// voice that is always outweighed under another one says so.
func testAudioPlanVoiceWithoutWords(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	h.putPipeline(fakePipeline(2))
	plan := h.publishVoicePlan([]int{2, 7}, map[int]string{2: "seule voix", 7: ""})
	h.startPlanVoice(plan, false, http.StatusOK)
	claimed := h.claimVoice(2)
	h.complete(claimed["voice.diarization.2"], trackDiarization(2, map[string][][2]float64{"SPEAKER_00": {{1, 1.8}}, "SPEAKER_01": {{1.8, 1.9}}}, "SPEAKER_00", "SPEAKER_01"), http.StatusOK)
	h.complete(claimed["voice.diarization.7"], trackDiarization(7, map[string][][2]float64{"SPEAKER_00": {{0.5, 1.5}}}, "SPEAKER_00"), http.StatusOK)
	transcript := h.planTranscript()
	if len(transcript.Transcription.Segments) != 1 || len(transcript.Transcription.LocalSpeakers) != 3 {
		t.Fatalf("every detected voice is listed although track 7 has no word: %+v", transcript.Transcription)
	}
	var two, seven []int64
	for _, s := range transcript.Transcription.LocalSpeakers {
		if s.Track == 7 {
			seven = append(seven, s.ID)
		} else {
			two = append(two, s.ID)
		}
	}
	if status := transcript.coverage(7, seven[0]); status != "no_transcript" {
		t.Fatalf("a detected voice without any word keeps a visible diagnostic, got %q", status)
	}
	if transcript.coverage(2, two[0]) != "words" || transcript.coverage(2, two[1]) != "speech_without_words" {
		t.Fatalf("an outweighed voice is speech without words: %+v", transcript.Transcription.Coverage)
	}
}

// A failure on one track leaves the other and the published text alone; a new start only redoes what is missing.
func testAudioPlanVoiceFailureAndResume(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	h.putPipeline(fakePipeline(2))
	plan := h.publishVoicePlan([]int{2, 7}, map[int]string{2: "voix deux", 7: "voix sept"})
	h.startPlanVoice(plan, false, http.StatusOK)
	if _, err := pool.Exec(context.Background(), `UPDATE jobs SET max_attempts=1 WHERE clip_id=$1 AND step LIKE 'voice.diarization.%'`, h.clipID); err != nil {
		t.Fatal(err)
	}
	claimed := h.claimVoice(2)
	two, seven := claimed["voice.diarization.2"], claimed["voice.diarization.7"]
	h.complete(two, trackDiarization(2, map[string][][2]float64{"SPEAKER_00": {{0.9, 2.1}}}, "SPEAKER_00"), http.StatusOK)
	h.workerCall(http.MethodPost, fmt.Sprintf("/api/worker/jobs/%d/fail", seven.ID), map[string]string{"error": "provider_unavailable"}, seven.Lease, http.StatusOK)

	// Track 2 continues to its fingerprints; track 7 failed with its cause; the transcript is still published.
	fp, ok := h.claim()
	if !ok || fp.Step != "voice.fingerprint.2" {
		t.Fatalf("the healthy track keeps going: %+v", fp)
	}
	h.complete(fp, trackFingerprints(2, two.ID, map[string][2]float64{"SPEAKER_00": {0.9, 2.1}}, "SPEAKER_00"), http.StatusOK)
	failed := h.planVoice(plan)
	if failed.Status != "failed" || failed.Cause != "provider_unavailable" || failed.track(2).Status != "done" || failed.track(7).Status != "failed" || failed.track(7).Cause != "provider_unavailable" {
		t.Fatalf("one track failed, the other is done: %+v", failed)
	}
	if len(h.planTranscript().Transcription.Segments) != 2 {
		t.Fatal("a voice failure must not unpublish the transcription")
	}
	doneJobs := h.count(`SELECT count(*) FROM jobs WHERE clip_id=$1 AND step LIKE 'voice.%' AND status='done'`, h.clipID)
	failedJob := failed.track(7).Diar.JobID

	// Automatic reconciliation leaves a terminal failure visible until an explicit retry.
	if _, err := h.api.startAnalysisPlanVoice(context.Background(), plan.ID, h.clipID, authorRef{Username: "analysis"}); err != nil {
		t.Fatal(err)
	}
	if got := h.planVoice(plan); got.track(7).Status != "failed" {
		t.Fatalf("automatic reconciliation resumed a terminal failure: %+v", got)
	}

	// Resume: same job, new cycle, nothing else is created or redone.
	h.startPlanVoice(plan, false, http.StatusOK)
	resumed := h.planVoice(plan)
	if resumed.track(7).Diar.JobID != failedJob || resumed.track(7).Status != "queued" || resumed.track(2).Diar.JobID != failed.track(2).Diar.JobID || resumed.track(2).Status != "done" {
		t.Fatalf("only the missing track is resumed, in place: %+v", resumed)
	}
	if h.count(`SELECT count(*) FROM jobs WHERE clip_id=$1 AND step LIKE 'voice.%' AND status='done'`, h.clipID) != doneJobs || h.count(`SELECT count(*) FROM jobs WHERE clip_id=$1 AND step LIKE 'voice.diarization.%'`, h.clipID) != 2 {
		t.Fatal("finished work must not be redone nor duplicated")
	}
	job, ok := h.claim()
	if !ok || job.ID != failedJob {
		t.Fatalf("the Worker gets the resumed job: %+v", job)
	}
	h.complete(job, trackDiarization(7, map[string][][2]float64{"SPEAKER_00": {{0, 1}}}, "SPEAKER_00"), http.StatusOK)
	fp, ok = h.claim()
	if !ok || fp.Step != "voice.fingerprint.7" {
		t.Fatalf("fingerprints of the resumed track: %+v", fp)
	}
	h.complete(fp, trackFingerprints(7, job.ID, map[string][2]float64{"SPEAKER_00": {0, 1}}, "SPEAKER_00"), http.StatusOK)
	if final := h.planVoice(plan); final.Status != "done" {
		t.Fatalf("resumed to completion: %+v", final)
	}
}

// Without a ready voice Worker the steps wait with their cause and the transcription stays published;
// cancelling then resuming works, and deleting the diarization steps respects the existing cascades.
func testAudioPlanVoiceWorkerAbsenceAndCancellation(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	h := newVoiceHarness(t, pool)
	h.putPipeline(fakePipeline(2))
	// A real local provider is active but no Worker holds its model: the steps can only wait.
	if _, err := pool.Exec(ctx, `INSERT INTO voice_pipeline_versions(version,config) VALUES('voice-real-fixture','{"provider":"pyannote","model":"fixture/diarization","model_version":"0123456789012345678901234567890123456789","min_fingerprint_seconds":1,"min_quality":0.5}'); UPDATE voice_pipeline_active SET version='voice-real-fixture' WHERE id=true`); err != nil {
		t.Fatal(err)
	}
	plan := h.publishVoicePlan([]int{2, 7}, map[int]string{2: "voix deux", 7: "voix sept"})
	h.startPlanVoice(plan, false, http.StatusOK)
	waiting := h.planVoice(plan)
	if waiting.Status != "queued" || waiting.Cause != "voice_worker_unavailable" || waiting.track(2).Cause != "voice_worker_unavailable" || waiting.track(7).Cause != "voice_worker_unavailable" {
		t.Fatalf("pending voice steps must say why they wait: %+v", waiting)
	}
	if _, ok := h.claim(); ok {
		t.Fatal("no Worker holds the model: nothing may be claimed")
	}
	if len(h.planTranscript().Transcription.Segments) != 2 {
		t.Fatal("the published transcription survives the absence of a voice Worker")
	}
	// Cancel one step: that track says so; the other keeps waiting.
	h.adminCall(http.MethodDelete, fmt.Sprintf("/api/admin/jobs/%d", waiting.track(7).Diar.JobID), nil, http.StatusNoContent)
	cancelled := h.planVoice(plan)
	if cancelled.track(7).Status != "cancelled" || cancelled.track(7).Cause != "cancelled_by_admin" || cancelled.track(2).Status != "queued" {
		t.Fatalf("cancellation is per track: %+v", cancelled)
	}
	// Resuming revives the same job rather than creating another.
	h.startPlanVoice(plan, false, http.StatusOK)
	if again := h.planVoice(plan); again.track(7).Status != "queued" || again.track(7).Diar.JobID != waiting.track(7).Diar.JobID {
		t.Fatalf("resume revives the cancelled step in place: %+v", again)
	}
	// Cancel everything, then delete the diarization step by step (the analysis run is over).
	for _, track := range []int{2, 7} {
		h.adminCall(http.MethodDelete, fmt.Sprintf("/api/admin/jobs/%d", h.planVoice(plan).track(track).Diar.JobID), nil, http.StatusNoContent)
	}
	if _, err := pool.Exec(ctx, `UPDATE clip_analysis_runs SET status='completed',finished_at=now() WHERE clip_id=$1`, h.clipID); err != nil {
		t.Fatal(err)
	}
	h.adminCall(http.MethodDelete, fmt.Sprintf("/api/admin/clips/%d/analysis", h.clipID), map[string]any{"steps": []string{"diarization"}, "confirm_loss": true}, http.StatusOK)
	if h.count(`SELECT count(*) FROM audio_plan_children WHERE plan_id=$1 AND stage IN ('diarization','fingerprint')`, plan.ID) != 0 {
		t.Fatal("deleting a voice step removes its bound child with it")
	}
	if cleared := h.planVoice(plan); cleared.Status != "not_started" || len(h.planTranscript().Transcription.Segments) != 2 {
		t.Fatalf("the plan is back to not started and the transcript stays published: %+v", cleared)
	}
}

// Human work on the replaced tracks: counted on every track, confirmed once, kept until a new result
// succeeds, and protected against validations made after the confirmation.
func testAudioPlanVoiceHumanWork(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	first, second := h.diarizeClip()
	var alice struct {
		Person struct {
			ID int64 `json:"id"`
		} `json:"person"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodPost, fmt.Sprintf("/api/local-speakers/%d/validation", first), map[string]any{"new_person_name": "Alice"}, http.StatusCreated), &alice); err != nil {
		t.Fatal(err)
	}
	oldJob := h.count(`SELECT diarization_job_id FROM local_speakers WHERE id=$1`, first)
	plan := h.publishVoicePlan([]int{2, 7}, map[int]string{2: "voix deux", 7: "voix sept"})

	preview := h.planVoice(plan)
	// The earlier diarization used the same pipeline: it is kept, so nothing is at stake there.
	if !preview.track(2).Reused || preview.track(2).Status != "done" || preview.Stake.ValidatedSpeakers != 0 {
		t.Fatalf("an identical earlier diarization is kept, so nothing is at stake there: %+v", preview)
	}
	// A different pipeline makes the validated voices replaceable: the stake is counted on that track only.
	h.putPipeline(map[string]any{"provider": "fake", "model": "fixture-diarization", "model_version": "fixture-v2", "min_fingerprint_seconds": 1.0, "min_quality": 0.5, "fake": map[string]any{"speakers": 2, "dimensions": 4}})
	preview = h.planVoice(plan)
	if preview.Stake.ValidatedSpeakers != 1 || preview.Stake.VoiceSamples != 1 || preview.Stake.AffectedPersons != 1 || len(preview.Stake.AudioStreamIndices) != 1 || preview.Stake.AudioStreamIndices[0] != 2 || preview.track(7).Stake.ValidatedSpeakers != 0 {
		t.Fatalf("validations and References are counted per replaced track: %+v", preview.Stake)
	}
	var refusal struct {
		Error string         `json:"error"`
		Stake voicePlanStake `json:"stake"`
	}
	if err := json.Unmarshal(h.startPlanVoice(plan, false, http.StatusConflict), &refusal); err != nil || refusal.Error != "voice_rediarization_confirmation_required" || refusal.Stake.ValidatedSpeakers != 1 {
		t.Fatalf("a start that replaces human work needs confirmation: %+v %v", refusal, err)
	}
	if h.count(`SELECT count(*) FROM jobs WHERE clip_id=$1 AND step='voice.diarization.7'`, h.clipID) != 0 {
		t.Fatal("a refused start schedules nothing, on any track")
	}
	assertAudioContract(t, "AudioPlanVoiceConfirmation", h.startPlanVoice(plan, false, http.StatusConflict))

	h.startPlanVoice(plan, true, http.StatusOK)
	claimed := h.claimVoice(2)
	// While it runs the old voices stay as they were, validation included.
	if h.count(`SELECT count(*) FROM local_speakers WHERE id=ANY($1) AND kind='diarized'`, []int64{first, second}) != 2 {
		t.Fatal("old voices stay until a new result succeeds")
	}
	// Concurrent human work after the confirmation: a validation of the other voice of the track.
	h.adminCall(http.MethodPost, fmt.Sprintf("/api/local-speakers/%d/validation", second), map[string]any{"new_person_name": "Bob"}, http.StatusCreated)
	h.complete(claimed["voice.diarization.2"], map[string]any{"provider": "fake", "model": "fixture-diarization", "model_version": "fixture-v2", "audio_stream_index": 2, "speakers": []map[string]any{{"label": "NEW", "turns": []map[string]any{{"start": 0.0, "end": 1.0}}}}}, http.StatusConflict)
	if h.count(`SELECT count(*) FROM local_speakers WHERE id=ANY($1) AND kind='diarized'`, []int64{first, second}) != 2 || h.count(`SELECT count(*) FROM local_speakers WHERE label='NEW'`) != 0 {
		t.Fatal("a validation made after the confirmation must lock the old voices")
	}
	// The healthy track is unaffected by the lock on track 2.
	other := trackDiarization(7, map[string][][2]float64{"SPEAKER_00": {{0, 1}}}, "SPEAKER_00")
	other["model_version"] = "fixture-v2"
	h.complete(claimed["voice.diarization.7"], other, http.StatusOK)

	// The lock fails the step; the administrator sees the new work in the next confirmation, then it succeeds.
	if _, err := pool.Exec(context.Background(), `UPDATE jobs SET status='dead',lease_owner=NULL,lease_expires_at=NULL,lease_token_hash=NULL WHERE id=$1`, claimed["voice.diarization.2"].ID); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(h.startPlanVoice(plan, false, http.StatusConflict), &refusal); err != nil || refusal.Stake.ValidatedSpeakers != 2 || refusal.Stake.AffectedPersons != 2 {
		t.Fatalf("the concurrent validation joins the stake: %+v %v", refusal, err)
	}
	h.startPlanVoice(plan, true, http.StatusOK)
	retry := h.claimVoice(2)
	rerun := retry["voice.diarization.2"]
	if rerun.ID != claimed["voice.diarization.2"].ID {
		t.Fatalf("the same step is resumed: %+v", rerun)
	}
	h.complete(rerun, map[string]any{"provider": "fake", "model": "fixture-diarization", "model_version": "fixture-v2", "audio_stream_index": 2, "speakers": []map[string]any{{"label": "NEW", "turns": []map[string]any{{"start": 0.0, "end": 1.0}}}}}, http.StatusOK)
	if h.count(`SELECT count(*) FROM local_speakers WHERE id=ANY($1) AND kind='archived' AND diarization_job_id=$2`, []int64{first, second}, oldJob) != 2 ||
		h.count(`SELECT count(*) FROM speaker_attributions WHERE local_speaker_id=ANY($1) AND validation_state='validated'`, []int64{first, second}) != 2 ||
		h.count(`SELECT count(*) FROM voice_samples WHERE local_speaker_id=ANY($1) AND retired_at IS NULL`, []int64{first, second}) != 2 {
		t.Fatal("after success the validated voices are archived with their validation and Voice samples, never deleted")
	}
	if h.count(`SELECT count(*) FROM persons`) != 2 {
		t.Fatal("no Person was created or merged by the voice analysis")
	}
}

// The designated microphone track keeps its direct attribution and is never diarized; the other
// kept track is analysed normally and the direct attribution does not spread to it.
func testAudioPlanVoiceMicrophone(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	h.putPipeline(fakePipeline(2))
	var sourceID int64
	if err := pool.QueryRow(context.Background(), `SELECT id FROM clip_sources ORDER BY id LIMIT 1`).Scan(&sourceID); err != nil {
		t.Fatal(err)
	}
	alice := h.newPerson("Alice Micro")
	h.adminCall(http.MethodPut, fmt.Sprintf("/api/admin/sources/%d/microphone-track", sourceID), map[string]any{"title": "Micro", "person_id": alice}, http.StatusOK)
	plan := h.publishVoicePlan([]int{1, 2}, map[int]string{1: "au micro", 2: "en jeu"})
	h.startPlanVoice(plan, false, http.StatusOK)
	state := h.planVoice(plan)
	if !state.track(1).Microphone || state.track(1).Status != "not_applicable" || state.track(1).Cause != "microphone_direct_attribution" || state.track(1).Diar != nil {
		t.Fatalf("the microphone track is attributed directly, never diarized: %+v", state.track(1))
	}
	if h.count(`SELECT count(*) FROM jobs WHERE clip_id=$1 AND step='voice.diarization.1'`, h.clipID) != 0 {
		t.Fatal("no diarization for the microphone track")
	}
	claimed := h.claimVoice(1)
	h.complete(claimed["voice.diarization.2"], trackDiarization(2, map[string][][2]float64{"SPEAKER_00": {{1, 2}}}, "SPEAKER_00"), http.StatusOK)
	transcript := h.planTranscript()
	segments := transcript.Transcription.Segments
	if len(segments) != 2 {
		t.Fatalf("both tracks are published: %+v", segments)
	}
	var micSpeaker, gameSpeaker int64
	for _, s := range transcript.Transcription.LocalSpeakers {
		if s.Track == 1 {
			micSpeaker = s.ID
		} else {
			gameSpeaker = s.ID
		}
	}
	if micSpeaker == 0 || gameSpeaker == 0 || segments[0].SpeakerID == nil || *segments[0].SpeakerID != micSpeaker || segments[1].SpeakerID == nil || *segments[1].SpeakerID != gameSpeaker {
		t.Fatalf("each segment takes the voice of its own track: %+v", segments)
	}
	if h.count(`SELECT count(*) FROM speaker_attributions WHERE local_speaker_id=$1 AND origin='microphone_track'`, micSpeaker) != 1 || h.count(`SELECT count(*) FROM speaker_attributions WHERE local_speaker_id=$1 AND person_id IS NOT NULL`, gameSpeaker) != 0 {
		t.Fatal("the direct attribution stays on the microphone voice only")
	}
}

// A plan that is not (or no longer) the published revision starts nothing.
func testAudioPlanVoiceNotPublished(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	h.putPipeline(fakePipeline(2))
	h.transcribe()
	var run int64
	if err := pool.QueryRow(context.Background(), `INSERT INTO clip_analysis_runs(clip_id,scope,status) VALUES($1,'full','active') RETURNING id`, h.clipID).Scan(&run); err != nil {
		t.Fatal(err)
	}
	var input audioplans.Input
	if err := json.Unmarshal(audioplans.ContractFixture(), &input); err != nil {
		t.Fatal(err)
	}
	plan, err := audioplans.New(pool).CreatePlan(context.Background(), run, input)
	if err != nil {
		t.Fatal(err)
	}
	path := fmt.Sprintf("/api/admin/clips/%d/audio-plans/%d/voice", h.clipID, plan.Id)
	h.adminCall(http.MethodPost, path, map[string]any{}, http.StatusConflict)
	var state planVoiceView
	if err = json.Unmarshal(h.adminCall(http.MethodGet, path, nil, http.StatusOK), &state); err != nil || state.Published || state.Status != "not_published" {
		t.Fatalf("an unpublished plan only reports: %+v %v", state, err)
	}
	h.adminCall(http.MethodGet, fmt.Sprintf("/api/admin/clips/%d/audio-plans/999999/voice", h.clipID), nil, http.StatusNotFound)
	if h.count(`SELECT count(*) FROM jobs WHERE clip_id=$1 AND step LIKE 'voice.%'`, h.clipID) != 0 {
		t.Fatal("no voice job for an unpublished plan")
	}
	// A legacy clip keeps the single-track voice view.
	var legacy map[string]any
	if err = json.Unmarshal(h.adminCall(http.MethodGet, fmt.Sprintf("/api/clips/%d/voice", h.clipID), nil, http.StatusOK), &legacy); err != nil {
		t.Fatal(err)
	}
	if _, has := legacy["tracks"]; has {
		t.Fatal("the legacy voice view is unchanged")
	}
}

// A restart after several days must still catch a missed post-publication hook.
func testAudioVoiceDelayedResume(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	plan := h.publishVoicePlan([]int{2}, map[int]string{2: "synthetic delayed voice"})
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `UPDATE clip_analysis_runs SET status='completed',finished_at=now()-interval '3 days' WHERE id=(SELECT run_id FROM audio_plans WHERE id=$1)`, plan.ID); err != nil {
		t.Fatal(err)
	}
	h.api.resumeAudioVoices(ctx)
	if n := h.count(`SELECT count(*) FROM audio_plan_voice_starts WHERE plan_id=$1`, plan.ID); n != 1 {
		t.Fatalf("delayed voice start: %d", n)
	}
	h.api.resumeAudioVoices(ctx)
	if n := h.count(`SELECT count(*) FROM audio_plan_voice_starts WHERE plan_id=$1`, plan.ID); n != 1 {
		t.Fatalf("repeated recovery duplicated the start: %d", n)
	}
}

// Forced diarization reuses durable child identities but recomputes their fingerprints.
func testAudioPlanVoiceForcedRun(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	h.putPipeline(fakePipeline(2))
	plan := h.publishVoicePlan([]int{2}, map[int]string{2: "voix deux"})
	h.startPlanVoice(plan, false, http.StatusOK)
	diar, ok := h.claim()
	if !ok {
		t.Fatal("diarization not claimable")
	}
	h.complete(diar, trackDiarization(2, map[string][][2]float64{"OLD": {{0.9, 2.1}}}, "OLD"), http.StatusOK)
	fp, ok := h.claim()
	if !ok {
		t.Fatal("fingerprint not claimable")
	}
	h.complete(fp, trackFingerprints(2, diar.ID, map[string][2]float64{"OLD": {0.9, 2.1}}, "OLD"), http.StatusOK)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `INSERT INTO clip_analysis_runs(clip_id,scope,mode,forced_steps,status) VALUES($1,'voice','steps',ARRAY['diarization'],'active')`, h.clipID); err != nil {
		t.Fatal(err)
	}
	if _, err := h.api.startAnalysisPlanVoice(ctx, plan.ID, h.clipID, authorRef{Username: "analysis"}); err != nil {
		t.Fatal(err)
	}
	rerun, ok := h.claim()
	if !ok || rerun.ID != diar.ID {
		t.Fatalf("durable diarization not resumed: %+v", rerun)
	}
	h.complete(rerun, trackDiarization(2, map[string][][2]float64{"NEW": {{1.0, 2.0}}}, "NEW"), http.StatusOK)
	renewed, ok := h.claim()
	if !ok || renewed.ID != fp.ID {
		t.Fatalf("plan fingerprint not resumed: %+v", renewed)
	}
	if got := h.planVoice(plan); got.track(2).Status == "done" {
		t.Fatal("stale fingerprint reported done")
	}
	h.complete(renewed, trackFingerprints(2, diar.ID, map[string][2]float64{"NEW": {1.0, 2.0}}, "NEW"), http.StatusOK)
	if got := h.planVoice(plan); got.Status != "done" {
		t.Fatalf("rerun incomplete: %+v", got)
	}
	if h.count(`SELECT count(*) FROM voice_fingerprints f JOIN local_speakers s ON s.id=f.local_speaker_id WHERE s.clip_id=$1 AND s.label='NEW'`, h.clipID) == 0 {
		t.Fatal("new speakers have no fingerprint")
	}
}
