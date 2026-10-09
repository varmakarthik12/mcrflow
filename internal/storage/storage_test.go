package storage

import (
	"testing"

	"github.com/varmakarthik12/mcrflow/internal/models"
)

func TestStorageMountsAndProbe(t *testing.T) {
	mgr := NewManager()

	// 1. List default mounts
	mounts := mgr.ListMounts()
	if len(mounts) < 2 {
		t.Fatalf("expected at least 2 default mounts, got %d", len(mounts))
	}

	// 2. Create custom mount
	newMount := &models.StorageMount{
		Name:       "Secondary QNAP NAS",
		Type:       models.StorageNFS,
		TargetPath: "192.168.1.55:/volume2/fast_pool",
	}
	created, err := mgr.CreateMount(newMount)
	if err != nil {
		t.Fatalf("failed to create mount: %v", err)
	}

	if created.ID == "" {
		t.Errorf("expected generated mount ID")
	}

	// 3. Browse directory
	entries, err := mgr.BrowseDirectory(created.ID, "/bollywood/2023")
	if err != nil {
		t.Fatalf("failed to browse directory: %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("expected file entries in browsed directory")
	}

	// 4. Probe media file
	res, err := mgr.ProbeMediaFile("Jawan.2023.1080p.Hindi.Atmos.mkv")
	if err != nil {
		t.Fatalf("failed to probe media file: %v", err)
	}

	if res.DurationSeconds <= 0 {
		t.Errorf("expected positive duration seconds")
	}
	if res.DurationString != "02:49:12" {
		t.Errorf("expected formatted duration 02:49:12, got %s", res.DurationString)
	}
	if len(res.AudioStreams) == 0 {
		t.Errorf("expected probed audio streams")
	}

	// 5. Delete mount
	if err := mgr.DeleteMount(created.ID); err != nil {
		t.Fatalf("failed to delete mount: %v", err)
	}
	if _, err := mgr.GetMount(created.ID); err != ErrMountNotFound {
		t.Errorf("expected ErrMountNotFound after deleting mount")
	}
}

func TestFormatDuration(t *testing.T) {
	cases := []struct {
		secs     int64
		expected string
	}{
		{0, "00:00:00"},
		{59, "00:00:59"},
		{60, "00:01:00"},
		{3661, "01:01:01"},
		{10152, "02:49:12"},
	}

	for _, c := range cases {
		got := FormatDuration(c.secs)
		if got != c.expected {
			t.Errorf("for %d secs expected %s, got %s", c.secs, c.expected, got)
		}
	}
}
