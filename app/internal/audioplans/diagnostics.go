package audioplans

import (
	"context"
	"errors"
	"strconv"

	"github.com/jackc/pgx/v5/pgconn"
)

// DiagnosticInputVersion is the input version of the durable job that examines one plan. It names the plan, so
// two plans never share (and never supersede) each other's diagnostic job; the observation itself still carries
// the plan digest. Replaying the request finds the same bound jobs.
func DiagnosticInputVersion(planID int64, planDigest string) string {
	return "audio-diagnostic-" + strconv.FormatInt(planID, 10) + "-" + planDigest
}

// QueueDiagnostics creates, in ONE statement, a pending audio.diagnostic.<real index> job and its immutable
// plan binding for every selected track that has none yet. Every inventoried selected track is examined, in
// whatever order or under whatever title: nothing is read from metadata. It is idempotent and keeps jobs already
// bound (done, retrying, dead) untouched: a retry is an administrator job control, never a duplicate. A pool is
// enough: the binding cannot lag the job. Reliable completed observations with an identical
// immutable input and clip are copied into a new plan-owned done job. Uncertain or failed
// observations are recomputed; cancellation never affects another plan's jobs.
// It does not run any detector and has no effect on transcripts.
func (s *Store) QueueDiagnostics(ctx context.Context, planID int64) ([]Child, error) {
	var found bool
	var digest string
	if err := s.db.QueryRow(ctx, `SELECT true,input_version FROM audio_plans WHERE id=$1`, planID).Scan(&found, &digest); err != nil {
		return nil, err
	}
	version := DiagnosticInputVersion(planID, digest)
	for attempt := 0; ; attempt++ {
		err := s.queueMissing(ctx, planID, version)
		var pgErr *pgconn.PgError
		// A concurrent identical request won the race: the plan is already bound, read it.
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && attempt < 3 {
			continue
		}
		if err != nil {
			return nil, err
		}
		break
	}
	state, err := s.GetPlan(ctx, planID)
	if err != nil {
		return nil, err
	}
	out := []Child{}
	for _, c := range state.Children {
		if c.Stage == "diagnostic" {
			out = append(out, c)
		}
	}
	return out, nil
}

func (s *Store) queueMissing(ctx context.Context, planID int64, version string) error {
	rows, err := s.db.Query(ctx, `WITH missing AS (
 SELECT p.clip_id,t.track_index,p.input_version AS digest,cached.payload,cached.result
 FROM audio_plans p JOIN audio_plan_tracks t ON t.plan_id=p.id
 LEFT JOIN LATERAL (
   SELECT d.payload,j.result FROM audio_plans previous
   JOIN audio_track_diagnostics d ON d.plan_id=previous.id AND d.track_index=t.track_index
   JOIN audio_plan_children c ON c.plan_id=previous.id AND c.track_index=t.track_index AND c.stage='diagnostic'
   JOIN jobs j ON j.id=c.job_id AND j.status='done' AND j.attempts=d.attempt
   WHERE previous.clip_id=p.clip_id AND previous.id<>p.id AND previous.input_version=p.input_version
   AND d.payload->>'status' IN ('speech','no_speech')
   ORDER BY previous.id DESC LIMIT 1
 ) cached ON true
 WHERE p.id=$1 AND NOT EXISTS(SELECT 1 FROM audio_plan_children c WHERE c.plan_id=p.id AND c.track_index=t.track_index AND c.stage='diagnostic')
 ORDER BY t.track_index), created AS (
 INSERT INTO jobs(clip_id,step,input_version,max_attempts,status,attempts,result)
 SELECT clip_id,'audio.diagnostic.'||track_index,$2,3,
 CASE WHEN payload IS NULL THEN 'pending' ELSE 'done' END,
 CASE WHEN payload IS NULL THEN 0 ELSE 1 END,result
 FROM missing RETURNING id,step), bound AS (
 INSERT INTO audio_plan_children(plan_id,track_index,stage,job_id,input_version)
 SELECT $1,substring(step FROM '[0-9]+$')::int,'diagnostic',id,$2 FROM created RETURNING track_index,job_id), observations AS (
 INSERT INTO audio_track_diagnostics(plan_id,track_index,attempt,payload)
 SELECT $1,b.track_index,1,m.payload || jsonb_build_object('plan_id',$1::bigint,'input_version',m.digest,'job_id',b.job_id,'attempt',1)
 FROM bound b JOIN missing m USING(track_index) WHERE m.payload IS NOT NULL)
 SELECT track_index,job_id FROM bound ORDER BY track_index`, planID, version)
	if err != nil {
		return err
	}
	rows.Close()
	return rows.Err()
}
