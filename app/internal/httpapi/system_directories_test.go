package httpapi

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRootVideosSkipsInaccessibleSystemDirectories(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"#recycle", "@eaDir", "games/#recycle", "games/@eaDir", "private"} {
		dir := filepath.Join(root, name)
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "clip.mp4"), []byte("synthetic"), 0600); err != nil {
			t.Fatal(err)
		}
		if name != "private" {
			if err := os.Chmod(dir, 0); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.Chmod(dir, 0700) })
			if _, err := os.ReadDir(dir); err == nil {
				t.Skip("permission checks require an unprivileged user")
			} else if !errors.Is(err, os.ErrPermission) {
				t.Fatalf("fixture must be inaccessible: %s: %v", name, err)
			}
		}
	}
	got, err := walkRootVideos(context.Background(), root, map[string]bool{"private": true}, true)
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for _, v := range got {
		paths = append(paths, v.rel)
	}
	if !reflect.DeepEqual(paths, []string{"private/clip.mp4"}) {
		t.Fatalf("videos = %v", paths)
	}
	got, err = walkRootVideos(context.Background(), root, map[string]bool{"private": true}, false)
	if err != nil || len(got) != 0 {
		t.Fatalf("ordinary admin: %v, %v", got, err)
	}
}

func TestRootVideosReportsOrdinaryDirectoryPermissionErrors(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "ordinary")
	if err := os.Mkdir(dir, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0700) })
	if _, err := os.ReadDir(dir); err == nil {
		t.Skip("permission checks require an unprivileged user")
	}
	_, err := walkRootVideos(context.Background(), root, nil, true)
	if !errors.Is(err, os.ErrPermission) {
		t.Fatalf("expected permission error, got %v", err)
	}
}
