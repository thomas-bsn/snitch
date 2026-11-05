package library

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestWalkMediaDirExcludesOnlySystemDirectories(t *testing.T) {
	root := t.TempDir()
	for _, rel := range []string{"#recycle/trash.mp4", "@eaDir/thumb.mp4", "nested/#recycle/trash.mp4", "nested/@eaDir/thumb.mp4", "private/clip.mp4", "#recycle.mp4", "@eaDir.mp4", "#recycle-other/clip.mp4"} {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("synthetic"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	var files []string
	err := WalkMediaDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() && IsSystemDirectory(entry.Name()) {
			t.Fatalf("system directory reached caller: %s", path)
		}
		if !entry.IsDir() {
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			files = append(files, filepath.ToSlash(rel))
		}
		return nil
	})
	want := []string{"#recycle-other/clip.mp4", "#recycle.mp4", "@eaDir.mp4", "private/clip.mp4"}
	if err != nil || !reflect.DeepEqual(files, want) {
		t.Fatalf("files = %v, err = %v", files, err)
	}
	for _, rel := range []string{"#recycle", "@eaDir/thumb.mp4", "nested/#recycle/trash.mp4"} {
		if _, err := ManagedPath(root, rel); !errors.Is(err, ErrInvalidSource) {
			t.Fatalf("ManagedPath(%q) = %v", rel, err)
		}
	}
	if _, err := ManagedPath(root, "private/clip.mp4"); err != nil {
		t.Fatal(err)
	}
	if err := WalkMediaDir(filepath.Join(root, "absent"), func(_ string, _ fs.DirEntry, err error) error { return err }); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing root: %v", err)
	}
}
