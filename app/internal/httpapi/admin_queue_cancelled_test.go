//go:build integration

package httpapi

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

type locationQueue struct {
	Active []struct {
		ClipID    int64  `json:"clip_id"`
		Status    string `json:"status"`
		Locations []struct {
			Source  string `json:"source"`
			Path    string `json:"path"`
			Private bool   `json:"private"`
		} `json:"locations"`
	} `json:"active"`
	Completed []struct {
		ClipID int64 `json:"clip_id"`
	} `json:"completed"`
	Tasks []struct {
		ClipID int64 `json:"clip_id"`
	} `json:"tasks"`
}

func (h *voiceHarness) locationQueue() (locationQueue, []byte) {
	h.t.Helper()
	raw := h.adminCall(http.MethodGet, "/api/admin/analysis/queue", nil, http.StatusOK)
	var q locationQueue
	if err := json.Unmarshal(raw, &q); err != nil {
		h.t.Fatal(err)
	}
	return q, raw
}

// A cancelled analysis keeps its clip on the board, never among the finished ones, until it is
// planned again; the card names where the original video lives (the persisted source path,
// never a Worker work file), and a private folder never gives its path away.
func testAdminQueueCancelledAndLocation(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	ctx := t.Context()
	h.startAnalysis(map[string]any{}, http.StatusCreated)
	h.analysisAction("/cancel", http.StatusOK)

	q, _ := h.locationQueue()
	if len(q.Active) != 1 || q.Active[0].ClipID != h.clipID || q.Active[0].Status != "cancelled" {
		t.Fatalf("a cancelled run stays on the board: %+v", q.Active)
	}
	if len(q.Completed) != 0 || len(q.Tasks) != 0 {
		t.Fatalf("a cancelled run is neither finished nor a loose task: %+v", q)
	}

	// The location is the indexed source path.
	var root, rel string
	if err := pool.QueryRow(ctx, `SELECT cs.root_path,sf.relative_path FROM source_files sf JOIN clip_sources cs ON cs.id=sf.source_id WHERE sf.clip_id=$1`, h.clipID).Scan(&root, &rel); err != nil {
		t.Fatal(err)
	}
	locs := q.Active[0].Locations
	if len(locs) != 1 || locs[0].Path != filepath.Join(root, rel) || locs[0].Source == "" || locs[0].Private {
		t.Fatalf("the card shows the persisted source path: %+v", locs)
	}

	// A file that went missing gives no misleading path.
	if _, err := pool.Exec(ctx, `UPDATE source_files SET status='missing' WHERE clip_id=$1`, h.clipID); err != nil {
		t.Fatal(err)
	}
	if q, _ = h.locationQueue(); len(q.Active[0].Locations) != 0 {
		t.Fatalf("a missing file has no path: %+v", q.Active[0].Locations)
	}

	// A private folder is named by its source only: the path is never sent.
	if _, err := pool.Exec(ctx, `UPDATE source_files SET status='ready',relative_path='secret/'||relative_path WHERE clip_id=$1`, h.clipID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO private_folders(source_id,folder) SELECT source_id,'secret' FROM source_files WHERE clip_id=$1`, h.clipID); err != nil {
		t.Fatal(err)
	}
	q, raw := h.locationQueue()
	if locs = q.Active[0].Locations; len(locs) != 1 || !locs[0].Private || locs[0].Path != "" || strings.Contains(string(raw), "secret") {
		t.Fatalf("a private folder keeps its path: %+v %s", locs, raw)
	}

	// Planning again replaces the cancelled card.
	h.startAnalysis(map[string]any{}, http.StatusCreated)
	if q, _ = h.locationQueue(); len(q.Active) != 1 || q.Active[0].Status != "active" {
		t.Fatalf("a new run replaces the cancelled one: %+v", q.Active)
	}
}
