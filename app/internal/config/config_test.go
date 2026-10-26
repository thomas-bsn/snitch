package config

import (
	"bytes"
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

func TestConfigNeedsNoMasterKey(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DATABASE_URL", "postgres://test")
	t.Setenv("MASTER_KEY", "")
	t.Setenv("SECRETS_DIR", dir)
	t.Setenv("PUBLIC_URL", "https://receipts.example")
	first, err := Load()
	if err != nil || len(first.MasterKey) != 32 || first.MasterKeyOrigin != "generated" {
		t.Fatalf("no generated key: %v %q", err, first.MasterKeyOrigin)
	}
	second, err := Load()
	if err != nil || !bytes.Equal(first.MasterKey, second.MasterKey) {
		t.Fatal("the generated key changed between starts", err)
	}
	explicit := bytes.Repeat([]byte{9}, 32)
	t.Setenv("MASTER_KEY", base64.StdEncoding.EncodeToString(explicit))
	third, err := Load()
	if err != nil || !bytes.Equal(third.MasterKey, explicit) || third.MasterKeyOrigin != "provided" {
		t.Fatal("an explicit key must take precedence", err)
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
	if got.ProbePath != "ffprobe" || got.FFmpegPath != "ffmpeg" || got.StabilityWindow != 2*time.Minute || got.ReconcileInterval != 10*time.Minute {
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

func TestAddressSettingsAreOptional(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://test:test@localhost:5432/receipts")
	t.Setenv("MASTER_KEY", base64.StdEncoding.EncodeToString(make([]byte, 32)))
	t.Setenv("PUBLIC_URL", "")
	t.Setenv("COOKIE_SECURE", "")
	t.Setenv("TRUSTED_PROXIES", "")
	got, err := Load()
	if err != nil {
		t.Fatalf("start without address settings: %v", err)
	}
	if got.PublicURL != "" || got.Secure || got.SecureExplicit || len(got.TrustedProxies) == 0 {
		t.Fatalf("unexpected defaults: %+v", got)
	}
	t.Setenv("PUBLIC_URL", "https://Snitch.Example/")
	if got, err = Load(); err != nil || got.PublicURL != "https://snitch.example" || got.Secure || got.SecureExplicit {
		t.Fatalf("pinned address: %+v %v", got, err)
	}
	t.Setenv("PUBLIC_URL", "http://snitch.example")
	t.Setenv("COOKIE_SECURE", "false")
	if got, err = Load(); err != nil || got.Secure || !got.SecureExplicit {
		t.Fatalf("legacy settings: %+v %v", got, err)
	}
}

func TestTrustedProxiesSetting(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://test:test@localhost:5432/receipts")
	t.Setenv("MASTER_KEY", base64.StdEncoding.EncodeToString(make([]byte, 32)))
	t.Setenv("TRUSTED_PROXIES", "none")
	got, err := Load()
	if err != nil || len(got.TrustedProxies) != 0 {
		t.Fatalf("none: %+v %v", got.TrustedProxies, err)
	}
	t.Setenv("TRUSTED_PROXIES", "203.0.113.7, 198.51.100.0/24 ,2001:db8::1")
	got, err = Load()
	if err != nil || len(got.TrustedProxies) != 3 {
		t.Fatalf("list: %+v %v", got.TrustedProxies, err)
	}
	for _, bad := range []string{"proxy.example", "203.0.113.0/40", "10.0.0.1;10.0.0.2"} {
		t.Setenv("TRUSTED_PROXIES", bad)
		if _, err := Load(); err == nil {
			t.Fatalf("%q accepted", bad)
		}
	}
}

func TestRejectInvalidRuntimeConfiguration(t *testing.T) {
	for _, tc := range []struct{ key, value string }{
		{"PUBLIC_URL", "https://user:password@example.org"},
		{"EMBEDDINGS_URL", "file:///tmp/model"},
		{"EMBEDDINGS_URL", "http://user:password@embed:11434/v1/embeddings"},
		{"EMBEDDINGS_URL", "http://embed:11434/v1/embeddings?token=secret"},
		{"LISTEN_ADDR", "localhost"},
		{"WORKER_IMAGE", "bad image\nname"},
		{"LISTEN_ADDR", ":invalid"},
		{"LISTEN_ADDR", ":70000"},
		{"DATABASE_URL", "https://database.example"},
	} {
		t.Run(tc.key+"="+tc.value, func(t *testing.T) {
			t.Setenv("DATABASE_URL", "postgres://test:test@localhost:5432/receipts")
			t.Setenv("MASTER_KEY", base64.StdEncoding.EncodeToString(make([]byte, 32)))
			t.Setenv("PUBLIC_URL", "https://receipts.example")
			t.Setenv("COOKIE_SECURE", "true")
			t.Setenv("LISTEN_ADDR", ":8080")
			t.Setenv("EMBEDDINGS_URL", "")
			t.Setenv(tc.key, tc.value)
			if _, err := Load(); err == nil {
				t.Fatal("invalid runtime configuration accepted")
			}
		})
	}
}
