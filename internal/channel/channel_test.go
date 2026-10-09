package channel_test

import (
	"path/filepath"
	"testing"

	"github.com/varmakarthik12/mcrflow/internal/channel"
	"github.com/varmakarthik12/mcrflow/internal/database"
	"github.com/varmakarthik12/mcrflow/internal/models"
)

func TestChannelService(t *testing.T) {
	tempDir := t.TempDir()
	db, err := database.Open(filepath.Join(tempDir, "ch_test.db"))
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	repo := database.NewRepository(db)
	svc := channel.NewService(repo)

	// Validation tests
	err = svc.CreateChannel(&models.Channel{Name: "", CallSign: "CS"})
	if err != channel.ErrNameRequired {
		t.Fatalf("expected ErrNameRequired, got: %v", err)
	}

	err = svc.CreateChannel(&models.Channel{Name: "DD Sports", CallSign: ""})
	if err != channel.ErrCallSignRequired {
		t.Fatalf("expected ErrCallSignRequired, got: %v", err)
	}

	// Create valid channel
	ch := &models.Channel{
		Name:     "DD Sports HD",
		CallSign: "MCR-DDS",
	}
	if err := svc.CreateChannel(ch); err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}

	list, err := svc.ListChannels()
	if err != nil {
		t.Fatalf("failed to list channels: %v", err)
	}
	if len(list) < 2 { // Seeded ch-01 + new DD Sports HD
		t.Fatalf("expected at least 2 channels, got %d", len(list))
	}
}
