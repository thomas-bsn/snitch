//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// syntheticFFmpegFixture stands in for ffmpeg: it records its arguments and prints a
// fixed synthetic WAV-looking payload. No real media or codec is involved.
func syntheticFFmpegFixture(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	log := filepath.Join(dir, "ffmpeg.log")
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> '" + strings.ReplaceAll(log, "'", "'\\''") + "'\nprintf 'RIFF....WAVEsynthetic-excerpt'\n"
	path := filepath.Join(dir, "ffmpeg-fixture")
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	return path, log
}

type voiceUser struct {
	name   string
	client *http.Client
	csrf   string
}

// newUser registers and signs in a regular (non administrator) user.
func (h *voiceHarness) newUser(name string) *voiceUser {
	h.t.Helper()
	if _, err := h.pool.Exec(context.Background(), `UPDATE installation SET registration_approval=false`); err != nil {
		h.t.Fatal(err)
	}
	jar, _ := cookiejar.New(nil)
	u := &voiceUser{name: name, client: &http.Client{Jar: jar}}
	credentials := map[string]string{"username": name, "password": "synthetic-password"}
	h.userCall(u, http.MethodPost, "/api/auth/register", credentials, http.StatusCreated)
	h.userCall(u, http.MethodPost, "/api/auth/login", credentials, http.StatusOK)
	var me struct {
		CSRF string `json:"csrf_token"`
	}
	if err := json.Unmarshal(h.userCall(u, http.MethodGet, "/api/auth/me", nil, http.StatusOK), &me); err != nil {
		h.t.Fatal(err)
	}
	u.csrf = me.CSRF
	return u
}

func (h *voiceHarness) userCall(u *voiceUser, method, path string, body any, want int) []byte {
	h.t.Helper()
	status, data := h.userStatus(u, method, path, body)
	if status != want {
		h.t.Fatalf("%s %s as %s: got %d want %d: %s", method, path, u.name, status, want, data)
	}
	return data
}

func (h *voiceHarness) userStatus(u *voiceUser, method, path string, body any) (int, []byte) {
	h.t.Helper()
	var payload []byte
	if body != nil {
		payload = mustJSON(h.t, body)
	}
	req, _ := http.NewRequest(method, h.ts.URL+path, bytes.NewReader(payload))
	req.Header.Set("Origin", h.ts.URL)
	req.Header.Set("Content-Type", "application/json")
	if u.csrf != "" {
		req.Header.Set("X-CSRF-Token", u.csrf)
	}
	resp, err := u.client.Do(req)
	if err != nil {
		h.t.Fatal(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, data
}

func (h *voiceHarness) adminUser() *voiceUser {
	return &voiceUser{name: "administrator", client: h.admin, csrf: h.csrf}
}

type attributionResponse struct {
	ID              int64  `json:"id"`
	LocalSpeakerID  int64  `json:"local_speaker_id"`
	DecisionLevel   string `json:"decision_level"`
	ValidationState string `json:"validation_state"`
	Revision        int    `json:"revision"`
	Person          *struct {
		ID          int64  `json:"id"`
		DisplayName string `json:"display_name"`
	} `json:"person"`
	Author *struct {
		Username string `json:"username"`
	} `json:"author"`
}

type conflictResponse struct {
	Error       string               `json:"error"`
	Attribution *attributionResponse `json:"attribution"`
}

// diarizeClip runs the synthetic worker through diarization and fingerprints and
// returns the ids of SPEAKER_00 and SPEAKER_01.
func (h *voiceHarness) diarizeClip() (int64, int64) {
	h.t.Helper()
	h.transcribe()
	h.putPipeline(fakePipeline(2))
	h.adminCall(http.MethodPost, fmt.Sprintf("/api/admin/clips/%d/voice", h.clipID), map[string]any{"audio_stream_index": 2}, http.StatusCreated)
	job, ok := h.claim()
	if !ok || job.Step != "voice.diarization.2" {
		h.t.Fatalf("expected diarization, got %+v", job)
	}
	h.complete(job, diarizationPayload("SPEAKER_00", "SPEAKER_01"), http.StatusOK)
	fp, ok := h.claim()
	if !ok || fp.Step != "voice.fingerprint.2" {
		h.t.Fatalf("expected fingerprints, got %+v", fp)
	}
	vec := []float64{0.5, 0.5, 0.5, 0.5}
	items := []map[string]any{}
	for i, label := range []string{"SPEAKER_00", "SPEAKER_01"} {
		items = append(items, map[string]any{"speaker_label": label, "scope": "speaker", "start": float64(i), "end": float64(i) + 1.5, "quality": 0.9, "embedding": vec})
	}
	h.complete(fp, map[string]any{"provider": "fake", "model": "fixture-diarization", "model_version": "fixture-v1", "embedding_model": "fixture-embedding", "embedding_model_version": "emb-v1", "audio_stream_index": 2, "diarization_job_id": job.ID, "dimensions": 4, "fingerprints": items}, http.StatusOK)
	view := h.voice()
	if len(view.Speakers) != 2 {
		h.t.Fatalf("expected two local speakers: %+v", view)
	}
	return view.Speakers[0].ID, view.Speakers[1].ID
}

func (h *voiceHarness) count(query string, args ...any) int {
	h.t.Helper()
	var n int
	if err := h.pool.QueryRow(context.Background(), query, args...).Scan(&n); err != nil {
		h.t.Fatal(err)
	}
	return n
}

func testPersonValidation(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	first, second := h.diarizeClip()
	speakerPath := func(id int64, leaf string) string { return fmt.Sprintf("/api/local-speakers/%d/%s", id, leaf) }
	admin := h.adminUser()
	alice := h.newUser("alice")
	bob := h.newUser("bob")

	// Excerpts and validations need a signed-in user; clip visibility is enforced.
	for _, route := range [][2]string{{http.MethodGet, speakerPath(first, "excerpt")}, {http.MethodGet, speakerPath(first, "history")}, {http.MethodPost, speakerPath(first, "validation")}, {http.MethodGet, "/api/persons"}} {
		anon := &voiceUser{name: "anonymous", client: &http.Client{}}
		if status, _ := h.userStatus(anon, route[0], route[1], nil); status != http.StatusUnauthorized {
			t.Fatalf("anonymous %s %s: %d", route[0], route[1], status)
		}
	}
	if _, err := pool.Exec(context.Background(), `UPDATE clips SET visibility='private' WHERE id=$1`, h.clipID); err != nil {
		t.Fatal(err)
	}
	for _, route := range [][2]string{{http.MethodGet, speakerPath(first, "excerpt")}, {http.MethodGet, speakerPath(first, "history")}} {
		h.userCall(alice, route[0], route[1], nil, http.StatusNotFound)
	}
	h.userCall(alice, http.MethodPost, speakerPath(first, "validation"), map[string]any{"new_person_name": "Hidden"}, http.StatusNotFound)
	h.userCall(alice, http.MethodGet, fmt.Sprintf("/api/clips/%d/voice", h.clipID), nil, http.StatusNotFound)
	if h.count(`SELECT count(*) FROM persons`) != 0 {
		t.Fatal("a refused validation must not create a person")
	}
	h.userCall(admin, http.MethodGet, speakerPath(first, "excerpt"), nil, http.StatusOK) // the owner and administrators still hear it
	if _, err := pool.Exec(context.Background(), `UPDATE clips SET visibility='shared' WHERE id=$1`, h.clipID); err != nil {
		t.Fatal(err)
	}
	body := h.userCall(alice, http.MethodGet, speakerPath(first, "excerpt"), nil, http.StatusOK)
	if !bytes.HasPrefix(body, []byte("RIFF")) {
		t.Fatalf("excerpt is not audio: %q", body)
	}
	log, _ := os.ReadFile(h.ffmpegLog)
	if !strings.Contains(string(log), "-ss 0.000 -t 1.500") || !strings.Contains(string(log), "-map 0:2") {
		t.Fatalf("excerpt must cut the speaker's turn from the selected audio stream: %s", log)
	}
	h.userCall(alice, http.MethodGet, speakerPath(first, "excerpt")+"?turn=9", nil, http.StatusNotFound)
	h.userCall(alice, http.MethodGet, "/api/local-speakers/999999/excerpt", nil, http.StatusNotFound)

	// Every speaker starts unknown and no sample exists.
	view := h.voiceAs(alice)
	if view.Speakers[0].DecisionLevel != "unknown" || view.Speakers[0].State != "unidentified" || view.Speakers[0].Person != nil || view.Speakers[0].Attribution != nil {
		t.Fatalf("a voice without attribution must be unknown: %+v", view.Speakers[0])
	}

	// Input validation.
	h.userCall(alice, http.MethodPost, speakerPath(first, "validation"), map[string]any{}, http.StatusBadRequest)
	h.userCall(alice, http.MethodPost, speakerPath(first, "validation"), map[string]any{"person_id": 1, "new_person_name": "Both"}, http.StatusBadRequest)
	h.userCall(alice, http.MethodPost, speakerPath(first, "validation"), map[string]any{"new_person_name": "   "}, http.StatusBadRequest)
	h.userCall(alice, http.MethodPost, speakerPath(first, "validation"), map[string]any{"person_id": 424242}, http.StatusNotFound)
	h.userCall(alice, http.MethodPost, speakerPath(first, "validation"), map[string]any{"new_person_name": "Early", "expected_person_id": 1}, http.StatusBadRequest)
	h.userCall(alice, http.MethodPost, speakerPath(first, "correction"), map[string]any{"new_person_name": "Early"}, http.StatusBadRequest)
	h.userCall(alice, http.MethodPost, speakerPath(first, "correction"), map[string]any{"new_person_name": "Early", "expected_person_id": 1}, http.StatusConflict) // not validated yet
	if h.count(`SELECT count(*) FROM persons`) != 0 || h.count(`SELECT count(*) FROM voice_samples`) != 0 {
		t.Fatal("refused requests left persons or samples behind")
	}

	// An automatic attribution from another origin (here a synthetic one: matching
	// itself is covered by the voice matching suite, and never overwrites foreign
	// origins) is not a sample and never blocks a human validation, even at high confidence.
	var created struct {
		ID          int64  `json:"id"`
		DisplayName string `json:"display_name"`
	}
	if err := json.Unmarshal(h.userCall(alice, http.MethodPost, "/api/persons", map[string]any{"display_name": "  Carol  "}, http.StatusCreated), &created); err != nil || created.DisplayName != "Carol" {
		t.Fatalf("person creation must trim the name: %+v %v", created, err)
	}
	carol := created.ID
	if _, err := pool.Exec(context.Background(), `INSERT INTO speaker_attributions(local_speaker_id,person_id,decision_level,validation_state,confidence,origin) VALUES($1,$2,'automatic','unvalidated',0.97,'synthetic_origin')`, second, carol); err != nil {
		t.Fatal(err)
	}
	view = h.voiceAs(alice)
	if s := view.Speakers[1]; s.DecisionLevel != "automatic" || s.State != "identified" || s.Person == nil || s.Person.DisplayName != "Carol" || s.Attribution.ValidationState != "unvalidated" {
		t.Fatalf("automatic attribution must be shown as such: %+v", s)
	}
	if h.count(`SELECT count(*) FROM voice_samples`) != 0 {
		t.Fatal("an automatic attribution must never become a validated sample")
	}

	// A diarization already leased before the first validation cannot replace the
	// validated speakers afterwards.
	h.putPipeline(fakePipeline(1))
	h.adminCall(http.MethodPost, fmt.Sprintf("/api/admin/clips/%d/voice", h.clipID), map[string]any{"audio_stream_index": 2}, http.StatusCreated)
	rerun, ok := h.claim()
	if !ok || rerun.Step != "voice.diarization.2" {
		t.Fatalf("expected the replanned diarization: %+v", rerun)
	}

	// Concurrent first validations of the same local speaker: one winner chosen by
	// PostgreSQL, losers see the existing attribution and its author, and create nothing.
	users := []*voiceUser{alice, bob, admin}
	const contenders = 9
	type outcome struct {
		status int
		body   []byte
	}
	outcomes := make([]outcome, contenders)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < contenders; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			status, data := h.userStatus(users[i%len(users)], http.MethodPost, speakerPath(first, "validation"), map[string]any{"new_person_name": fmt.Sprintf("Contender %d", i)})
			outcomes[i] = outcome{status, data}
		}(i)
	}
	close(start)
	wg.Wait()
	winners := 0
	var winner attributionResponse
	for _, o := range outcomes {
		switch o.status {
		case http.StatusCreated:
			winners++
			if err := json.Unmarshal(o.body, &winner); err != nil {
				t.Fatal(err)
			}
		case http.StatusConflict:
		default:
			t.Fatalf("unexpected concurrent outcome %d: %s", o.status, o.body)
		}
	}
	if winners != 1 || winner.DecisionLevel != "human" || winner.ValidationState != "validated" || winner.Person == nil || winner.Author == nil {
		t.Fatalf("exactly one validation must win: %d %+v", winners, winner)
	}
	for _, o := range outcomes {
		if o.status != http.StatusConflict {
			continue
		}
		var lost conflictResponse
		if err := json.Unmarshal(o.body, &lost); err != nil || lost.Error != "already_validated" || lost.Attribution == nil || lost.Attribution.ID != winner.ID || lost.Attribution.Person.ID != winner.Person.ID || lost.Attribution.Author == nil || lost.Attribution.Author.Username != winner.Author.Username {
			t.Fatalf("a losing validation must see the existing attribution and its author: %s", o.body)
		}
	}
	if h.count(`SELECT count(*) FROM persons WHERE display_name LIKE 'Contender %'`) != 1 || h.count(`SELECT count(*) FROM voice_samples WHERE local_speaker_id=$1`, first) != 1 || h.count(`SELECT count(*) FROM attribution_events WHERE local_speaker_id=$1`, first) != 1 {
		t.Fatal("losing validations must leave no person, sample or history entry")
	}
	if status, _ := h.userStatus(bob, http.MethodPost, speakerPath(first, "validation"), map[string]any{"person_id": carol}); status != http.StatusConflict {
		t.Fatalf("a later validation also loses: %d", status)
	}

	// The shared view marks the validated voice as human and keeps the unknown one apart.
	view = h.voiceAs(bob)
	if s := view.Speakers[0]; s.DecisionLevel != "human" || s.State != "identified" || s.Person == nil || s.Person.ID != winner.Person.ID || s.Attribution.Author == nil {
		t.Fatalf("validated voice not shown as human: %+v", s)
	}
	var detail struct {
		Transcription struct {
			Segments []struct {
				PersonName *string `json:"local_speaker_person_name"`
				Level      *string `json:"local_speaker_decision_level"`
			} `json:"segments"`
		} `json:"transcription"`
	}
	if err := json.Unmarshal(h.userCall(bob, http.MethodGet, fmt.Sprintf("/api/clips/%d", h.clipID), nil, http.StatusOK), &detail); err != nil {
		t.Fatal(err)
	}
	segments := detail.Transcription.Segments
	if segments[0].PersonName == nil || *segments[0].PersonName != winner.Person.DisplayName || *segments[0].Level != "human" || segments[1].PersonName == nil || *segments[1].Level != "automatic" || segments[2].Level != nil {
		t.Fatalf("segments must carry the person and decision level of their speaker: %+v", segments)
	}

	// A human validation pins the speakers: a new diarization is refused, by schedule and by result.
	h.putPipeline(fakePipeline(3))
	h.adminCall(http.MethodPost, fmt.Sprintf("/api/admin/clips/%d/voice", h.clipID), map[string]any{"audio_stream_index": 2}, http.StatusConflict)
	if body = h.complete(rerun, diarizationPayload("SOLO"), http.StatusConflict); !bytes.Contains(body, []byte("diarization_locked")) {
		t.Fatalf("a diarization over validated speakers must be refused: %s", body)
	}
	if after := h.voice(); len(after.Speakers) != 2 || after.Speakers[0].ID != first || after.Speakers[0].DecisionLevel != "human" {
		t.Fatalf("validated speakers were replaced: %+v", after)
	}
	if _, err := pool.Exec(context.Background(), `DELETE FROM local_speakers WHERE id=$1`, first); err == nil {
		t.Fatal("the database must refuse deleting a validated local speaker while its clip exists")
	}

	// Validating over an automatic attribution keeps what the machine said in the history.
	other := h.userCall(bob, http.MethodPost, speakerPath(second, "validation"), map[string]any{"person_id": carol}, http.StatusCreated)
	var confirmed attributionResponse
	if err := json.Unmarshal(other, &confirmed); err != nil || confirmed.Person.ID != carol || confirmed.DecisionLevel != "human" {
		t.Fatalf("validation of a voice that matched automatically: %s", other)
	}
	if h.count(`SELECT count(*) FROM speaker_attributions WHERE local_speaker_id=$1`, second) != 2 || h.count(`SELECT count(*) FROM voice_samples WHERE local_speaker_id=$1 AND retired_at IS NULL`, second) != 1 {
		t.Fatal("the machine row stays distinct from the human validation, which creates one sample")
	}

	// Explicit correction: stale expectation refused, same person refused, then applied.
	wrong := h.userCall(alice, http.MethodPost, speakerPath(first, "correction"), map[string]any{"person_id": carol, "expected_person_id": carol}, http.StatusConflict)
	var changed conflictResponse
	if err := json.Unmarshal(wrong, &changed); err != nil || changed.Error != "attribution_changed" || changed.Attribution.Person.ID != winner.Person.ID {
		t.Fatalf("a correction based on stale data must show the current attribution: %s", wrong)
	}
	h.userCall(alice, http.MethodPost, speakerPath(first, "correction"), map[string]any{"person_id": winner.Person.ID, "expected_person_id": winner.Person.ID}, http.StatusConflict)
	var corrected attributionResponse
	if err := json.Unmarshal(h.userCall(bob, http.MethodPost, speakerPath(first, "correction"), map[string]any{"person_id": carol, "expected_person_id": winner.Person.ID}, http.StatusOK), &corrected); err != nil {
		t.Fatal(err)
	}
	if corrected.ID != winner.ID || corrected.Person.ID != carol || corrected.Revision != 2 || corrected.Author.Username != "bob" {
		t.Fatalf("correction result: %+v", corrected)
	}
	if h.count(`SELECT count(*) FROM voice_samples WHERE local_speaker_id=$1`, first) != 2 ||
		h.count(`SELECT count(*) FROM voice_samples WHERE local_speaker_id=$1 AND retired_at IS NULL AND person_id=$2`, first, carol) != 1 ||
		h.count(`SELECT count(*) FROM voice_samples WHERE local_speaker_id=$1 AND retired_at IS NOT NULL AND person_id=$2 AND retired_reason='corrected'`, first, winner.Person.ID) != 1 {
		t.Fatal("a correction must retire the old sample from the references and create a new one")
	}
	var people struct {
		Items []struct {
			ID                   int64  `json:"id"`
			DisplayName          string `json:"display_name"`
			ValidatedSampleCount int    `json:"validated_sample_count"`
			LocalSpeakerCount    int    `json:"local_speaker_count"`
		} `json:"items"`
	}
	if err := json.Unmarshal(h.userCall(alice, http.MethodGet, "/api/persons", nil, http.StatusOK), &people); err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, p := range people.Items {
		counts[p.DisplayName] = p.ValidatedSampleCount
	}
	if counts["Carol"] != 2 || counts[winner.Person.DisplayName] != 0 {
		t.Fatalf("validated sample counts after correction: %+v", people.Items)
	}
	if err := json.Unmarshal(h.userCall(alice, http.MethodGet, "/api/persons?q=carol", nil, http.StatusOK), &people); err != nil || len(people.Items) != 1 || people.Items[0].LocalSpeakerCount != 2 {
		t.Fatalf("person search: %+v %v", people.Items, err)
	}
	h.userCall(alice, http.MethodGet, fmt.Sprintf("/api/persons/%d", carol), nil, http.StatusOK)
	h.userCall(alice, http.MethodGet, "/api/persons/424242", nil, http.StatusNotFound)

	// The history keeps old value, new value, author and date, and is immutable.
	var history struct {
		Items []struct {
			Action    string `json:"action"`
			OldPerson *struct {
				ID int64 `json:"id"`
			} `json:"old_person"`
			OldLevel  string `json:"old_decision_level"`
			NewPerson *struct {
				ID int64 `json:"id"`
			} `json:"new_person"`
			Author    *struct{ Username string } `json:"author"`
			CreatedAt string                     `json:"created_at"`
		} `json:"items"`
	}
	if err := json.Unmarshal(h.userCall(alice, http.MethodGet, speakerPath(first, "history"), nil, http.StatusOK), &history); err != nil {
		t.Fatal(err)
	}
	if len(history.Items) != 2 {
		t.Fatalf("history: %+v", history)
	}
	latest, earliest := history.Items[0], history.Items[1]
	if latest.Action != "correct" || latest.OldPerson == nil || latest.OldPerson.ID != winner.Person.ID || latest.NewPerson == nil || latest.NewPerson.ID != carol || latest.Author == nil || latest.Author.Username != "bob" || latest.CreatedAt == "" || latest.OldLevel != "human" {
		t.Fatalf("correction history entry: %+v", latest)
	}
	if earliest.Action != "validate" || earliest.OldPerson != nil || earliest.OldLevel != "unknown" || earliest.NewPerson.ID != winner.Person.ID {
		t.Fatalf("validation history entry: %+v", earliest)
	}
	if err := json.Unmarshal(h.userCall(alice, http.MethodGet, speakerPath(second, "history"), nil, http.StatusOK), &history); err != nil || len(history.Items) != 1 || history.Items[0].OldLevel != "automatic" || history.Items[0].OldPerson == nil || history.Items[0].OldPerson.ID != carol {
		t.Fatalf("validating over an automatic attribution must record what the machine said: %+v %v", history, err)
	}
	if _, err := pool.Exec(context.Background(), `UPDATE attribution_events SET author_username='tampered'`); err == nil {
		t.Fatal("history must be immutable")
	}
	if _, err := pool.Exec(context.Background(), `DELETE FROM attribution_events`); err == nil {
		t.Fatal("history must not be deletable")
	}

	// Deleting the clip erases derived data, history included, and keeps Persons.
	if _, err := pool.Exec(context.Background(), `DELETE FROM source_files WHERE clip_id=$1`, h.clipID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `DELETE FROM clips WHERE id=$1`, h.clipID); err != nil {
		t.Fatalf("clip deletion must cascade through validated speakers: %v", err)
	}
	if h.count(`SELECT count(*) FROM speaker_attributions`)+h.count(`SELECT count(*) FROM voice_samples`)+h.count(`SELECT count(*) FROM attribution_events`) != 0 || h.count(`SELECT count(*) FROM persons`) < 2 {
		t.Fatal("clip deletion must erase attributions, samples and history but not Persons")
	}
}

func (h *voiceHarness) voiceAs(u *voiceUser) voiceView {
	h.t.Helper()
	var v voiceView
	if err := json.Unmarshal(h.userCall(u, http.MethodGet, fmt.Sprintf("/api/clips/%d/voice", h.clipID), nil, http.StatusOK), &v); err != nil {
		h.t.Fatal(err)
	}
	return v
}
