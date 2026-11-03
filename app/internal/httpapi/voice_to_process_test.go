//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

type toProcessClipT struct {
	ClipID    int64 `json:"clip_id"`
	ToConfirm int   `json:"to_confirm"`
	Ambiguous int   `json:"ambiguous"`
	Unknown   int   `json:"unknown"`
	Groups    int   `json:"groups"`

	Folder           string   `json:"folder"`
	RecordedAt       string   `json:"recorded_at"`
	DurationSeconds  *float64 `json:"duration_seconds"`
	AnalysedRecently bool     `json:"analysed_recently"`
}

type toProcessT struct {
	Items  []toProcessClipT `json:"items"`
	Total  int              `json:"total"`
	Totals struct {
		Clips     int `json:"clips"`
		ToConfirm int `json:"to_confirm"`
		Ambiguous int `json:"ambiguous"`
		Unknown   int `json:"unknown"`
		Groups    int `json:"groups"`
	} `json:"totals"`
}

func (h *voiceHarness) toProcess(u *voiceUser, query string) toProcessT {
	h.t.Helper()
	var v toProcessT
	if err := json.Unmarshal(h.userCall(u, http.MethodGet, "/api/people/to-process"+query, nil, http.StatusOK), &v); err != nil {
		h.t.Fatal(err)
	}
	return v
}

func (v toProcessT) byClip(id int64) (toProcessClipT, bool) {
	for _, c := range v.Items {
		if c.ClipID == id {
			return c, true
		}
	}
	return toProcessClipT{}, false
}

func testVoiceToProcess(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarnessClips(t, pool, 3)
	if _, err := pool.Exec(context.Background(), `UPDATE voice_matching_settings SET auto_enabled=false,automatic_threshold=0.90,suggestion_threshold=0.75,ambiguity_margin=0.05,max_candidates=5`); err != nil {
		t.Fatal(err)
	}
	clipA, clipB, clipC := h.clipIDs[0], h.clipIDs[1], h.clipIDs[2]
	admin := h.adminUser()
	alice := h.newUser("alice")

	h.validate(h.fixtureSpeaker(clipA, "REF_ALICE", usableAt(0)), "Alice")
	h.validate(h.fixtureSpeaker(clipA, "REF_BOB", usableAt(20)), "Bob")
	h.fixtureSpeaker(clipB, "AMBIGUOUS", usableAt(10))
	h.fixtureSpeaker(clipC, "SUGGESTED", usableAt(40))
	h.fixtureSpeaker(clipB, "U1", usableAt(120))
	h.fixtureSpeaker(clipC, "U2", usableAt(125))
	u4 := h.fixtureSpeaker(clipA, "U4", usableAt(122))
	u3 := h.fixtureSpeaker(clipA, "U3", usableAt(200))
	h.adminCall(http.MethodPost, "/api/admin/voice-matching/rematch", nil, http.StatusOK)

	// One row per clip with a voice to settle; validated reference voices count for nothing.
	all := h.toProcess(admin, "")
	if all.Total != 3 || len(all.Items) != 3 {
		t.Fatalf("three clips to process: %+v", all)
	}
	want := map[int64]toProcessClipT{
		clipA: {ToConfirm: 0, Ambiguous: 0, Unknown: 2, Groups: 1},
		clipB: {ToConfirm: 0, Ambiguous: 1, Unknown: 1, Groups: 2},
		clipC: {ToConfirm: 1, Ambiguous: 0, Unknown: 1, Groups: 2},
	}
	for id, w := range want {
		got, ok := all.byClip(id)
		if !ok || got.ToConfirm != w.ToConfirm || got.Ambiguous != w.Ambiguous || got.Unknown != w.Unknown || got.Groups != w.Groups {
			t.Fatalf("clip %d = %+v, want %+v", id, got, w)
		}
	}
	if all.Totals.Clips != 3 || all.Totals.ToConfirm != 1 || all.Totals.Ambiguous != 1 || all.Totals.Unknown != 4 {
		t.Fatalf("totals: %+v", all.Totals)
	}

	// Under the title: the folder of the clip, its recording date, its duration, a recent analysis.
	for _, q := range []string{
		`UPDATE source_files SET relative_path='2026/octobre/parking/a.mp4' WHERE clip_id=` + strconv.FormatInt(clipA, 10),
		`UPDATE clips SET duration_seconds=2292 WHERE id=` + strconv.FormatInt(clipA, 10),
		`INSERT INTO clip_analysis_runs(clip_id,scope,status,finished_at) VALUES(` + strconv.FormatInt(clipA, 10) + `,'full','completed',now())`,
	} {
		if _, err := pool.Exec(context.Background(), q); err != nil {
			t.Fatal(err)
		}
	}
	detail := h.toProcess(admin, "")
	a, _ := detail.byClip(clipA)
	if a.Folder != "2026/octobre/parking" || a.RecordedAt == "" || a.DurationSeconds == nil || *a.DurationSeconds != 2292 || !a.AnalysedRecently {
		t.Fatalf("clip A details: %+v", a)
	}
	if b, _ := detail.byClip(clipB); b.Folder != "" || b.AnalysedRecently {
		t.Fatalf("clip B has no folder and no recent analysis: %+v", b)
	}

	// Filters are applied by the database; the totals ignore them.
	if f := h.toProcess(admin, "?filter=to_confirm"); f.Total != 1 || f.Items[0].ClipID != clipC || f.Totals.Clips != 3 {
		t.Fatalf("to_confirm filter: %+v", f)
	}
	if f := h.toProcess(admin, "?filter=unknown_ambiguous"); f.Total != 3 {
		t.Fatalf("unknown_ambiguous filter: %+v", f)
	}
	if f := h.toProcess(admin, "?filter=group"); f.Total != 3 {
		t.Fatalf("group filter: %+v", f)
	}
	if f := h.toProcess(admin, "?page=2&page_size=2"); f.Total != 3 || len(f.Items) != 1 {
		t.Fatalf("pagination: %+v", f)
	}
	for _, q := range []string{"?filter=nope", "?page=0", "?page_size=101"} {
		if status, _ := h.userStatus(admin, http.MethodGet, "/api/people/to-process"+q, nil); status != http.StatusBadRequest {
			t.Fatalf("%s must be refused, got %d", q, status)
		}
	}

	// A hidden clip is neither listed nor counted.
	if _, err := pool.Exec(context.Background(), `UPDATE clips SET visibility='private' WHERE id=$1`, clipC); err != nil {
		t.Fatal(err)
	}
	seen := h.toProcess(alice, "")
	if _, leaked := seen.byClip(clipC); leaked || seen.Total != 2 || seen.Totals.ToConfirm != 0 || seen.Totals.Unknown != 3 {
		t.Fatalf("hidden clip leaked: %+v", seen)
	}
	if _, err := pool.Exec(context.Background(), `UPDATE clips SET visibility='shared' WHERE id=$1`, clipC); err != nil {
		t.Fatal(err)
	}

	// Ignored and Random voices do not count: a clip with nothing left leaves the list,
	// and restoring a voice brings it back.
	h.setDisposition(u4, "ignored", http.StatusOK)
	h.setDisposition(u3, "random", http.StatusOK)
	after := h.toProcess(admin, "")
	if _, still := after.byClip(clipA); still || after.Total != 2 || after.Totals.Unknown != 2 {
		t.Fatalf("clip A must leave the list: %+v", after)
	}
	h.restoreDisposition(u4)
	if back := h.toProcess(admin, ""); back.Total != 3 {
		t.Fatalf("restored voice must bring its clip back: %+v", back)
	}
}
