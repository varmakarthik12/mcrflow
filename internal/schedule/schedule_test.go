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
	if err := svc.CreateItem(item1, schedule.ActionNone); err != nil {
		t.Fatalf("failed to schedule item1: %v", err)
	}
	if item1.EndTime != baseTime.Add(3600*time.Second) {
		t.Fatalf("end time was not auto-calculated correctly: got %v", item1.EndTime)
	}

	// Try to schedule overlapping item: 12:30 -> 13:30 without resolution action
	item2 := &models.ScheduleItem{
		ChannelID:       "ch-01",
		ProgramTitle:    "Conflicting Feature Film",
		MediaPath:       "/media/film.mp4",
		StartTime:       baseTime.Add(1800 * time.Second),
		DurationSeconds: 3600,
	}
	err = svc.CreateItem(item2, schedule.ActionNone)
	if err == nil || !errors.Is(err, schedule.ErrScheduleConflict) {
		t.Fatalf("expected ErrScheduleConflict, got %v", err)
	}

	// Verify CheckConflicts report
	report, err := svc.CheckConflicts("ch-01", item2.StartTime, item2.EndTime, "")
	if err != nil {
		t.Fatalf("CheckConflicts error: %v", err)
	}
	if !report.HasConflict || len(report.Conflicts) == 0 {
		t.Fatalf("expected conflict report to flag conflict, got %+v", report)
	}
	if !report.SuggestedStart.Equal(item1.EndTime) {
		t.Fatalf("expected suggested start to be %v, got %v", item1.EndTime, report.SuggestedStart)
	}

	// Schedule non-overlapping item: 13:00 -> 14:00
	item3 := &models.ScheduleItem{
		ChannelID:       "ch-01",
		ProgramTitle:    "Afternoon Sports Show",
		MediaPath:       "/media/sports.mp4",
		StartTime:       baseTime.Add(3600 * time.Second),
		DurationSeconds: 3600,
	}
	if err := svc.CreateItem(item3, schedule.ActionNone); err != nil {
		t.Fatalf("failed to schedule non-overlapping item3: %v", err)
	}
}

func TestScheduleConflictRipple(t *testing.T) {
	tempDir := t.TempDir()
	db, err := database.Open(filepath.Join(tempDir, "sched_ripple.db"))
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	repo := database.NewRepository(db)
	svc := schedule.NewService(repo)

	baseTime := time.Date(2026, 10, 10, 10, 0, 0, 0, time.UTC)

	// Item A: 10:00 -> 11:00
	itemA := &models.ScheduleItem{
		ChannelID:       "ch-01",
		ProgramTitle:    "Morning Show",
		MediaPath:       "/media/morning.mp4",
		StartTime:       baseTime,
		DurationSeconds: 3600,
	}
	_ = svc.CreateItem(itemA, schedule.ActionNone)

	// Item B: 11:00 -> 12:00
	itemB := &models.ScheduleItem{
		ChannelID:       "ch-01",
		ProgramTitle:    "Noon Class",
		MediaPath:       "/media/noon.mp4",
		StartTime:       baseTime.Add(3600 * time.Second),
		DurationSeconds: 3600,
	}
	_ = svc.CreateItem(itemB, schedule.ActionNone)

	// Insert breaking news at 10:30 for 1 hour with ActionRipple
	breakNews := &models.ScheduleItem{
		ChannelID:       "ch-01",
		ProgramTitle:    "Breaking Special Report",
		MediaPath:       "/media/special.mp4",
		StartTime:       baseTime.Add(1800 * time.Second), // 10:30 -> 11:30
		DurationSeconds: 3600,
	}
	if err := svc.CreateItem(breakNews, schedule.ActionRipple); err != nil {
		t.Fatalf("failed to insert item with ripple: %v", err)
	}

	// Verify items were rippled forward
	items, err := svc.ListByChannel("ch-01")
	if err != nil {
		t.Fatalf("failed to list items: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}

	// Verify no items overlap now
	for i := 0; i < len(items)-1; i++ {
		if items[i].EndTime.After(items[i+1].StartTime) {
			t.Fatalf("items %s (%v-%v) and %s (%v-%v) still overlap after ripple",
				items[i].ProgramTitle, items[i].StartTime, items[i].EndTime,
				items[i+1].ProgramTitle, items[i+1].StartTime, items[i+1].EndTime)
		}
	}
}

func TestScheduleConflictSnapStart(t *testing.T) {
	tempDir := t.TempDir()
	db, err := database.Open(filepath.Join(tempDir, "sched_snap.db"))
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	repo := database.NewRepository(db)
	svc := schedule.NewService(repo)

	baseTime := time.Date(2026, 10, 10, 14, 0, 0, 0, time.UTC)

	// Existing: 14:00 -> 16:00
	item1 := &models.ScheduleItem{
		ChannelID:       "ch-01",
		ProgramTitle:    "Matinee Movie",
		MediaPath:       "/media/matinee.mp4",
		StartTime:       baseTime,
		DurationSeconds: 7200,
	}
	_ = svc.CreateItem(item1, schedule.ActionNone)

	// Proposed with collision: 15:00 -> 17:00, use ActionAdjustStart
	item2 := &models.ScheduleItem{
		ChannelID:       "ch-01",
		ProgramTitle:    "Evening Drama",
		MediaPath:       "/media/drama.mp4",
		StartTime:       baseTime.Add(3600 * time.Second),
		DurationSeconds: 7200,
	}
	if err := svc.CreateItem(item2, schedule.ActionAdjustStart); err != nil {
		t.Fatalf("failed to snap start time: %v", err)
	}

	if !item2.StartTime.Equal(item1.EndTime) {
		t.Fatalf("expected start time to snap to %v, got %v", item1.EndTime, item2.StartTime)
	}
}

func TestScheduleConflictTrimEnd(t *testing.T) {
	tempDir := t.TempDir()
	db, err := database.Open(filepath.Join(tempDir, "sched_trim.db"))
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	repo := database.NewRepository(db)
	svc := schedule.NewService(repo)

	baseTime := time.Date(2026, 10, 10, 18, 0, 0, 0, time.UTC)

	// Existing fixed broadcast at 19:00 -> 20:00
	liveEvent := &models.ScheduleItem{
		ChannelID:       "ch-01",
		ProgramTitle:    "Live Football Match",
		MediaPath:       "/media/live.mp4",
		StartTime:       baseTime.Add(3600 * time.Second),
		DurationSeconds: 3600,
	}
	_ = svc.CreateItem(liveEvent, schedule.ActionNone)

	// New item starting at 18:00 with 2-hour duration, use ActionAdjustEnd
	feature := &models.ScheduleItem{
		ChannelID:       "ch-01",
		ProgramTitle:    "Pre-Match Featurette",
		MediaPath:       "/media/feature.mp4",
		StartTime:       baseTime,
		DurationSeconds: 7200, // Proposed 2 hours, should be trimmed to 1 hour (3600s)
	}
	if err := svc.CreateItem(feature, schedule.ActionAdjustEnd); err != nil {
		t.Fatalf("failed to trim end time: %v", err)
	}

	if feature.DurationSeconds != 3600 || !feature.EndTime.Equal(liveEvent.StartTime) {
		t.Fatalf("expected duration 3600 and end time %v, got dur=%d end=%v",
			liveEvent.StartTime, feature.DurationSeconds, feature.EndTime)
	}
}

func TestScheduleConflictForcePreempt(t *testing.T) {
	tempDir := t.TempDir()
	db, err := database.Open(filepath.Join(tempDir, "sched_preempt.db"))
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	repo := database.NewRepository(db)
	svc := schedule.NewService(repo)

	baseTime := time.Date(2026, 10, 10, 20, 0, 0, 0, time.UTC)

	// Existing item: 20:00 -> 22:00
	filler := &models.ScheduleItem{
		ChannelID:       "ch-01",
		ProgramTitle:    "Night Filler Reel",
		MediaPath:       "/media/filler.mp4",
		StartTime:       baseTime,
		DurationSeconds: 7200,
	}
	_ = svc.CreateItem(filler, schedule.ActionNone)

	// Force preempt with blockbuster premiere: 20:30 -> 21:30
	blockbuster := &models.ScheduleItem{
		ChannelID:       "ch-01",
		ProgramTitle:    "Jawan Premiere",
		MediaPath:       "/media/jawan.mp4",
		StartTime:       baseTime.Add(1800 * time.Second),
		DurationSeconds: 3600,
	}
	if err := svc.CreateItem(blockbuster, schedule.ActionForcePreempt); err != nil {
		t.Fatalf("failed to force preempt: %v", err)
	}

	items, err := svc.ListByChannel("ch-01")
	if err != nil {
		t.Fatalf("failed to list items: %v", err)
	}

	foundPremiere := false
	for _, it := range items {
		if it.ProgramTitle == "Jawan Premiere" {
			foundPremiere = true
		}
	}
	if !foundPremiere {
		t.Fatalf("expected Jawan Premiere to be scheduled")
	}
}

func TestScheduleGetActiveAndNextProgram(t *testing.T) {
	tempDir := t.TempDir()
	db, err := database.Open(filepath.Join(tempDir, "sched_active.db"))
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	repo := database.NewRepository(db)
	svc := schedule.NewService(repo)

	now := time.Now().UTC()

	// Active program: started 15 min ago, ends 45 min in future
	active := &models.ScheduleItem{
		ChannelID:       "ch-01",
		ProgramTitle:    "Current Live Show",
		MediaPath:       "/media/live.mp4",
		StartTime:       now.Add(-15 * time.Minute),
		DurationSeconds: 3600,
	}
	_ = svc.CreateItem(active, schedule.ActionNone)

	// Next program: starts in 45 min
	next := &models.ScheduleItem{
		ChannelID:       "ch-01",
		ProgramTitle:    "Upcoming News Hour",
		MediaPath:       "/media/upcoming.mp4",
		StartTime:       active.EndTime,
		DurationSeconds: 1800,
	}
	_ = svc.CreateItem(next, schedule.ActionNone)

	gotActive, err := svc.GetActiveProgram("ch-01", now)
	if err != nil || gotActive == nil {
		t.Fatalf("expected active program, got %v (err: %v)", gotActive, err)
	}
	if gotActive.ProgramTitle != "Current Live Show" {
		t.Fatalf("expected 'Current Live Show', got '%s'", gotActive.ProgramTitle)
	}

	gotNext, err := svc.GetNextProgram("ch-01", now)
	if err != nil || gotNext == nil {
		t.Fatalf("expected next program, got %v (err: %v)", gotNext, err)
	}
	if gotNext.ProgramTitle != "Upcoming News Hour" {
		t.Fatalf("expected 'Upcoming News Hour', got '%s'", gotNext.ProgramTitle)
	}
}
