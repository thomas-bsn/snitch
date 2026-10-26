//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers/legacy"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/database"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"snitch/internal/config"
	"snitch/internal/setup"
	"snitch/migrations"
)

const pgImage = "pgvector/pgvector:0.8.7-pg18-bookworm@sha256:2358fcba361ed2233a5ed81b5fe4ca779ccb304120ce531a3bf51c0ed7e2bc11"

func TestFoundation(t *testing.T) {
	ctx := context.Background()
	pg, err := postgres.Run(ctx, pgImage, postgres.WithDatabase("receipts"), postgres.WithUsername("receipts"), postgres.WithPassword("synthetic-only"), postgres.BasicWaitStrategies())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pg.Terminate(ctx) })
	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	db, err := goose.OpenDBWithDriver("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	provider, err := goose.NewProvider(database.DialectPostgres, db, migrations.Files)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = provider.Up(ctx); err != nil {
		t.Fatal(err)
	}
	replay, err := provider.Up(ctx)
	if err != nil || len(replay) != 0 {
		t.Fatal("migration replay", err)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var pgVersion, vectorVersion string
	if err = pool.QueryRow(ctx, `SELECT version(),(SELECT extversion FROM pg_extension WHERE extname='vector')`).Scan(&pgVersion, &vectorVersion); err != nil {
		t.Fatal(err)
	}
	t.Log("database:", pgVersion, "vector:", vectorVersion)
	t.Run("failed migration is rejected", func(t *testing.T) {
		bad := fstest.MapFS{"99999_bad.sql": &fstest.MapFile{Data: []byte("-- +goose Up\nTHIS IS NOT SQL;\n-- +goose Down\nSELECT 1;")}}
		p, e := goose.NewProvider(database.DialectPostgres, db, bad)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = p.Up(ctx); e == nil {
			t.Fatal("bad migration accepted")
		}
	})
	reset := func(t *testing.T) {
		t.Helper()
		_, e := pool.Exec(ctx, `UPDATE embedding_index_state SET active_profile_id=NULL,candidate_profile_id=NULL WHERE id=1; DELETE FROM transcript_embeddings; DELETE FROM embedding_tasks; DELETE FROM embedding_profiles; TRUNCATE sessions,oauth_states,settings,users,transcription_vocabulary_terms,transcription_vocabulary_versions RESTART IDENTITY CASCADE; TRUNCATE workers RESTART IDENTITY CASCADE; UPDATE job_control SET paused=false,max_concurrency=1,updated_at=now() WHERE id=true; DELETE FROM setup_steps; UPDATE installation SET initialized=false,bootstrap_hash=NULL,bootstrap_expires=NULL,registration_approval=true,setup_banner_dismissed=false`)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = pool.Exec(ctx, `UPDATE voice_pipeline_active SET version='voice-v1' WHERE id=true`); e != nil {
			t.Fatal(e)
		}
		if _, e = pool.Exec(ctx, `UPDATE voice_matching_settings SET auto_enabled=false,automatic_threshold=0.90,suggestion_threshold=0.75,ambiguity_margin=0.05,max_candidates=5,precision_threshold=0.95,min_confirmations=50,auto_changed_at=NULL,auto_changed_by=NULL WHERE id=true; DELETE FROM voice_automation_events`); e != nil {
			t.Fatal(e)
		}
	}
	t.Run("optional embeddings do not block foundation", func(t *testing.T) {
		provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		endpoint := provider.URL
		provider.Close() // A configured but unreachable service must not block the foundation.
		for _, endpoint := range []string{"", endpoint} {
			s := New(pool, config.Config{MasterKey: bytes.Repeat([]byte{1}, 32), EmbeddingsURL: endpoint, EmbeddingsModel: "fixture-only"}, slog.New(slog.NewJSONHandler(io.Discard, nil)))
			ts := httptest.NewServer(s.Handler())
			for path, want := range map[string]string{"/api/health/live": `"status":"ok"`, "/api/health/ready": `"status":"degraded"`, "/api/setup": `"initialized":false`} {
				resp, err := ts.Client().Get(ts.URL + path)
				if err != nil {
					t.Fatal(err)
				}
				body, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				if resp.StatusCode != 200 || !bytes.Contains(body, []byte(want)) {
					t.Fatalf("%s with embeddings %q: %d %s", path, endpoint, resp.StatusCode, body)
				}
			}
			ts.Close()
		}
	})
	t.Run("local authentication security", func(t *testing.T) {
		reset(t)
		testLocalAuthentication(t, pool)
	})
	t.Run("account preferences", func(t *testing.T) {
		reset(t)
		testAccountPreferences(t, pool)
	})
	t.Run("setup progress", func(t *testing.T) {
		reset(t)
		testSetupProgress(t, pool)
	})
	t.Run("public address deduced from the request", func(t *testing.T) {
		reset(t)
		testPublicAddress(t, pool)
	})
	t.Run("registration and approval", func(t *testing.T) {
		reset(t)
		testRegistration(t, pool)
	})
	t.Run("Discord OAuth", func(t *testing.T) { reset(t); testDiscordOAuth(t, pool) })
	t.Run("Discord settings", func(t *testing.T) {
		reset(t)
		testDiscordSettings(t, pool)
	})
	t.Run("master key", func(t *testing.T) {
		reset(t)
		testMasterKey(t, pool)
	})
	t.Run("roles", func(t *testing.T) {
		reset(t)
		testRoles(t, pool, provider)
	})
	t.Run("folder sources", func(t *testing.T) {
		reset(t)
		testFolderSources(t, pool)
	})
	t.Run("source reconciliation", func(t *testing.T) {
		reset(t)
		testSourceReconciliation(t, pool)
	})
	t.Run("resumable uploads", func(t *testing.T) {
		reset(t)
		testResumableUploads(t, pool)
	})
	t.Run("clip library", func(t *testing.T) {
		reset(t)
		testClipLibrary(t, pool)
	})
	t.Run("clip progress and filter options", func(t *testing.T) {
		reset(t)
		testClipProgress(t, pool)
	})
	t.Run("transcription pipeline", func(t *testing.T) {
		reset(t)
		testTranscriptionPipeline(t, pool)
	})
	t.Run("voice diarization", func(t *testing.T) {
		reset(t)
		testVoiceDiarization(t, pool)
	})
	t.Run("persons and attributions", func(t *testing.T) {
		reset(t)
		testPersonValidation(t, pool)
	})
	t.Run("voice matching", func(t *testing.T) {
		reset(t)
		testVoiceMatching(t, pool)
	})
	t.Run("voice automation", func(t *testing.T) {
		reset(t)
		testVoiceAutomation(t, pool)
	})
	t.Run("voice groups", func(t *testing.T) {
		reset(t)
		testVoiceGroups(t, pool)
	})
	t.Run("voice verification", func(t *testing.T) {
		reset(t)
		testVoiceVerification(t, pool)
	})
	t.Run("microphone track", func(t *testing.T) {
		reset(t)
		testMicrophoneTrack(t, pool)
	})
	t.Run("person corrections", func(t *testing.T) {
		reset(t)
		testPersonCorrections(t, pool)
	})
	t.Run("person voices", func(t *testing.T) {
		reset(t)
		testPersonVoices(t, pool)
	})
	t.Run("person filter", func(t *testing.T) {
		reset(t)
		testPersonFilter(t, pool)
	})
	t.Run("voice profile deletion", func(t *testing.T) {
		reset(t)
		testVoiceProfileDeletion(t, pool)
	})
	t.Run("segment corrections", func(t *testing.T) {
		reset(t)
		testSegmentCorrections(t, pool)
	})
	t.Run("segment retranscription", func(t *testing.T) {
		reset(t)
		testSegmentRetranscription(t, pool)
	})
	t.Run("segment reindexing", func(t *testing.T) {
		reset(t)
		testSegmentReindex(t, pool)
	})
	t.Run("segment concurrent corrections", func(t *testing.T) {
		reset(t)
		testSegmentConcurrency(t, pool)
	})
	t.Run("search corrections", func(t *testing.T) {
		reset(t)
		testSearchCorrections(t, pool)
	})
	t.Run("segment history", func(t *testing.T) {
		reset(t)
		testSegmentHistory(t, pool)
	})
	t.Run("segment export", func(t *testing.T) {
		reset(t)
		testSegmentExport(t, pool)
	})
	t.Run("segment speaker", func(t *testing.T) {
		reset(t)
		testSegmentSpeaker(t, pool)
	})
	t.Run("segment review fixes", func(t *testing.T) {
		testSegmentReviewFixes(t, pool, reset)
	})
	t.Run("voice vectors not persisted in jobs", func(t *testing.T) {
		reset(t)
		testVoiceVectorsNeverPersistedInJobs(t, pool)
	})
	t.Run("voice group visibility is not transitive", func(t *testing.T) {
		reset(t)
		testVoiceGroupVisibilityIsNotTransitive(t, pool)
	})
	t.Run("voice group naming refuses deleted profile", func(t *testing.T) {
		reset(t)
		testVoiceGroupNamingRefusesDeletedProfile(t, pool)
	})
	t.Run("voice group links ignore known neighbors", func(t *testing.T) {
		reset(t)
		testVoiceGroupLinksIgnoreKnownNeighbors(t, pool)
	})
	t.Run("library unknown filter matches search", func(t *testing.T) {
		reset(t)
		testLibraryUnknownFilterMatchesSearch(t, pool)
	})
	t.Run("person corrections respect visibility", func(t *testing.T) {
		reset(t)
		testPersonCorrectionsRespectVisibility(t, pool)
	})
	t.Run("voice result with expired lease", func(t *testing.T) {
		reset(t)
		testVoiceResultWithExpiredLease(t, pool)
	})
	t.Run("job queue", func(t *testing.T) {
		reset(t)
		testJobQueue(t, pool)
	})
	t.Run("job retry and cancellation", func(t *testing.T) {
		reset(t)
		testJobRetryOperations(t, pool)
	})
	t.Run("job pause and concurrency controls", func(t *testing.T) {
		reset(t)
		testJobControls(t, pool)
	})
	t.Run("worker authentication and registration", func(t *testing.T) {
		reset(t)
		testWorkerRegistration(t, pool)
	})
	t.Run("worker declared models", func(t *testing.T) {
		reset(t)
		testWorkerModels(t, pool)
	})
	t.Run("worker administration", func(t *testing.T) {
		reset(t)
		testWorkerAdministration(t, pool)
	})
	t.Run("HTTP contract and permissions", func(t *testing.T) {
		reset(t)
		clipRoot := t.TempDir()
		code, e := setup.Issue(ctx, pool)
		if e != nil {
			t.Fatal(e)
		}
		logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
		s := New(pool, config.Config{MasterKey: bytes.Repeat([]byte{1}, 32), ClipRoots: []string{clipRoot}, ProbePath: "ffprobe", StabilityWindow: time.Hour}, logger)
		// High test allowance isolates security tests from test-suite request volume.
		ts := httptest.NewUnstartedServer(s.Handler())
		s.Config.PublicURL = "http://" + ts.Listener.Addr().String()
		ts.Start()
		defer ts.Close()
		jar, _ := cookiejar.New(nil)
		client := &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
		anon := &http.Client{CheckRedirect: client.CheckRedirect}
		spec, e := openapi3.NewLoader().LoadFromFile("../../../contracts/openapi.yaml")
		if e != nil {
			t.Fatal(e)
		}
		if e = spec.Validate(ctx); e != nil {
			t.Fatal(e)
		}
		spec.Servers = nil
		router, e := legacy.NewRouter(spec)
		if e != nil {
			t.Fatal(e)
		}
		seen := map[string]bool{}
		workerToken := ""
		leaseToken := ""
		call := func(c *http.Client, method, path string, body any, csrf string, want int) []byte {
			t.Helper()
			s.rates.Clear()
			var data []byte
			if body != nil {
				data, _ = json.Marshal(body)
			}
			req, e := http.NewRequest(method, ts.URL+path, bytes.NewReader(data))
			if e != nil {
				t.Fatal(e)
			}
			req.Header.Set("Origin", ts.URL)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-CSRF-Token", csrf)
			if workerToken != "" {
				req.Header.Set("Authorization", "Bearer "+workerToken)
			}
			if leaseToken != "" {
				req.Header.Set("X-Worker-Lease-Token", leaseToken)
			}
			resp, e := c.Do(req)
			if e != nil {
				t.Fatal(e)
			}
			defer resp.Body.Close()
			b, _ := io.ReadAll(resp.Body)
			if resp.StatusCode != want {
				t.Fatalf("%s %s: got %d want %d: %s", method, path, resp.StatusCode, want, b)
			}
			route, params, e := router.FindRoute(req)
			if e != nil {
				t.Fatal(e)
			}
			req.Body = io.NopCloser(bytes.NewReader(data))
			input := &openapi3filter.RequestValidationInput{Request: req, PathParams: params, Route: route, Options: &openapi3filter.Options{AuthenticationFunc: openapi3filter.NoopAuthenticationFunc}}
			if e = openapi3filter.ValidateRequest(ctx, input); e != nil {
				t.Fatalf("request contract %s %s: %v", method, path, e)
			}
			out := &openapi3filter.ResponseValidationInput{RequestValidationInput: input, Status: resp.StatusCode, Header: resp.Header, Body: io.NopCloser(bytes.NewReader(b))}
			if e = openapi3filter.ValidateResponse(ctx, out); e != nil {
				t.Fatalf("contract %s %s: %v", method, path, e)
			}
			seen[method+" "+route.Path] = true
			return b
		}
		creds := map[string]string{"username": "administrator", "password": "synthetic-password"}
		call(anon, "GET", "/api/health/live", nil, "", 200)
		call(anon, "GET", "/api/health/ready", nil, "", 200)
		call(anon, "GET", "/api/setup", nil, "", 200)
		call(anon, "GET", "/api/admin/settings", nil, "", 401)
		call(anon, "GET", "/api/admin/sources", nil, "", 401)
		call(anon, "POST", "/api/auth/register", creds, "", 409)
		call(client, "POST", "/api/setup", map[string]string{"code": "wrong", "username": "administrator", "password": "synthetic-password"}, "", 409)
		call(client, "POST", "/api/setup", map[string]string{"code": code, "username": "administrator", "password": "synthetic-password"}, "", 201)
		call(client, "POST", "/api/setup", map[string]string{"code": code, "username": "administrator", "password": "synthetic-password"}, "", 409)
		noRoots := New(pool, config.Config{MasterKey: bytes.Repeat([]byte{1}, 32)}, logger)
		noRootsTS := httptest.NewServer(noRoots.Handler())
		defer noRootsTS.Close()
		noRootsResponse, e := client.Get(noRootsTS.URL + "/api/admin/sources")
		if e != nil {
			t.Fatal(e)
		}
		var noRootsBody struct {
			AllowedRoots []string `json:"allowed_roots"`
		}
		if noRootsResponse.StatusCode != http.StatusOK || json.NewDecoder(noRootsResponse.Body).Decode(&noRootsBody) != nil {
			noRootsResponse.Body.Close()
			t.Fatalf("unconfigured source roots response: status %d", noRootsResponse.StatusCode)
		}
		noRootsResponse.Body.Close()
		if noRootsBody.AllowedRoots == nil || len(noRootsBody.AllowedRoots) != 0 {
			t.Fatalf("unconfigured source roots must be an empty array, got %#v", noRootsBody.AllowedRoots)
		}
		var me struct {
			CSRF string `json:"csrf_token"`
		}
		json.Unmarshal(call(client, "GET", "/api/auth/me", nil, "", 200), &me)
		// Account language (behavior in the account preferences suite).
		call(client, "GET", "/api/account/preferences", nil, "", 200)
		call(client, "PUT", "/api/account/preferences", map[string]any{"language": "fr"}, me.CSRF, 200)
		call(client, "GET", "/api/admin/job-controls", nil, "", 200)
		call(client, "PUT", "/api/admin/job-controls", map[string]any{"paused": false, "max_concurrency": 2}, me.CSRF, 200)
		call(client, "GET", "/api/admin/embeddings", nil, "", 200)
		call(client, "POST", "/api/admin/embeddings/backfill", map[string]any{}, me.CSRF, 202)
		call(client, "POST", "/api/admin/embeddings/rebuild", map[string]string{"model": "fixture-only"}, me.CSRF, 503)
		call(client, "GET", "/api/clips", nil, "", 200)
		call(client, "GET", "/api/clips/999999", nil, "", 404)
		var adminID, contractClipID int64
		if e = pool.QueryRow(ctx, `SELECT id FROM users WHERE username='administrator'`).Scan(&adminID); e != nil {
			t.Fatal(e)
		}
		if e = pool.QueryRow(ctx, `INSERT INTO clips(owner_id,original_name,size_bytes,modified_at,recorded_at) VALUES($1,'contract-synthetic.mp4',1,now(),now()) RETURNING id`, adminID).Scan(&contractClipID); e != nil {
			t.Fatal(e)
		}
		_, e = pool.Exec(ctx, `UPDATE clips SET tracks=$2::jsonb WHERE id=$1`, contractClipID, `[{"index":1,"type":"audio","codec":"aac","title":"Game","language":"fra","channels":2,"channel_layout":"stereo"},{"index":2,"type":"audio","codec":"aac","title":"Voice","language":"fra","channels":2,"channel_layout":"stereo"}]`)
		if e != nil {
			t.Fatal(e)
		}
		call(client, "PUT", "/api/admin/clips/"+strconv64(contractClipID)+"/audio-track", map[string]int{"audio_stream_index": 2}, me.CSRF, 200)
		call(client, "DELETE", "/api/admin/clips/"+strconv64(contractClipID)+"/audio-track", nil, me.CSRF, 200)
		// Voice endpoints: contract conformity here, full behavior in the voice suite.
		call(client, "GET", "/api/admin/voice-pipeline", nil, "", 200)
		call(client, "PUT", "/api/admin/voice-pipeline", map[string]any{"provider": "fake", "model": "fixture-diarization", "model_version": "fixture-v1", "access_confirmed": false, "min_fingerprint_seconds": 1.0, "min_quality": 0.5, "fake": map[string]any{"speakers": 2, "dimensions": 4}}, me.CSRF, 200)
		call(client, "POST", "/api/admin/clips/"+strconv64(contractClipID)+"/voice", map[string]any{}, me.CSRF, 409)
		call(client, "GET", "/api/clips/"+strconv64(contractClipID)+"/voice", nil, "", 200)
		// Person and attribution endpoints, on a synthetic local speaker.
		var contractSpeakerID int64
		if e = pool.QueryRow(ctx, `WITH j AS (INSERT INTO jobs(clip_id,step,input_version,status) VALUES($1,'voice.diarization.2','contract-diarization','done') RETURNING id) INSERT INTO local_speakers(clip_id,audio_stream_index,diarization_job_id,label,ordinal,speaking_seconds,turn_count) SELECT $1,2,id,'SPEAKER_00',0,1.5,1 FROM j RETURNING id`, contractClipID).Scan(&contractSpeakerID); e != nil {
			t.Fatal(e)
		}
		speakerPath := "/api/local-speakers/" + strconv64(contractSpeakerID)
		var contractPerson struct {
			ID int64 `json:"id"`
		}
		json.Unmarshal(call(client, "POST", "/api/persons", map[string]any{"display_name": "Contract person"}, me.CSRF, 201), &contractPerson)
		call(client, "GET", "/api/persons?q=contract", nil, "", 200)
		call(client, "GET", "/api/persons/"+strconv64(contractPerson.ID), nil, "", 200)
		call(client, "GET", "/api/persons/"+strconv64(contractPerson.ID)+"/voices", nil, "", 200)
		call(client, "GET", "/api/clip-filters", nil, "", 200)
		call(client, "POST", speakerPath+"/validation", map[string]any{"person_id": contractPerson.ID}, me.CSRF, 201)
		call(client, "POST", speakerPath+"/validation", map[string]any{"new_person_name": "Late"}, me.CSRF, 409)
		call(client, "POST", speakerPath+"/correction", map[string]any{"new_person_name": "Corrected", "expected_person_id": contractPerson.ID}, me.CSRF, 200)
		call(client, "GET", speakerPath+"/history", nil, "", 200)
		call(client, "GET", speakerPath+"/excerpt", nil, "", 404)
		// Person corrections (behavior in the person corrections suite).
		var absorbed, kept struct {
			ID int64 `json:"id"`
		}
		json.Unmarshal(call(client, "POST", "/api/persons", map[string]any{"display_name": "Absorbed"}, me.CSRF, 201), &absorbed)
		json.Unmarshal(call(client, "POST", "/api/persons", map[string]any{"display_name": "Kept"}, me.CSRF, 201), &kept)
		call(client, "POST", "/api/persons/"+strconv64(absorbed.ID)+"/separation", map[string]any{"local_speaker_ids": []int64{contractSpeakerID}}, me.CSRF, 409)
		call(client, "POST", "/api/persons/"+strconv64(absorbed.ID)+"/merge", map[string]any{"target_person_id": kept.ID}, me.CSRF, 200)
		call(client, "GET", "/api/person-events?person_id="+strconv64(absorbed.ID), nil, "", 200)
		call(client, "DELETE", "/api/persons/"+strconv64(kept.ID), nil, me.CSRF, 200)
		call(client, "DELETE", "/api/persons/"+strconv64(kept.ID), nil, me.CSRF, 404)
		// Voice matching settings and rematch (behavior in the voice matching suite).
		call(client, "GET", "/api/voice-matching", nil, "", 200)
		call(client, "PUT", "/api/admin/voice-matching", map[string]any{"automatic_threshold": 0.9, "suggestion_threshold": 0.75, "ambiguity_margin": 0.05, "max_candidates": 5}, me.CSRF, 200)
		call(client, "PUT", "/api/admin/voice-matching", map[string]any{"automatic_threshold": 0.5, "suggestion_threshold": 0.75, "ambiguity_margin": 0.05, "max_candidates": 5}, me.CSRF, 400)
		call(client, "POST", "/api/admin/voice-matching/rematch", nil, me.CSRF, 200)
		// Account link and voice profile deletion (behavior in the voice profile suite).
		personPath := "/api/admin/persons/" + strconv64(contractPerson.ID) + "/account"
		call(client, "GET", "/api/me/person", nil, "", 404)
		call(client, "PUT", personPath, map[string]any{"username": "administrator"}, me.CSRF, 200)
		call(client, "PUT", personPath, map[string]any{"username": "nobody"}, me.CSRF, 404)
		call(client, "GET", "/api/me/person", nil, "", 200)
		call(client, "DELETE", personPath, nil, me.CSRF, 200)
		call(client, "DELETE", "/api/persons/"+strconv64(contractPerson.ID)+"/voice-profile", nil, me.CSRF, 200)
		call(client, "DELETE", "/api/persons/999999/voice-profile", nil, me.CSRF, 404)
		// Accuracy measure and automation switch (behavior in the voice automation suite).
		call(client, "GET", "/api/admin/voice-accuracy", nil, me.CSRF, 200)
		call(client, "PUT", "/api/admin/voice-automation/criteria", map[string]any{"precision_threshold": 0.95, "min_confirmations": 50}, me.CSRF, 200)
		call(client, "PUT", "/api/admin/voice-automation", map[string]any{"enabled": true}, me.CSRF, 409)
		call(client, "PUT", "/api/admin/voice-automation", map[string]any{"enabled": false}, me.CSRF, 200)
		// Groups of unknown voices (behavior in the voice groups suite).
		call(client, "GET", "/api/voice-groups", nil, "", 200)
		call(client, "GET", "/api/voice-groups?sort=members&proposed=true&page=1&page_size=5", nil, "", 200)
		call(client, "GET", "/api/voice-groups/999999", nil, "", 404)
		call(client, "POST", "/api/voice-groups/999999/naming", map[string]any{"new_person_name": "Nobody", "local_speaker_ids": []int64{999999}}, me.CSRF, 404)
		call(client, "PUT", speakerPath+"/group-exclusion", nil, me.CSRF, 409)
		call(client, "DELETE", speakerPath+"/group-exclusion", nil, me.CSRF, 200)
		// Review of unknown and ambiguous voices (behavior in the voice verification suite).
		call(client, "GET", "/api/voice-verification", nil, "", 200)
		call(client, "GET", "/api/voice-verification?level=ambiguous&page=1&page_size=5", nil, "", 200)
		// Microphone track of a clip: view, replacement, disabling and removal.
		micPath := "/api/clips/" + strconv64(contractClipID) + "/microphone-track"
		call(client, "GET", micPath, nil, "", 200)
		call(client, "PUT", "/api/admin/clips/"+strconv64(contractClipID)+"/microphone-track", map[string]any{"audio_stream_index": 1, "person_id": contractPerson.ID}, me.CSRF, 200)
		call(client, "PUT", "/api/admin/clips/"+strconv64(contractClipID)+"/microphone-track", map[string]any{"disabled": true}, me.CSRF, 200)
		call(client, "DELETE", "/api/admin/clips/"+strconv64(contractClipID)+"/microphone-track", nil, me.CSRF, 200)
		contractJobBody := call(client, "POST", "/api/admin/jobs", map[string]any{"clip_id": contractClipID, "step": "contract.synthetic", "input_version": "contract-v1"}, me.CSRF, 201)
		var contractJob struct {
			ID int64 `json:"id"`
		}
		if e = json.Unmarshal(contractJobBody, &contractJob); e != nil || contractJob.ID < 1 {
			t.Fatalf("contract worker job: %s %v", contractJobBody, e)
		}
		call(client, "GET", "/api/admin/jobs?status=pending", nil, "", 200)
		contractJobPath := "/api/admin/jobs/" + strconv64(contractJob.ID)
		call(client, "GET", contractJobPath+"/history", nil, "", 200)
		workerBody := call(client, "POST", "/api/admin/workers", map[string]any{"name": "contract-worker", "max_concurrency": 1}, me.CSRF, 201)
		var contractWorker struct {
			ID    string `json:"id"`
			Token string `json:"token"`
		}
		if e = json.Unmarshal(workerBody, &contractWorker); e != nil || contractWorker.Token == "" {
			t.Fatalf("contract worker registration: %s %v", workerBody, e)
		}
		call(client, "GET", "/api/admin/workers", nil, "", 200)
		call(client, "GET", "/api/admin/workers/"+contractWorker.ID, nil, "", 200)
		renewedBody := call(client, "POST", "/api/admin/workers/"+contractWorker.ID+"/token", nil, me.CSRF, 201)
		if e = json.Unmarshal(renewedBody, &contractWorker); e != nil || contractWorker.Token == "" {
			t.Fatalf("contract worker token renewal: %s %v", renewedBody, e)
		}
		workerToken = contractWorker.Token
		call(client, "POST", "/api/worker/pulse", map[string]any{"capacity": 1, "supported_steps": []string{"contract.synthetic"}}, "", 200)
		claimedBody := call(client, "POST", "/api/worker/jobs/claim", nil, "", 200)
		var claimedContractJob struct {
			LeaseToken string `json:"lease_token"`
			Job        struct {
				ID int64 `json:"id"`
			} `json:"job"`
		}
		if e = json.Unmarshal(claimedBody, &claimedContractJob); e != nil || claimedContractJob.Job.ID != contractJob.ID || claimedContractJob.LeaseToken == "" {
			t.Fatalf("contract worker claim: %s %v", claimedBody, e)
		}
		leaseToken = claimedContractJob.LeaseToken
		workerJobPath := "/api/worker/jobs/" + strconv64(contractJob.ID)
		call(client, "GET", workerJobPath+"/input", nil, "", 200)
		call(client, "GET", workerJobPath+"/media", nil, "", 404)
		call(client, "POST", workerJobPath+"/heartbeat", map[string]any{}, "", 200)
		call(client, "POST", workerJobPath+"/complete", map[string]any{"input_version": "contract-v1", "result": map[string]any{"fixture": true}}, "", 200)
		failedJobBody := call(client, "POST", "/api/admin/jobs", map[string]any{"clip_id": contractClipID, "step": "contract.failure", "input_version": "failure-v1", "max_attempts": 1}, me.CSRF, 201)
		var failedContractJob struct {
			ID int64 `json:"id"`
		}
		if e = json.Unmarshal(failedJobBody, &failedContractJob); e != nil || failedContractJob.ID < 1 {
			t.Fatalf("contract failure job: %s %v", failedJobBody, e)
		}
		call(client, "POST", "/api/worker/pulse", map[string]any{"capacity": 1, "supported_steps": []string{"contract.failure"}}, "", 200)
		failedClaim := call(client, "POST", "/api/worker/jobs/claim", nil, "", 200)
		var failedJob struct {
			LeaseToken string `json:"lease_token"`
			Job        struct {
				ID int64 `json:"id"`
			} `json:"job"`
		}
		if e = json.Unmarshal(failedClaim, &failedJob); e != nil || failedJob.Job.ID < 1 || failedJob.LeaseToken == "" {
			t.Fatalf("second worker claim: %s %v", failedClaim, e)
		}
		leaseToken = failedJob.LeaseToken
		call(client, "POST", "/api/worker/jobs/"+strconv64(failedJob.Job.ID)+"/fail", map[string]any{"error": "provider_unavailable"}, "", 200)
		workerToken = ""
		failedPath := "/api/admin/jobs/" + strconv64(failedContractJob.ID)
		call(client, "GET", failedPath+"/history", nil, "", 200)
		call(client, "POST", failedPath+"/retry", nil, me.CSRF, 200)
		call(client, "DELETE", failedPath, nil, me.CSRF, 204)
		call(client, "DELETE", "/api/admin/workers/"+contractWorker.ID, nil, me.CSRF, 204)
		call(client, "POST", "/api/auth/logout", nil, "wrong", 403)
		call(client, "GET", "/api/admin/vocabulary", nil, "", 200)
		vocabularyBody := call(client, "POST", "/api/admin/vocabulary", map[string]any{"term": "Synthetic term", "variants": []string{"synthetic variant"}, "preferred_spelling": "Synthetic term", "active": true}, me.CSRF, 201)
		var vocabularyTerm struct {
			ID int64 `json:"id"`
		}
		if e = json.Unmarshal(vocabularyBody, &vocabularyTerm); e != nil || vocabularyTerm.ID < 1 {
			t.Fatalf("vocabulary response: %s %v", vocabularyBody, e)
		}
		call(client, "PUT", "/api/admin/vocabulary/"+strconv64(vocabularyTerm.ID), map[string]any{"term": "Synthetic term", "variants": []string{"synthetic variant"}, "preferred_spelling": "Synthetic variant", "active": false}, me.CSRF, 200)
		call(client, "DELETE", "/api/admin/vocabulary/"+strconv64(vocabularyTerm.ID), nil, me.CSRF, 204)
		call(client, "PUT", "/api/admin/settings", map[string]any{"client_id": "test-id", "client_secret": "secret-test-only", "registration_approval": true}, me.CSRF, 200)
		call(client, "GET", "/api/admin/sources", nil, "", 200)
		sourceBody := map[string]string{"name": "contract-fixture", "path": clipRoot}
		sourceResponse := call(client, "POST", "/api/admin/sources", sourceBody, me.CSRF, 201)
		var createdSource struct {
			ID int64 `json:"id"`
		}
		if e = json.Unmarshal(sourceResponse, &createdSource); e != nil || createdSource.ID == 0 {
			t.Fatalf("source response: %v %s", e, sourceResponse)
		}
		call(client, "PUT", "/api/admin/sources/"+strconv64(createdSource.ID)+"/audio-track-rule", map[string]string{"title": "Voice"}, me.CSRF, 200)
		sourcePath := fmt.Sprintf("/api/admin/sources/%d", createdSource.ID)
		call(client, "GET", sourcePath+"/microphone-track", nil, "", 200)
		call(client, "PUT", sourcePath+"/microphone-track", map[string]any{"title": "Voice", "person_id": contractPerson.ID}, me.CSRF, 200)
		call(client, "DELETE", sourcePath+"/microphone-track", nil, me.CSRF, 200)
		call(client, "GET", sourcePath+"/files", nil, "", 200)
		call(client, "POST", sourcePath+"/scan", map[string]any{}, me.CSRF, 202)
		call(client, "DELETE", sourcePath, nil, me.CSRF, 200)
		b := call(client, "GET", "/api/admin/settings", nil, "", 200)
		if bytes.Contains(b, []byte("secret-test-only")) {
			t.Fatal("secret leaked")
		}
		var stored []byte
		if e = pool.QueryRow(ctx, `SELECT value FROM settings WHERE key='discord_client_secret'`).Scan(&stored); e != nil || bytes.Contains(stored, []byte("secret-test-only")) {
			t.Fatal("unencrypted setting", e)
		}
		call(anon, "POST", "/api/auth/register", map[string]string{"username": "player", "password": "synthetic-password"}, "", 201)
		call(anon, "POST", "/api/auth/login", map[string]string{"username": "player", "password": "synthetic-password"}, "", 401)
		var users []struct {
			ID int64 `json:"id"`
		}
		json.Unmarshal(call(client, "GET", "/api/admin/users", nil, "", 200), &users)
		if len(users) != 1 {
			t.Fatal("pending", string(b))
		}
		call(client, "POST", fmt.Sprintf("/api/admin/users/%d/approve", users[0].ID), nil, me.CSRF, 200)
		userJar, _ := cookiejar.New(nil)
		user := &http.Client{Jar: userJar, CheckRedirect: client.CheckRedirect}
		call(user, "POST", "/api/auth/login", map[string]string{"username": "player", "password": "synthetic-password"}, "", 200)
		call(user, "GET", "/api/admin/settings", nil, "", 403)
		call(user, "GET", "/api/admin/vocabulary", nil, "", 403)
		call(user, "GET", "/api/admin/users", nil, "", 403)
		call(client, "POST", "/api/admin/users/99999/approve", nil, me.CSRF, 404)
		call(client, "POST", "/api/auth/logout", nil, me.CSRF, 200)
		call(client, "GET", "/api/auth/me", nil, "", 401)
		call(client, "POST", "/api/auth/login", creds, "", 200)
		json.Unmarshal(call(client, "GET", "/api/auth/me", nil, "", 200), &me)
		// Simulated Discord: verify scope, state, browser binding, replay, approval and explicit linking.
		mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/token" {
				r.ParseForm()
				if r.Form.Get("client_secret") != "secret-test-only" {
					t.Error("wrong OAuth secret")
				}
				reply(w, 200, map[string]string{"access_token": "mock-token"})
				return
			}
			if r.Header.Get("Authorization") != "Bearer mock-token" {
				t.Error("missing OAuth bearer")
			}
			reply(w, 200, map[string]string{"id": "123456"})
		}))
		defer mock.Close()
		s.Config.DiscordAuthorizeURL = mock.URL + "/authorize"
		s.Config.DiscordTokenURL = mock.URL + "/token"
		s.Config.DiscordUserURL = mock.URL + "/me"
		var redirect struct {
			URL string `json:"url"`
		}
		start := func(c *http.Client, path, csrf string) string {
			json.Unmarshal(call(c, "POST", path, nil, csrf, 200), &redirect)
			req, _ := http.NewRequest("GET", redirect.URL, nil)
			if req.URL.Query().Get("scope") != "identify" {
				t.Fatal("OAuth scope")
			}
			return req.URL.Query().Get("state")
		}
		state := start(user, "/api/auth/discord/start", "")
		call(anon, "GET", "/api/auth/discord/callback?state="+state+"&code=mock", nil, "", 400)
		call(user, "GET", "/api/auth/discord/callback?state="+state+"&code=mock", nil, "", 303)
		call(user, "GET", "/api/auth/discord/callback?state="+state+"&code=mock", nil, "", 400)
		var discordUID int64
		if e = pool.QueryRow(ctx, `SELECT id FROM users WHERE discord_id='123456' AND NOT approved`).Scan(&discordUID); e != nil {
			t.Fatal("Discord approval", e)
		}
		// Linking never merges an identity belonging to another account.
		state = start(client, "/api/auth/discord/link", me.CSRF)
		call(client, "GET", "/api/auth/discord/callback?state="+state+"&code=mock", nil, "", 409)
		_, e = pool.Exec(ctx, `DELETE FROM users WHERE id=$1`, discordUID)
		if e != nil {
			t.Fatal(e)
		}
		state = start(client, "/api/auth/discord/link", me.CSRF)
		call(client, "GET", "/api/auth/discord/callback?state="+state+"&code=mock", nil, "", 303)
		call(client, "POST", "/api/auth/logout", nil, me.CSRF, 200)
		state = start(client, "/api/auth/discord/start", "")
		call(client, "GET", "/api/auth/discord/callback?state="+state+"&code=mock", nil, "", 303)
		call(client, "GET", "/api/auth/me", nil, "", 200)
		// Segment corrections: contract conformity here, behavior in the segment corrections suite.
		json.Unmarshal(call(client, "GET", "/api/auth/me", nil, "", 200), &me)
		call(client, "PUT", "/api/clips/"+strconv64(contractClipID)+"/segments/0/text", map[string]any{"text": "corrigé", "expected_version": 0}, me.CSRF, 404)
		if _, e = pool.Exec(ctx, `INSERT INTO jobs(clip_id,step,input_version,status,result) VALUES($1,'transcription.audio.2','contract-transcription','done',$2::jsonb)`, contractClipID, `{"provider":"fixture","model":"m","audio_stream_index":2,"segments":[{"start":0,"end":1,"text":"original","words":[]}]}`); e != nil {
			t.Fatal(e)
		}
		call(client, "PUT", "/api/clips/"+strconv64(contractClipID)+"/segments/0/text", map[string]any{"text": "corrigé", "expected_version": 0, "reason": "contrat"}, me.CSRF, 200)
		call(client, "PUT", "/api/clips/"+strconv64(contractClipID)+"/segments/0/text", map[string]any{"text": "encore", "expected_version": 0}, me.CSRF, 409)
		call(client, "GET", "/api/clips/"+strconv64(contractClipID)+"/segments/0/history", nil, "", 200)
		call(client, "GET", "/api/clips/"+strconv64(contractClipID)+"/obsolete-corrections", nil, "", 200)
		call(client, "POST", "/api/clips/"+strconv64(contractClipID)+"/segments/0/restore", map[string]any{"original": true, "expected_version": 0}, me.CSRF, 409)
		call(client, "POST", "/api/clips/"+strconv64(contractClipID)+"/segments/0/restore", map[string]any{"original": true, "expected_version": 1, "reason": "contrat"}, me.CSRF, 200)
		call(client, "GET", "/api/admin/segment-corrections/export", nil, "", 200)
		call(client, "POST", "/api/admin/master-key/export", nil, me.CSRF, 200) // behavior in the master key suite
		call(client, "PUT", "/api/clips/"+strconv64(contractClipID)+"/segments/9/speaker", map[string]any{"local_speaker_id": contractSpeakerID, "expected_version": 0}, me.CSRF, 404)
		call(client, "PUT", "/api/clips/"+strconv64(contractClipID)+"/segments/0/speaker", map[string]any{"local_speaker_id": contractSpeakerID, "expected_version": 2, "reason": "contrat"}, me.CSRF, 200)
		call(client, "PUT", "/api/clips/"+strconv64(contractClipID)+"/segments/0/speaker", map[string]any{"local_speaker_id": nil, "expected_version": 2}, me.CSRF, 409)
		for path, item := range spec.Paths.Map() {
			for method := range item.Operations() {
				if strings.HasPrefix(path, "/api/uploads") || path == "/api/clips/{id}/media" || path == "/api/admin/clips/{id}/transcription" || path == "/api/search" || strings.HasPrefix(path, "/api/admin/transcriptions/batch-") || path == "/api/admin/transcription-pipeline" {
					continue // Upload bodies, media ranges, transcription and batch scheduling, and transcript search have dedicated suites.
				}
				if !seen[method+" "+path] {
					t.Errorf("untested contract operation %s %s", method, path)
				}
			}
		}
		// Origin and rate limiter are tested separately from the contract helper.
		req, _ := http.NewRequest("POST", ts.URL+"/api/auth/login", strings.NewReader(`{}`))
		req.Header.Set("Origin", "https://attacker.invalid")
		resp, e := anon.Do(req)
		if e != nil {
			t.Fatal(e)
		}
		resp.Body.Close()
		if resp.StatusCode != 403 {
			t.Fatal("origin check")
		}
		s.rates.Clear()
		throttled := false
		for i := 0; i < 12; i++ {
			req, _ = http.NewRequest("POST", ts.URL+"/api/auth/login", strings.NewReader(`{}`))
			req.Header.Set("Origin", ts.URL)
			req.Header.Set("Content-Type", "application/json")
			resp, e = anon.Do(req)
			if e != nil {
				t.Fatal(e)
			}
			resp.Body.Close()
			if resp.StatusCode == 429 {
				throttled = true
			}
		}
		if !throttled {
			t.Fatal("no rate limit")
		}
		resp, e = client.Get(ts.URL + "/")
		if e != nil {
			t.Fatal(e)
		}
		html, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if !bytes.Contains(html, []byte("<div id=\"root\"")) {
			t.Fatal("embedded SPA missing")
		}
		// Every screen address returns the application, and the old interface keeps its own address.
		for _, path := range []string{"/library", "/search", "/people", "/import", "/login", "/admin/workers", "/clips/12", "/account/settings"} {
			resp, e = client.Get(ts.URL + path)
			if e != nil {
				t.Fatal(e)
			}
			deep, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if resp.StatusCode != 200 || !bytes.Equal(deep, html) {
				t.Fatalf("screen address %s does not return the application: %d", path, resp.StatusCode)
			}
		}
		for _, path := range []string{"/legacy/", "/legacy/anything"} {
			resp, e = client.Get(ts.URL + path)
			if e != nil {
				t.Fatal(e)
			}
			old, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if resp.StatusCode != 200 || !bytes.Contains(old, []byte("<div id=\"root\"")) || bytes.Equal(old, html) {
				t.Fatalf("old interface missing at %s: %d", path, resp.StatusCode)
			}
		}
	})
	if os.Getenv("SNITCH_E2E") == "1" {
		suites := []string{"foundation.spec.ts", "local-auth.spec.ts", "registration.spec.ts", "shell-auth.spec.ts", "shell-preferences.spec.ts", "shell-accessibility.spec.ts", "setup-wizard.spec.ts", "account-settings.spec.ts", "admin-workers.spec.ts", "discord-settings.spec.ts", "discord-oauth.spec.ts", "sources.spec.ts", "uploads.spec.ts", "library.spec.ts", "jobs.spec.ts", "queue-controls.spec.ts", "workers.spec.ts", "transcription.spec.ts", "batch-transcriptions.spec.ts", "search.spec.ts", "vocabulary.spec.ts", "voice.spec.ts", "persons.spec.ts", "person-filter.spec.ts", "microphone.spec.ts", "matching.spec.ts", "voice-groups.spec.ts", "automation.spec.ts", "voice-profile.spec.ts", "voice-verification.spec.ts", "person-corrections.spec.ts", "people.spec.ts", "segment-corrections.spec.ts", "segment-reindex.spec.ts", "segment-conflicts.spec.ts", "search-corrections.spec.ts", "segment-history.spec.ts", "segment-export.spec.ts", "segment-speaker.spec.ts", "segment-replace-corrections.spec.ts"}
		if os.Getenv("SNITCH_E2E_FOUNDATION") == "1" {
			suites = suites[:1]
		}
		if os.Getenv("SNITCH_E2E_SHELL") == "1" {
			suites = []string{"shell-auth.spec.ts", "shell-preferences.spec.ts", "shell-accessibility.spec.ts"}
		}
		if os.Getenv("SNITCH_E2E_ACCOUNT") == "1" {
			suites = []string{"account-settings.spec.ts", "discord-oauth.spec.ts", "voice-profile.spec.ts"}
		}
		if os.Getenv("SNITCH_E2E_WIZARD") == "1" {
			suites = []string{"setup-wizard.spec.ts"}
		}
		if os.Getenv("SNITCH_E2E_SOURCES") == "1" {
			suites = []string{"sources.spec.ts"}
		}
		if os.Getenv("SNITCH_E2E_UPLOADS") == "1" {
			suites = []string{"uploads.spec.ts"}
		}
		if os.Getenv("SNITCH_E2E_LIBRARY") == "1" {
			suites = []string{"library.spec.ts"}
		}
		if os.Getenv("SNITCH_E2E_JOBS") == "1" {
			suites = []string{"jobs.spec.ts"}
		}
		if os.Getenv("SNITCH_E2E_WORKERS") == "1" {
			suites = []string{"workers.spec.ts"}
		}
		if os.Getenv("SNITCH_E2E_CONTROLS") == "1" {
			suites = []string{"queue-controls.spec.ts"}
		}
		if os.Getenv("SNITCH_E2E_TRANSCRIPTION") == "1" {
			suites = []string{"transcription.spec.ts"}
		}
		if os.Getenv("SNITCH_E2E_BATCH_TRANSCRIPTIONS") == "1" {
			suites = []string{"batch-transcriptions.spec.ts"}
		}
		if os.Getenv("SNITCH_E2E_SEARCH") == "1" {
			suites = []string{"search.spec.ts"}
		}
		if os.Getenv("SNITCH_E2E_VOCABULARY") == "1" {
			suites = []string{"vocabulary.spec.ts"}
		}
		if os.Getenv("SNITCH_E2E_VOICE") == "1" {
			suites = []string{"voice.spec.ts"}
		}
		if os.Getenv("SNITCH_E2E_PERSONS") == "1" {
			suites = []string{"persons.spec.ts"}
		}
		if os.Getenv("SNITCH_E2E_MATCHING") == "1" {
			suites = []string{"matching.spec.ts"}
		}
		if os.Getenv("SNITCH_E2E_VOICE_PROFILE") == "1" {
			suites = []string{"voice-profile.spec.ts"}
		}
		if os.Getenv("SNITCH_E2E_AUTOMATION") == "1" {
			suites = []string{"automation.spec.ts"}
		}
		if os.Getenv("SNITCH_E2E_MICROPHONE") == "1" {
			suites = []string{"microphone.spec.ts"}
		}
		if os.Getenv("SNITCH_E2E_PERSON_FILTER") == "1" {
			suites = []string{"person-filter.spec.ts"}
		}
		if os.Getenv("SNITCH_E2E_VOICE_GROUPS") == "1" {
			suites = []string{"voice-groups.spec.ts"}
		}
		if os.Getenv("SNITCH_E2E_VOICE_VERIFICATION") == "1" {
			suites = []string{"voice-verification.spec.ts"}
		}
		if os.Getenv("SNITCH_E2E_PEOPLE") == "1" {
			suites = []string{"people.spec.ts"}
		}
		if os.Getenv("SNITCH_E2E_PERSON_CORRECTIONS") == "1" {
			suites = []string{"person-corrections.spec.ts", "segment-corrections.spec.ts"}
		}
		if os.Getenv("SNITCH_E2E_SEGMENT_CORRECTIONS") == "1" {
			suites = []string{"segment-corrections.spec.ts", "segment-conflicts.spec.ts"}
		}
		if os.Getenv("SNITCH_E2E_SEGMENT_REINDEX") == "1" {
			suites = []string{"segment-reindex.spec.ts"}
		}
		if os.Getenv("SNITCH_E2E_SEARCH_CORRECTIONS") == "1" {
			suites = []string{"search-corrections.spec.ts"}
		}
		if os.Getenv("SNITCH_E2E_SEGMENT_HISTORY") == "1" {
			suites = []string{"segment-history.spec.ts"}
		}
		if os.Getenv("SNITCH_E2E_SEGMENT_EXPORT") == "1" {
			suites = []string{"segment-export.spec.ts"}
		}
		if os.Getenv("SNITCH_E2E_SEGMENT_SPEAKER") == "1" {
			suites = []string{"segment-speaker.spec.ts"}
		}
		if os.Getenv("SNITCH_E2E_SEGMENT_RETRANSCRIPTION") == "1" {
			suites = []string{"segment-replace-corrections.spec.ts"}
		}
		for _, suite := range suites {
			t.Run("browser/"+suite, func(t *testing.T) {
				reset(t)
				code, e := setup.Issue(ctx, pool)
				if e != nil {
					t.Fatal(e)
				}
				config := config.Config{MasterKey: bytes.Repeat([]byte{1}, 32)}
				sourceRoot := ""
				if suite == "sources.spec.ts" || suite == "microphone.spec.ts" {
					sourceRoot = t.TempDir()
					if e = os.WriteFile(filepath.Join(sourceRoot, "browser-fixture.mp4"), []byte("synthetic-browser-video"), 0o400); e != nil {
						t.Fatal(e)
					}
					config.ClipRoots = []string{sourceRoot}
					config.UploadRoot = t.TempDir()
					config.ProbePath = syntheticProbeFixture(t)
					config.StabilityWindow = 10 * time.Millisecond
					config.ReconcileInterval = 100 * time.Millisecond
				}
				if suite == "uploads.spec.ts" || suite == "library.spec.ts" || suite == "jobs.spec.ts" || suite == "transcription.spec.ts" || suite == "batch-transcriptions.spec.ts" || suite == "search.spec.ts" || suite == "voice.spec.ts" || suite == "persons.spec.ts" || suite == "person-filter.spec.ts" || suite == "matching.spec.ts" || suite == "voice-groups.spec.ts" || suite == "automation.spec.ts" || suite == "voice-profile.spec.ts" || suite == "voice-verification.spec.ts" || suite == "person-corrections.spec.ts" || suite == "people.spec.ts" || suite == "segment-corrections.spec.ts" || suite == "segment-reindex.spec.ts" || suite == "segment-conflicts.spec.ts" || suite == "search-corrections.spec.ts" || suite == "segment-history.spec.ts" || suite == "segment-export.spec.ts" || suite == "segment-speaker.spec.ts" || suite == "segment-replace-corrections.spec.ts" {
					config.UploadRoot = t.TempDir()
					config.ProbePath = syntheticProbeFixture(t)
					config.UploadMaxBytes = 1 << 20
					config.UploadChunkBytes = 16 * 1024
					config.UploadSessionTTL = time.Hour
					config.FFmpegPath, _ = syntheticFFmpegFixture(t)
				}
				s := New(pool, config, slog.New(slog.NewJSONHandler(io.Discard, nil)))
				if suite == "sources.spec.ts" || suite == "matching.spec.ts" || suite == "automation.spec.ts" || suite == "microphone.spec.ts" || suite == "uploads.spec.ts" || suite == "library.spec.ts" || suite == "jobs.spec.ts" || suite == "transcription.spec.ts" || suite == "batch-transcriptions.spec.ts" || suite == "search.spec.ts" || suite == "voice.spec.ts" || suite == "persons.spec.ts" || suite == "person-filter.spec.ts" || suite == "voice-groups.spec.ts" || suite == "voice-profile.spec.ts" || suite == "voice-verification.spec.ts" || suite == "person-corrections.spec.ts" || suite == "people.spec.ts" || suite == "segment-corrections.spec.ts" || suite == "segment-reindex.spec.ts" || suite == "segment-conflicts.spec.ts" || suite == "search-corrections.spec.ts" || suite == "segment-history.spec.ts" || suite == "segment-export.spec.ts" || suite == "segment-speaker.spec.ts" || suite == "segment-replace-corrections.spec.ts" {
					if e = s.Start(ctx); e != nil {
						t.Fatal(e)
					}
					defer s.Close()
				}
				if suite == "discord-oauth.spec.ts" || suite == "account-settings.spec.ts" {
					browserDiscord(t, s)
				}
				ts := httptest.NewUnstartedServer(s.Handler())
				s.Config.PublicURL = "http://" + ts.Listener.Addr().String()
				ts.Start()
				defer ts.Close()
				args := []string{"run", "test:e2e", "--", suite}
				cmd := exec.Command("npm", args...)
				cmd.Dir = "../../../web"
				cmd.Env = append(os.Environ(), "SNITCH_TEST_URL="+ts.URL, "SNITCH_TEST_CODE="+code, "SNITCH_TEST_SOURCE_ROOT="+sourceRoot, "SNITCH_TEST_UPLOAD_CHUNK=16384")
				out, e := cmd.CombinedOutput()
				t.Log(string(out))
				if e != nil {
					t.Fatal(e)
				}
			})
		}
	}
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	t.Logf("Go test process heap in use %.1f MiB (NOT production RSS)", float64(mem.HeapInuse)/1024/1024)
	stats, e := exec.Command("docker", "stats", "--no-stream", "--format", "{{.MemUsage}}", pg.GetContainerID()).CombinedOutput()
	if e == nil {
		t.Log("PostgreSQL container at test end:", string(stats))
	}
}
