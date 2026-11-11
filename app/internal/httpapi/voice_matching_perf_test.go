//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Dimensions of the background matching measurement (#392): the order of the library aimed at.
const (
	perfVoiceClips          = 3000
	perfVoiceSpeakersByClip = 10 // 30 000 local speakers
	perfVoicePersons        = 200
	perfVoiceSamples        = 10 // validated samples per Person: 2 000 references
	perfVoiceDimensions     = 256
)

// seedVoiceLibrary writes a synthetic voice library straight into the database: clips, one diarization and one
// fingerprint job each, local speakers with one comparable and contributive speaker fingerprint, and Persons
// with validated samples. Four speakers out of five belong to a Person (her center plus noise), the others are
// unknown voices. Never real data.
func seedVoiceLibrary(t *testing.T, pool *pgxpool.Pool, adminID int64) {
	t.Helper()
	ctx := context.Background()
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO clips(owner_id,original_name,size_bytes,modified_at,recorded_at) SELECT $1,'perf-voice-'||g||'.mp4',12,now(),now() FROM generate_series(1,$2) g`, adminID, perfVoiceClips)
	exec(`INSERT INTO jobs(clip_id,step,input_version,status) SELECT id,s,'perf-voice','done' FROM clips CROSS JOIN (VALUES ('voice.diarization.1'),('voice.fingerprint.1')) v(s) WHERE original_name LIKE 'perf-voice-%'`)
	exec(`INSERT INTO local_speakers(clip_id,audio_stream_index,diarization_job_id,label,ordinal,speaking_seconds,turn_count)
 SELECT j.clip_id,1,j.id,'SPEAKER_0'||o,o,5,3 FROM jobs j CROSS JOIN generate_series(0,$1-1) o WHERE j.step='voice.diarization.1' AND j.input_version='perf-voice'`, perfVoiceSpeakersByClip)
	exec(`INSERT INTO persons(display_name,created_by) SELECT 'Perf person '||p,$1 FROM generate_series(1,$2) p`, adminID, perfVoicePersons)
	// Centers in [-0.5,0.5]^d; a speaker is its Person's center plus uniform noise of width 0.6.
	exec(`CREATE TEMP TABLE perf_centers AS SELECT p.id AS person,row_number() OVER (ORDER BY p.id)-1 AS k,(SELECT array_agg(random()-0.5) FROM generate_series(1,$1) i WHERE p.id>0) AS c FROM persons p WHERE p.display_name LIKE 'Perf person %'`, perfVoiceDimensions)
	exec(`CREATE TEMP TABLE perf_speakers AS SELECT x.id,x.clip_id,pc.person
 FROM (SELECT ls.id,ls.clip_id,row_number() OVER (ORDER BY ls.id) AS n FROM local_speakers ls JOIN clips c ON c.id=ls.clip_id WHERE c.original_name LIKE 'perf-voice-%') x
 LEFT JOIN perf_centers pc ON x.n % 5 <> 0 AND pc.k = (x.n - x.n/5) % $1`, perfVoicePersons)
	exec(`INSERT INTO voice_fingerprints(local_speaker_id,clip_id,fingerprint_job_id,scope,start_seconds,end_seconds,duration_seconds,quality,usable,unusable_reason,contributive,model,model_version,dimensions,embedding,net_speech_seconds)
 SELECT s.id,s.clip_id,fj.id,'speaker',0,5,5,0.9,true,'',true,'perf-embedding','perf-v1',$1,
  (SELECT array_agg(COALESCE(pc.c[i],0)+0.6*(random()-0.5) ORDER BY i) FROM generate_series(1,$1) i WHERE s.id>0)::vector,5
 FROM perf_speakers s JOIN jobs fj ON fj.clip_id=s.clip_id AND fj.step='voice.fingerprint.1' AND fj.input_version='perf-voice' LEFT JOIN perf_centers pc ON pc.person=s.person`, perfVoiceDimensions)
	// The first samples of each Person are validated by a human: they make her Reference.
	exec(`CREATE TEMP TABLE perf_validated AS SELECT id,clip_id,person FROM (SELECT s.*,row_number() OVER (PARTITION BY s.person ORDER BY s.id) AS n FROM perf_speakers s WHERE s.person IS NOT NULL) x WHERE n<=$1`, perfVoiceSamples)
	exec(`INSERT INTO speaker_attributions(local_speaker_id,person_id,decision_level,validation_state,origin,author_id,author_username) SELECT id,person,'human','validated','human',$1,'perf-admin' FROM perf_validated`, adminID)
	exec(`INSERT INTO voice_samples(person_id,local_speaker_id,clip_id,attribution_id,created_by) SELECT v.person,v.id,v.clip_id,a.id,$1 FROM perf_validated v JOIN speaker_attributions a ON a.local_speaker_id=v.id`, adminID)
	exec(`ANALYZE`)
}

// testVoiceMatchingPerformance measures a full background matching pass over a synthetic library of 30 000 local
// speakers and 200 Persons. It is a measurement, not a regression test: it runs only with SNITCH_PERF=1 and logs
// its figures (docs/performance-rapprochement.md).
func testVoiceMatchingPerformance(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if os.Getenv("SNITCH_PERF") != "1" {
		t.Skip("measurement only: set SNITCH_PERF=1")
	}
	ctx := context.Background()
	phases := &phaseLog{}
	h := newVoiceHarnessWith(t, pool, 1, slog.New(slog.NewJSONHandler(phases, &slog.HandlerOptions{Level: slog.LevelDebug})), true)
	h.waitMatching()
	var adminID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM users WHERE username='administrator'`).Scan(&adminID); err != nil {
		t.Fatal(err)
	}
	seeded := time.Now()
	seedVoiceLibrary(t, pool, adminID)
	var speakers, persons, samples int
	if err := pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM voice_fingerprints WHERE model='perf-embedding'),(SELECT count(DISTINCT person_id) FROM voice_samples),(SELECT count(*) FROM voice_samples)`).Scan(&speakers, &persons, &samples); err != nil {
		t.Fatal(err)
	}
	t.Logf("seeded %d speaker fingerprints (%d dims), %d Persons, %d validated samples in %s; %s/%s, %d CPU", speakers, perfVoiceDimensions, persons, samples, time.Since(seeded).Round(time.Millisecond), runtime.GOOS, runtime.GOARCH, runtime.NumCPU())
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM voice_samples; DELETE FROM speaker_attributions; DELETE FROM voice_fingerprints; DELETE FROM local_speakers; DELETE FROM jobs; DELETE FROM persons; DELETE FROM clips WHERE original_name LIKE 'perf-voice-%'`)
	}()

	pass := func(label string) {
		t.Helper()
		started := time.Now()
		stats, err := h.api.rematchNow(ctx)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%s: full pass %s; %+v; phases %s", label, time.Since(started).Round(time.Millisecond), stats, phases.last())
	}
	waitPass := func(label string) {
		t.Helper()
		deadline := time.Now().Add(10 * time.Minute)
		for {
			v := h.matchingStatus()
			if !v.Pending && !v.Running {
				t.Logf("%s: background pass %d ms for %d request(s); phases %s", label, *v.LastDurationMS, *v.LastRequests, phases.last())
				return
			}
			if time.Now().After(deadline) {
				t.Fatalf("%s: matching still in progress", label)
			}
			time.Sleep(20 * time.Millisecond)
		}
	}

	pass("cumulative, first pass (every suggestion written)")
	pass("cumulative, second pass (nothing changes)")

	// A validation answers while a full pass runs; the pass it asks for follows in the background.
	var target fixtureSpeaker
	var person int64
	if err := pool.QueryRow(ctx, `SELECT f.local_speaker_id,f.clip_id,(SELECT min(person_id) FROM voice_samples) FROM voice_fingerprints f WHERE f.model='perf-embedding' AND NOT EXISTS (SELECT 1 FROM speaker_attributions a WHERE a.local_speaker_id=f.local_speaker_id AND a.validation_state='validated') ORDER BY f.local_speaker_id LIMIT 1`).Scan(&target.ID, &target.ClipID, &person); err != nil {
		t.Fatal(err)
	}
	h.api.matchingChanged(ctx, 0)
	time.Sleep(200 * time.Millisecond) // the requested pass is under way
	asked := time.Now()
	h.adminCall(http.MethodPost, fmt.Sprintf("/api/local-speakers/%d/validation", target.ID), map[string]any{"person_id": person}, http.StatusCreated)
	t.Logf("validation answered in %s while a full pass ran", time.Since(asked).Round(time.Millisecond))
	waitPass("cumulative, after the validation")

	h.scoreMode(http.MethodPut, map[string]any{"mode": "best_sample"}, http.StatusOK)
	waitPass("best_sample, first pass")
	pass("best_sample, second pass (nothing changes)")
}

// phaseLog keeps the phase timings the matching logs at debug level (voice_matching_phases).
type phaseLog struct {
	mu    sync.Mutex
	lines []string
}

func (p *phaseLog) Write(b []byte) (int, error) {
	if bytes.Contains(b, []byte(`"voice_matching_phases"`)) {
		var line struct {
			Targets int              `json:"targets"`
			Phases  map[string]int64 `json:"phases"`
		}
		if json.Unmarshal(b, &line) == nil {
			p.mu.Lock()
			p.lines = append(p.lines, fmt.Sprintf("targets %d, elapsed at end of: scoring %d ms, writing %d ms, commit %d ms, index %d ms, group links %d ms, reviews %d ms", line.Targets, line.Phases["scored_ms"], line.Phases["written_ms"], line.Phases["committed_ms"], line.Phases["indexed_ms"], line.Phases["linked_ms"], line.Phases["reviewed_ms"]))
			p.mu.Unlock()
		}
	}
	return len(b), nil
}

func (p *phaseLog) last() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.lines) == 0 {
		return "(none)"
	}
	return strings.TrimSpace(p.lines[len(p.lines)-1])
}
