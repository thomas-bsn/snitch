//go:build integration

package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// A voice result sent after the lease expired is refused with 409 and stores nothing,
// for both the diarization and the fingerprint step. (A result that was superseded by
// a newer diarization is covered by testVoiceDiarization.)
func testVoiceResultWithExpiredLease(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	ctx := context.Background()
	expire := func(id int64) {
		if _, err := pool.Exec(ctx, `UPDATE jobs SET lease_expires_at=now()-interval '1 minute' WHERE id=$1`, id); err != nil {
			t.Fatal(err)
		}
	}
	h.transcribe()
	h.putPipeline(fakePipeline(2))
	h.adminCall(http.MethodPost, fmt.Sprintf("/api/admin/clips/%d/voice", h.clipID), map[string]any{"audio_stream_index": 2}, http.StatusCreated)
	job, ok := h.claim()
	if !ok || job.Step != "voice.diarization.2" {
		t.Fatalf("expected diarization, got %+v", job)
	}
	expire(job.ID)
	h.complete(job, diarizationPayload("SPEAKER_00", "SPEAKER_01"), http.StatusConflict)
	if h.count(`SELECT count(*) FROM local_speakers`)+h.count(`SELECT count(*) FROM speaker_turns`)+h.count(`SELECT count(*) FROM jobs WHERE step='voice.fingerprint.2'`) != 0 || h.count(`SELECT count(*) FROM jobs WHERE id=$1 AND status='done'`, job.ID) != 0 {
		t.Fatal("an expired diarization lease must leave no partial rows")
	}
	// The job is claimable again and a valid lease completes normally.
	if _, err := pool.Exec(ctx, `UPDATE jobs SET status='pending',lease_owner=NULL,lease_expires_at=NULL,lease_token_hash=NULL,next_attempt_at=now() WHERE id=$1`, job.ID); err != nil {
		t.Fatal(err)
	}
	job, ok = h.claim()
	if !ok {
		t.Fatal("the diarization must be claimable again")
	}
	h.complete(job, diarizationPayload("SPEAKER_00", "SPEAKER_01"), http.StatusOK)
	fp, ok := h.claim()
	if !ok || fp.Step != "voice.fingerprint.2" {
		t.Fatalf("expected fingerprints, got %+v", fp)
	}
	expire(fp.ID)
	items := []map[string]any{{"speaker_label": "SPEAKER_00", "scope": "speaker", "start": 0, "end": 1.5, "quality": 0.9, "embedding": []float64{0.5, 0.5, 0.5, 0.5}}}
	h.complete(fp, map[string]any{"provider": "fake", "model": "fixture-diarization", "model_version": "fixture-v1", "embedding_model": "fixture-embedding", "embedding_model_version": "emb-v1", "audio_stream_index": 2, "diarization_job_id": job.ID, "dimensions": 4, "fingerprints": items}, http.StatusConflict)
	if h.count(`SELECT count(*) FROM voice_fingerprints`) != 0 || h.count(`SELECT count(*) FROM jobs WHERE id=$1 AND status='done'`, fp.ID) != 0 {
		t.Fatal("an expired fingerprint lease must leave no partial rows")
	}
}
