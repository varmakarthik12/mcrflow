package storage_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/varmakarthik12/mcrflow/internal/database"
	"github.com/varmakarthik12/mcrflow/internal/models"
	"github.com/varmakarthik12/mcrflow/internal/storage"
)

func TestStorageManager(t *testing.T) {
	tempDir := t.TempDir()
	db, err := database.Open(filepath.Join(tempDir, "storage_test.db"))
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	repo := database.NewRepository(db)
	mgr := storage.NewManager(repo)

	// Create a real media folder with test files
	mediaDir := filepath.Join(tempDir, "media_library")
	if err := os.MkdirAll(mediaDir, 0755); err != nil {
		t.Fatalf("failed to create media directory: %v", err)
	}
	testFile := filepath.Join(mediaDir, "sample_feature.mp4")
	if err := os.WriteFile(testFile, []byte("fake mp4 content for testing duration calculation"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Register mount
	mount := models.StorageMount{
		ID:        "mount-test",
		Name:      "Test Media Dir",
		MountType: "local",
		MountPath: mediaDir,
		IsActive:  true,
	}
	mgr.RegisterMount(mount)

	// Test browse
	entries, err := mgr.Browse("mount-test", "")
	if err != nil {
		t.Fatalf("failed to browse mount: %v", err)
	}
	if len(entries) != 1 || entries[0].Name != "sample_feature.mp4" {
		t.Fatalf("unexpected browse entries: %+v", entries)
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
