package config

import (
	"encoding/base64"
	"path/filepath"
	"testing"
	"time"
)

func TestConfig(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://test")
	t.Setenv("MASTER_KEY", base64.StdEncoding.EncodeToString(make([]byte, 32)))
	t.Setenv("PUBLIC_URL", "https://receipts.example")
	t.Setenv("COOKIE_SECURE", "true")
	if _, err := Load(); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PUBLIC_URL", "http://receipts.example")
	if _, err := Load(); err == nil {
		t.Fatal("insecure production origin")
	}
	t.Setenv("COOKIE_SECURE", "false")
	if _, err := Load(); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MASTER_KEY", "wrong")
	if _, err := Load(); err == nil {
		t.Fatal("invalid key accepted")
	}
}

func TestSourceRuntimeConfiguration(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://test:test@localhost:5432/receipts")
	t.Setenv("MASTER_KEY", base64.StdEncoding.EncodeToString(make([]byte, 32)))
	t.Setenv("PUBLIC_URL", "https://receipts.example")
	t.Setenv("COOKIE_SECURE", "true")
	t.Setenv("CLIPS_ROOTS", filepath.Join("relative", "media")+string(filepath.ListSeparator)+"/mnt/captures")
	t.Setenv("CLIP_STABILITY_WINDOW", "2m")
	t.Setenv("SOURCE_RECONCILE_INTERVAL", "10m")
	t.Setenv("CLIP_EXCLUDE_PATTERNS", "*.tmp, *.part")
	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(got.ClipRoots) != 2 || !filepath.IsAbs(got.ClipRoots[0]) || got.ClipRoots[1] != "/mnt/captures" {
		t.Fatalf("unexpected roots: %#v", got.ClipRoots)
	}
	if got.ProbePath != "ffprobe" || got.StabilityWindow != 2*time.Minute || got.ReconcileInterval != 10*time.Minute {
		t.Fatalf("unexpected source settings: %+v", got)
	}
	if len(got.ClipExcludePatterns) != 2 || got.ClipExcludePatterns[0] != "*.tmp" || got.ClipExcludePatterns[1] != "*.part" {
		t.Fatalf("unexpected exclude patterns: %#v", got.ClipExcludePatterns)
	}
	for _, tc := range []struct{ key, value string }{
		{"CLIP_STABILITY_WINDOW", "-1s"},
		{"CLIP_STABILITY_WINDOW", "25h"},
		{"SOURCE_RECONCILE_INTERVAL", "500ms"},
		{"SOURCE_RECONCILE_INTERVAL", "25h"},
	} {
		t.Run(tc.key+"="+tc.value, func(t *testing.T) {
			t.Setenv(tc.key, tc.value)
			if _, err := Load(); err == nil {
				t.Fatal("invalid source setting accepted")
			}
		})
	}
}

func TestRejectInvalidRuntimeConfiguration(t *testing.T) {
	for _, tc := range []struct{ key, value string }{
		{"PUBLIC_URL", "https://user:password@example.org"},
		{"EMBEDDINGS_URL", "file:///tmp/model"},
		{"EMBEDDINGS_URL", "http://user:password@embed:11434/v1/embeddings"},
		{"EMBEDDINGS_URL", "http://embed:11434/v1/embeddings?token=secret"},
		{"LISTEN_ADDR", "localhost"},
		{"LISTEN_ADDR", ":invalid"},
		{"LISTEN_ADDR", ":70000"},
		{"DISPLAY_NAME", "   "},
		{"DATABASE_URL", "https://database.example"},
	} {
		t.Run(tc.key+"="+tc.value, func(t *testing.T) {
			t.Setenv("DATABASE_URL", "postgres://test:test@localhost:5432/receipts")
			t.Setenv("MASTER_KEY", base64.StdEncoding.EncodeToString(make([]byte, 32)))
			t.Setenv("PUBLIC_URL", "https://receipts.example")
			t.Setenv("COOKIE_SECURE", "true")
			t.Setenv("LISTEN_ADDR", ":8080")
			t.Setenv("DISPLAY_NAME", "Receipts")
			t.Setenv("EMBEDDINGS_URL", "")
			t.Setenv(tc.key, tc.value)
			if _, err := Load(); err == nil {
				t.Fatal("invalid runtime configuration accepted")
			}
		})
	}
}
