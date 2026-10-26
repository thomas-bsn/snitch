package secrets

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// MasterKeyFile is the file name of the generated master key inside the secrets directory.
// It holds the same base64 text an operator would give in MASTER_KEY, so an export is a valid explicit value.
const MasterKeyFile = "master_key"

const (
	// MasterKeyProvided: the key comes from the MASTER_KEY variable and is never written to the volume.
	MasterKeyProvided = "provided"
	// MasterKeyGenerated: the application generated the key and keeps it in the secrets directory.
	MasterKeyGenerated = "generated"
)

// ResolveMasterKey returns the key that encrypts the settings secrets and where it comes from.
//
// An explicit, well-formed value always wins and is not copied to the volume. Without it, the key kept in dir is
// used; when there is none yet, a random 32-byte key is generated and kept there. A key file that cannot be used is
// set aside (never overwritten) and replaced, so a damaged volume does not prevent the start: the secrets it
// protected are then reported as unreadable instead.
func ResolveMasterKey(dir, explicit string) ([]byte, string, error) {
	if explicit != "" {
		key, err := decodeMasterKey(explicit)
		if err != nil {
			return nil, "", err
		}
		return key, MasterKeyProvided, nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, "", errors.New("MASTER_KEY is not set and the secrets directory cannot be created")
	}
	path := filepath.Join(dir, MasterKeyFile)
	if raw, err := os.ReadFile(path); err == nil {
		if key, err := decodeMasterKey(strings.TrimSpace(string(raw))); err == nil {
			return key, MasterKeyGenerated, nil
		}
		if err := os.Rename(path, path+".unusable"); err != nil {
			return nil, "", errors.New("the master key file is unusable and cannot be set aside")
		}
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, "", errors.New("cannot generate a master key")
	}
	if err := createPrivateFile(dir, path, base64.StdEncoding.EncodeToString(key)); err != nil {
		// A concurrent start may have created it first: that key is the one to use.
		if raw, readErr := os.ReadFile(path); readErr == nil {
			if existing, decodeErr := decodeMasterKey(strings.TrimSpace(string(raw))); decodeErr == nil {
				return existing, MasterKeyGenerated, nil
			}
		}
		return nil, "", errors.New("MASTER_KEY is not set and the secrets directory is not writable")
	}
	return key, MasterKeyGenerated, nil
}

func decodeMasterKey(value string) ([]byte, error) {
	key, err := base64.StdEncoding.DecodeString(value)
	if err != nil || len(key) != 32 {
		return nil, errors.New("MASTER_KEY must encode 32 bytes in base64")
	}
	return key, nil
}

// createPrivateFile creates path with mode 0600, failing when it already exists, and never exposes a partial file.
func createPrivateFile(dir, path, value string) error {
	tmp, err := os.CreateTemp(dir, ".master-key-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err = tmp.WriteString(value); err == nil {
		err = tmp.Chmod(0o600)
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Link(tmp.Name(), path)
}
