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
	// Refusals and Dispositions count what the deleted voices carried: the refused Persons and
	// the ignored or random voices. Deleting the voices drops them (ADR 0031).
	Refusals     int `json:"refusals"`
	Dispositions int `json:"dispositions"`
}

func (st analysisDeletionStake) empty() bool {
	return st.Segments == 0 && st.Corrections == nil && st.Voices == 0 && st.ValidatedVoices == 0 && st.VoiceSamples == 0 && st.Persons == 0 &&
		st.Refusals == 0 && st.Dispositions == 0
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

// analysisActive reports whether an analysis run or one of its jobs (transcription, voice) is
// still active on the clip. Reading it under the clip lock is what makes the check hold.
func analysisActive(ctx context.Context, q queryer, clipID int64) (bool, error) {
	var busy bool
	err := q.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM clip_analysis_runs WHERE clip_id=$1 AND status='active')
 OR EXISTS(SELECT 1 FROM jobs WHERE clip_id=$1 AND status IN ('pending','leased','retry_wait') AND (step LIKE 'transcription.audio.%' OR step LIKE 'voice.%'))`, clipID).Scan(&busy)
	return busy, err
}

// analysisDeletionLoss computes the stake of the steps on the clip, reading through q.
// Only the speakers produced by the clip's diarization are at stake; microphone and archived
// speakers survive a diarization deletion, unless the whole analysis goes (all).
func (s *Server) analysisDeletionLoss(ctx context.Context, q queryer, clipID int64, steps []string, all bool) (analysisDeletionStake, error) {
	var stake analysisDeletionStake
	if slices.Contains(steps, "transcription") {
		if err := q.QueryRow(ctx, `SELECT COALESCE(sum(CASE WHEN jsonb_typeof(result->'segments')='array' THEN jsonb_array_length(result->'segments') ELSE 0 END),0)::int FROM jobs WHERE clip_id=$1 AND step LIKE 'transcription.audio.%' AND status='done'`, clipID).Scan(&stake.Segments); err != nil {
			return stake, err
		}
		corrections, found, err := s.clipCorrections(ctx, q, clipID)
		if err != nil {
			return stake, err
		}
		if found {
			stake.Corrections = &corrections
		}
	}
	if slices.Contains(steps, "diarization") {
		if err := q.QueryRow(ctx, `SELECT
 (SELECT count(*) FROM local_speakers WHERE clip_id=$1 AND (kind='diarized' OR $2)),
 (SELECT count(DISTINCT a.local_speaker_id) FROM speaker_attributions a JOIN local_speakers ls ON ls.id=a.local_speaker_id WHERE ls.clip_id=$1 AND (ls.kind='diarized' OR $2) AND a.validation_state='validated'),
 (SELECT count(*) FROM voice_samples vs JOIN local_speakers ls ON ls.id=vs.local_speaker_id WHERE vs.clip_id=$1 AND vs.retired_at IS NULL AND (ls.kind='diarized' OR $2)),
 (SELECT count(DISTINCT p.person_id) FROM (
   SELECT a.person_id FROM speaker_attributions a JOIN local_speakers ls ON ls.id=a.local_speaker_id WHERE ls.clip_id=$1 AND (ls.kind='diarized' OR $2) AND a.validation_state='validated'
   UNION SELECT vs.person_id FROM voice_samples vs JOIN local_speakers ls ON ls.id=vs.local_speaker_id WHERE vs.clip_id=$1 AND vs.retired_at IS NULL AND (ls.kind='diarized' OR $2)) p),
 (SELECT count(*) FROM voice_refusals vr JOIN local_speakers ls ON ls.id=vr.local_speaker_id WHERE ls.clip_id=$1 AND (ls.kind='diarized' OR $2)),
 (SELECT count(*) FROM local_speaker_dispositions dsp JOIN local_speakers ls ON ls.id=dsp.local_speaker_id WHERE ls.clip_id=$1 AND (ls.kind='diarized' OR $2))`, clipID, all).Scan(&stake.Voices, &stake.ValidatedVoices, &stake.VoiceSamples, &stake.Persons, &stake.Refusals, &stake.Dispositions); err != nil {
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
	if busy, err := analysisActive(ctx, tx, clipID); err != nil {
		return analysisDeletionStake{}, err
	} else if busy {
		return analysisDeletionStake{}, errAnalysisActive
	}
	stake, err := s.analysisDeletionLoss(ctx, tx, clipID, steps, all)
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
	// Deleting the voices or the vectors of validated samples changes the References they belonged to.
	var referencesChange bool
	if has("diarization") || has("fingerprints") {
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM voice_samples WHERE clip_id=$1 AND retired_at IS NULL)`, clipID).Scan(&referencesChange); err != nil {
			return stake, err
		}
	}
	if has("transcription") {
		if _, err = tx.Exec(ctx, `DELETE FROM jobs WHERE clip_id=$1 AND step LIKE 'transcription.audio.%'`, clipID); err != nil {
			return stake, err
		}
	}
	// The speakers of this clip's diarization go; microphone and archived speakers stay, and so
	// do the jobs they still point to (a deleted job would cascade to its speakers).
	if has("diarization") {
		if _, err = tx.Exec(ctx, `DELETE FROM local_speakers WHERE clip_id=$1 AND (kind='diarized' OR $2)`, clipID, all); err != nil {
			return stake, err
		}
		if _, err = tx.Exec(ctx, `DELETE FROM jobs WHERE clip_id=$1 AND step LIKE 'voice.diarization.%' AND id NOT IN (SELECT diarization_job_id FROM local_speakers WHERE diarization_job_id IS NOT NULL)`, clipID); err != nil {
			return stake, err
		}
	}
	if has("fingerprints") {
		// The vectors of the clip's diarized speakers go; those of archived speakers stay.
		if _, err = tx.Exec(ctx, `DELETE FROM voice_fingerprints WHERE clip_id=$1 AND local_speaker_id IN (SELECT id FROM local_speakers WHERE clip_id=$1 AND (kind='diarized' OR $2))`, clipID, all); err != nil {
			return stake, err
		}
	}
	if has("diarization") || has("fingerprints") {
		if _, err = tx.Exec(ctx, `DELETE FROM jobs WHERE clip_id=$1 AND step LIKE 'voice.fingerprint.%' AND id NOT IN (SELECT fingerprint_job_id FROM voice_fingerprints)`, clipID); err != nil {
			return stake, err
		}
	}
	if has("diarization") || has("fingerprints") || has("matching") {
		// The automatic matching goes; validated attributions are human work and stay unless their voice goes.
		if _, err = tx.Exec(ctx, `DELETE FROM speaker_attributions WHERE validation_state='unvalidated' AND local_speaker_id IN (SELECT id FROM local_speakers WHERE clip_id=$1 AND (kind='diarized' OR $2))`, clipID, all); err != nil {
			return stake, err
		}
	}
	if all {
		for _, query := range []string{
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
	if _, err = tx.Exec(ctx, `INSERT INTO clip_analysis_deletions(clip_id,steps,deleted_by,deleted_by_username,lost_segments,lost_corrections,lost_voices,lost_validated_voices,lost_persons,lost_refusals,lost_dispositions) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		clipID, steps, userID, username, stake.Segments, corrections, stake.Voices, stake.ValidatedVoices, stake.Persons, stake.Refusals, stake.Dispositions); err != nil {
		return stake, err
	}
	if err = tx.Commit(ctx); err != nil {
		return stake, err
	}
	if referencesChange {
		s.matchingChanged(ctx, 0)
	}
	return stake, nil
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
	stake, err := s.analysisDeletionLoss(r.Context(), s.Pool, clipID, steps, r.URL.Query().Get("all") == "true")
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{"steps": steps, "stake": stake})
}
