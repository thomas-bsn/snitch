//go:build integration

package httpapi

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// This opt-in smoke crosses the real Go/Python HTTP boundary, including leased
// downloads, ffmpeg, pinned Silero inference, completion and atomic publication.
// Only the recognizer is a deterministic fixture; no ASR service is required.
func testAudioPythonWorker(t *testing.T, pool *pgxpool.Pool, duplicate bool) {
	python := os.Getenv("SNITCH_TEST_WORKER_PYTHON")
	model := os.Getenv("SNITCH_TEST_VAD_DIRECTORY")
	if python == "" || model == "" {
		t.Skip("explicit Python environment and installed Silero model required")
	}
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	h := newVoiceHarness(t, pool)
	var media string
	err = pool.QueryRow(context.Background(), `SELECT cs.root_path||'/'||sf.relative_path FROM source_files sf JOIN clip_sources cs ON cs.id=sf.source_id WHERE sf.clip_id=$1`, h.clipID).Scan(&media)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Chmod(media, 0600); err != nil {
		t.Fatal(err)
	}
	firstAudio := []string{"-f", "lavfi", "-i", "sine=frequency=150:sample_rate=16000:duration=8"}
	if duplicate {
		firstAudio = []string{"-i", filepath.Join(root, "worker/tests/fixtures/acoustic/voice.wav")}
	}
	args := []string{"-v", "error", "-y", "-f", "lavfi", "-i", "color=s=16x16:d=8"}
	args = append(args, firstAudio...)
	args = append(args, "-i", filepath.Join(root, "worker/tests/fixtures/acoustic/voice.wav"), "-map", "0:v", "-map", "1:a", "-map", "2:a", "-c:v", "mpeg4", "-c:a", "aac", "-t", "8", media)
	cmd := exec.Command("ffmpeg", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("synthetic media: %v: %s", err, out)
	}
	stat, err := os.Stat(media)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(context.Background(), `UPDATE clips SET size_bytes=$2,duration_seconds=8,tracks=(SELECT jsonb_agg(t || CASE WHEN t->>'type'='audio' THEN jsonb_build_object('start_seconds',0,'duration_seconds',8) ELSE '{}'::jsonb END) FROM jsonb_array_elements(tracks) t) WHERE id=$1;`, h.clipID, stat.Size()); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(context.Background(), `UPDATE source_files SET size_bytes=$2,modified_at=$3 WHERE clip_id=$1`, h.clipID, stat.Size(), stat.ModTime().UTC().Truncate(time.Microsecond)); err != nil {
		t.Fatal(err)
	}
	h.readyAudioWorker()
	h.startMultitrackAnalysis()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	cmd = exec.CommandContext(ctx, python, "-c", pythonAudioSmoke, h.ts.URL, h.token, model, t.TempDir(), fmt.Sprint(duplicate))
	cmd.Dir = filepath.Join(root, "worker")
	cmd.Env = append(os.Environ(), "PYTHONPATH="+strings.Join([]string{filepath.Join(root, "worker"), filepath.Join(root, "worker/snitch_worker/api"), os.Getenv("SNITCH_TEST_ACOUSTIC_SITE")}, string(os.PathListSeparator)))
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Python Worker: %v\n%s", err, out)
	} else {
		t.Logf("Python Worker:\n%s", out)
	}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if run := h.analysis(); run != nil && run.Status == "completed" {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if got := h.visibleTexts(); len(got) == 0 || got[0] != "synthetic worker speech" {
		var jobs string
		_ = pool.QueryRow(context.Background(), `SELECT jsonb_agg(to_jsonb(j))::text FROM jobs j WHERE clip_id=$1`, h.clipID).Scan(&jobs)
		t.Logf("jobs: %s", jobs)
		t.Fatalf("publication from Python: %v", got)
	}
	if n := h.count(`SELECT count(*) FROM jobs WHERE clip_id=$1 AND step='transcription.audio.1'`, h.clipID); !duplicate && n != 0 {
		t.Fatalf("non-speech track transcribed: %d", n)
	}
	if duplicate {
		if n := h.count(`SELECT count(*) FROM jobs WHERE clip_id=$1 AND step LIKE 'audio.dedup.%' AND status='done' AND result::text LIKE '%exact_pcm%'`, h.clipID); n != 1 {
			t.Fatalf("real acoustic deduplication was not completed: %d", n)
		}
		if n := h.count(`SELECT count(*) FROM audio_publications p JOIN transcript_revisions r ON r.id=p.revision_id CROSS JOIN LATERAL jsonb_array_elements(r.manifest->'segments') s WHERE p.clip_id=$1 AND jsonb_array_length(s->'sources')=2`, h.clipID); n == 0 {
			t.Fatal("real PCM duplicates were not grouped with both provenances")
		}
	}
	if h.exactCount("synthetic worker speech") != len(h.visibleTexts()) {
		t.Fatal("Python publication is not searchable")
	}
}

const pythonAudioSmoke = `
import os, sys, time, wave
from snitch_worker import audio_diagnostic, runner, log
from snitch_client.client import AuthenticatedClient
log.configure('DEBUG')
os.environ['SNITCH_VOICE_EMBEDDING_MODEL_PATH'] = os.path.join(sys.argv[4], 'embedding')
audio_diagnostic.install(audio_diagnostic.model_directory(), source=os.path.join(sys.argv[3], audio_diagnostic.MODEL))
class Recognizer:
    model = 'synthetic-smoke'
    def transcribe(self, path, vocabulary):
        with wave.open(path) as wav:
            assert wav.getframerate() == 16000 and wav.getnchannels() == 1
            end = min(0.5, wav.getnframes()/16000)
        return dict(provider='fixture', model=self.model, version=self.model, language='en', segments=[dict(start=0, end=end, text='synthetic worker speech', words=[dict(word=word, start=i*end/3, end=min(end, (i+1)*end/3)) for i, word in enumerate(('synthetic', 'worker', 'speech'))])])
client = AuthenticatedClient(base_url=sys.argv[1], token=sys.argv[2], timeout=30)
def check_http(response):
    if response.status_code >= 400:
        print('HTTP failure', response.status_code, response.request.url.path, flush=True)
client.get_httpx_client().event_hooks['response'].append(check_http)
processed = 0
for attempt in range(20):
    if runner.run_once(client, transcriber=Recognizer()):
        processed += 1
    else:
        if processed >= (5 if sys.argv[5] == 'true' else 3): break
        time.sleep(0.1)
assert processed >= (5 if sys.argv[5] == 'true' else 3), processed
print('real Python Worker completed', processed, 'jobs')
`
