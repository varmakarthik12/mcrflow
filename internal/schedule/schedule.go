package schedule

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/varmakarthik12/mcrflow/internal/models"
)

var (
	ErrChannelIDRequired    = errors.New("channel id is required")
	ErrProgramTitleRequired = errors.New("program title is required")
	ErrStartTimeRequired    = errors.New("start time is required")
	ErrScheduleConflict     = errors.New("schedule conflict: overlapping program already scheduled in this time slot")
)

// Store defines persistence operations for schedule items
type Store interface {
	CreateScheduleItem(s *models.ScheduleItem) error
	GetScheduleItemByID(id string) (*models.ScheduleItem, error)
	ListScheduleByChannel(channelID string) ([]models.ScheduleItem, error)
	ListScheduleBetween(channelID string, start, end time.Time) ([]models.ScheduleItem, error)
	UpdateScheduleItem(s *models.ScheduleItem) error
	DeleteScheduleItem(id string) error
}

// Service provides broadcast timeline operations with conflict checking
type Service struct {
	store Store
}

// NewService creates a new schedule service
func NewService(store Store) *Service {
	return &Service{store: store}
}

// CreateItem adds a program to the channel schedule and verifies conflict freedom
func (s *Service) CreateItem(item *models.ScheduleItem, allowOverlap bool) error {
	item.ChannelID = strings.TrimSpace(item.ChannelID)
	if item.ChannelID == "" {
		return ErrChannelIDRequired
	}
	item.ProgramTitle = strings.TrimSpace(item.ProgramTitle)
	if item.ProgramTitle == "" {
		return ErrProgramTitleRequired
	}
	if item.StartTime.IsZero() {
		return ErrStartTimeRequired
	}
	if item.DurationSeconds <= 0 {
		item.DurationSeconds = 3600 // default 1 hour
	}
	item.EndTime = item.StartTime.Add(time.Duration(item.DurationSeconds) * time.Second)

	if !allowOverlap {
		conflicts, err := s.FindConflicts(item.ChannelID, item.StartTime, item.EndTime, "")
		if err != nil {
			return fmt.Errorf("failed to check conflicts: %w", err)
		}
		if len(conflicts) > 0 {
			return fmt.Errorf("%w: collides with '%s' (%s - %s)",
				ErrScheduleConflict, conflicts[0].ProgramTitle,
				conflicts[0].StartTime.Format("15:04:05"), conflicts[0].EndTime.Format("15:04:05"))
		}
	}

	return s.store.CreateScheduleItem(item)
}

// UpdateItem updates a program and verifies conflict freedom
func (s *Service) UpdateItem(item *models.ScheduleItem, allowOverlap bool) error {
	item.ChannelID = strings.TrimSpace(item.ChannelID)
	if item.ChannelID == "" {
		return ErrChannelIDRequired
	}
	item.ProgramTitle = strings.TrimSpace(item.ProgramTitle)
	if item.ProgramTitle == "" {
		return ErrProgramTitleRequired
	}
	if item.StartTime.IsZero() {
		return ErrStartTimeRequired
	}
	if item.DurationSeconds <= 0 {
		item.DurationSeconds = 3600
	}
	item.EndTime = item.StartTime.Add(time.Duration(item.DurationSeconds) * time.Second)

	if !allowOverlap {
		conflicts, err := s.FindConflicts(item.ChannelID, item.StartTime, item.EndTime, item.ID)
		if err != nil {
			return fmt.Errorf("failed to check conflicts: %w", err)
		}
		if len(conflicts) > 0 {
			return fmt.Errorf("%w: collides with '%s'", ErrScheduleConflict, conflicts[0].ProgramTitle)
		}
	}

	return s.store.UpdateScheduleItem(item)
}

// FindConflicts returns schedule items overlapping with [start, end)
func (s *Service) FindConflicts(channelID string, start, end time.Time, excludeID string) ([]models.ScheduleItem, error) {
	items, err := s.store.ListScheduleBetween(channelID, start, end)
	if err != nil {
		return nil, err
	}

	conflicts := make([]models.ScheduleItem, 0)
	for _, it := range items {
		if excludeID != "" && it.ID == excludeID {
			continue
		}
		// Strict overlap check: it.StartTime < end && it.EndTime > start
		if it.StartTime.Before(end) && it.EndTime.After(start) {
			conflicts = append(conflicts, it)
		}
	}
	return conflicts, nil
}

// GetItemByID returns item by ID
func (s *Service) GetItemByID(id string) (*models.ScheduleItem, error) {
	return s.store.GetScheduleItemByID(id)
}

// ListByChannel returns all scheduled items for a channel
func (s *Service) ListByChannel(channelID string) ([]models.ScheduleItem, error) {
	return s.store.ListScheduleByChannel(channelID)
}

// ListBetween returns scheduled items within a time window
func (s *Service) ListBetween(channelID string, start, end time.Time) ([]models.ScheduleItem, error) {
	return s.store.ListScheduleBetween(channelID, start, end)
}

// DeleteItem removes a schedule item
func (s *Service) DeleteItem(id string) error {
	return s.store.DeleteScheduleItem(id)
}
