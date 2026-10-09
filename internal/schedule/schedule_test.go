package schedule

import (
	"errors"
	"testing"
	"time"

	"mcrflow/internal/models"
)

func TestScheduleLifecycleAndConflictResolution(t *testing.T) {
	store := NewStore()

	items := store.ListByChannel("ch-01")
	if len(items) < 2 {
		t.Fatalf("expected at least 2 default items on ch-01")
	}

	existingItem := items[0] // 16:15 to ~19:04

	// 1. Attempt overlapping item without conflict resolution action -> should fail
	overlapItem := &models.ScheduleItem{
		ChannelID:       "ch-01",
		MediaFilePath:   "/movies/Avengers.mkv",
		StartTime:       existingItem.StartTime.Add(30 * time.Minute), // Inside existing slot
		DurationSeconds: 7200,                                         // 2 hours
	}

	_, err := store.CreateItem(overlapItem, "")
	if !errors.Is(err, ErrConflictDetected) {
		t.Fatalf("expected ErrConflictDetected, got: %v", err)
	}

	// 2. Resolve with QUEUE_AFTER
	queuedItem, err := store.CreateItem(overlapItem, models.ConflictQueueAfter)
	if err != nil {
		t.Fatalf("failed to queue after conflict: %v", err)
	}

	if !queuedItem.StartTime.Equal(existingItem.EndTime) {
		t.Errorf("expected queued start time %v to match existing end time %v",
			queuedItem.StartTime, existingItem.EndTime)
	}

	// Verify duration and end time calculation
	expectedEnd := queuedItem.StartTime.Add(7200 * time.Second)
	if !queuedItem.EndTime.Equal(expectedEnd) {
		t.Errorf("expected calculated end time %v, got %v", expectedEnd, queuedItem.EndTime)
	}

	// 3. Resolve with FORCE_OVERWRITE on a new slot
	now := time.Now().UTC()
	slotStart := now.Add(24 * time.Hour)
	baseItem := &models.ScheduleItem{
		ChannelID:       "ch-02",
		StartTime:       slotStart,
		DurationSeconds: 3600,
	}
	base, err := store.CreateItem(baseItem, "")
	if err != nil {
		t.Fatalf("failed to create base item: %v", err)
	}
	t.Logf("base item created with ID: %s, Start: %v, End: %v", base.ID, base.StartTime, base.EndTime)

	collidingItem := &models.ScheduleItem{
		ChannelID:       "ch-02",
		StartTime:       slotStart.Add(15 * time.Minute),
		DurationSeconds: 3600,
	}
	overwritten, err := store.CreateItem(collidingItem, models.ConflictForceOverwrite)
	if err != nil {
		t.Fatalf("failed to force overwrite: %v", err)
	}
	t.Logf("overwritten item ID: %s", overwritten.ID)

	// Base item should be removed
	if itm, err := store.GetItem(base.ID); err == nil {
		t.Fatalf("expected base item %s to be deleted after force overwrite, but found it with Start: %v, End: %v", base.ID, itm.StartTime, itm.EndTime)
	}

	if overwritten.ID == "" {
		t.Errorf("expected valid ID for overwritten item")
	}
}
