package schedule_test

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/varmakarthik12/mcrflow/internal/database"
	"github.com/varmakarthik12/mcrflow/internal/models"
	"github.com/varmakarthik12/mcrflow/internal/schedule"
)

func TestScheduleConflictDetection(t *testing.T) {
	tempDir := t.TempDir()
	db, err := database.Open(filepath.Join(tempDir, "sched_test.db"))
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	repo := database.NewRepository(db)
	svc := schedule.NewService(repo)

	baseTime := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)

	// Create initial item: 12:00 -> 13:00
	item1 := &models.ScheduleItem{
		ChannelID:       "ch-01",
		ProgramTitle:    "Prime Time News",
		MediaPath:       "/media/news1.mp4",
		StartTime:       baseTime,
		DurationSeconds: 3600,
	}
	if err := svc.CreateItem(item1, false); err != nil {
		t.Fatalf("failed to schedule item1: %v", err)
	}
	if item1.EndTime != baseTime.Add(3600*time.Second) {
		t.Fatalf("end time was not auto-calculated correctly: got %v", item1.EndTime)
	}

	// Try to schedule overlapping item: 12:30 -> 13:30
	item2 := &models.ScheduleItem{
		ChannelID:       "ch-01",
		ProgramTitle:    "Conflicting Feature Film",
		MediaPath:       "/media/film.mp4",
		StartTime:       baseTime.Add(1800 * time.Second),
		DurationSeconds: 3600,
	}
	err = svc.CreateItem(item2, false)
	if err == nil || !errors.Is(err, schedule.ErrScheduleConflict) {
		t.Fatalf("expected ErrScheduleConflict, got %v", err)
	}

	// Schedule non-overlapping item: 13:00 -> 14:00
	item3 := &models.ScheduleItem{
		ChannelID:       "ch-01",
		ProgramTitle:    "Afternoon Sports Show",
		MediaPath:       "/media/sports.mp4",
		StartTime:       baseTime.Add(3600 * time.Second),
		DurationSeconds: 3600,
	}
	if err := svc.CreateItem(item3, false); err != nil {
		t.Fatalf("failed to schedule non-overlapping item3: %v", err)
	}
}
