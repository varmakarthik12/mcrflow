package bot

import (
	"testing"

	"github.com/varmakarthik12/mcrflow/internal/models"
	"github.com/varmakarthik12/mcrflow/internal/schedule"
	"github.com/varmakarthik12/mcrflow/internal/storage"
)

func TestParseTimeToken(t *testing.T) {
	cases := []struct {
		input       string
		expectedHour int
		expectedMin  int
	}{
		{"9 AM", 9, 0},
		{"9:30 AM", 9, 30},
		{"9 PM", 21, 0},
		{"12 PM", 12, 0},
		{"12 AM", 0, 0},
		{"16:45", 16, 45},
	}

	for _, c := range cases {
		parsed, err := ParseTimeToken(c.input)
		if err != nil {
			t.Fatalf("unexpected error for %s: %v", c.input, err)
		}
		if parsed.Hour() != c.expectedHour || parsed.Minute() != c.expectedMin {
			t.Errorf("for input %s expected %02d:%02d, got %02d:%02d",
				c.input, c.expectedHour, c.expectedMin, parsed.Hour(), parsed.Minute())
		}
	}
}

func TestProcessNaturalLanguageCommand(t *testing.T) {
	schedStore := schedule.NewStore()
	storageMgr := storage.NewManager()
	botStore := NewStore(schedStore, storageMgr)

	// 1. Process English command with fuzzy match
	res, err := botStore.ProcessNaturalLanguageCommand("Schedule Jawan at 11 AM", "ch-01")
	if err != nil {
		t.Fatalf("failed to process NLP command: %v", err)
	}

	if res.ParsedMovieTitle != "Jawan" {
		t.Errorf("expected parsed title Jawan, got %s", res.ParsedMovieTitle)
	}
	if len(res.MatchedFiles) == 0 {
		t.Errorf("expected fuzzy matched media files from storage")
	}

	// 2. Test conflict detection with existing item at 16:15
	conflictRes, err := botStore.ProcessNaturalLanguageCommand("Schedule Avengers at 16:30", "ch-01")
	if err != nil {
		t.Fatalf("failed to process NLP command: %v", err)
	}

	if !conflictRes.ConflictDetected {
		t.Errorf("expected conflict detected at 16:30 with on-air movie")
	}
	if conflictRes.ConflictingItem == nil {
		t.Errorf("expected conflicting item reference")
	}

	// 3. Save new bot
	newBot := &models.BotConfig{
		Platform: "SLACK",
		BotName:  "MCR-Slack-Bot",
		APIToken: "xoxb-123456789-abcdef",
	}
	saved, err := botStore.SaveBot(newBot)
	if err != nil {
		t.Fatalf("failed to save bot: %v", err)
	}
	if saved.ID == "" {
		t.Errorf("expected generated bot ID")
	}
}
