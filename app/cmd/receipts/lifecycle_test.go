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
	"time"

	"github.com/testcontainers/testcontainers-go/modules/postgres"
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
	binary := filepath.Join(t.TempDir(), "receipts")
	if out, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %s: %v", out, err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	listener.Close()
	env := []string{"DATABASE_URL=" + dsn, "MASTER_KEY=" + base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1}, 32)), "PUBLIC_URL=http://" + address, "COOKIE_SECURE=false", "LISTEN_ADDR=" + address, "DISPLAY_NAME=Fixture Receipts"}
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
	if body := get("/api/setup", 200); !bytes.Contains(body, []byte(`"initialized":false`)) || !bytes.Contains(body, []byte("Fixture Receipts")) {
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
