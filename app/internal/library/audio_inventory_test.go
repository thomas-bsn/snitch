package library

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestProbeAudioStreamInventory(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg required")
	}
	path := filepath.Join(t.TempDir(), "inventory.mp4")
	cmd := exec.Command("ffmpeg", "-v", "error", "-f", "lavfi", "-i", "color=s=16x16:d=2", "-f", "lavfi", "-i", "anullsrc=r=16000:cl=mono", "-map", "0:v", "-map", "1:a", "-t", "1", "-c:a", "aac", "-metadata:s:a:0", "title=Game with dialogue", "-metadata:s:a:0", "language=fra", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("fixture: %v %s", err, out)
	}
	m, err := ProbeFile(context.Background(), "ffprobe", path)
	if err != nil {
		t.Fatal(err)
	}
	a := m.Tracks[1]
	if a.Index != 1 || a.Codec != "aac" || a.Language != "fra" || a.Channels != 1 || a.DurationSeconds == nil || *a.DurationSeconds <= 0 || a.StartSeconds == nil {
		t.Fatalf("stream metadata: %+v", a)
	}
	// Matroska often has no per-stream duration. Never substitute the container duration.
	mkv := filepath.Join(t.TempDir(), "unknown.mkv")
	if out, err := exec.Command("ffmpeg", "-v", "error", "-i", path, "-c", "copy", mkv).CombinedOutput(); err != nil {
		t.Fatalf("remux: %v %s", err, out)
	}
	m, err = ProbeFile(context.Background(), "ffprobe", mkv)
	if err != nil || m.Tracks[1].DurationSeconds != nil {
		t.Fatalf("unknown stream duration: %+v %v", m, err)
	}
}
