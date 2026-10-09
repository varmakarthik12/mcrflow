package resolution

import (
	"strings"
	"testing"

	"mcrflow/internal/models"
)

func TestDefaultIndianCablePresets(t *testing.T) {
	store := NewStore()
	presets := store.ListPresets()

	if len(presets) < 5 {
		t.Fatalf("expected at least 5 default presets, got %d", len(presets))
	}

	// Verify standard 1080i50 PAL HD exists
	p1080i, err := store.GetPreset("res-1080i50-pal-hd")
	if err != nil {
		t.Fatalf("expected 1080i50 preset: %v", err)
	}

	if p1080i.Width != 1920 || p1080i.Height != 1080 {
		t.Errorf("expected 1920x1080, got %dx%d", p1080i.Width, p1080i.Height)
	}

	if p1080i.ScanningMode != "interlaced" {
		t.Errorf("expected interlaced scanning, got %s", p1080i.ScanningMode)
	}

	// Verify 576i50 SD 4:3 exists
	p576i, err := store.GetPreset("res-576i50-sd-4x3")
	if err != nil {
		t.Fatalf("expected 576i50 4:3 preset: %v", err)
	}

	if p576i.AspectRatio != "4:3" {
		t.Errorf("expected 4:3 aspect ratio, got %s", p576i.AspectRatio)
	}

	// Verify 576i50 SD 16:9 Anamorphic exists
	p576iAnamorphic, err := store.GetPreset("res-576i50-sd-16x9")
	if err != nil {
		t.Fatalf("expected 576i50 16:9 anamorphic preset: %v", err)
	}

	if p576iAnamorphic.Width != 720 || p576iAnamorphic.Height != 576 {
		t.Errorf("expected 720x576, got %dx%d", p576iAnamorphic.Width, p576iAnamorphic.Height)
	}
}

func TestCustomPresetLifecycle(t *testing.T) {
	store := NewStore()

	// 1. Create custom preset
	custom := &models.ResolutionPreset{
		Name:             "900p Custom OTT Stream",
		Category:         "Custom",
		Width:            1600,
		Height:           900,
		FrameRate:        30.0,
		AspectRatio:      "16:9",
		ScanningMode:     "progressive",
		VideoBitrateKbps: 4500,
	}

	created, err := store.CreatePreset(custom)
	if err != nil {
		t.Fatalf("failed to create custom preset: %v", err)
	}

	if created.ID == "" {
		t.Errorf("expected generated ID")
	}

	if created.IsSystem {
		t.Errorf("custom preset should not be marked as system")
	}

	// 2. Update custom preset
	created.Name = "900p30 Custom Sports Stream"
	created.VideoBitrateKbps = 5000
	updated, err := store.UpdatePreset(created.ID, created)
	if err != nil {
		t.Fatalf("failed to update preset: %v", err)
	}

	if updated.VideoBitrateKbps != 5000 {
		t.Errorf("expected bitrate 5000, got %d", updated.VideoBitrateKbps)
	}

	// 3. Delete custom preset
	if err := store.DeletePreset(created.ID); err != nil {
		t.Fatalf("failed to delete custom preset: %v", err)
	}

	// Verify deleted
	if _, err := store.GetPreset(created.ID); err != ErrPresetNotFound {
		t.Errorf("expected ErrPresetNotFound after deletion")
	}

	// 4. System preset protection
	if err := store.DeletePreset("res-1080i50-pal-hd"); err != ErrCannotDeleteSys {
		t.Errorf("expected ErrCannotDeleteSys when deleting system preset, got %v", err)
	}
}

func TestGenerateFfmpegVideoArgs(t *testing.T) {
	store := NewStore()
	p1080i, _ := store.GetPreset("res-1080i50-pal-hd")

	args := GenerateFfmpegVideoArgs(p1080i)
	joined := strings.Join(args, " ")

	if !strings.Contains(joined, "scale=1920:1080") {
		t.Errorf("expected scale filter 1920:1080 in args: %s", joined)
	}

	if !strings.Contains(joined, "tinterlace") {
		t.Errorf("expected tinterlace filter in args: %s", joined)
	}

	if !strings.Contains(joined, "-b:v 8500k") {
		t.Errorf("expected bitrate 8500k in args: %s", joined)
	}
}
