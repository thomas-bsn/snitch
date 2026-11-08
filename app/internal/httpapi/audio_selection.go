package httpapi

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"snitch/internal/analysis"
	"snitch/internal/audioplans"
	"snitch/internal/httpapi/contract"
)

// AudioSelectionEngine is the activation seam for #339/#344. Context supplies current
// versions and diagnostic evidence; Preview supplies the actual execution closure and
// human-work fingerprint (not only counts). Start binds jobs/steps to the persisted
// plan inside the supplied transaction. It must not commit, publish, or use the pool.
// Publication must recheck Effects.Losses and StakeVersion against later human edits.
// Leaving this nil preserves the production single-track workflow.
type AudioSelectionEngine interface {
	Context(context.Context, pgx.Tx, int64) (contract.AudioSelectionContext, error)
	Preview(context.Context, pgx.Tx, int64, audioplans.Input, contract.AudioAnalysisChoice) (contract.AudioSelectionEffects, error)
	Start(context.Context, pgx.Tx, audioplans.Plan, contract.AudioAnalysisChoice, contract.AudioSelectionEffects) error
}

type audioReceipt struct {
	Clip      int64                          `json:"clip"`
	Actor     int64                          `json:"actor"`
	At        time.Time                      `json:"at"`
	Selection contract.AudioSelectionRequest `json:"selection"`
	Effects   contract.AudioSelectionEffects `json:"effects"`
}

func (s *Server) getAudioSelection(w http.ResponseWriter, r *http.Request) {
	clip, ok := s.authorizedClip(w, r)
	if !ok {
		return
	}
	if s.AudioSelection == nil {
		reply(w, 200, contract.AudioSelectionContext{Inventory: []contract.AudioTrackInventory{}, Diagnostics: []contract.AudioSelectionDiagnostic{}})
		return
	}
	tx, err := s.Pool.Begin(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	current, err := s.audioSelectionContext(r.Context(), tx, clip.ID)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, 200, current)
}

func (s *Server) audioSelectionContext(ctx context.Context, tx pgx.Tx, id int64) (contract.AudioSelectionContext, error) {
	current, err := s.AudioSelection.Context(ctx, tx, id)
	if err != nil {
		return current, err
	}
	// The API always verifies actual ffprobe indices, irrespective of the engine's cache.
	var raw []byte
	var latest int64
	err = tx.QueryRow(ctx, `SELECT tracks,COALESCE((SELECT max(id) FROM audio_plans WHERE clip_id=clips.id),0) FROM clips WHERE id=$1`, id).Scan(&raw, &latest)
	if err != nil {
		return current, err
	}
	var tracks []audioTrack
	if err = json.Unmarshal(raw, &tracks); err != nil {
		return current, err
	}
	base := contract.AudioPlanInput{}
	if current.Input != nil {
		base = *current.Input
	}
	current.Input = &base
	current.Input.Inventory = []contract.AudioTrackInventory{}
	current.Input.TrackIndices = []int{}
	current.Input.Decisions = []contract.AudioTrackDecision{}
	current.Input.Mode = "auto"
	current.Input.Inventory = audioInventoryOf(tracks)
	for _, t := range current.Input.Inventory {
		current.Input.TrackIndices = append(current.Input.TrackIndices, t.Index)
	}
	current.Inventory = current.Input.Inventory
	if current.Diagnostics == nil {
		current.Diagnostics = []contract.AudioSelectionDiagnostic{}
	}
	// A clip with a single audio track keeps the historical single-track flow: nothing to choose.
	current.Enabled = len(current.Inventory) > 1
	if !current.Enabled {
		current.Inventory, current.Diagnostics = []contract.AudioTrackInventory{}, []contract.AudioSelectionDiagnostic{}
	}
	current.Version = ""
	b, _ := json.Marshal(struct {
		Context contract.AudioSelectionContext
		Latest  int64
	}{current, latest})
	sum := sha256.Sum256(b)
	current.Version = hex.EncodeToString(sum[:])
	return current, nil
}

func selectionInput(current contract.AudioSelectionContext, request contract.AudioSelectionRequest, actor int64, at time.Time) (audioplans.Input, error) {
	if current.Input == nil {
		return audioplans.Input{}, errors.New("invalid_audio_selection")
	}
	input := *current.Input
	if request.Version != current.Version {
		return input, errors.New("audio_selection_stale")
	}
	if request.Analysis.Scope != "full" && request.Analysis.Scope != "transcription" && request.Analysis.Scope != "voice" {
		return input, errors.New("invalid_audio_selection")
	}
	if request.Analysis.Mode != "missing" && request.Analysis.Mode != "full" && request.Analysis.Mode != "steps" {
		return input, errors.New("invalid_audio_selection")
	}
	if request.Analysis.Mode == "steps" && len(request.Analysis.Steps) == 0 {
		return input, errors.New("invalid_audio_selection")
	}
	for _, step := range request.Analysis.Steps {
		if !slices.Contains([]string{"transcription", "indexing", "diarization"}, step) {
			return input, errors.New("invalid_audio_selection")
		}
	}
	input.Mode = contract.AudioPlanInputMode(request.Mode)
	if request.Mode == "manual" {
		input.TrackIndices = append([]int{}, request.TrackIndices...)
	} else if request.Mode != "auto" || len(request.TrackIndices) > 0 {
		return input, errors.New("invalid_audio_selection")
	}
	for _, d := range request.Decisions {
		if d.Action == "exclude" && request.Mode == "auto" {
			input.TrackIndices = slices.DeleteFunc(input.TrackIndices, func(i int) bool { return i == d.TrackIndex })
		}
		input.Decisions = append(input.Decisions, contract.AudioTrackDecision{TrackIndex: d.TrackIndex, Action: contract.AudioTrackDecisionAction(d.Action), Forced: d.Forced, Reason: d.Reason, ActorId: actor, DecidedAt: at})
	}
	for _, d := range current.Diagnostics {
		if d.Freshness != "current" {
			continue
		}
		decision, found := contract.AudioTrackDecision{}, false
		for _, v := range input.Decisions {
			if v.TrackIndex == d.TrackIndex {
				decision = v
				found = true
			}
		}
		if d.Status == "ambiguous" && !found {
			return input, errors.New("audio_decision_required")
		}
		if request.Mode == "manual" && slices.Contains(input.TrackIndices, d.TrackIndex) && d.Status == "no_speech" && (!found || !decision.Forced) {
			return input, errors.New("audio_force_required")
		}
	}
	canonical, err := audioplans.CanonicalInput(input)
	if err != nil {
		return input, errors.New("invalid_audio_selection")
	}
	return canonical.Input, nil
}

func (s *Server) previewAudioSelection(w http.ResponseWriter, r *http.Request) {
	s.mutateAudioSelection(w, r, false)
}
func (s *Server) startAudioSelection(w http.ResponseWriter, r *http.Request) {
	s.mutateAudioSelection(w, r, true)
}
func (s *Server) mutateAudioSelection(w http.ResponseWriter, r *http.Request, start bool) {
	clip, ok := s.authorizedClip(w, r)
	if !ok {
		return
	}
	if s.AudioSelection == nil {
		fail(w, 503, "audio_selection_disabled")
		return
	}
	actor := r.Context().Value(sessionKey{}).(session).User.ID
	at := time.Now().UTC().Truncate(time.Microsecond)
	var request contract.AudioSelectionRequest
	var previous audioReceipt
	confirmed := false
	if start {
		var body contract.AudioSelectionStart
		if !decode(w, r, &body) {
			return
		}
		request = body.Selection
		confirmed = body.ConfirmLosses
		if !s.readAudioReceipt(body.PreviewToken, &previous) || previous.Clip != clip.ID || previous.Actor != actor || at.Sub(previous.At) > 15*time.Minute || previous.At.After(at) {
			fail(w, 409, "audio_preview_expired")
			return
		}
		a, _ := json.Marshal(previous.Selection)
		b, _ := json.Marshal(request)
		if !hmac.Equal(a, b) {
			fail(w, 409, "audio_preview_changed")
			return
		}
		at = previous.At
	} else if !decode(w, r, &request) {
		return
	}
	tx, err := s.Pool.Begin(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	// Serialize starts, then read inventory, evidence and losses again after taking the lock.
	if _, err = tx.Exec(r.Context(), `SELECT id FROM clips WHERE id=$1 FOR UPDATE`, clip.ID); err != nil {
		s.internal(w, err)
		return
	}
	current, err := s.audioSelectionContext(r.Context(), tx, clip.ID)
	if err != nil {
		s.internal(w, err)
		return
	}
	if !current.Enabled {
		fail(w, 503, "audio_selection_disabled")
		return
	}
	input, err := selectionInput(current, request, actor, at)
	if err != nil {
		status := 400
		if err.Error() == "audio_selection_stale" {
			status = 409
		}
		fail(w, status, err.Error())
		return
	}
	effects, err := s.AudioSelection.Preview(r.Context(), tx, clip.ID, input, request.Analysis)
	if err != nil {
		if errors.Is(err, analysis.ErrTranscriptionRequired) {
			fail(w, 409, "transcription_required")
			return
		}
		s.internal(w, err)
		return
	}
	// This timestamp is part of the publication receipt, not a general consent.
	effects.Losses.PreviewedAt = at
	effects.Losses.ConfirmationRequired = effects.Losses.CorrectionCount > 0 || effects.Losses.ValidatedVoiceCount > 0 || effects.Losses.ConfirmationRequired
	receipt := audioReceipt{clip.ID, actor, at, request, effects}
	if !start {
		reply(w, 200, contract.AudioSelectionPreview{Input: input, Effects: effects, PreviewToken: s.signAudioReceipt(receipt), ExpiresAt: at.Add(15 * time.Minute)})
		return
	}
	before, _ := json.Marshal(previous.Effects)
	after, _ := json.Marshal(effects)
	if !hmac.Equal(before, after) {
		fail(w, 409, "audio_preview_changed")
		return
	}
	if effects.Losses.ConfirmationRequired && !confirmed {
		fail(w, 409, "audio_confirmation_required")
		return
	}
	var reuse int64
	if engine, ok := s.AudioSelection.(interface {
		ReusablePlan(context.Context, pgx.Tx, int64, audioplans.Input, contract.AudioAnalysisChoice) (int64, error)
	}); ok {
		reuse, err = engine.ReusablePlan(r.Context(), tx, clip.ID, input, request.Analysis)
		if err != nil {
			s.internal(w, err)
			return
		}
	}
	resolved, err := analysis.Resolve(string(request.Analysis.Scope), analysis.Options{Mode: string(request.Analysis.Mode), Steps: request.Analysis.Steps}, analysis.Existing{Transcription: reuse != 0, Voices: true})
	if err != nil {
		fail(w, 400, "invalid_audio_selection")
		return
	}
	var runID int64
	err = tx.QueryRow(r.Context(), `INSERT INTO clip_analysis_runs(clip_id,scope,started_by,mode,forced_steps,confirmed_replace_corrections,confirmed_replace_validations,confirmed_stake_at) VALUES($1,$2,$3,$4,$5,$6,$6,$7) RETURNING id`, clip.ID, resolved.Scope, actor, request.Analysis.Mode, resolved.Forced, confirmed, at).Scan(&runID)
	var pgerr *pgconn.PgError
	if errors.As(err, &pgerr) && pgerr.Code == "23505" {
		fail(w, 409, "analysis_already_active")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	var state contract.AudioPlanState
	if reuse != 0 {
		for _, step := range []string{analysis.StepPreparation, analysis.StepTranscription, analysis.StepIndexing, analysis.StepDiarization, analysis.StepFingerprints, analysis.StepMatching} {
			status, reason := "waiting", "awaiting_previous_step"
			if (step == analysis.StepIndexing && resolved.Scope != analysis.ScopeFull) || ((step == analysis.StepDiarization || step == analysis.StepFingerprints || step == analysis.StepMatching) && resolved.Scope == analysis.ScopeTranscription) {
				status, reason = "not_applicable", "outside_scope"
			}
			_, err = tx.Exec(r.Context(), `INSERT INTO clip_analysis_steps(run_id,step,status,reason,finished_at) VALUES($1,$2,$3,$4,CASE WHEN $3='not_applicable' THEN now() END)`, runID, step, status, reason)
			if err != nil {
				s.internal(w, err)
				return
			}
		}
		state, err = audioplans.New(tx).GetPlan(r.Context(), reuse)
	} else {
		var plan audioplans.Plan
		plan, err = audioplans.New(tx).CreatePlan(r.Context(), runID, input)
		if err == nil {
			err = s.AudioSelection.Start(r.Context(), tx, plan, request.Analysis, effects)
		}
		if err == nil {
			state, err = audioplans.New(tx).GetPlan(r.Context(), plan.Id)
		}
	}
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	if reuse != 0 && slices.Contains(resolved.Forced, analysis.StepIndexing) {
		if err := (analysisHooks{s}).Reindex(r.Context(), clip.ID); err != nil {
			s.internal(w, err)
			return
		}
	}
	// The run exists now: let its orchestrator queue the diagnostics' follow-up without waiting for the ticker.
	s.reconcileAnalysis(r.Context(), clip.ID)
	reply(w, 200, contract.AudioSelectionStarted{State: state, WorkerCompatible: effects.WorkerCompatible})
}
func (s *Server) signAudioReceipt(receipt audioReceipt) string {
	raw, _ := json.Marshal(receipt)
	mac := hmac.New(sha256.New, s.Config.MasterKey)
	mac.Write([]byte("audio-selection-v1:"))
	mac.Write(raw)
	return base64.RawURLEncoding.EncodeToString(raw) + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
func (s *Server) readAudioReceipt(token string, out *audioReceipt) bool {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return false
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil || json.Unmarshal(raw, out) != nil {
		return false
	}
	return hmac.Equal([]byte(s.signAudioReceipt(*out)), []byte(token))
}
