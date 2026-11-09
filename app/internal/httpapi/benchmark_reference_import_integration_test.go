//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// External references (#287, ADR 0061): a hand-written STM and RTTM replace the production reference of a test clip.
func testAIBenchmarkReferenceImport(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	h := newVoiceHarnessClips(t, pool, 3)
	clip, twoTracks, plain := h.clipIDs[0], h.clipIDs[1], h.clipIDs[2]
	// A production transcript on one track, with a validated voice; a clip transcribed on two tracks; a clip with nothing.
	if _, err := pool.Exec(ctx, `INSERT INTO jobs(clip_id,step,input_version,status,result) VALUES($1,'transcription.audio.9','p','done',$2::jsonb)`, clip,
		`{"provider":"fixture","model":"m","version":"v","audio_stream_index":9,"segments":[{"start":0,"end":5,"text":"texte de production","words":[]}]}`); err != nil {
		t.Fatal(err)
	}
	for _, track := range []int{1, 2} {
		if _, err := pool.Exec(ctx, `INSERT INTO jobs(clip_id,step,input_version,status,result) VALUES($1,$2,'p','done',$3::jsonb)`, twoTracks, fmt.Sprintf("transcription.audio.%d", track),
			fmt.Sprintf(`{"provider":"fixture","model":"m","version":"v","audio_stream_index":%d,"segments":[{"start":0,"end":5,"text":"x","words":[]}]}`, track)); err != nil {
			t.Fatal(err)
		}
	}
	datasets := "/api/admin/benchmarks/datasets"
	const stm = ";; fixture\nclip 1 alice 0.0 5.0 <o,f0,female> Bonjour tout le monde\nclip 1 bob 5.0 10.0 salut Alice\n"
	const rttm = "SPEAKER clip 1 0.0 5.0 <NA> <NA> alice <NA> <NA>\nSPEAKER clip 1 5.0 5.0 <NA> <NA> bob <NA> <NA>\n"
	corpus := func(name string, tests []int64, imports ...map[string]any) map[string]any {
		return map[string]any{"name": name, "test_clip_ids": tests, "reference_clip_ids": []int64{}, "verified_clip_ids": []int64{}, "reference_imports": imports}
	}
	imp := func(id int64, stm, rttm string) map[string]any {
		m := map[string]any{"clip_id": id}
		if stm != "" {
			m["stm"] = stm
		}
		if rttm != "" {
			m["rttm"] = rttm
		}
		return m
	}
	refuse := func(name string, status int, code string, tests []int64, imports ...map[string]any) {
		t.Helper()
		body := h.adminCall(http.MethodPost, datasets, corpus(name, tests, imports...), status)
		if !strings.Contains(string(body), `"`+code+`"`) {
			t.Fatalf("%s: want %s, got %s", name, code, body)
		}
	}
	refuse("Format", http.StatusUnprocessableEntity, "benchmark_import_format", []int64{clip}, imp(clip, "pas un stm", ""))
	refuse("Temps", http.StatusUnprocessableEntity, "benchmark_import_times", []int64{clip}, imp(clip, "", "SPEAKER clip 1 0 -3 <NA> <NA> alice <NA> <NA>"))
	refuse("Locuteurs", http.StatusUnprocessableEntity, "benchmark_import_speakers", []int64{clip}, imp(clip, stm, strings.ReplaceAll(strings.ReplaceAll(rttm, "alice", "carol"), "bob", "dave")))
	refuse("Taille", http.StatusUnprocessableEntity, "benchmark_import_size", []int64{clip}, imp(clip, strings.Repeat("a", 2<<20+1), ""))
	refuse("Piste ambiguë", http.StatusUnprocessableEntity, "benchmark_import_track", []int64{twoTracks}, imp(twoTracks, stm, ""))
	refuse("Sans piste", http.StatusUnprocessableEntity, "benchmark_import_track", []int64{plain}, imp(plain, stm, ""))
	refuse("Clip hors test", http.StatusUnprocessableEntity, "benchmark_import_clip", []int64{clip}, imp(plain, stm, ""))
	refuse("Deux fois", http.StatusUnprocessableEntity, "benchmark_import_clip", []int64{clip}, imp(clip, stm, ""), imp(clip, "", rttm))
	refuse("Vide", http.StatusUnprocessableEntity, "benchmark_import_format", []int64{clip}, imp(clip, "", ""))

	var created struct {
		ID     int64    `json:"id"`
		Stages []string `json:"stages"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodPost, datasets, corpus("Importé", []int64{clip}, imp(clip, stm, rttm)), http.StatusCreated), &created); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(created.Stages) != "[transcription diarization e2e]" {
		t.Fatalf("an imported reference certifies the clip and measures the three stages, got %v", created.Stages)
	}
	var text, speaker string
	if err := pool.QueryRow(ctx, `SELECT text,speaker FROM benchmark_reference_segments WHERE dataset_id=$1 AND ordinal=0 AND audio_stream_index=9 AND NOT corrected`, created.ID).Scan(&text, &speaker); err != nil || text != "Bonjour tout le monde" || speaker != "alice" {
		t.Fatalf("the reference text is the import, on the production track: %q %q %v", text, speaker, err)
	}
	if n := h.count(`SELECT count(*) FROM benchmark_reference_segments WHERE dataset_id=$1`, created.ID); n != 2 {
		t.Fatalf("only the imported segments are frozen, got %d", n)
	}
	if n := h.count(`SELECT count(*) FROM benchmark_reference_turns WHERE dataset_id=$1 AND audio_stream_index=9 AND speaker IN ('alice','bob')`, created.ID); n != 2 {
		t.Fatalf("the reference turns are the import, got %d", n)
	}
	if n := h.count(`SELECT count(*) FROM benchmark_dataset_clips WHERE dataset_id=$1 AND verified AND reference_source='import'`, created.ID); n != 1 {
		t.Fatalf("the source is remembered per clip, got %d", n)
	}

	// A text alone: no turns, no diarization; the speaker comes from the file.
	var textOnly struct {
		ID     int64    `json:"id"`
		Stages []string `json:"stages"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodPost, datasets, corpus("Texte seul", []int64{clip}, imp(clip, stm, "")), http.StatusCreated), &textOnly); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(textOnly.Stages) != "[transcription]" || h.count(`SELECT count(*) FROM benchmark_reference_turns WHERE dataset_id=$1`, textOnly.ID) != 0 {
		t.Fatalf("a text alone measures the transcription only, got %v", textOnly.Stages)
	}

	// A production reference keeps its source, and the list tells which sources a corpus relies on.
	if err := pool.QueryRow(ctx, `SELECT 1`).Scan(new(int)); err != nil {
		t.Fatal(err)
	}
	var production struct {
		ID int64 `json:"id"`
	}
	body := map[string]any{"name": "Production", "test_clip_ids": []int64{clip}, "reference_clip_ids": []int64{}, "verified_clip_ids": []int64{clip}}
	if err := json.Unmarshal(h.adminCall(http.MethodPost, datasets, body, http.StatusCreated), &production); err != nil {
		t.Fatal(err)
	}
	var list struct {
		Datasets []struct {
			ID               int64    `json:"id"`
			ReferenceSources []string `json:"reference_sources"`
		} `json:"datasets"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, datasets, nil, http.StatusOK), &list); err != nil {
		t.Fatal(err)
	}
	sources := map[int64]string{}
	for _, d := range list.Datasets {
		sources[d.ID] = fmt.Sprint(d.ReferenceSources)
	}
	if sources[created.ID] != "[import]" || sources[production.ID] != "[production]" {
		t.Fatalf("the corpus list tells the reference sources: %v", sources)
	}
}
