//go:build integration

package httpapi

import (
	"context"
	"fmt"
	"net/url"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// The filters of the notifications page run on the server, so they never depend on how many
// notifications the client holds: an old unread one, or the active actions, stay reachable
// however long the history is.
func testNotificationFilters(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `DELETE FROM notifications`); err != nil {
		t.Fatal(err)
	}
	h := newVoiceHarnessClips(t, pool, 1)
	clip := h.clipIDs[0]
	admin := h.adminUser()
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	// The oldest notification of all is an unread resolved voice review; 230 newer ones were read.
	exec(`INSERT INTO notifications(kind,clip_id,state,resolved_at,created_at,updated_at) VALUES('needs_speaker_review',$1,'resolved',now(),now()-interval '30 days',now()-interval '30 days')`, clip)
	exec(`INSERT INTO notifications(kind,clip_id,state,resolved_at,created_at,updated_at) SELECT 'job_failed',$1,'resolved',now(),now()-interval '1 day',now()-(g * interval '1 minute') FROM generate_series(1,230) g`, clip)
	exec(`INSERT INTO notification_recipients(notification_id,user_id,read_at) SELECT n.id,(SELECT id FROM users WHERE role IN ('admin','super_admin') ORDER BY id LIMIT 1),CASE WHEN n.kind='job_failed' THEN now() END FROM notifications n`)
	exec(`INSERT INTO notifications(kind,clip_id,state) VALUES('needs_speaker_review',$1,'active')`, clip)
	exec(`INSERT INTO notification_recipients(notification_id,user_id) SELECT id,(SELECT id FROM users WHERE role IN ('admin','super_admin') ORDER BY id LIMIT 1) FROM notifications WHERE state='active'`)

	list := func(query url.Values) notificationsT {
		t.Helper()
		return h.notifications(admin, "?"+query.Encode())
	}
	unread := list(url.Values{"filter": {"unread"}})
	if len(unread.Items) != 2 || unread.Total != 2 {
		t.Fatalf("an old unread notification must be reachable: %+v", unread)
	}
	actions := list(url.Values{"filter": {"actions"}})
	if len(actions.Items) != 1 || actions.Items[0].State != "active" || actions.Total != 1 {
		t.Fatalf("active actions are always retrievable: %+v", actions)
	}
	errorsOnly := list(url.Values{"filter": {"errors"}, "limit": {"200"}})
	if len(errorsOnly.Items) != 200 || errorsOnly.Total != 230 {
		t.Fatalf("errors page one: %d of %d", len(errorsOnly.Items), errorsOnly.Total)
	}
	rest := list(url.Values{"filter": {"errors"}, "limit": {"200"}, "offset": {"200"}})
	if len(rest.Items) != 30 || rest.Total != 230 {
		t.Fatalf("errors page two: %d of %d", len(rest.Items), rest.Total)
	}
	all := list(url.Values{"limit": {"50"}, "offset": {"200"}})
	if all.Total != 232 || len(all.Items) != 32 {
		t.Fatalf("all, paged beyond the first 200: %d of %d", len(all.Items), all.Total)
	}
	h.userCall(admin, "GET", "/api/notifications?filter=weird", nil, 400)
	h.userCall(admin, "GET", fmt.Sprintf("/api/notifications?offset=%d", -1), nil, 400)
}
