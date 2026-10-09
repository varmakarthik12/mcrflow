package bot_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/varmakarthik12/mcrflow/internal/bot"
	"github.com/varmakarthik12/mcrflow/internal/channel"
	"github.com/varmakarthik12/mcrflow/internal/database"
	"github.com/varmakarthik12/mcrflow/internal/models"
	"github.com/varmakarthik12/mcrflow/internal/schedule"
	"github.com/varmakarthik12/mcrflow/internal/tmdb"
)

func TestChatOpsBot(t *testing.T) {
	tempDir := t.TempDir()
	db, err := database.Open(filepath.Join(tempDir, "bot_test.db"))
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	repo := database.NewRepository(db)
	schedSvc := schedule.NewService(repo)
	chanSvc := channel.NewService(repo)
	tmdbClient := tmdb.NewClient("")

	botSvc := bot.NewService(schedSvc, chanSvc, tmdbClient)

	// 1. Test help
	resp := botSvc.ProcessCommand(models.BotMessageRequest{Message: "help"})
	if !resp.Success || !strings.Contains(resp.Reply, "MCRFlow ChatOps Commands") {
		t.Fatalf("help command failed: %+v", resp)
	}

	// 2. Test channels
	resp = botSvc.ProcessCommand(models.BotMessageRequest{Message: "channels"})
	if !resp.Success || !strings.Contains(resp.Reply, "DD National HD") {
		t.Fatalf("channels command failed: %+v", resp)
	}

	// 3. Test search
	resp = botSvc.ProcessCommand(models.BotMessageRequest{Message: "search RRR"})
	if !resp.Success || !strings.Contains(resp.Reply, "RRR") {
		t.Fatalf("search command failed: %+v", resp)
	}

	// 4. Test schedule command: schedule Kantara at 21:00 on ch-01
	resp = botSvc.ProcessCommand(models.BotMessageRequest{Message: "schedule Kantara at 21:00 on ch-01"})
	if !resp.Success || !strings.Contains(resp.Reply, "Successfully scheduled 'Kantara'") {
		t.Fatalf("schedule command failed: %+v", resp)
	}

	// 5. Test status
	resp = botSvc.ProcessCommand(models.BotMessageRequest{Message: "status ch-01"})
	if !resp.Success || !strings.Contains(resp.Reply, "DD National HD") {
		t.Fatalf("status command failed: %+v", resp)
	}
}
