package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"snitch/internal/jobs"
)

const (
	maxVoiceResultBytes = 4 << 20
	maxVoiceSpeakers    = 32
	maxVoiceTurns       = 2000
	maxVoiceFingerprint = 400
	maxVoiceDimensions  = 1024
)

type voiceTurn struct {
	Start *float64 `json:"start"`
	End   *float64 `json:"end"`
}

type diarizationResult struct {
	Provider         string `json:"provider"`
	Model            string `json:"model"`
	ModelVersion     string `json:"model_version"`
	AudioStreamIndex *int   `json:"audio_stream_index"`
	Speakers         *[]struct {
		Label string       `json:"label"`
		Turns *[]voiceTurn `json:"turns"`
	} `json:"speakers"`
}

type fingerprintResult struct {
	EmbeddingModel        string `json:"embedding_model"`
	EmbeddingModelVersion string `json:"embedding_model_version"`
	AudioStreamIndex      *int   `json:"audio_stream_index"`
	DiarizationJobID      *int64 `json:"diarization_job_id"`
	Dimensions            int    `json:"dimensions"`
	Fingerprints          *[]struct {
		SpeakerLabel string    `json:"speaker_label"`
		Scope        string    `json:"scope"`
		TurnIndex    *int      `json:"turn_index"`
		Start        *float64  `json:"start"`
		End          *float64  `json:"end"`
		Quality      *float64  `json:"quality"`
		Embedding    []float64 `json:"embedding"`
	} `json:"fingerprints"`
}

func validVoiceTurn(t voiceTurn) bool {
	return t.Start != nil && t.End != nil && validTime(*t.Start) && validTime(*t.End) && *t.End >= *t.Start
}

func parseDiarization(raw json.RawMessage, config voicePipeline, index int) (diarizationResult, bool) {
	var result diarizationResult
	if json.Unmarshal(raw, &result) != nil || result.Provider != config.Provider || result.Model != config.Model || result.ModelVersion != config.ModelVersion || result.AudioStreamIndex == nil || *result.AudioStreamIndex != index || result.Speakers == nil || len(*result.Speakers) > maxVoiceSpeakers {
		return result, false
	}
	seen := map[string]bool{}
	turns := 0
	for _, speaker := range *result.Speakers {
		if speaker.Label == "" || len(speaker.Label) > 64 || seen[speaker.Label] || speaker.Turns == nil {
			return result, false
		}
		seen[speaker.Label] = true
		turns += len(*speaker.Turns)
		for _, t := range *speaker.Turns {
			if !validVoiceTurn(t) {
				return result, false
			}
		}
	}
	return result, turns <= maxVoiceTurns
}

func parseFingerprints(raw json.RawMessage, index int, diarizationJobID int64) (fingerprintResult, bool) {
	var result fingerprintResult
	if json.Unmarshal(raw, &result) != nil || result.EmbeddingModel == "" || len(result.EmbeddingModel) > 100 || result.EmbeddingModelVersion == "" || len(result.EmbeddingModelVersion) > 64 || result.AudioStreamIndex == nil || *result.AudioStreamIndex != index || result.DiarizationJobID == nil || *result.DiarizationJobID != diarizationJobID || result.Dimensions < 1 || result.Dimensions > maxVoiceDimensions || result.Fingerprints == nil || len(*result.Fingerprints) > maxVoiceFingerprint {
		return result, false
	}
	for _, fp := range *result.Fingerprints {
		if fp.SpeakerLabel == "" || (fp.Scope != "speaker" && fp.Scope != "turn") || fp.Start == nil || fp.End == nil || !validTime(*fp.Start) || !validTime(*fp.End) || *fp.End < *fp.Start || fp.Quality == nil || *fp.Quality < 0 || *fp.Quality > 1 || len(fp.Embedding) != result.Dimensions || (fp.TurnIndex != nil && *fp.TurnIndex < 0) {
			return result, false
		}
		for _, v := range fp.Embedding {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				return result, false
			}
		}
	}
	return result, true
}

// fingerprintSummary is what jobs.result keeps of a fingerprint result: never a
// vector, which would survive the deletion of a voice profile.
func fingerprintSummary(r fingerprintResult) json.RawMessage {
	out, _ := json.Marshal(map[string]any{"embedding_model": r.EmbeddingModel, "embedding_model_version": r.EmbeddingModelVersion, "audio_stream_index": r.AudioStreamIndex, "diarization_job_id": r.DiarizationJobID, "dimensions": r.Dimensions, "fingerprint_count": len(*r.Fingerprints)})
	return out
}

// completeVoiceJob validates and records a voice result. Validation happens only
// while the lease is held; the projection into tables runs inside the same
// transaction as the job completion, so a rejected or stale result stores nothing.
func (s *Server) completeVoiceJob(w http.ResponseWriter, r *http.Request, id int64, step, status, inputVersion string, result json.RawMessage) {
	var project func(ctx context.Context, tx pgx.Tx, job jobs.Job) error
	var summarize func(json.RawMessage) json.RawMessage
	if status != "done" {
		leased, err := s.Jobs.Input(r.Context(), id, currentWorker(r).ID, workerLeaseToken(r))
		if errors.Is(err, jobs.ErrLeaseNotOwned) {
			fail(w, http.StatusConflict, "lease_not_owned")
			return
		}
		if err != nil {
			s.internal(w, err)
			return
		}
		pipelineVersion, diarizationID, ok := jobs.VoiceInputVersion(leased.Job.InputVersion)
		prefix := "voice.diarization."
		if strings.HasPrefix(step, "voice.fingerprint.") {
			prefix = "voice.fingerprint."
		}
		index, indexErr := strconv.Atoi(strings.TrimPrefix(step, prefix))
		if !ok || indexErr != nil || (prefix == "voice.fingerprint." && diarizationID == 0) {
			fail(w, http.StatusBadRequest, "invalid_voice_result")
			return
		}
		config, err := s.voiceConfig(r.Context(), pipelineVersion)
		if err != nil {
			s.internal(w, err)
			return
		}
		if prefix == "voice.diarization." {
			parsed, valid := parseDiarization(result, config, index)
			if !valid {
				fail(w, http.StatusBadRequest, "invalid_voice_result")
				return
			}
			project = func(ctx context.Context, tx pgx.Tx, job jobs.Job) error {
				return projectDiarization(ctx, tx, job, parsed, pipelineVersion, index)
			}
		} else {
			parsed, valid := parseFingerprints(result, index, diarizationID)
			if !valid {
				fail(w, http.StatusBadRequest, "invalid_voice_result")
				return
			}
			summarize = func(json.RawMessage) json.RawMessage { return fingerprintSummary(parsed) }
			project = func(ctx context.Context, tx pgx.Tx, job jobs.Job) error {
				return projectFingerprints(ctx, tx, job, parsed, config, index, diarizationID)
			}
		}
	}
	job, err := s.Jobs.CompleteWith(r.Context(), id, currentWorker(r).ID, workerLeaseToken(r), inputVersion, result, jobs.CompleteOptions{MaxResultBytes: maxVoiceResultBytes, Summarize: summarize, Project: project})
	switch {
	case errors.Is(err, jobs.ErrInvalidInput):
		fail(w, http.StatusBadRequest, "invalid_voice_result")
	case errors.Is(err, errDiarizationLocked):
		fail(w, http.StatusConflict, "diarization_locked")
	case errors.Is(err, jobs.ErrStaleResult):
		fail(w, http.StatusConflict, "stale_voice_result")
	case errors.Is(err, jobs.ErrLeaseNotOwned):
		fail(w, http.StatusConflict, "lease_not_owned")
	case err != nil:
		s.internal(w, err)
	default:
		if strings.HasPrefix(step, "voice.fingerprint.") {
			// New fingerprints are matched against the existing references, in the
			// database only. A failure never undoes the stored result.
			s.matchingChanged(r.Context(), job.ClipID)
		}
		s.reconcileAnalysis(r.Context(), job.ClipID)
		reply(w, http.StatusOK, job)
	}
}

// errDiarizationLocked refuses a diarization result for a designated microphone track.
var errDiarizationLocked = errors.New("diarization locked by human validation")

// projectDiarization replaces the clip's local speakers for the audio stream and
// queues the fingerprint step against this exact diarization.
func projectDiarization(ctx context.Context, tx pgx.Tx, job jobs.Job, result diarizationResult, pipelineVersion string, index int) error {
	if _, err := tx.Exec(ctx, `SELECT id FROM clips WHERE id=$1 FOR UPDATE`, job.ClipID); err != nil {
		return fmt.Errorf("lock clip for diarization: %w", err)
	}
	// Validations take the clip row FOR SHARE, so none can race with this replacement.
	// A designated microphone track is never diarized (microphone track).
	var microphone bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM local_speakers WHERE clip_id=$1 AND audio_stream_index=$2 AND kind='microphone')`, job.ClipID, index).Scan(&microphone); err != nil {
		return fmt.Errorf("check microphone track: %w", err)
	}
	if microphone {
		return errDiarizationLocked
	}
	// Speaker corrections point to the replaced speakers: they become obsolete (history kept).
	if err := dropSpeakerOverrides(ctx, tx, `ls.clip_id=$1 AND ls.audio_stream_index=$2 AND ls.kind='diarized'`, job.ClipID, index); err != nil {
		return fmt.Errorf("record dropped speaker corrections: %w", err)
	}
	// The speakers a human validated are archived, never deleted: the validated
	// Attribution, the Voice samples and the speaker fingerprints stay linked to the audio.
	// An archived speaker keeps no turn and no machine attribution.
	if _, err := tx.Exec(ctx, `WITH archived AS (
 UPDATE local_speakers ls SET kind='archived',archived_at=now()
 WHERE ls.clip_id=$1 AND ls.audio_stream_index=$2 AND ls.kind='diarized'
   AND EXISTS (SELECT 1 FROM speaker_attributions a WHERE a.local_speaker_id=ls.id AND a.validation_state='validated')
 RETURNING ls.id), cleaned AS (
 DELETE FROM speaker_attributions a USING archived WHERE a.local_speaker_id=archived.id AND a.validation_state='unvalidated' RETURNING a.id)
DELETE FROM speaker_turns st USING archived WHERE st.local_speaker_id=archived.id`, job.ClipID, index); err != nil {
		return fmt.Errorf("archive validated speakers: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM local_speakers WHERE clip_id=$1 AND audio_stream_index=$2 AND kind='diarized'`, job.ClipID, index); err != nil {
		return fmt.Errorf("replace local speakers: %w", err)
	}
	for ordinal, speaker := range *result.Speakers {
		seconds := 0.0
		for _, t := range *speaker.Turns {
			seconds += *t.End - *t.Start
		}
		var speakerID int64
		if err := tx.QueryRow(ctx, `INSERT INTO local_speakers(clip_id,audio_stream_index,diarization_job_id,label,ordinal,speaking_seconds,turn_count) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id`, job.ClipID, index, job.ID, speaker.Label, ordinal, seconds, len(*speaker.Turns)).Scan(&speakerID); err != nil {
			return fmt.Errorf("insert local speaker: %w", err)
		}
		for turnIndex, t := range *speaker.Turns {
			if _, err := tx.Exec(ctx, `INSERT INTO speaker_turns(local_speaker_id,clip_id,turn_index,start_seconds,end_seconds) VALUES($1,$2,$3,$4,$5)`, speakerID, job.ClipID, turnIndex, *t.Start, *t.End); err != nil {
				return fmt.Errorf("insert speaker turn: %w", err)
			}
		}
	}
	step := fmt.Sprintf("voice.fingerprint.%d", index)
	version := fmt.Sprintf("f5-a%d-p%s-d%d", index, pipelineVersion, job.ID)
	if _, err := tx.Exec(ctx, `UPDATE jobs SET status='failed',lease_owner=NULL,lease_expires_at=NULL,lease_token_hash=NULL,last_error='input_version_superseded',updated_at=now() WHERE clip_id=$1 AND step=$2 AND input_version<>$3 AND status IN ('pending','leased','retry_wait')`, job.ClipID, step, version); err != nil {
		return fmt.Errorf("supersede fingerprint step: %w", err)
	}
	if len(*result.Speakers) == 0 {
		return nil
	}
	if _, err := tx.Exec(ctx, `INSERT INTO jobs(clip_id,step,input_version,priority,max_attempts) VALUES($1,$2,$3,0,3) ON CONFLICT (clip_id,step,input_version) WHERE status IN ('pending','leased','retry_wait') DO NOTHING`, job.ClipID, step, version); err != nil {
		return fmt.Errorf("queue fingerprint step: %w", err)
	}
	return nil
}

// projectFingerprints replaces the fingerprints of the current diarization. A
// result computed against a replaced diarization is stale and stores nothing.
func projectFingerprints(ctx context.Context, tx pgx.Tx, job jobs.Job, result fingerprintResult, config voicePipeline, index int, diarizationID int64) error {
	if _, err := tx.Exec(ctx, `SELECT id FROM clips WHERE id=$1 FOR UPDATE`, job.ClipID); err != nil {
		return fmt.Errorf("lock clip for fingerprints: %w", err)
	}
	var current int64
	err := tx.QueryRow(ctx, `SELECT id FROM jobs WHERE clip_id=$1 AND step=$2 AND status='done' ORDER BY updated_at DESC,id DESC LIMIT 1`, job.ClipID, fmt.Sprintf("voice.diarization.%d", index)).Scan(&current)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && current != diarizationID) {
		return jobs.ErrStaleResult
	}
	if err != nil {
		return fmt.Errorf("read current diarization: %w", err)
	}
	speakers := map[string]int64{}
	rows, err := tx.Query(ctx, `SELECT id,label FROM local_speakers WHERE diarization_job_id=$1`, diarizationID)
	if err != nil {
		return fmt.Errorf("read diarization speakers: %w", err)
	}
	for rows.Next() {
		var id int64
		var label string
		if err = rows.Scan(&id, &label); err != nil {
			rows.Close()
			return err
		}
		speakers[label] = id
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM voice_fingerprints WHERE clip_id=$1 AND local_speaker_id IN (SELECT id FROM local_speakers WHERE diarization_job_id=$2)`, job.ClipID, diarizationID); err != nil {
		return fmt.Errorf("replace fingerprints: %w", err)
	}
	for _, fp := range *result.Fingerprints {
		speakerID, known := speakers[fp.SpeakerLabel]
		if !known {
			return jobs.ErrInvalidInput
		}
		var turnID *int64
		if fp.TurnIndex != nil {
			var id int64
			if err = tx.QueryRow(ctx, `SELECT id FROM speaker_turns WHERE local_speaker_id=$1 AND turn_index=$2`, speakerID, *fp.TurnIndex).Scan(&id); errors.Is(err, pgx.ErrNoRows) {
				return jobs.ErrInvalidInput
			} else if err != nil {
				return fmt.Errorf("find fingerprint turn: %w", err)
			}
			turnID = &id
		}
		duration := *fp.End - *fp.Start
		reason := ""
		if duration < config.MinFingerprintSeconds {
			reason = "too_short"
		} else if *fp.Quality < config.MinQuality {
			reason = "low_quality"
		}
		values := make([]string, len(fp.Embedding))
		for i, v := range fp.Embedding {
			values[i] = strconv.FormatFloat(float64(float32(v)), 'g', -1, 32)
		}
		vector := "[" + strings.Join(values, ",") + "]"
		if _, err = tx.Exec(ctx, `INSERT INTO voice_fingerprints(local_speaker_id,clip_id,fingerprint_job_id,scope,turn_id,start_seconds,end_seconds,duration_seconds,quality,usable,unusable_reason,model,model_version,dimensions,embedding) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15::vector)`, speakerID, job.ClipID, job.ID, fp.Scope, turnID, *fp.Start, *fp.End, duration, *fp.Quality, reason == "", reason, result.EmbeddingModel, result.EmbeddingModelVersion, result.Dimensions, vector); err != nil {
			return fmt.Errorf("insert fingerprint: %w", err)
		}
	}
	return nil
}
