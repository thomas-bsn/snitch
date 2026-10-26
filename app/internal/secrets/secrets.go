// Package secrets produces and reads the secrets a deployment keeps in its own volume.
// Nothing here ever logs or returns a secret in an error message.
package secrets

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// PostgresPasswordFile is the file name of the database password inside the secrets directory.
const PostgresPasswordFile = "postgres_password"

// EnsurePostgresPassword makes sure dir holds the database password.
// A non-empty explicit value always wins and is written back, so an installation created with an explicit
// password keeps working, including after the variable is removed. Without it, an existing password is kept
// and a new random one is produced only when none exists yet.
func EnsurePostgresPassword(dir, explicit string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return errors.New("cannot create the secrets directory")
	}
	path := filepath.Join(dir, PostgresPasswordFile)
	current, readErr := ReadPostgresPassword(path)
	switch {
	case explicit != "":
		if readErr == nil && current == explicit {
			return nil
		}
		return write(dir, path, explicit)
	case readErr == nil:
		return nil
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return errors.New("cannot produce a random password")
	}
	return write(dir, path, base64.RawURLEncoding.EncodeToString(raw))
}

// ReadPostgresPassword returns the password stored in path, without trailing line breaks.
func ReadPostgresPassword(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", errors.New("cannot read the database password file")
	}
	password := strings.TrimRight(string(raw), "\r\n")
	if password == "" {
		return "", errors.New("the database password file is empty")
	}
	return password, nil
}

// write replaces the file atomically; it is readable by the database (another user) and the application.
func write(dir, path, value string) error {
	tmp, err := os.CreateTemp(dir, ".password-*")
	if err != nil {
		return errors.New("cannot write the database password file")
	}
	defer os.Remove(tmp.Name())
	if _, err = tmp.WriteString(value); err == nil {
		err = tmp.Chmod(0o444)
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(tmp.Name(), path)
	}
	if err != nil {
		return errors.New("cannot write the database password file")
	}
	return nil
}
