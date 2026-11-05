package library

import (
	"io/fs"
	"path/filepath"
)

// IsSystemDirectory identifies NAS metadata and recycle directories by their exact
// name, at any depth. It is independent of user-configured Dossiers privés.
func IsSystemDirectory(name string) bool {
	return name == "#recycle" || name == "@eaDir"
}

// WalkMediaDir prunes system directories before WalkDir opens them. All other
// entries and errors are passed to visit, preserving each caller's error policy.
func WalkMediaDir(root string, visit fs.WalkDirFunc) error {
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err == nil && entry.IsDir() && IsSystemDirectory(entry.Name()) {
			return filepath.SkipDir
		}
		return visit(path, entry, err)
	})
}
