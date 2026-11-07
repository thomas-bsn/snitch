package audioplans

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"snitch/internal/httpapi/contract"
)

var (
	ErrIncomplete           = errors.New("audio plan is incomplete")
	ErrPublicationChanged   = errors.New("publication preview is stale")
	ErrConfirmationRequired = errors.New("publication replaces human work")
)

// PublicationPreview binds a loss preview to the exact correction history seen.
// Obtain a fresh preview when ErrPublicationChanged is returned. Confirmation must
// come from the administrator; the orchestration must persist this preview as-is.
type PublicationPreview struct {
	contract.AudioLossPreview
	PlanID int64  `json:"plan_id"`
	Guard  string `json:"guard"`
}

// PreviewPublication is independent of run mode: changing tracks in missing mode
// replaces the same human work as a forced retranscription.
func (s *Store) PreviewPublication(ctx context.Context, planID int64) (PublicationPreview, error) {
	var out PublicationPreview
	out.PlanID = planID
	out.InvalidatedSteps = []string{"transcription", "indexing", "diarization", "fingerprints", "matching"}
	out.ReplacedTrackIndices = []int{}
	var clip int64
	if err := s.db.QueryRow(ctx, `SELECT clip_id FROM audio_plans WHERE id=$1`, planID).Scan(&clip); err != nil {
		return out, err
	}
	// A single statement sees one snapshot, including restorations (event identity).
	var raw []byte
	err := s.db.QueryRow(ctx, `SELECT COALESCE((SELECT id FROM audio_visible_revisions WHERE clip_id=$1),''),clock_timestamp(),
 (SELECT count(*) FROM speaker_attributions a JOIN local_speakers ls ON ls.id=a.local_speaker_id WHERE ls.clip_id=$1 AND a.validation_state='validated'),
 (SELECT count(*) FROM segment_corrections WHERE clip_id=$1 AND stale_at IS NULL AND (text IS NOT NULL OR speaker_override_id IS NOT NULL)),
 COALESCE((SELECT jsonb_agg(DISTINCT stream ORDER BY stream) FROM audio_visible_segments WHERE clip_id=$1),'[]'::jsonb),
 jsonb_build_object('voices',COALESCE((SELECT jsonb_agg(jsonb_build_array(a.id,a.revision,a.validation_state) ORDER BY a.id) FROM speaker_attributions a JOIN local_speakers ls ON ls.id=a.local_speaker_id WHERE ls.clip_id=$1),'[]'::jsonb),'events',COALESCE((SELECT max(id) FROM segment_correction_events WHERE clip_id=$1),0),
 'states',COALESCE((SELECT jsonb_agg(jsonb_build_array(transcription_job_id,segment_index,version,stale_at) ORDER BY transcription_job_id,segment_index) FROM segment_corrections WHERE clip_id=$1),'[]'::jsonb))`, clip).Scan(&out.ExpectedRevisionId, &out.PreviewedAt, &out.ValidatedVoiceCount, &out.CorrectionCount, &raw, &out.Guard)
	if err != nil {
		return out, err
	}
	if err = json.Unmarshal(raw, &out.ReplacedTrackIndices); err != nil {
		return out, err
	}
	digest := sha256.Sum256([]byte(out.Guard))
	out.Guard = hex.EncodeToString(digest[:])
	out.ConfirmationRequired = out.CorrectionCount > 0 || out.ValidatedVoiceCount > 0
	return out, nil
}

// PublishRevision atomically validates a staged manifest and switches all readers.
// A pool starts a transaction; an existing pgx.Tx uses a savepoint. Clip locking
// serializes with correction routes and legacy job completion. No child publishes.
func (s *Store) PublishRevision(ctx context.Context, id string, preview PublicationPreview, confirmed bool) error {
	beginner, ok := s.db.(interface {
		Begin(context.Context) (pgx.Tx, error)
	})
	if !ok {
		return errors.New("publication requires transactional database")
	}
	tx, err := beginner.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	repo := New(tx)
	r, err := repo.GetRevision(ctx, id)
	if err != nil {
		return err
	}
	if r.PlanId == nil || *r.PlanId != preview.PlanID {
		return ErrConflict
	}
	var clip int64
	if err = tx.QueryRow(ctx, `SELECT id FROM clips WHERE id=$1 FOR UPDATE`, r.ClipId).Scan(&clip); err != nil {
		return err
	}
	var current string
	err = tx.QueryRow(ctx, `SELECT revision_id FROM audio_publications WHERE clip_id=$1`, clip).Scan(&current)
	if err == nil && current == id {
		return tx.Commit(ctx)
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	latest, err := repo.PreviewPublication(ctx, *r.PlanId)
	if err != nil {
		return err
	}
	if latest.ExpectedRevisionId != preview.ExpectedRevisionId || latest.Guard != preview.Guard {
		return ErrPublicationChanged
	}
	if latest.ConfirmationRequired && !confirmed {
		return ErrConfirmationRequired
	}
	state, err := repo.GetPlan(ctx, *r.PlanId)
	if err != nil {
		return err
	}
	var active bool
	if err = tx.QueryRow(ctx, `SELECT status='active' AND NOT EXISTS(SELECT 1 FROM audio_plans newer WHERE newer.run_id=clip_analysis_runs.id AND newer.id>$2) FROM clip_analysis_runs WHERE id=$1 FOR UPDATE`, state.Plan.RunId, state.Plan.Id).Scan(&active); err != nil {
		return err
	}
	if !active {
		return ErrIncomplete
	}
	included := map[int]int64{}
	for _, diagnostic := range state.Diagnostics {
		decision := ""
		forced := false
		for _, d := range state.Plan.Input.Decisions {
			if d.TrackIndex == diagnostic.TrackIndex {
				decision = string(d.Action)
				forced = d.Forced
			}
		}
		if decision == "exclude" {
			continue
		}
		if !forced {
			var valid bool
			if diagnostic.JobId == nil {
				return ErrIncomplete
			}
			if err = tx.QueryRow(ctx, `SELECT status='done' AND attempts=$2 FROM jobs WHERE id=$1`, *diagnostic.JobId, diagnostic.Attempt).Scan(&valid); err != nil || !valid {
				return ErrIncomplete
			}

			if diagnostic.Status == "no_speech" && decision != "include" {
				continue
			}
			if diagnostic.Status != "speech" {
				return ErrIncomplete
			}
		}
		found := false
		for _, c := range state.Children {
			if c.TrackIndex == diagnostic.TrackIndex && c.Stage == "transcription" && c.Status == "done" {
				included[c.TrackIndex] = c.JobId
				found = true
			}
		}
		if !found {
			return ErrIncomplete
		}
	}
	// Every expected source segment must be represented exactly once, including
	// sources retained under an acoustically deduplicated segment. No partial/empty
	// manifest can masquerade as a complete nonempty result.
	seen := map[[2]int64]bool{}
	for _, segment := range r.Segments {
		for _, source := range segment.Sources {
			if included[source.TrackIndex] != source.JobId {
				return ErrIncomplete
			}
			key := [2]int64{source.JobId, int64(source.SegmentIndex)}
			if seen[key] {
				return ErrConflict
			}
			seen[key] = true
		}
	}
	for _, job := range included {
		var count int
		if err = tx.QueryRow(ctx, `SELECT jsonb_array_length(result->'segments') FROM jobs WHERE id=$1 AND status='done'`, job).Scan(&count); err != nil {
			return ErrIncomplete
		}
		// Empty ASR is not an explicit no-speech diagnosis.
		if count == 0 {
			return ErrIncomplete
		}
		for i := 0; i < count; i++ {
			if !seen[[2]int64{job, int64(i)}] {
				return ErrIncomplete
			}
		}
	}
	if len(included) == 0 {
		if !r.Empty || len(state.Diagnostics) == 0 {
			return ErrIncomplete
		}
		for _, d := range state.Diagnostics {
			if d.Status != "no_speech" {
				return ErrIncomplete
			}
		}
	}
	if _, err = tx.Exec(ctx, `UPDATE segment_corrections SET stale_at=clock_timestamp(),reindex_pending=false WHERE clip_id=$1 AND stale_at IS NULL`, clip); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO audio_publications(clip_id,revision_id) VALUES($1,$2) ON CONFLICT(clip_id) DO UPDATE SET revision_id=EXCLUDED.revision_id,published_at=clock_timestamp()`, clip, id); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `SELECT id FROM embedding_index_state WHERE id=1 FOR SHARE`); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO embedding_tasks(profile_id,transcription_job_id)
 SELECT p.id,s.job_id FROM embedding_index_state st JOIN embedding_profiles p ON p.id IN(st.active_profile_id,st.candidate_profile_id)
 CROSS JOIN (SELECT DISTINCT job_id FROM audio_visible_segments WHERE clip_id=$1) s WHERE st.id=1 AND p.state<>'failed' ON CONFLICT DO NOTHING`, clip); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
