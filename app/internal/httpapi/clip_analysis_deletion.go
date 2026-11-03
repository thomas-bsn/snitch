package httpapi

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"snitch/internal/analysis"
)

// The steps an administrator can delete (ADR 0030), in dependency order. Deleting one also
// deletes the steps that depend on it: see analysisDeletionFollows.
var deletableAnalysisSteps = []string{"transcription", "diarization", "fingerprints", "matching"}

// analysisDeletionFollows lists, for each deleted step, the run steps it empties. The
// indexing follows the transcription; the fingerprints and the matching follow the diarization.
var analysisDeletionFollows = map[string][]string{
	"transcription": {analysis.StepTranscription, analysis.StepIndexing},
	"diarization":   {analysis.StepDiarization, analysis.StepFingerprints, analysis.StepMatching},
	"fingerprints":  {analysis.StepFingerprints, analysis.StepMatching},
	"matching":      {analysis.StepMatching},
}

var (
	errAnalysisActive               = errors.New("an analysis is active on the clip")
	errAnalysisDeletionConfirmation = errors.New("the analysis deletion needs its losses confirmed")
)

// analysisDeletionStake is what a deletion would lose: the human work and the transcript. It
// is computed before anything is deleted, and nothing is deleted without its confirmation.
type analysisDeletionStake struct {
	Segments        int                 `json:"segments"`
	Corrections     *correctionsAtStake `json:"corrections"`
	Voices          int                 `json:"voices"`
	ValidatedVoices int                 `json:"validated_voices"`
	VoiceSamples    int                 `json:"voice_samples"`
	Persons         int                 `json:"persons"`
}

func (st analysisDeletionStake) empty() bool {
	return st.Segments == 0 && st.Corrections == nil && st.Voices == 0 && st.ValidatedVoices == 0 && st.VoiceSamples == 0 && st.Persons == 0
}

// resolveDeletionSteps turns the request into the steps to delete, in dependency order.
func resolveDeletionSteps(all bool, requested []string) ([]string, bool) {
	if all {
		return slices.Clone(deletableAnalysisSteps), true
	}
	if len(requested) == 0 {
		return nil, false
	}
	out := []string{}
	for _, step := range deletableAnalysisSteps {
		if slices.Contains(requested, step) {
			out = append(out, step)
		}
	}
	if len(out) != len(requested) {
		return nil, false
	}
	return out, true
}

// analysisDeletionLoss computes the stake of the steps on the clip, reading through q.
func (s *Server) analysisDeletionLoss(ctx context.Context, q queryer, clipID int64, steps []string) (analysisDeletionStake, error) {
	var stake analysisDeletionStake
	if slices.Contains(steps, "transcription") {
		if err := q.QueryRow(ctx, `SELECT COALESCE(sum(CASE WHEN jsonb_typeof(result->'segments')='array' THEN jsonb_array_length(result->'segments') ELSE 0 END),0)::int FROM jobs WHERE clip_id=$1 AND step LIKE 'transcription.audio.%' AND status='done'`, clipID).Scan(&stake.Segments); err != nil {
			return stake, err
		}
		corrections, found, err := s.clipCorrections(ctx, clipID)
		if err != nil {
			return stake, err
		}
		if found {
			stake.Corrections = &corrections
		}
	}
	if slices.Contains(steps, "diarization") {
		if err := q.QueryRow(ctx, `SELECT
 (SELECT count(*) FROM local_speakers WHERE clip_id=$1),
 (SELECT count(DISTINCT a.local_speaker_id) FROM speaker_attributions a JOIN local_speakers ls ON ls.id=a.local_speaker_id WHERE ls.clip_id=$1 AND a.validation_state='validated'),
 (SELECT count(*) FROM voice_samples WHERE clip_id=$1 AND retired_at IS NULL),
 (SELECT count(DISTINCT p.person_id) FROM (
   SELECT a.person_id FROM speaker_attributions a JOIN local_speakers ls ON ls.id=a.local_speaker_id WHERE ls.clip_id=$1 AND a.validation_state='validated'
   UNION SELECT person_id FROM voice_samples WHERE clip_id=$1 AND retired_at IS NULL) p)`, clipID).Scan(&stake.Voices, &stake.ValidatedVoices, &stake.VoiceSamples, &stake.Persons); err != nil {
			return stake, err
		}
	}
	return stake, nil
}

// deleteAnalysisSteps deletes the steps of a clip in one transaction, after the stake was
// confirmed. The clip is locked, so no run starts meanwhile; an active run or an active job
// of the analysis refuses the deletion. Only this transaction may delete a validated voice.
func (s *Server) deleteAnalysisSteps(ctx context.Context, clipID int64, steps []string, all bool, userID int64, username string, confirm bool) (analysisDeletionStake, error) {
	tx, err := s.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return analysisDeletionStake{}, err
	}
	defer tx.Rollback(ctx)
	var locked int64
	if err = tx.QueryRow(ctx, `SELECT id FROM clips WHERE id=$1 FOR UPDATE`, clipID).Scan(&locked); errors.Is(err, pgx.ErrNoRows) {
		return analysisDeletionStake{}, analysis.ErrClipNotFound
	} else if err != nil {
		return analysisDeletionStake{}, err
	}
	var busy bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM clip_analysis_runs WHERE clip_id=$1 AND status='active')
 OR EXISTS(SELECT 1 FROM jobs WHERE clip_id=$1 AND status IN ('pending','leased','retry_wait') AND (step LIKE 'transcription.audio.%' OR step LIKE 'voice.%'))`, clipID).Scan(&busy); err != nil {
		return analysisDeletionStake{}, err
	}
	if busy {
		return analysisDeletionStake{}, errAnalysisActive
	}
	stake, err := s.analysisDeletionLoss(ctx, tx, clipID, steps)
	if err != nil {
		return analysisDeletionStake{}, err
	}
	if !stake.empty() && !confirm {
		return stake, errAnalysisDeletionConfirmation
	}
	// Only this transaction lifts the guard of validated voices; the setting ends with it.
	if _, err = tx.Exec(ctx, `SELECT set_config('snitch.analysis_deletion','on',true)`); err != nil {
		return analysisDeletionStake{}, err
	}
	has := func(step string) bool { return slices.Contains(steps, step) }
	if has("transcription") {
		if _, err = tx.Exec(ctx, `DELETE FROM jobs WHERE clip_id=$1 AND step LIKE 'transcription.audio.%'`, clipID); err != nil {
			return stake, err
		}
	}
	if has("diarization") {
		if _, err = tx.Exec(ctx, `DELETE FROM local_speakers WHERE clip_id=$1`, clipID); err != nil {
			return stake, err
		}
		if _, err = tx.Exec(ctx, `DELETE FROM jobs WHERE clip_id=$1 AND (step LIKE 'voice.diarization.%' OR step LIKE 'voice.fingerprint.%')`, clipID); err != nil {
			return stake, err
		}
	} else if has("fingerprints") {
		if _, err = tx.Exec(ctx, `DELETE FROM jobs WHERE clip_id=$1 AND step LIKE 'voice.fingerprint.%'`, clipID); err != nil {
			return stake, err
		}
	}
	if has("diarization") || has("fingerprints") || has("matching") {
		// The automatic matching goes; validated attributions are human work and stay unless their voice goes.
		if _, err = tx.Exec(ctx, `DELETE FROM speaker_attributions WHERE validation_state='unvalidated' AND local_speaker_id IN (SELECT id FROM local_speakers WHERE clip_id=$1)`, clipID); err != nil {
			return stake, err
		}
	}
	if all {
		for _, query := range []string{
			`DELETE FROM voice_diarization_trials WHERE clip_id=$1`,
			`DELETE FROM notifications WHERE clip_id=$1`,
			`DELETE FROM clip_analysis_runs WHERE clip_id=$1`,
		} {
			if _, err = tx.Exec(ctx, query, clipID); err != nil {
				return stake, err
			}
		}
	} else {
		// The latest run keeps its record of the deleted steps, which now wait for a new run.
		affected := []string{}
		for _, step := range steps {
			for _, run := range analysisDeletionFollows[step] {
				if !slices.Contains(affected, run) {
					affected = append(affected, run)
				}
			}
		}
		if _, err = tx.Exec(ctx, `UPDATE clip_analysis_steps SET status='waiting',reason='analysis_deleted',job_id=NULL,attempt=0,started_at=NULL,finished_at=NULL,updated_at=now()
 WHERE run_id=(SELECT id FROM clip_analysis_runs WHERE clip_id=$1 ORDER BY id DESC LIMIT 1) AND step=ANY($2) AND status<>'not_applicable'`, clipID, affected); err != nil {
			return stake, err
		}
	}
	corrections := 0
	if stake.Corrections != nil {
		corrections = stake.Corrections.TextSegments + stake.Corrections.SpeakerSegments
	}
	if _, err = tx.Exec(ctx, `INSERT INTO clip_analysis_deletions(clip_id,steps,deleted_by,deleted_by_username,lost_segments,lost_corrections,lost_voices,lost_validated_voices,lost_persons) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		clipID, steps, userID, username, stake.Segments, corrections, stake.Voices, stake.ValidatedVoices, stake.Persons); err != nil {
		return stake, err
	}
	return stake, tx.Commit(ctx)
}

// deleteClipAnalysis deletes the chosen steps of a clip, or the whole analysis. Reserved to
// the super-administrator: the request is refused with 409 analysis_deletion_confirmation_required
// (and the stake) until the losses are confirmed with confirm_loss.
func (s *Server) deleteClipAnalysis(w http.ResponseWriter, r *http.Request) {
	clipID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || clipID < 1 {
		fail(w, http.StatusBadRequest, "invalid_clip_id")
		return
	}
	var input struct {
		Steps       []string `json:"steps"`
		All         bool     `json:"all"`
		ConfirmLoss bool     `json:"confirm_loss"`
	}
	if !decode(w, r, &input) {
		return
	}
	ss := r.Context().Value(sessionKey{}).(session)
	if !isSuperAdministrator(ss.User.Role) {
		fail(w, http.StatusForbidden, "forbidden")
		return
	}
	steps, ok := resolveDeletionSteps(input.All, input.Steps)
	if !ok {
		fail(w, http.StatusBadRequest, "invalid_analysis_step")
		return
	}
	stake, err := s.deleteAnalysisSteps(r.Context(), clipID, steps, input.All, ss.User.ID, ss.User.Username, input.ConfirmLoss)
	switch {
	case errors.Is(err, analysis.ErrClipNotFound):
		fail(w, http.StatusNotFound, "clip_not_found")
	case errors.Is(err, errAnalysisActive):
		fail(w, http.StatusConflict, "analysis_active")
	case errors.Is(err, errAnalysisDeletionConfirmation):
		reply(w, http.StatusConflict, map[string]any{"error": "analysis_deletion_confirmation_required", "stake": stake})
	case err != nil:
		s.internal(w, err)
	default:
		run, latestErr := s.Analysis.Latest(r.Context(), clipID)
		if latestErr != nil {
			s.internal(w, latestErr)
			return
		}
		reply(w, http.StatusOK, map[string]any{"run": run, "deleted_steps": steps, "stake": stake})
	}
}

// getClipAnalysisDeletionPreview is the stake of a deletion, read before the administrator
// confirms it. Query: steps (comma-separated) or all=true.
func (s *Server) getClipAnalysisDeletionPreview(w http.ResponseWriter, r *http.Request) {
	clipID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || clipID < 1 {
		fail(w, http.StatusBadRequest, "invalid_clip_id")
		return
	}
	if !s.clipExists(w, r, clipID) {
		return
	}
	var requested []string
	for _, step := range strings.Split(r.URL.Query().Get("steps"), ",") {
		if step != "" {
			requested = append(requested, step)
		}
	}
	steps, ok := resolveDeletionSteps(r.URL.Query().Get("all") == "true", requested)
	if !ok {
		fail(w, http.StatusBadRequest, "invalid_analysis_step")
		return
	}
	stake, err := s.analysisDeletionLoss(r.Context(), s.Pool, clipID, steps)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{"steps": steps, "stake": stake})
}
