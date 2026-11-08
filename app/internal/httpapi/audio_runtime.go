package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"
	"snitch/internal/analysis"
	"snitch/internal/audioplans"
	"snitch/internal/httpapi/contract"
	"snitch/internal/jobs"
	"snitch/internal/multitrack"
)

// The activation of the multitrack pipeline (#344, ADR 0054). Everything that plans the transcription of a clip
// with several audio tracks goes through an immutable audio plan: the legacy single-track scheduler keeps serving
// the clips with one audio track (and the results already produced). This file owns the pieces that need the
// server: inventory and versions of a plan, the engine behind the selection routes, the publisher of the
// complete plan and the entry points of the legacy flows.

// The pinned speech detector, as the Worker reports it (worker/snitch_worker/audio_diagnostic.py: VERSION and
// CONFIG_VERSION; a test keeps both in step). A plan names them, so a new detector is a new plan.
const (
	audioDetectorVersion       = "silero-v6.2.1-7e30209a3e901f9842f81b225f3e93d8199902b1"
	audioDetectorConfigVersion = "channels-16k-512-hi050-lo020-min096-v1"
)

// multitrackClip tells whether a clip has several audio tracks: the only clips the plan pipeline takes over.
func multitrackClip(tracks []audioTrack) bool {
	n := 0
	for _, t := range tracks {
		if t.Type == "audio" {
			n++
		}
	}
	return n > 1
}

func audioInventoryOf(tracks []audioTrack) []contract.AudioTrackInventory {
	out := []contract.AudioTrackInventory{}
	for _, t := range tracks {
		if t.Type != "audio" {
			continue
		}
		start := 0.0
		evidence := []string{}
		if t.StartSeconds == nil {
			evidence = append(evidence, "timeline_unknown")
		}
		if t.StartSeconds != nil {
			start = *t.StartSeconds
		}
		out = append(out, contract.AudioTrackInventory{Index: t.Index, Title: t.Title, Codec: t.Codec, Channels: t.Channels, ChannelLayout: t.ChannelLayout, Language: t.Language, DurationSeconds: t.DurationSeconds, StartSeconds: start, Evidence: evidence})
	}
	return out
}

// currentVocabularyVersion records the active vocabulary snapshot and returns its version.
func (s *Server) currentVocabularyVersion(ctx context.Context) (string, error) {
	vocabulary, err := s.activeVocabulary(ctx)
	if err != nil {
		return "", err
	}
	version := vocabularyVersion(vocabulary)
	raw, err := json.Marshal(vocabulary)
	if err != nil {
		return "", err
	}
	_, err = s.Pool.Exec(ctx, `INSERT INTO transcription_vocabulary_versions(version,terms) VALUES($1,$2) ON CONFLICT(version) DO NOTHING`, version, raw)
	return version, err
}

// baseAudioInput is the plan input without tracks: the versions that identify the strategy.
func (s *Server) baseAudioInput(ctx context.Context) (audioplans.Input, error) {
	vocabulary, err := s.currentVocabularyVersion(ctx)
	if err != nil {
		return audioplans.Input{}, err
	}
	pipeline, err := s.activeTranscriptionPipeline(ctx)
	if err != nil {
		return audioplans.Input{}, err
	}
	return audioplans.Input{
		Mode: "auto", Decisions: []contract.AudioTrackDecision{}, Inventory: []contract.AudioTrackInventory{}, TrackIndices: []int{},
		DetectorVersion: audioDetectorVersion, DetectorConfigVersion: audioDetectorConfigVersion, MergeVersion: audioplans.MergeVersion(),
		PipelineVersion: pipeline, VocabularyVersion: vocabulary,
	}, nil
}

// legacyAudioInput is the plan a legacy flow asks for: every audio track examined (auto), or the one track a
// source rule or a clip override names (manual: the diagnostic still decides, a blocked track needs a human).
func (s *Server) legacyAudioInput(ctx context.Context, clipID int64, explicit *int) (audioplans.Input, error) {
	tracks, err := s.clipAudioTracks(ctx, clipID)
	if err != nil {
		return audioplans.Input{}, err
	}
	input, err := s.baseAudioInput(ctx)
	if err != nil {
		return input, err
	}
	input.Inventory = audioInventoryOf(tracks)
	index := explicit
	if index == nil {
		selection, selErr := s.audioSelection(ctx, clipID, tracks)
		if selErr != nil {
			return input, selErr
		}
		index = selection.AudioStreamIndex
	}
	if index != nil {
		if !hasAudioTrack(tracks, *index) {
			return input, errors.New("audio stream not found")
		}
		input.Mode, input.TrackIndices = "manual", []int{*index}
		return input, nil
	}
	for _, t := range input.Inventory {
		input.TrackIndices = append(input.TrackIndices, t.Index)
	}
	return input, nil
}

// audioRunStart describes the run a plan belongs to.
type audioRunStart struct {
	Scope         string
	Mode          string
	ForcedSteps   []string
	Starter       int64
	ConfirmLosses bool
	StakeAt       *time.Time
}

// startAudioPlanTx makes the plan of a run in the caller's transaction: the plan, the diagnostic jobs bound in the
// same statement, the steps of the run and the confirmation of the losses the administrator accepted. The clip is
// locked first, like every correction, so the losses seen are the ones the publication will check.
func (s *Server) startAudioPlanTx(ctx context.Context, tx pgx.Tx, runID, clipID int64, input audioplans.Input, confirmCorrections, confirmValidations bool) (audioplans.Plan, error) {
	if _, err := tx.Exec(ctx, `SELECT id FROM clips WHERE id=$1 FOR UPDATE`, clipID); err != nil {
		return audioplans.Plan{}, err
	}
	store := audioplans.New(tx)
	plan, err := store.CreatePlan(ctx, runID, input)
	if err != nil {
		return plan, err
	}
	if _, err = store.QueueDiagnostics(ctx, plan.Id); err != nil {
		return plan, err
	}
	var scope string
	if err := tx.QueryRow(ctx, `SELECT scope FROM clip_analysis_runs WHERE id=$1`, runID).Scan(&scope); err != nil {
		return plan, err
	}
	for _, step := range []string{analysis.StepPreparation, analysis.StepTranscription, analysis.StepIndexing, analysis.StepDiarization, analysis.StepFingerprints, analysis.StepMatching} {
		status, reason := "not_applicable", "audio_plan"
		if step == analysis.StepPreparation || step == analysis.StepTranscription || scope == analysis.ScopeFull || scope == analysis.ScopeVoice && step != analysis.StepIndexing {
			status = "waiting"
		}
		if _, err = tx.Exec(ctx, `INSERT INTO clip_analysis_steps(run_id,step,status,reason,finished_at) VALUES($1,$2,$3,$4,CASE WHEN $3='not_applicable' THEN now() END)
 ON CONFLICT(run_id,step) DO UPDATE SET status=EXCLUDED.status,reason=EXCLUDED.reason,finished_at=EXCLUDED.finished_at,updated_at=now()`, runID, step, status, reason); err != nil {
			return plan, err
		}
	}
	preview, err := store.PreviewClipPublication(ctx, clipID, plan.Id)
	if err != nil {
		return plan, err
	}
	err = store.RecordConfirmation(ctx, audioplans.Confirmation{PlanID: plan.Id, ExpectedRevisionID: preview.ExpectedRevisionId, Guard: preview.Guard,
		Confirmed: preview.ConfirmationRequired && (preview.CorrectionCount == 0 || confirmCorrections) && (preview.ValidatedVoiceCount == 0 || confirmValidations)})
	return plan, err
}

// AttachAudio gives the run a plan when the clip has several audio tracks and the run needs a transcription (the
// legacy hooks never choose a track for such a clip). The confirmation comes from what the administrator accepted
// when the run started; an automatic run accepts nothing, so human work is never replaced silently.
func (h analysisHooks) AttachAudio(ctx context.Context, run analysis.Run) (bool, error) {
	s := h.s
	if run.Scope == analysis.ScopeVoice {
		return false, nil
	}
	tracks, err := s.clipAudioTracks(ctx, run.ClipID)
	if err != nil || !multitrackClip(tracks) {
		return false, err
	}
	has, err := h.HasTranscription(ctx, run.ClipID)
	if err != nil {
		return false, err
	}
	if has && !slices.Contains(run.ForcedSteps, analysis.StepTranscription) {
		return false, nil
	}
	var ready bool
	if err = s.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM uploaded_media WHERE clip_id=$1) OR EXISTS(SELECT 1 FROM source_files WHERE clip_id=$1 AND status='ready')`, run.ClipID).Scan(&ready); err != nil || !ready {
		return false, err
	}
	input, err := s.legacyAudioInput(ctx, run.ClipID, nil)
	if err != nil {
		return false, err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SELECT id FROM clips WHERE id=$1 FOR UPDATE`, run.ClipID); err != nil {
		return false, err
	}
	var exists bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM audio_plans WHERE run_id=$1) OR NOT EXISTS(SELECT 1 FROM clip_analysis_runs WHERE id=$1 AND status='active')`, run.ID).Scan(&exists); err != nil || exists {
		return exists, err
	}
	if _, err = s.startAudioPlanTx(ctx, tx, run.ID, run.ClipID, input, run.ConfirmedReplaceCorrections, run.ConfirmedReplaceValidations); err != nil {
		return false, err
	}
	return true, tx.Commit(ctx)
}

// scheduleMultitrack is the legacy "make sure this clip is transcribed" for a clip with several audio tracks:
// it starts (or finds) the plan of the clip instead of a single-track job. confirmLosses is the confirmation the
// caller already obtained from the administrator (corrections guard, batch with corrected clips). It returns the
// same shape as the legacy scheduler: created is false when an identical plan is already active or published.
func (s *Server) scheduleMultitrack(ctx context.Context, clipID int64, explicit *int, actor int64, confirmLosses bool) (jobs.Job, bool, error) {
	input, err := s.legacyAudioInput(ctx, clipID, explicit)
	if err != nil {
		return jobs.Job{}, false, err
	}
	canonical, err := audioplans.CanonicalInput(input)
	if err != nil {
		return jobs.Job{}, false, err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return jobs.Job{}, false, err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SELECT id FROM clips WHERE id=$1 FOR UPDATE`, clipID); err != nil {
		return jobs.Job{}, false, err
	}
	pseudo := jobs.Job{ClipID: clipID, Step: "audio.plan", Status: "pending"}
	var same int64
	// The same plan already active, or already the published revision: nothing to do (idempotent callers).
	err = tx.QueryRow(ctx, `SELECT p.id FROM audio_plans p WHERE p.clip_id=$1 AND p.input_version=$2 AND (
 EXISTS(SELECT 1 FROM clip_analysis_runs r WHERE r.id=p.run_id AND r.status='active') OR
 EXISTS(SELECT 1 FROM audio_publications ap JOIN transcript_revisions tr ON tr.id=ap.revision_id WHERE ap.clip_id=p.clip_id AND tr.plan_id=p.id)) ORDER BY p.id DESC LIMIT 1`, clipID, canonical.Version).Scan(&same)
	if err == nil {
		return pseudo, false, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return jobs.Job{}, false, err
	}
	var starter *int64
	if actor > 0 {
		starter = &actor
	}
	// A newer request for other inputs replaces an older automatic one that no administrator decided (the legacy
	// scheduler supersedes the pending job of an older input version the same way). Plans that carry administrator
	// decisions, and runs of other kinds, are never superseded.
	var olderRun int64
	err = tx.QueryRow(ctx, `SELECT r.id FROM clip_analysis_runs r JOIN audio_plans p ON p.run_id=r.id WHERE r.clip_id=$1 AND r.status='active' AND r.scope='transcription'
 AND jsonb_array_length(p.input->'decisions')=0 AND NOT EXISTS(SELECT 1 FROM audio_plans n WHERE n.run_id=r.id AND n.id>p.id) ORDER BY p.id DESC LIMIT 1`, clipID).Scan(&olderRun)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return jobs.Job{}, false, err
	}
	if err == nil {
		if err = analysis.CancelAudioRun(ctx, tx, olderRun); err != nil {
			return jobs.Job{}, false, err
		}
	}
	var runID int64
	if err = tx.QueryRow(ctx, `INSERT INTO clip_analysis_runs(clip_id,scope,started_by,mode,confirmed_replace_corrections,confirmed_replace_validations) VALUES($1,'transcription',$2,'missing',$3,false) ON CONFLICT (clip_id) WHERE status='active' DO NOTHING RETURNING id`, clipID, starter, confirmLosses).Scan(&runID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Another analysis is running on the clip: it owns the transcription, nothing is scheduled twice.
			return pseudo, false, nil
		}
		return jobs.Job{}, false, err
	}
	if _, err = s.startAudioPlanTx(ctx, tx, runID, clipID, input, confirmLosses, false); err != nil {
		return jobs.Job{}, false, err
	}
	if err = tx.Commit(ctx); err != nil {
		return jobs.Job{}, false, err
	}
	s.reconcileAnalysis(ctx, clipID)
	return pseudo, true, nil
}

// ---- publication --------------------------------------------------------------------------------------------

// audioPublisher assembles the revision of a complete plan and publishes it with #340's guards, in the
// transaction of the orchestrator. It is the only publisher: children never publish.
type audioPublisher struct{ s *Server }

type childOutput struct {
	Segments []struct {
		Start float64 `json:"start"`
		End   float64 `json:"end"`
		Words []struct {
			Start float64 `json:"start"`
			End   float64 `json:"end"`
		} `json:"words"`
	} `json:"segments"`
}

func (p audioPublisher) PublishAudio(ctx context.Context, tx pgx.Tx, r multitrack.Result) error {
	store := audioplans.New(tx)
	revision := audioplans.Revision{ClipId: r.Plan.ClipId, PlanId: &r.Plan.Id, TimeBase: "clip", Empty: r.Empty, Segments: []contract.TranscriptManifestSegment{}}
	var sources []audioplans.SourceSegment
	for _, track := range r.Tracks {
		var output childOutput
		if err := json.Unmarshal(track.Output, &output); err != nil {
			return &analysis.PublicationBlocked{Cause: "invalid_audio_result"}
		}
		for i, segment := range output.Segments {
			source := audioplans.SourceSegment{Source: contract.TranscriptSource{TrackIndex: track.TrackIndex, JobId: track.JobID, SegmentIndex: i}, Start: segment.Start, End: segment.End}
			for j, word := range segment.Words {
				index := j
				source.Words = append(source.Words, contract.TranscriptManifestWord{Start: word.Start, End: word.End, Sources: []contract.TranscriptSource{{TrackIndex: track.TrackIndex, JobId: track.JobID, SegmentIndex: i, WordIndex: &index}}})
			}
			if source.Words == nil {
				source.Words = []contract.TranscriptManifestWord{}
			}
			sources = append(sources, source)
		}
	}
	// A retained track with no words still participates in voice analysis. It
	// cannot, by itself, justify publishing a no-speech revision.
	if len(sources) == 0 && !r.Empty {
		return &analysis.PublicationBlocked{Cause: "audio_transcription_empty"}
	}
	if len(r.Evidence) > 0 {
		report, err := audioplans.ParseEvidence(r.Evidence)
		if err != nil {
			return &analysis.PublicationBlocked{Cause: "invalid_audio_evidence"}
		}
		merged, err := audioplans.MergeAcoustic(sources, report)
		if err != nil {
			return &analysis.PublicationBlocked{Cause: "invalid_audio_evidence"}
		}
		revision.Segments = merged.Segments
	} else {
		for _, source := range sources {
			revision.Segments = append(revision.Segments, contract.TranscriptManifestSegment{Start: source.Start, End: source.End, Sources: []contract.TranscriptSource{source.Source}, Words: source.Words})
		}
	}
	staged, err := store.StageRevision(ctx, revision)
	if err != nil {
		return err
	}
	confirmation, err := store.GetConfirmation(ctx, r.Plan.Id)
	if errors.Is(err, pgx.ErrNoRows) {
		return &analysis.PublicationBlocked{Cause: "audio_confirmation_required"}
	}
	if err != nil {
		return err
	}
	preview := audioplans.PublicationPreview{AudioLossPreview: contract.AudioLossPreview{ExpectedRevisionId: confirmation.ExpectedRevisionID}, PlanID: r.Plan.Id, Guard: confirmation.Guard}
	switch err = store.PublishRevision(ctx, staged.Id, preview, confirmation.Confirmed); {
	case err == nil:
		return nil
	case errors.Is(err, audioplans.ErrPublicationChanged):
		return &analysis.PublicationBlocked{Cause: "audio_publication_changed"}
	case errors.Is(err, audioplans.ErrConfirmationRequired):
		return &analysis.PublicationBlocked{Cause: "audio_confirmation_required"}
	case errors.Is(err, audioplans.ErrIncomplete):
		return &analysis.PublicationBlocked{Cause: "audio_publication_incomplete"}
	}
	return err
}

// afterAudioPublication starts the voice analysis of the published plan when the run asked for it. A failure
// here never undoes the publication: the periodic sweep (resumeAudioVoices) retries it.
func (s *Server) afterAudioPublication(ctx context.Context, planID, clipID int64) {
	var scope string
	var starter *int64
	if err := s.Pool.QueryRow(ctx, `SELECT r.scope,r.started_by FROM clip_analysis_runs r JOIN audio_plans p ON p.run_id=r.id WHERE p.id=$1`, planID).Scan(&scope, &starter); err != nil {
		s.Log.Warn("read run of the published audio plan", "plan", planID, "error", err)
		return
	}
	if scope == analysis.ScopeTranscription {
		return
	}
	author := authorRef{ID: starter, Username: "analysis"}
	if _, err := s.startAnalysisPlanVoice(ctx, planID, clipID, author); err != nil {
		var needs *voiceConfirmationError
		if errors.As(err, &needs) {
			s.Log.Info("voice analysis of a published audio plan waits for confirmation", "plan", planID, "clip", clipID)
			return
		}
		s.Log.Warn("start voice analysis of a published audio plan", "plan", planID, "clip", clipID, "error", err)
	}
}

// resumeAudioVoices is the safety net of afterAudioPublication: published plans of full or voice runs whose voice
// analysis never started (a restart between the two, a Worker error) are started again. It is idempotent.
func (s *Server) resumeAudioVoices(ctx context.Context) {
	rows, err := s.Pool.Query(ctx, `SELECT p.id,p.clip_id FROM audio_plans p JOIN clip_analysis_runs r ON r.id=p.run_id
 JOIN audio_publications ap ON ap.clip_id=p.clip_id JOIN transcript_revisions tr ON tr.id=ap.revision_id AND tr.plan_id=p.id
 WHERE r.status IN ('active','completed') AND r.scope<>'transcription' AND NOT EXISTS(SELECT 1 FROM audio_plan_voice_starts v WHERE v.plan_id=p.id)
 AND NOT EXISTS(SELECT 1 FROM audio_plan_children c WHERE c.plan_id=p.id AND c.stage IN ('diarization','fingerprint'))
 ORDER BY r.finished_at,p.id`)
	if err != nil {
		s.Log.Warn("list published audio plans without voices", "error", err)
		return
	}
	type pair struct{ plan, clip int64 }
	var todo []pair
	for rows.Next() {
		var p pair
		if rows.Scan(&p.plan, &p.clip) == nil {
			todo = append(todo, p)
		}
	}
	rows.Close()
	for _, p := range todo {
		s.afterAudioPublication(ctx, p.plan, p.clip)
	}
}

// serveAudioVoices runs the safety net of the voice analysis of published plans at start and on every tick.
func (s *Server) serveAudioVoices(ctx context.Context, every time.Duration) {
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		s.resumeAudioVoices(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// ---- selection engine ---------------------------------------------------------------------------------------

// audioEngine is the real AudioSelectionEngine: diagnostics of the latest plan, effects of a choice and the
// start of a plan inside the transaction the route supplies.
type audioEngine struct{ s *Server }

func (e audioEngine) Context(ctx context.Context, tx pgx.Tx, clipID int64) (contract.AudioSelectionContext, error) {
	base, err := e.s.baseAudioInput(ctx)
	if err != nil {
		return contract.AudioSelectionContext{}, err
	}
	out := contract.AudioSelectionContext{Input: &base, Diagnostics: []contract.AudioSelectionDiagnostic{}, Inventory: []contract.AudioTrackInventory{}}
	var raw []byte
	if err = tx.QueryRow(ctx, `SELECT tracks FROM clips WHERE id=$1`, clipID).Scan(&raw); err != nil {
		return out, err
	}
	var tracks []audioTrack
	if err = json.Unmarshal(raw, &tracks); err != nil {
		return out, err
	}
	inventory := audioInventoryOf(tracks)
	var latest int64
	var active bool
	err = tx.QueryRow(ctx, `SELECT p.id,r.status='active' FROM audio_plans p JOIN clip_analysis_runs r ON r.id=p.run_id WHERE p.clip_id=$1 ORDER BY p.id DESC LIMIT 1`, clipID).Scan(&latest, &active)
	if errors.Is(err, pgx.ErrNoRows) {
		out.StateVersion = "no-plan"
		return out, nil
	}
	if err != nil {
		return out, err
	}
	state, err := audioplans.New(tx).GetPlan(ctx, latest)
	if err != nil {
		return out, err
	}
	// A diagnostic is current only for the detector and the track metadata that produced it.
	plan := state.Plan
	known := map[int]contract.AudioTrackInventory{}
	for _, t := range plan.Input.Inventory {
		known[t.Index] = t
	}
	for _, d := range state.Diagnostics {
		fresh := contract.AudioSelectionDiagnosticFreshnessCurrent
		current, ok := inventoryEntry(inventory, d.TrackIndex)
		previous := known[d.TrackIndex]
		a, _ := json.Marshal(current)
		b, _ := json.Marshal(previous)
		if d.Status == "unexamined" {
			fresh = contract.AudioSelectionDiagnosticFreshnessMissing
		} else if !ok || string(a) != string(b) || plan.Input.DetectorVersion != base.DetectorVersion || plan.Input.DetectorConfigVersion != base.DetectorConfigVersion {
			fresh = contract.AudioSelectionDiagnosticFreshnessStale
		}
		out.Diagnostics = append(out.Diagnostics, contract.AudioSelectionDiagnostic{TrackIndex: d.TrackIndex, Status: contract.AudioSelectionDiagnosticStatus(d.Status), Freshness: fresh, Cause: d.Cause})
	}
	out.StateVersion = fmt.Sprintf("plan:%d:%s:%t", latest, plan.InputVersion, active)
	return out, nil
}

func inventoryEntry(inventory []contract.AudioTrackInventory, index int) (contract.AudioTrackInventory, bool) {
	for _, t := range inventory {
		if t.Index == index {
			return t, true
		}
	}
	return contract.AudioTrackInventory{}, false
}

// ReusablePlan keeps a limited run on the already published immutable selection.
func (e audioEngine) ReusablePlan(ctx context.Context, tx pgx.Tx, clipID int64, input audioplans.Input, choice contract.AudioAnalysisChoice) (int64, error) {
	var id int64
	err := tx.QueryRow(ctx, `SELECT COALESCE(r.plan_id,0) FROM audio_publications p JOIN transcript_revisions r ON r.id=p.revision_id WHERE p.clip_id=$1`, clipID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	if err != nil || id == 0 {
		return 0, err
	}
	state, err := audioplans.New(tx).GetPlan(ctx, id)
	if err != nil {
		return 0, err
	}
	old := state.Plan.Input
	// Decisions retain their audit identity on the published plan; only their meaning matters here.
	normalize := func(in audioplans.Input) audioplans.Input {
		for i := range in.Decisions {
			in.Decisions[i].ActorId = 0
			in.Decisions[i].DecidedAt = time.Time{}
			in.Decisions[i].Reason = ""
		}
		return in
	}
	old.Decisions = append([]contract.AudioTrackDecision{}, old.Decisions...)
	input.Decisions = append([]contract.AudioTrackDecision{}, input.Decisions...)
	a, _ := json.Marshal(normalize(old))
	b, _ := json.Marshal(normalize(input))
	if !bytes.Equal(a, b) {
		return 0, nil
	}
	resolved, err := analysis.Resolve(string(choice.Scope), analysis.Options{Mode: string(choice.Mode), Steps: choice.Steps}, analysis.Existing{Transcription: true, Voices: true})
	if err != nil {
		return 0, err
	}
	if slices.Contains(resolved.Forced, analysis.StepTranscription) {
		return 0, nil
	}
	return id, nil
}

func (e audioEngine) Preview(ctx context.Context, tx pgx.Tx, clipID int64, input audioplans.Input, choice contract.AudioAnalysisChoice) (contract.AudioSelectionEffects, error) {
	preview, err := audioplans.New(tx).PreviewClipPublication(ctx, clipID, 0)
	if err != nil {
		return contract.AudioSelectionEffects{}, err
	}
	reuse, err := e.ReusablePlan(ctx, tx, clipID, input, choice)
	if err != nil {
		return contract.AudioSelectionEffects{}, err
	}
	if choice.Scope == analysis.ScopeVoice && reuse == 0 {
		return contract.AudioSelectionEffects{}, analysis.ErrTranscriptionRequired
	}
	resolved, err := analysis.Resolve(string(choice.Scope), analysis.Options{Mode: string(choice.Mode), Steps: choice.Steps}, analysis.Existing{Transcription: reuse != 0, Voices: true})
	if err != nil {
		return contract.AudioSelectionEffects{}, err
	}
	added := []string{}
	for _, i := range input.TrackIndices {
		if reuse != 0 {
			break
		}
		added = append(added, fmt.Sprintf("audio.diagnostic.%d", i), fmt.Sprintf("transcription.audio.%d", i))
	}
	if reuse == 0 && len(input.TrackIndices) > 1 {
		added = append(added, "audio.dedup")
	}
	if resolved.Scope != "transcription" {
		for _, i := range input.TrackIndices {
			added = append(added, fmt.Sprintf("voice.diarization.%d", i), fmt.Sprintf("voice.fingerprint.%d", i))
		}
	}
	if reuse != 0 {
		preview.CorrectionCount = 0
		preview.ReplacedTrackIndices = []int{}
		preview.InvalidatedSteps = []string{}
		if !slices.Contains(resolved.Forced, analysis.StepDiarization) {
			preview.ValidatedVoiceCount = 0
		} else {
			preview.InvalidatedSteps = append(preview.InvalidatedSteps, "diarization", "fingerprints", "matching")
		}
		if resolved.Scope == analysis.ScopeFull {
			added = append(added, "indexing")
		}
		preview.ConfirmationRequired = preview.ValidatedVoiceCount > 0
	}
	compatible, err := analysis.AudioWorkerCompatible(ctx, tx)
	if err != nil {
		return contract.AudioSelectionEffects{}, err
	}
	return contract.AudioSelectionEffects{
		AddedSteps: added, InvalidatedSteps: preview.InvalidatedSteps, TrackIndices: append([]int{}, input.TrackIndices...),
		StakeVersion: preview.Guard, WorkerCompatible: compatible,
		Losses: contract.AudioLossPreview{ExpectedRevisionId: preview.ExpectedRevisionId, CorrectionCount: preview.CorrectionCount, ValidatedVoiceCount: preview.ValidatedVoiceCount, InvalidatedSteps: preview.InvalidatedSteps, ReplacedTrackIndices: preview.ReplacedTrackIndices, ConfirmationRequired: preview.ConfirmationRequired, PreviewedAt: preview.PreviewedAt},
	}, nil
}

// Start runs in the route's transaction, after it created the run and the plan: it queues the diagnostics, sets
// the steps and records the confirmation the preview covers (the guard the publication will check).
func (e audioEngine) Start(ctx context.Context, tx pgx.Tx, plan audioplans.Plan, _ contract.AudioAnalysisChoice, effects contract.AudioSelectionEffects) error {
	var confirmed bool
	if err := tx.QueryRow(ctx, `SELECT confirmed_replace_corrections FROM clip_analysis_runs WHERE id=$1`, plan.RunId).Scan(&confirmed); err != nil {
		return err
	}
	store := audioplans.New(tx)
	if _, err := store.QueueDiagnostics(ctx, plan.Id); err != nil {
		return err
	}
	var scope string
	if err := tx.QueryRow(ctx, `SELECT scope FROM clip_analysis_runs WHERE id=$1`, plan.RunId).Scan(&scope); err != nil {
		return err
	}
	for _, step := range []string{analysis.StepPreparation, analysis.StepTranscription, analysis.StepIndexing, analysis.StepDiarization, analysis.StepFingerprints, analysis.StepMatching} {
		status, reason := "not_applicable", "audio_plan"
		if step == analysis.StepPreparation || step == analysis.StepTranscription || scope == analysis.ScopeFull || scope == analysis.ScopeVoice && step != analysis.StepIndexing {
			status = "waiting"
		}
		if _, err := tx.Exec(ctx, `INSERT INTO clip_analysis_steps(run_id,step,status,reason,finished_at) VALUES($1,$2,$3,$4,CASE WHEN $3='not_applicable' THEN now() END)
 ON CONFLICT(run_id,step) DO UPDATE SET status=EXCLUDED.status,reason=EXCLUDED.reason,finished_at=EXCLUDED.finished_at,updated_at=now()`, plan.RunId, step, status, reason); err != nil {
			return err
		}
	}
	return store.RecordConfirmation(ctx, audioplans.Confirmation{PlanID: plan.Id, ExpectedRevisionID: effects.Losses.ExpectedRevisionId, Guard: effects.StakeVersion, Confirmed: confirmed && effects.Losses.ConfirmationRequired})
}

// installAudio wires the activated pipeline into a server: the durable orchestrator behind the analysis runs,
// the real engine behind the selection routes and the voice analysis after a publication.
func (s *Server) installAudio() {
	s.Audio = analysis.NewAudio(s.Pool, analysis.WorkerAudioDiagnostics{}, audioPublisher{s})
	s.Audio.OnPublished(s.afterAudioPublication)
	s.Analysis.SetAudio(s.Audio)
	s.AudioSelection = audioEngine{s}
}
