package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// maxReindexAttempts bounds the attempts to reindex one correction; after that the
// segment stays pending and is reported to administrators until a backfill resets it.
const maxReindexAttempts = 5

const reindexLease = 2 * time.Minute

// reindexNext reindexes one corrected segment: it embeds only the effective text of
// that segment and replaces its vector (same profile/job/segment key, never a second
// row) in the active index and in the candidate index of a running rebuild. The
// transcription is not redone and the other segments are not touched. The correction
// is recorded before and independently of the embeddings provider: while it is
// unavailable the segment stays reindex_pending and is retried with a growing delay,
// at most maxReindexAttempts times. Claims use a lease like embedding_tasks, so a
// consumer that died is replaced when its lease expires.
func (s *Server) reindexNext(ctx context.Context) error {
	settings, err := s.embeddingSettings(ctx)
	if err != nil {
		return err
	}
	if settings.URL == "" {
		return nil
	}
	var jobID, clipID int64
	var index, version int
	var corrected *string
	err = s.Pool.QueryRow(ctx, `WITH c AS (SELECT transcription_job_id,segment_index FROM segment_corrections
 WHERE reindex_pending AND stale_at IS NULL AND reindex_attempts<$1 AND (reindex_lease_expires_at IS NULL OR reindex_lease_expires_at<now())
 ORDER BY reindex_requested_at,transcription_job_id,segment_index FOR UPDATE SKIP LOCKED LIMIT 1)
 UPDATE segment_corrections sc SET reindex_lease_expires_at=now()+make_interval(secs=>$2),reindex_attempts=sc.reindex_attempts+1
 FROM c WHERE sc.transcription_job_id=c.transcription_job_id AND sc.segment_index=c.segment_index
 RETURNING sc.transcription_job_id,sc.segment_index,sc.clip_id,sc.text_version,sc.text`, maxReindexAttempts, reindexLease.Seconds()).Scan(&jobID, &index, &clipID, &version, &corrected)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}
	done := func() error {
		_, err := s.Pool.Exec(ctx, `UPDATE segment_corrections SET reindex_pending=false,reindex_error='',reindex_lease_expires_at=NULL WHERE transcription_job_id=$1 AND segment_index=$2 AND text_version=$3`, jobID, index, version)
		return err
	}
	// A correction of a transcription that is no longer the visible one is obsolete.
	var raw []byte
	err = s.Pool.QueryRow(ctx, `SELECT seg FROM audio_visible_segments WHERE clip_id=$1 AND job_id=$2 AND idx=$3`, clipID, jobID, index).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return done()
	}
	if err != nil {
		return s.reindexFailed(ctx, jobID, index, version, err)
	}
	var segment indexSegment
	if err = json.Unmarshal(raw, &segment); err != nil {
		return s.reindexFailed(ctx, jobID, index, version, err)
	}
	text := segment.Text
	if corrected != nil {
		text = *corrected
	}
	text = strings.TrimSpace(text)
	profiles, err := s.Pool.Query(ctx, `SELECT p.id,p.model FROM embedding_index_state st JOIN embedding_profiles p ON p.id IN (st.active_profile_id,st.candidate_profile_id) WHERE st.id=1 AND p.state<>'failed' ORDER BY p.id`)
	if err != nil {
		return s.reindexFailed(ctx, jobID, index, version, err)
	}
	type target struct {
		id     int64
		model  string
		vector string
		dims   int
	}
	var targets []target
	for profiles.Next() {
		var t target
		if err = profiles.Scan(&t.id, &t.model); err != nil {
			profiles.Close()
			return s.reindexFailed(ctx, jobID, index, version, err)
		}
		targets = append(targets, t)
	}
	profiles.Close()
	if err = profiles.Err(); err != nil {
		return s.reindexFailed(ctx, jobID, index, version, err)
	}
	if len(targets) == 0 {
		return s.reindexFailed(ctx, jobID, index, version, fmt.Errorf("no embedding profile"))
	}
	if text != "" {
		for i := range targets {
			if targets[i].vector, targets[i].dims, err = s.embedOne(ctx, targets[i].model, text); err != nil {
				return s.reindexFailed(ctx, jobID, index, version, err)
			}
		}
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return s.reindexFailed(ctx, jobID, index, version, err)
	}
	defer tx.Rollback(ctx)
	for _, t := range targets {
		if text == "" {
			// An empty text is not indexed: drop the stale vector.
			if _, err = tx.Exec(ctx, `DELETE FROM transcript_embeddings WHERE profile_id=$1 AND transcription_job_id=$2 AND segment_index=$3`, t.id, jobID, index); err != nil {
				return s.reindexFailed(ctx, jobID, index, version, err)
			}
			continue
		}
		var dimensions int
		if err = tx.QueryRow(ctx, `SELECT COALESCE(dimensions,0) FROM embedding_profiles WHERE id=$1 FOR UPDATE`, t.id).Scan(&dimensions); err != nil {
			return s.reindexFailed(ctx, jobID, index, version, err)
		}
		if t.dims > 4000 || (dimensions > 0 && dimensions != t.dims) {
			return s.reindexFailed(ctx, jobID, index, version, fmt.Errorf("invalid embedding dimension"))
		}
		if _, err = tx.Exec(ctx, `UPDATE embedding_profiles SET dimensions=$2,updated_at=now() WHERE id=$1 AND dimensions IS NULL`, t.id, t.dims); err != nil {
			return s.reindexFailed(ctx, jobID, index, version, err)
		}
		if _, err = tx.Exec(ctx, `INSERT INTO transcript_embeddings(profile_id,transcription_job_id,clip_id,segment_index,segment_start,segment_end,text,embedding) VALUES($1,$2,$3,$4,$5,$6,$7,$8::vector)
 ON CONFLICT(profile_id,transcription_job_id,segment_index) DO UPDATE SET segment_start=EXCLUDED.segment_start,segment_end=EXCLUDED.segment_end,text=EXCLUDED.text,embedding=EXCLUDED.embedding`, t.id, jobID, clipID, index, segment.Start, segment.End, text, t.vector); err != nil {
			return s.reindexFailed(ctx, jobID, index, version, err)
		}
	}
	// A text correction made meanwhile (higher text version) keeps the segment pending.
	if _, err = tx.Exec(ctx, `UPDATE segment_corrections SET reindex_pending=false,reindex_error='',reindex_lease_expires_at=NULL WHERE transcription_job_id=$1 AND segment_index=$2 AND text_version=$3`, jobID, index, version); err != nil {
		return s.reindexFailed(ctx, jobID, index, version, err)
	}
	return tx.Commit(ctx)
}

// reindexFailed records the failure and delays the next attempt (the lease column
// doubles as "not before"): 30 s, 60 s, 90 s... A newer correction resets the delay.
func (s *Server) reindexFailed(ctx context.Context, jobID int64, index, version int, cause error) error {
	_, err := s.Pool.Exec(ctx, `UPDATE segment_corrections SET reindex_error=left($4,240),reindex_lease_expires_at=now()+make_interval(secs=>30*reindex_attempts) WHERE transcription_job_id=$1 AND segment_index=$2 AND text_version=$3 AND reindex_pending`, jobID, index, version, cause.Error())
	if err != nil {
		return err
	}
	return cause
}
