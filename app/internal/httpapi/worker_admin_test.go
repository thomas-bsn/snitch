//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"snitch/internal/config"
	"snitch/internal/setup"
)

// testWorkerAdministration covers what the administration screens need from the Worker API:
// the library's transcription profile fixed by the first Worker, the deployment files without a
// mandatory model name, the Worker detail (typed recent tasks), token regeneration and revocation.
func testWorkerAdministration(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `UPDATE transcription_pipeline_active SET version='pipeline-v1' WHERE id=true`); err != nil {
		t.Fatal(err)
	}
	code, err := setup.Issue(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	server := New(pool, config.Config{MasterKey: bytes.Repeat([]byte{1}, 32)}, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	ts := httptest.NewTLSServer(server.Handler())
	defer ts.Close()
	jar, _ := cookiejar.New(nil)
	admin := ts.Client()
	admin.Jar = jar
	anon := &http.Client{Transport: ts.Client().Transport}
	call := func(client *http.Client, method, path string, body any, csrf, token string, want int) []byte {
		t.Helper()
		var payload []byte
		if body != nil {
			payload = mustJSON(t, body)
		}
		req, _ := http.NewRequest(method, ts.URL+path, bytes.NewReader(payload))
		req.Header.Set("Origin", ts.URL)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if csrf != "" {
			req.Header.Set("X-CSRF-Token", csrf)
		}
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		resp, requestErr := client.Do(req)
		if requestErr != nil {
			t.Fatal(requestErr)
		}
		defer resp.Body.Close()
		data, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != want {
			t.Fatalf("%s %s: got %d want %d: %s", method, path, resp.StatusCode, want, data)
		}
		return data
	}
	errorCode := func(data []byte) string {
		var e struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(data, &e)
		return e.Error
	}
	call(admin, http.MethodPost, "/api/setup", map[string]string{"code": code, "username": "administrator", "password": "synthetic-password"}, "", "", http.StatusCreated)
	var me struct {
		CSRF string `json:"csrf_token"`
	}
	if err = json.Unmarshal(call(admin, http.MethodGet, "/api/auth/me", nil, "", "", http.StatusOK), &me); err != nil {
		t.Fatal(err)
	}

	type page struct {
		FirstWorker   bool   `json:"first_worker"`
		SnitchURL     string `json:"snitch_url"`
		Transcription struct {
			Mode            string `json:"mode"`
			Endpoint        string `json:"endpoint"`
			SettingsVersion string `json:"settings_version"`
			Model           string `json:"model"`
			Language        string `json:"language"`
			Version         string `json:"version"`
		} `json:"transcription"`
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	list := func() page {
		t.Helper()
		var p page
		if e := json.Unmarshal(call(admin, http.MethodGet, "/api/admin/workers", nil, "", "", http.StatusOK), &p); e != nil {
			t.Fatal(e)
		}
		return p
	}
	var before page = list()
	if !before.FirstWorker || before.SnitchURL != ts.URL || len(before.Items) != 0 {
		t.Fatalf("listing before any Worker: %+v", before)
	}
	if before.Transcription.Mode != "unconfigured" {
		t.Fatalf("unexpected initial transcription mode: %+v", before.Transcription)
	}

	// Settings are chosen in Transcription, before any Worker exists. The registration
	// API rejects legacy per-worker overrides rather than silently diverging.
	call(admin, http.MethodPut, "/api/admin/transcription-settings", map[string]any{"mode": "enabled", "endpoint": "http://host.docker.internal:8000", "model": "whisper-large-v3", "language": "fr"}, me.CSRF, "", http.StatusOK)
	register := func(body map[string]any, want int) []byte {
		t.Helper()
		return call(admin, http.MethodPost, "/api/admin/workers", body, me.CSRF, "", want)
	}
	for _, body := range []map[string]any{
		{"name": "gpu-1", "max_concurrency": 1, "transcription_url": "http://other:8000"},
		{"name": "gpu-1", "max_concurrency": 1, "transcription": map[string]any{"model": "other", "language": "en"}},
	} {
		if got := errorCode(register(body, http.StatusBadRequest)); got != "invalid_request" {
			t.Fatalf("legacy override accepted: %q", got)
		}
	}
	if after := list(); !after.FirstWorker || len(after.Items) != 0 || after.Transcription.Mode != "enabled" || after.Transcription.Endpoint != "http://host.docker.internal:8000" || after.Transcription.SettingsVersion == "" {
		t.Fatalf("central settings not visible before Worker: %+v", after)
	}

	created := register(map[string]any{"name": "gpu-1", "max_concurrency": 1}, http.StatusCreated)
	var first struct {
		ID     string `json:"id"`
		Token  string `json:"token"`
		Bundle struct {
			Compose string `json:"compose"`
			Env     string `json:"env"`
		} `json:"bundle"`
	}
	if err = json.Unmarshal(created, &first); err != nil {
		t.Fatal(err)
	}
	env := "\n" + first.Bundle.Env + "\n"
	for _, line := range []string{"TRANSCRIPTION_API_URL=http://host.docker.internal:8000", "SNITCH_WORKER_TOKEN=" + first.Token} {
		if !strings.Contains(env, "\n"+line+"\n") {
			t.Fatalf("generated .env lacks %q:\n%s", line, first.Bundle.Env)
		}
	}
	if strings.Contains(env, "HF_TOKEN") {
		t.Fatalf("a new Worker downloads no voice model, its .env carries no token:\n%s", first.Bundle.Env)
	}
	// The deployment files never require a model name (it is the library's).
	if strings.Contains(env, "\nTRANSCRIPTION_MODEL=") || strings.Contains(first.Bundle.Compose, "TRANSCRIPTION_MODEL: ${TRANSCRIPTION_MODEL:?") {
		t.Fatalf("the deployment files require a model name:\n%s", first.Bundle.Env)
	}
	// The library profile was chosen before the first Worker; registration does not change it.
	var pipeline struct {
		Version string `json:"version"`
		Config  struct {
			Model    string `json:"model"`
			Language string `json:"language"`
			VAD      struct {
				Mode string `json:"mode"`
			} `json:"vad"`
		} `json:"config"`
	}
	if err = json.Unmarshal(call(admin, http.MethodGet, "/api/admin/transcription-pipeline", nil, "", "", http.StatusOK), &pipeline); err != nil {
		t.Fatal(err)
	}
	if pipeline.Config.Model != "whisper-large-v3" || pipeline.Config.Language != "fr" || pipeline.Config.VAD.Mode != "off" || pipeline.Version == "pipeline-v1" {
		t.Fatalf("central profile changed unexpectedly: %+v", pipeline)
	}
	afterFirst := list()
	if afterFirst.FirstWorker || afterFirst.Transcription.Model != "whisper-large-v3" || afterFirst.Transcription.Language != "fr" || afterFirst.Transcription.Version != pipeline.Version {
		t.Fatalf("listing after the first Worker: %+v", afterFirst)
	}

	// Later Workers inherit the same settings; per-Worker URL is not accepted here.
	if got := errorCode(register(map[string]any{"name": "gpu-2", "max_concurrency": 1, "transcription_url": "https://whisper.example.org"}, http.StatusBadRequest)); got != "invalid_request" {
		t.Fatalf("second Worker with manual URL: %q", got)
	}
	var second struct {
		ID    string `json:"id"`
		Token string `json:"token"`
	}
	if err = json.Unmarshal(register(map[string]any{"name": "gpu-2", "max_concurrency": 2}, http.StatusCreated), &second); err != nil {
		t.Fatal(err)
	}
	if again := list(); again.Transcription != afterFirst.Transcription || len(again.Items) != 2 {
		t.Fatalf("listing after the second Worker: %+v", again)
	}

	// A Worker may use a different name on its own Whisper server without changing the library profile.
	path := "/api/admin/workers/" + second.ID + "/transcription-model"
	call(anon, http.MethodPut, path, map[string]string{"model": "server-model"}, "", "", http.StatusUnauthorized)
	call(admin, http.MethodPut, path, map[string]string{"model": "bad model!"}, me.CSRF, "", http.StatusBadRequest)
	call(admin, http.MethodPut, path, map[string]string{"model": "server-model"}, me.CSRF, "", http.StatusOK)
	var updated struct {
		TranscriptionModel string `json:"transcription_model"`
	}
	if err = json.Unmarshal(call(admin, http.MethodGet, "/api/admin/workers/"+second.ID, nil, "", "", http.StatusOK), &updated); err != nil || updated.TranscriptionModel != "server-model" {
		t.Fatalf("worker model not persisted: %+v: %v", updated, err)
	}
	if again := list(); again.Transcription != afterFirst.Transcription {
		t.Fatalf("worker override changed library profile: %+v", again.Transcription)
	}

	// Worker detail: deployment file with a token placeholder, never the token itself.
	call(anon, http.MethodGet, "/api/admin/workers/"+first.ID, nil, "", "", http.StatusUnauthorized)
	call(admin, http.MethodGet, "/api/admin/workers/wk_unknownunknownunknownunknown", nil, "", "", http.StatusNotFound)
	detailBody := call(admin, http.MethodGet, "/api/admin/workers/"+first.ID, nil, "", "", http.StatusOK)
	if bytes.Contains(detailBody, []byte(first.Token)) {
		t.Fatal("the Worker detail exposed the token")
	}
	var detail struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		RecentTasks []struct {
			JobID     int64  `json:"job_id"`
			Kind      string `json:"kind"`
			Outcome   string `json:"outcome"`
			ErrorCode string `json:"error_code"`
		} `json:"recent_tasks"`
		Bundle struct {
			Env string `json:"env"`
		} `json:"bundle"`
	}
	if err = json.Unmarshal(detailBody, &detail); err != nil {
		t.Fatal(err)
	}
	if detail.ID != first.ID || detail.Name != "gpu-1" || len(detail.RecentTasks) != 0 || !strings.Contains("\n"+detail.Bundle.Env+"\n", "\nSNITCH_WORKER_TOKEN=A_RENSEIGNER\n") {
		t.Fatalf("unexpected detail: %s", detailBody)
	}

	// Recent tasks are typed: transcription, diarization, fingerprints (and anything else).
	var ownerID, clipID int64
	if err = pool.QueryRow(ctx, `SELECT id FROM users WHERE username='administrator'`).Scan(&ownerID); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO clips(owner_id,original_name,size_bytes,modified_at,recorded_at) VALUES($1,'admin-fixture.mp4',25,now(),now()) RETURNING id`, ownerID).Scan(&clipID); err != nil {
		t.Fatal(err)
	}
	type fixture struct{ step, event, errCode string }
	jobIDs := map[string]int64{}
	for _, f := range []fixture{
		{"transcription.audio.0", "completed", ""},
		{"voice.diarization.2", "marked_dead", "provider_unavailable"},
		{"voice.fingerprint.2", "attempt_started", ""},
		{"fixture.echo", "retry_scheduled", "temporary"},
	} {
		var id int64
		if err = pool.QueryRow(ctx, `INSERT INTO jobs(clip_id,step,input_version,status) VALUES($1,$2,'admin-v1','done') RETURNING id`, clipID, f.step).Scan(&id); err != nil {
			t.Fatal(err)
		}
		jobIDs[f.step] = id
		if _, err = pool.Exec(ctx, `INSERT INTO job_events(job_id,event,cycle,attempt,actor,actor_id,error_code) VALUES($1,'attempt_started',0,1,'worker',$2,''),($1,$3,0,1,'worker',$2,$4)`, id, first.ID, f.event, f.errCode); err != nil {
			t.Fatal(err)
		}
	}
	// Another Worker's work does not show up here.
	if _, err = pool.Exec(ctx, `INSERT INTO job_events(job_id,event,cycle,attempt,actor,actor_id) VALUES($1,'completed',0,1,'worker',$2)`, jobIDs["fixture.echo"], second.ID); err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(call(admin, http.MethodGet, "/api/admin/workers/"+first.ID, nil, "", "", http.StatusOK), &detail); err != nil {
		t.Fatal(err)
	}
	want := map[string][2]string{"transcription": {"completed", ""}, "diarization": {"failed", "provider_unavailable"}, "fingerprint": {"started", ""}, "other": {"retry", "temporary"}}
	if len(detail.RecentTasks) != len(want) {
		t.Fatalf("recent tasks: %+v", detail.RecentTasks)
	}
	for _, task := range detail.RecentTasks {
		if w, ok := want[task.Kind]; !ok || w[0] != task.Outcome || w[1] != task.ErrorCode {
			t.Fatalf("unexpected task %+v", task)
		}
	}

	// A new token replaces the old one, which stops working at once; the new one is shown once,
	// with its deployment files.
	call(anon, http.MethodPost, "/api/worker/pulse", map[string]any{"capacity": 1, "supported_steps": []string{"fixture.*"}}, "", first.Token, http.StatusOK)
	call(anon, http.MethodPost, "/api/admin/workers/"+first.ID+"/token", nil, "", "", http.StatusUnauthorized)
	call(admin, http.MethodPost, "/api/admin/workers/wk_unknownunknownunknownunknown/token", nil, me.CSRF, "", http.StatusNotFound)
	var renewed struct {
		ID     string `json:"id"`
		Token  string `json:"token"`
		Bundle struct {
			Env string `json:"env"`
		} `json:"bundle"`
	}
	if err = json.Unmarshal(call(admin, http.MethodPost, "/api/admin/workers/"+first.ID+"/token", nil, me.CSRF, "", http.StatusCreated), &renewed); err != nil {
		t.Fatal(err)
	}
	if renewed.ID == first.ID || !strings.HasPrefix(renewed.Token, "rw_") || renewed.Token == first.Token || !strings.Contains("\n"+renewed.Bundle.Env+"\n", "\nSNITCH_WORKER_TOKEN="+renewed.Token+"\n") {
		t.Fatalf("unexpected renewed token: %+v", renewed)
	}
	call(anon, http.MethodPost, "/api/worker/pulse", map[string]any{"capacity": 1, "supported_steps": []string{"fixture.*"}}, "", first.Token, http.StatusUnauthorized)
	call(anon, http.MethodPost, "/api/worker/pulse", map[string]any{"capacity": 1, "supported_steps": []string{"fixture.*"}}, "", renewed.Token, http.StatusOK)
	if bytes.Contains(call(admin, http.MethodGet, "/api/admin/workers/"+renewed.ID, nil, "", "", http.StatusOK), []byte(renewed.Token)) {
		t.Fatal("the Worker detail exposed the new token")
	}
	// A lost token means a new identity: the old one is revoked with a minimal audit and cannot be renewed again.
	var old struct {
		RevokedAt    *string `json:"revoked_at"`
		RevokeReason string  `json:"revoke_reason"`
	}
	if err = json.Unmarshal(call(admin, http.MethodGet, "/api/admin/workers/"+first.ID, nil, "", "", http.StatusOK), &old); err != nil || old.RevokedAt == nil || old.RevokeReason != "token_renewed" {
		t.Fatalf("old identity must be revoked by the renewal: %v %+v", err, old)
	}
	call(admin, http.MethodPost, "/api/admin/workers/"+first.ID+"/token", nil, me.CSRF, "", http.StatusNotFound)
	first.ID = renewed.ID

	// Deletion takes effect at once and removes the Worker from the list; its token is refused and it
	// cannot get a new one.
	call(admin, http.MethodDelete, "/api/admin/workers/"+first.ID, nil, me.CSRF, "", http.StatusNoContent)
	call(anon, http.MethodPost, "/api/worker/pulse", map[string]any{"capacity": 1, "supported_steps": []string{"fixture.*"}}, "", renewed.Token, http.StatusUnauthorized)
	call(admin, http.MethodPost, "/api/admin/workers/"+first.ID+"/token", nil, me.CSRF, "", http.StatusNotFound)
	call(admin, http.MethodGet, "/api/admin/workers/"+first.ID, nil, "", "", http.StatusNotFound)
}
