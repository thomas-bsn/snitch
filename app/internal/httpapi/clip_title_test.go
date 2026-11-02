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

type clipMeta struct {
	OriginalName string  `json:"original_name"`
	DisplayTitle *string `json:"display_title"`
	Game         *string `json:"game"`
	GameOrigin   *string `json:"game_origin"`
	RecordedAt   *string `json:"recorded_at"`
	Detected     *struct {
		Game       *string `json:"game"`
		RecordedAt *string `json:"recorded_at"`
	} `json:"detected"`
}

// testClipDisplayTitle: the display title is an optional label kept apart from the physical file name, edited by
// administrators only, validated, shown wherever the clip is named, and never lost to a rescan.
func testClipDisplayTitle(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	h := newVoiceHarness(t, pool)
	path := fmt.Sprintf("/api/admin/clips/%d/metadata", h.clipID)
	get := func() clipMeta {
		t.Helper()
		var c clipMeta
		if err := json.Unmarshal(h.adminCall(http.MethodGet, fmt.Sprintf("/api/clips/%d", h.clipID), nil, http.StatusOK), &c); err != nil {
			t.Fatal(err)
		}
		return c
	}
	before := get()
	if before.DisplayTitle != nil {
		t.Fatalf("no title by default: %+v", before)
	}

	// Only administrators edit; members are refused by the API itself.
	member := h.newUser("title-member")
	h.userCall(member, http.MethodPut, path, map[string]any{"display_title": "x", "display_title_changed": true}, http.StatusForbidden)

	// A Unicode title is stored and shown; the physical name and technical data stay untouched.
	title := "Clutch 1v4 — Mirage 🎮 é"
	h.adminCall(http.MethodPut, path, map[string]any{"display_title": "  " + title + "  ", "display_title_changed": true}, http.StatusOK)
	after := get()
	if after.DisplayTitle == nil || *after.DisplayTitle != title || after.OriginalName != before.OriginalName {
		t.Fatalf("title not kept apart from the file name: %+v", after)
	}
	var fromList struct {
		Items []clipMeta `json:"items"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/clips", nil, http.StatusOK), &fromList); err != nil || len(fromList.Items) == 0 || fromList.Items[0].DisplayTitle == nil || *fromList.Items[0].DisplayTitle != title {
		t.Fatalf("library list lacks the title: %+v %v", fromList, err)
	}

	// Invalid titles are refused and change nothing.
	for name, bad := range map[string]string{"control": "line\nbreak", "tab": "a\tb", "too long": strings.Repeat("é", 201), "nul": "a\x00b", "bidi override": "abc\u202edef", "separator": "a\u2028b"} {
		if got := string(h.adminCall(http.MethodPut, path, map[string]any{"display_title": bad, "display_title_changed": true}, http.StatusBadRequest)); !strings.Contains(got, "invalid_clip_title") {
			t.Fatalf("%s: %s", name, got)
		}
	}
	if got := get(); got.DisplayTitle == nil || *got.DisplayTitle != title {
		t.Fatalf("a refused title changed the clip: %+v", got)
	}
	// 200 characters (not bytes) is accepted.
	long := strings.Repeat("é", 200)
	h.adminCall(http.MethodPut, path, map[string]any{"display_title": long, "display_title_changed": true}, http.StatusOK)

	// Every view that names a clip uses the title, with the file name as fallback.
	h.adminCall(http.MethodPut, path, map[string]any{"display_title": title, "display_title_changed": true}, http.StatusOK)
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`DELETE FROM notifications`)
	exec(`INSERT INTO notifications(kind,clip_id,state) VALUES('needs_speaker_review',$1,'active')`, h.clipID)
	exec(`INSERT INTO notification_recipients(notification_id,user_id) SELECT id,(SELECT id FROM users WHERE role IN ('admin','super_admin') ORDER BY id LIMIT 1) FROM notifications`)
	notificationClip := func() string {
		t.Helper()
		var list struct {
			Items []struct {
				ClipName string `json:"clip_name"`
			} `json:"items"`
		}
		if err := json.Unmarshal(h.adminCall(http.MethodGet, "/api/notifications", nil, http.StatusOK), &list); err != nil || len(list.Items) == 0 {
			t.Fatalf("notifications: %+v %v", list, err)
		}
		return list.Items[0].ClipName
	}
	if got := notificationClip(); got != title {
		t.Fatalf("notification names the clip %q, want the title", got)
	}
	h.startAnalysis(map[string]any{}, http.StatusCreated)
	if q := h.queue(); len(q.Active) == 0 || q.Active[0].ClipName != title {
		t.Fatalf("Task Queue names the clip %+v, want the title", q.Active)
	}
	// Clearing the title falls back to the file name everywhere.
	h.adminCall(http.MethodPut, path, map[string]any{"display_title": "", "display_title_changed": true}, http.StatusOK)
	if got := get(); got.DisplayTitle != nil {
		t.Fatalf("an empty title clears it: %+v", got)
	}
	if got := notificationClip(); got != before.OriginalName {
		t.Fatalf("fallback to the file name: %q", got)
	}

	// Manual game/date edits keep the detected values so they can be restored; a rescan never overwrites an edit.
	exec(`UPDATE clips SET game='Detected Game',game_origin='filename',game_edited=false,capture_time=timestamptz '2026-01-02 03:04:05+00',capture_time_origin='container',capture_time_edited=false,detected_metadata=NULL WHERE id=$1`, h.clipID)
	h.adminCall(http.MethodPut, path, map[string]any{"game": "Edited Game", "game_changed": true, "recorded_at": "2027-05-06T07:08:09Z", "recorded_at_changed": true}, http.StatusOK)
	edited := get()
	if edited.Game == nil || *edited.Game != "Edited Game" || edited.GameOrigin == nil || *edited.GameOrigin != "manual" || edited.Detected == nil || edited.Detected.Game == nil || *edited.Detected.Game != "Detected Game" || edited.Detected.RecordedAt == nil {
		t.Fatalf("edit must keep the detected values: %+v", edited)
	}
	h.adminCall(http.MethodPut, path, map[string]any{"game": "Edited Again", "game_changed": true}, http.StatusOK)
	if again := get(); again.Detected == nil || again.Detected.Game == nil || *again.Detected.Game != "Detected Game" {
		t.Fatalf("a second edit must not replace the detected value with an edited one: %+v", again)
	}
	h.adminCall(http.MethodPut, path, map[string]any{"game_reset": true}, http.StatusOK)
	reset := get()
	if reset.Game == nil || *reset.Game != "Detected Game" || reset.GameOrigin == nil || *reset.GameOrigin != "filename" || reset.RecordedAt == nil || !strings.HasPrefix(*reset.RecordedAt, "2027-05-06") {
		t.Fatalf("reset restores only the game: %+v", reset)
	}
	h.adminCall(http.MethodPut, path, map[string]any{"recorded_at_reset": true}, http.StatusOK)
	if reset = get(); reset.RecordedAt == nil || !strings.HasPrefix(*reset.RecordedAt, "2026-01-02") {
		t.Fatalf("reset restores the date: %+v", reset)
	}
	// After a restore the snapshot is dropped: a rescan may change the automatic value, and the next edit
	// snapshots the new one rather than restoring an obsolete value later.
	if reset.Detected != nil {
		t.Fatalf("a restored field must not keep a stale snapshot: %+v", reset.Detected)
	}
	exec(`UPDATE clips SET game='Rescanned Game' WHERE id=$1`, h.clipID)
	h.adminCall(http.MethodPut, path, map[string]any{"game": "Edited Third", "game_changed": true}, http.StatusOK)
	h.adminCall(http.MethodPut, path, map[string]any{"game_reset": true}, http.StatusOK)
	if got := get(); got.Game == nil || *got.Game != "Rescanned Game" {
		t.Fatalf("restore must bring back the value of the last edit, not an obsolete one: %+v", got)
	}
	// Nothing detected to restore: the reset is refused rather than guessed.
	exec(`UPDATE clips SET detected_metadata=NULL,game_edited=true WHERE id=$1`, h.clipID)
	h.adminCall(http.MethodPut, path, map[string]any{"game_reset": true}, http.StatusConflict)
	h.adminCall(http.MethodPut, path, map[string]any{"game_reset": true, "game_changed": true}, http.StatusBadRequest)
}
