package resolution_test

import (
	"path/filepath"
	"testing"

	"github.com/varmakarthik12/mcrflow/internal/database"
	"github.com/varmakarthik12/mcrflow/internal/models"
	"github.com/varmakarthik12/mcrflow/internal/resolution"
)

func TestResolutionService(t *testing.T) {
	tempDir := t.TempDir()
	db, err := database.Open(filepath.Join(tempDir, "res_test.db"))
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	repo := database.NewRepository(db)
	svc := resolution.NewService(repo)

	// List seeded presets
	list, err := svc.ListResolutions()
	if err != nil {
		t.Fatalf("failed to list resolutions: %v", err)
	}
	if len(list) < 4 {
		t.Fatalf("expected at least 4 presets, got %d", len(list))
	}

	// Cannot delete built-in preset
	err = svc.DeleteResolution("res-in-1080i50")
	if err != resolution.ErrCannotDeletePreset {
		t.Fatalf("expected ErrCannotDeletePreset, got %v", err)
	}

	// Create custom preset
	custom := &models.ResolutionPreset{
		Name:      "Vertical OTT 1080x1920",
		Width:     1080,
		Height:    1920,
		FrameRate: 30,
		IsPreset:  false,
	}
	if err := svc.CreateResolution(custom); err != nil {
		t.Fatalf("failed to create custom preset: %v", err)
	}

	// Can delete custom preset
	if err := svc.DeleteResolution(custom.ID); err != nil {
		t.Fatalf("failed to delete custom preset: %v", err)
	}
}
