//go:build integration

package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"testing/fstest"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/database"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"snitch/migrations"
)

// The executable is exercised without Node or a dev server, against a dedicated database.
func TestExecutableLifecycle(t *testing.T) {
	ctx := context.Background()
	pg, err := postgres.Run(ctx, "pgvector/pgvector:0.8.7-pg18-bookworm@sha256:2358fcba361ed2233a5ed81b5fe4ca779ccb304120ce531a3bf51c0ed7e2bc11", postgres.WithDatabase("receipts"), postgres.WithUsername("receipts"), postgres.WithPassword("synthetic-private-password"), postgres.BasicWaitStrategies())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pg.Terminate(ctx) })
	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(t.TempDir(), "snitch")
	if out, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %s: %v", out, err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	listener.Close()
	env := []string{"DATABASE_URL=" + dsn, "MASTER_KEY=" + base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1}, 32)), "PUBLIC_URL=http://" + address, "COOKIE_SECURE=false", "LISTEN_ADDR=" + address, "DISPLAY_NAME=Fixture Snitch", "UPLOAD_ROOT=" + t.TempDir()}
	command := func(arg string) *exec.Cmd {
		cmd := exec.Command(binary, arg)
		cmd.Env = append(os.Environ(), env...)
		return cmd
	}
	t.Run("serve does not migrate an empty database", func(t *testing.T) {
		out, err := command("serve").CombinedOutput()
		if err == nil || !bytes.Contains(out, []byte("migrations missing")) {
			t.Fatalf("serve should reject unmigrated database: %s: %v", out, err)
		}
		if bytes.Contains(out, []byte("synthetic-private-password")) {
			t.Fatal("database password leaked")
		}
	})

	t.Run("serve rejects older migrated schema without upgrading it", func(t *testing.T) {
		db, err := goose.OpenDBWithDriver("pgx", dsn)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		first, err := migrations.Files.ReadFile("00001_foundation.sql")
		if err != nil {
			t.Fatal(err)
		}
		old, err := goose.NewProvider(database.DialectPostgres, db, fstest.MapFS{"00001_foundation.sql": &fstest.MapFile{Data: first}})
		if err != nil {
			t.Fatal(err)
		}
		if _, err = old.Up(ctx); err != nil {
			t.Fatal(err)
		}
		deadline, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		cmd := exec.CommandContext(deadline, binary, "serve")
		cmd.Env = command("serve").Env
		out, err := cmd.CombinedOutput()
		if err == nil || !bytes.Contains(out, []byte("migrations missing")) {
			t.Fatal("serve did not clearly reject an outdated schema")
		}
		version, err := old.GetDBVersion(ctx)
		if err != nil || version != 1 {
			t.Fatalf("serve modified schema: %d %v", version, err)
		}
	})
	for i := 0; i < 2; i++ {
		if out, err := command("migrate").CombinedOutput(); err != nil {
			t.Fatalf("migration/replay: %s: %v", out, err)
		}
	}
	// Logs go to a file so the test never reads a buffer concurrently with the server.
	logPath := filepath.Join(t.TempDir(), "server.log")
	logFile, err := os.Create(logPath)
	if err != nil {
		t.Fatal(err)
	}
	defer logFile.Close()
	server := command("serve")
	server.Stdout = logFile
	server.Stderr = logFile
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	stopped := false
	t.Cleanup(func() {
		if !stopped {
			_ = server.Process.Kill()
			_ = server.Wait()
		}
	})
	client := http.Client{Timeout: 3 * time.Second}
	get := func(path string, want int) []byte {
		t.Helper()
		resp, err := client.Get("http://" + address + path)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != want {
			t.Fatalf("%s: %d %s", path, resp.StatusCode, body)
		}
		return body
	}
	deadline := time.Now().Add(15 * time.Second)
	for {
		resp, err := client.Get("http://" + address + "/api/health/live")
		if err == nil {
			resp.Body.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("server did not start")
		}
		time.Sleep(50 * time.Millisecond)
	}
	if out, err := command("healthcheck").CombinedOutput(); err != nil {
		t.Fatalf("healthcheck must use LISTEN_ADDR: %s: %v", out, err)
	}
	if body := get("/api/health/ready", 200); !bytes.Contains(body, []byte(`"degraded"`)) {
		t.Fatalf("missing embeddings should degrade readiness: %s", body)
	}
	if body := get("/api/setup", 200); !bytes.Contains(body, []byte(`"initialized":false`)) || !bytes.Contains(body, []byte("Fixture Snitch")) {
		t.Fatalf("new installation: %s", body)
	}
	if body := get("/installation", 200); !bytes.Contains(body, []byte(`<div id="root"`)) {
		t.Fatal("embedded SPA missing")
	}
	get("/api/unknown", 404)
	if err := pg.Stop(ctx, nil); err != nil {
		t.Fatal(err)
	}
	get("/api/health/live", 200)
	get("/api/health/ready", 503)
	if err := server.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- server.Wait() }()
	select {
	case err := <-done:
		stopped = true
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(12 * time.Second):
		t.Fatal("shutdown timed out")
	}
	logs, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(logs)), "\n") {
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("unstructured log: %s", line)
		}
	}
	for _, secret := range []string{"synthetic-private-password", base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1}, 32))} {
		if bytes.Contains(logs, []byte(secret)) {
			t.Fatal("secret in logs")
		}
	}
}

// The deployment starts without any MASTER_KEY: the executable produces the key in the secrets directory itself,
// and refuses (clearly) only when it could not keep what it generated.
func TestExecutableGeneratesItsMasterKey(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "snitch")
	if out, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %s: %v", out, err)
	}
	run := func(secretsDir string) (string, error) {
		cmd := exec.Command(binary, "serve")
		// The database is deliberately unreachable: the key is resolved before the first connection.
		cmd.Env = append(os.Environ(), "DATABASE_URL=postgres://receipts@127.0.0.1:1/receipts?sslmode=disable&connect_timeout=1", "MASTER_KEY=", "SECRETS_DIR="+secretsDir, "PUBLIC_URL=http://localhost:8080", "COOKIE_SECURE=false", "UPLOAD_ROOT="+t.TempDir())
		out, err := cmd.CombinedOutput()
		return string(out), err
	}
	dir := t.TempDir()
	if _, err := run(dir); err == nil {
		t.Fatal("an unreachable database must stop the server")
	}
	raw, err := os.ReadFile(filepath.Join(dir, "master_key"))
	if err != nil {
		t.Fatalf("no key generated at the first start: %v", err)
	}
	if key, err := base64.StdEncoding.DecodeString(string(raw)); err != nil || len(key) != 32 {
		t.Fatal("generated key is malformed")
	}
	out, _ := run(dir)
	if strings.Contains(out, string(raw)) {
		t.Fatal("the key appeared in the output")
	}
	again, _ := os.ReadFile(filepath.Join(dir, "master_key"))
	if string(again) != string(raw) {
		t.Fatal("the second start replaced the key")
	}
	occupied := filepath.Join(t.TempDir(), "occupied")
	if err := os.WriteFile(occupied, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	out, err = run(filepath.Join(occupied, "secrets"))
	if err == nil || !strings.Contains(out, "MASTER_KEY is not set and the secrets directory") {
		t.Fatalf("a key that cannot be kept must be refused with an explanation: %s", out)
	}
}
