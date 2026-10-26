package secrets

import (
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
