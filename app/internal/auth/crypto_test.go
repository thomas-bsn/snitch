package auth

import (
	"bytes"
	"testing"
)

func TestPassword(t *testing.T) {
	h := Password("synthetic-password")
	if !Verify(h, "synthetic-password") || Verify(h, "wrong") || Verify("bad", "synthetic-password") {
		t.Fatal("password verification")
	}
	if h == Password("synthetic-password") {
		t.Fatal("salt reused")
	}
}
func TestEncryption(t *testing.T) {
	key := bytes.Repeat([]byte{1}, 32)
	b, err := Seal(key, "discord", "secret")
	if err != nil {
		t.Fatal(err)
	}
	v, err := Open(key, "discord", b)
	if err != nil || v != "secret" {
		t.Fatal("roundtrip", err)
	}
	if _, err = Open(key, "other", b); err == nil {
		t.Fatal("associated data ignored")
	}
	b[len(b)-1] ^= 1
	if _, err = Open(key, "discord", b); err == nil {
		t.Fatal("tamper accepted")
	}
	if _, err = Open(key, "discord", nil); err == nil {
		t.Fatal("truncated accepted")
	}
}
