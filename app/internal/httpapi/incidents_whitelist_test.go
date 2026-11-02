//go:build integration

package httpapi

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// A raw last_error (a path, a secret, a stack trace) never becomes the error code of a
// notification: only the stable codes of the failure catalogue are kept, anything else is
// the generic worker_failed.
func testIncidentErrorCodeWhitelist(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	h := newVoiceHarnessClips(t, pool, 1)
	clip := h.clipIDs[0]
	for _, q := range []string{`DELETE FROM notifications`, `DELETE FROM jobs`} {
		if _, err := pool.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	malicious := `open /srv/media/secret-clip.mkv: token=hunter2-synthetic`
	for step, lastError := range map[string]string{"leaky.test": malicious, "known.test": "job_dead_known_code_placeholder"} {
		if _, err := pool.Exec(ctx, `INSERT INTO jobs(clip_id,step,input_version,status,last_error) VALUES($1,$2,'v-'||gen_random_uuid()::text,'dead',$3)`, clip, step, lastError); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(ctx, `INSERT INTO jobs(clip_id,step,input_version,status,last_error) VALUES($1,'catalogue.test','v-cat','dead','model_missing')`, clip); err != nil {
		t.Fatal(err)
	}
	h.api.syncIncidents(ctx)
	items, raw := h.incidents(h.adminUser())
	if strings.Contains(raw, "hunter2") || strings.Contains(raw, "/srv/media") || strings.Contains(raw, "job_dead_known_code_placeholder") {
		t.Fatalf("a raw job error reached a notification: %s", raw)
	}
	codes := map[string]string{}
	for _, i := range items {
		codes[i.Step] = i.ErrorCode
	}
	if codes["leaky.test"] != "worker_failed" || codes["known.test"] != "worker_failed" {
		t.Fatalf("unknown errors must become worker_failed: %+v", items)
	}
}
