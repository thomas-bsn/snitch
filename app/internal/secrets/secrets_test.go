package secrets

import (
	"bytes"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

func TestPostgresPasswordIsProducedOnceAndKept(t *testing.T) {
	dir := t.TempDir()
	if err := EnsurePostgresPassword(dir, ""); err != nil {
		t.Fatal(err)
	}
	first, err := ReadPostgresPassword(filepath.Join(dir, PostgresPasswordFile))
	if err != nil || len(first) < 32 {
		t.Fatalf("weak or unreadable password: %d %v", len(first), err)
	}
	if err := EnsurePostgresPassword(dir, ""); err != nil {
		t.Fatal(err)
	}
	second, _ := ReadPostgresPassword(filepath.Join(dir, PostgresPasswordFile))
	if second != first {
		t.Fatal("second start changed the password")
	}
}

func TestExplicitPasswordTakesPrecedenceAndSurvivesItsRemoval(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, PostgresPasswordFile)
	explicit := "synthetic:@/?#%+ password"
	if err := EnsurePostgresPassword(dir, "generated-before-upgrade"); err != nil {
		t.Fatal(err)
	}
	if err := EnsurePostgresPassword(dir, explicit); err != nil {
		t.Fatal(err)
	}
	if got, _ := ReadPostgresPassword(path); got != explicit {
		t.Fatalf("explicit password not honoured: %q", got)
	}
	if err := EnsurePostgresPassword(dir, ""); err != nil {
		t.Fatal(err)
	}
	if got, _ := ReadPostgresPassword(path); got != explicit {
		t.Fatal("removing the variable changed the password of an existing installation")
	}
}

func TestPasswordFileIsReadableByTheDatabaseAndTheApplicationAndWrittenWithoutNewline(t *testing.T) {
	dir := t.TempDir()
	if err := EnsurePostgresPassword(dir, "plain"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, PostgresPasswordFile)
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o444 {
		t.Fatalf("unexpected mode: %v %v", info, err)
	}
	if raw, _ := os.ReadFile(path); string(raw) != "plain" {
		t.Fatalf("unexpected content length %d", len(raw))
	}
}

func TestReadPasswordRejectsMissingOrEmptyFile(t *testing.T) {
	dir := t.TempDir()
	if _, err := ReadPostgresPassword(filepath.Join(dir, "absent")); err == nil {
		t.Fatal("missing file accepted")
	}
	empty := filepath.Join(dir, "empty")
	if err := os.WriteFile(empty, []byte("\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadPostgresPassword(empty); err == nil {
		t.Fatal("empty password accepted")
	}
}

func TestMasterKeyIsGeneratedOnceKeptAndReadableOnlyByTheApplication(t *testing.T) {
	dir := t.TempDir()
	first, origin, err := ResolveMasterKey(dir, "")
	if err != nil || len(first) != 32 || origin != MasterKeyGenerated {
		t.Fatalf("generated key: %d %q %v", len(first), origin, err)
	}
	info, err := os.Stat(filepath.Join(dir, MasterKeyFile))
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("key file must be private to the application: %v %v", info, err)
	}
	second, origin, err := ResolveMasterKey(dir, "")
	if err != nil || origin != MasterKeyGenerated || !bytes.Equal(first, second) {
		t.Fatal("second start changed the key", err)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 1 {
		t.Fatalf("unexpected leftovers in the secrets directory: %d", len(entries))
	}
}

func TestExplicitMasterKeyTakesPrecedenceAndIsNeverWrittenToTheVolume(t *testing.T) {
	dir := t.TempDir()
	generated, _, err := ResolveMasterKey(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	explicit := bytes.Repeat([]byte{7}, 32)
	key, origin, err := ResolveMasterKey(dir, base64.StdEncoding.EncodeToString(explicit))
	if err != nil || origin != MasterKeyProvided || !bytes.Equal(key, explicit) {
		t.Fatalf("explicit key not honoured: %q %v", origin, err)
	}
	stored, _ := os.ReadFile(filepath.Join(dir, MasterKeyFile))
	if decoded, _ := base64.StdEncoding.DecodeString(string(stored)); !bytes.Equal(decoded, generated) {
		t.Fatal("explicit key replaced the generated one")
	}
	// Without the variable the generated key is back: nothing was lost.
	if again, _, _ := ResolveMasterKey(dir, ""); !bytes.Equal(again, generated) {
		t.Fatal("generated key lost after an explicit run")
	}
	empty := t.TempDir()
	if _, _, err = ResolveMasterKey(empty, base64.StdEncoding.EncodeToString(explicit)); err != nil {
		t.Fatal(err)
	}
	if entries, _ := os.ReadDir(empty); len(entries) != 0 {
		t.Fatal("explicit key written to the volume")
	}
}

func TestMalformedExplicitMasterKeyIsRefused(t *testing.T) {
	for _, value := range []string{"wrong", base64.StdEncoding.EncodeToString(make([]byte, 16))} {
		if _, _, err := ResolveMasterKey(t.TempDir(), value); err == nil {
			t.Fatalf("malformed key %q accepted", value)
		}
	}
}

func TestUnusableKeyFileIsSetAsideNotOverwritten(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, MasterKeyFile)
	if err := os.WriteFile(path, []byte("truncated"), 0o600); err != nil {
		t.Fatal(err)
	}
	key, origin, err := ResolveMasterKey(dir, "")
	if err != nil || len(key) != 32 || origin != MasterKeyGenerated {
		t.Fatalf("unusable file blocks the start: %v", err)
	}
	if kept, _ := os.ReadFile(path + ".unusable"); string(kept) != "truncated" {
		t.Fatal("unusable file was not kept aside")
	}
}

func TestMasterKeyNeedsAWritableDirectory(t *testing.T) {
	file := filepath.Join(t.TempDir(), "occupied")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ResolveMasterKey(filepath.Join(file, "secrets"), ""); err == nil {
		t.Fatal("a key that cannot be kept must not be generated silently")
	}
}
