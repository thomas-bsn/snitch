//go:build integration

package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func testDeleteClip(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarnessClips(t, pool, 2)
	var root, relative string
	if err := pool.QueryRow(context.Background(), `SELECT cs.root_path,sf.relative_path FROM source_files sf JOIN clip_sources cs ON cs.id=sf.source_id WHERE sf.clip_id=$1`, h.clipID).Scan(&root, &relative); err != nil {
		t.Fatal(err)
	}
	filename := filepath.Join(root, relative)
	path := fmt.Sprintf("/api/admin/clips/%d", h.clipID)
	ordinary := h.newUser("ordinary")
	h.userCall(ordinary, http.MethodDelete, path, nil, http.StatusForbidden)
	if _, err := os.Stat(filename); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `UPDATE users SET role='admin' WHERE username='ordinary'`); err != nil {
		t.Fatal(err)
	}
	h.userCall(ordinary, http.MethodDelete, path, nil, http.StatusForbidden)
	if _, err := pool.Exec(context.Background(), `UPDATE source_files SET status='missing' WHERE clip_id=$1`, h.clipID); err != nil {
		t.Fatal(err)
	}
	h.adminCall(http.MethodDelete, path, nil, http.StatusConflict)
	if _, err := os.Stat(filename); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `UPDATE source_files SET status='ready' WHERE clip_id=$1`, h.clipID); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filename, 0600); err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, []byte("replacement"), 0600); err != nil {
		t.Fatal(err)
	}
	h.adminCall(http.MethodDelete, path, nil, http.StatusConflict)
	if got, err := os.ReadFile(filename); err != nil || string(got) != "replacement" {
		t.Fatalf("replacement lost: %q %v", got, err)
	}
	if err := os.WriteFile(filename, original, 0600); err != nil {
		t.Fatal(err)
	}
	h.adminCall(http.MethodDelete, path, nil, http.StatusOK)
	if _, err := os.Stat(filename); !os.IsNotExist(err) {
		t.Fatalf("original still exists: %v", err)
	}
	h.adminCall(http.MethodGet, fmt.Sprintf("/api/clips/%d", h.clipID), nil, http.StatusNotFound)
	// Simulate interruption after the original was renamed but before the journal advanced.
	second := h.clipIDs[1]
	if err := pool.QueryRow(context.Background(), `SELECT cs.root_path,sf.relative_path FROM source_files sf JOIN clip_sources cs ON cs.id=sf.source_id WHERE sf.clip_id=$1`, second).Scan(&root, &relative); err != nil {
		t.Fatal(err)
	}
	staged := ".snitch-delete-synthetic-recovery"
	if _, err := pool.Exec(context.Background(), `INSERT INTO clip_deletion_files VALUES($1,$2,$3,$4,'moving')`, second, root, relative, staged); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(root, relative), filepath.Join(root, staged)); err != nil {
		t.Fatal(err)
	}
	h.adminCall(http.MethodDelete, fmt.Sprintf("/api/admin/clips/%d", second), nil, http.StatusOK)
	if _, err := os.Stat(filepath.Join(root, staged)); !os.IsNotExist(err) {
		t.Fatalf("staged original still exists: %v", err)
	}

}
