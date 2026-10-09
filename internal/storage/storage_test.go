package storage_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/varmakarthik12/mcrflow/internal/storage"
)

func TestStorageManager(t *testing.T) {
	tempDir := t.TempDir()
	mediaDir := filepath.Join(tempDir, "media")
	if err := os.MkdirAll(mediaDir, 0755); err != nil {
		t.Fatalf("failed to create media directory: %v", err)
	}

	testFile := filepath.Join(mediaDir, "sample_feature.mp4")
	if err := os.WriteFile(testFile, []byte("fake mp4 content for testing duration calculation"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	mgr := storage.NewManager(mediaDir)

	// Test browse
	entries, err := mgr.Browse("")
	if err != nil {
		t.Fatalf("failed to browse media directory: %v", err)
	}
	found := false
	for _, entry := range entries {
		if entry.Name == "sample_feature.mp4" && !entry.IsDir {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected sample_feature.mp4 in browse entries: %+v", entries)
	}

	// Test probe
	probeRes, err := mgr.ProbeFile(testFile)
	if err != nil {
		t.Fatalf("failed to probe file: %v", err)
	}
	if probeRes.DurationSeconds <= 0 || probeRes.VideoCodec == "" || len(probeRes.AudioTracks) == 0 {
		t.Fatalf("invalid probe result: %+v", probeRes)
	}
}
