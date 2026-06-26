package synapse

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultLogPathsReturnsNewestFirst(t *testing.T) {
	localAppData := t.TempDir()
	t.Setenv("LOCALAPPDATA", localAppData)

	logDir := filepath.Join(localAppData, "Razer", "RazerAppEngine", "User Data", "Logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}

	oldPath := filepath.Join(logDir, "systray_systrayv1.log")
	newPath := filepath.Join(logDir, "systray_systrayv2.log")
	for _, path := range []string{oldPath, newPath} {
		if err := os.WriteFile(path, []byte("test"), 0o644); err != nil {
			t.Fatalf("WriteFile returned error: %v", err)
		}
	}

	now := time.Now()
	if err := os.Chtimes(oldPath, now.Add(-time.Minute), now.Add(-time.Minute)); err != nil {
		t.Fatalf("Chtimes returned error: %v", err)
	}
	if err := os.Chtimes(newPath, now, now); err != nil {
		t.Fatalf("Chtimes returned error: %v", err)
	}

	paths, err := DefaultLogPaths()
	if err != nil {
		t.Fatalf("DefaultLogPaths returned error: %v", err)
	}

	if len(paths) != 2 {
		t.Fatalf("expected 2 paths, got %d", len(paths))
	}
	if paths[0] != newPath {
		t.Fatalf("expected newest path first, got %q", paths[0])
	}
}
