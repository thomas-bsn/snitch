package config

import (
	"encoding/base64"
	"testing"
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
