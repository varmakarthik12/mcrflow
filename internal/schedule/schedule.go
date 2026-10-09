package schedule

import (
	"errors"
	"fmt"
	"sort"
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

const (
	ActionNone         = ""
	ActionRipple       = "RIPPLE"
	ActionAdjustStart  = "ADJUST_START"
	ActionAdjustEnd    = "ADJUST_END"
	ActionForcePreempt = "FORCE_PREEMPT"
	ActionForceLegacy  = "FORCE_OVERWRITE"
)

// ConflictReport details collisions and suggests valid adjustment parameters
type ConflictReport struct {
	HasConflict       bool                  `json:"has_conflict"`
	Conflicts         []models.ScheduleItem `json:"conflicts"`
	SuggestedStart    time.Time             `json:"suggested_start"`
	SuggestedDuration int                   `json:"suggested_duration"`
	SuggestedEnd      time.Time             `json:"suggested_end"`
}

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

// CheckConflicts evaluates whether [start, end) collides with existing items and computes suggestions
func (s *Service) CheckConflicts(channelID string, start, end time.Time, excludeID string) (*ConflictReport, error) {
	conflicts, err := s.FindConflicts(channelID, start, end, excludeID)
	if err != nil {
		return nil, err
	}

	report := &ConflictReport{
		HasConflict: len(conflicts) > 0,
		Conflicts:   conflicts,
	}

	if len(conflicts) == 0 {
		return report, nil
	}

	// Sort conflicts ascending by start time
	sort.Slice(conflicts, func(i, j int) bool {
		return conflicts[i].StartTime.Before(conflicts[j].StartTime)
	})

	// 1. Suggested Start: latest end time of conflicting items that start before or at our proposed start
	suggestedStart := start
	for _, c := range conflicts {
		if c.EndTime.After(suggestedStart) {
			suggestedStart = c.EndTime
		}
	}
	report.SuggestedStart = suggestedStart

	// 2. Suggested Duration: trim to the start of the earliest upcoming item
	origDur := int(end.Sub(start).Seconds())
	if origDur <= 0 {
		origDur = 3600
	}
	suggestedDur := origDur
	for _, c := range conflicts {
		if c.StartTime.After(start) {
			availableSecs := int(c.StartTime.Sub(start).Seconds())
			if availableSecs > 0 && availableSecs < suggestedDur {
				suggestedDur = availableSecs
			}
		}
	}
	report.SuggestedDuration = suggestedDur
	report.SuggestedEnd = suggestedStart.Add(time.Duration(origDur) * time.Second)

	return report, nil
}

// CreateItem adds a program to the channel schedule, handling conflict resolution actions
func (s *Service) CreateItem(item *models.ScheduleItem, action string) error {
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

	act := strings.ToUpper(strings.TrimSpace(action))

	conflicts, err := s.FindConflicts(item.ChannelID, item.StartTime, item.EndTime, "")
	if err != nil {
		return fmt.Errorf("failed to check conflicts: %w", err)
	}

	if len(conflicts) > 0 {
		switch act {
		case ActionRipple:
			// Ripple subsequent overlapping and following programs down the timeline
			if err := s.applyRippleShift(item.ChannelID, item.StartTime, item.EndTime, ""); err != nil {
				return fmt.Errorf("failed to ripple schedule: %w", err)
			}

		case ActionAdjustStart:
			// Snap start time after the latest conflicting program
			report, _ := s.CheckConflicts(item.ChannelID, item.StartTime, item.EndTime, "")
			if report != nil && !report.SuggestedStart.IsZero() {
				item.StartTime = report.SuggestedStart
				item.EndTime = item.StartTime.Add(time.Duration(item.DurationSeconds) * time.Second)
			}

		case ActionAdjustEnd:
			// Trim duration to avoid collision with the next scheduled program
			report, _ := s.CheckConflicts(item.ChannelID, item.StartTime, item.EndTime, "")
			if report != nil && report.SuggestedDuration > 0 {
				item.DurationSeconds = report.SuggestedDuration
				item.EndTime = item.StartTime.Add(time.Duration(item.DurationSeconds) * time.Second)
			}

		case ActionForcePreempt, ActionForceLegacy:
			// Preempt / trim / overwrite conflicting items
			if err := s.applyForcePreempt(conflicts, item.StartTime, item.EndTime); err != nil {
				return fmt.Errorf("failed to force preempt schedule: %w", err)
			}

		default:
			// No action specified: return conflict error
			return fmt.Errorf("%w: collides with '%s' (%s - %s)",
				ErrScheduleConflict, conflicts[0].ProgramTitle,
				conflicts[0].StartTime.Format("15:04:05"), conflicts[0].EndTime.Format("15:04:05"))
		}
	}

	return s.store.CreateScheduleItem(item)
}

// UpdateItem updates a program, handling conflict resolution actions
func (s *Service) UpdateItem(item *models.ScheduleItem, action string) error {
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

	act := strings.ToUpper(strings.TrimSpace(action))

	conflicts, err := s.FindConflicts(item.ChannelID, item.StartTime, item.EndTime, item.ID)
	if err != nil {
		return fmt.Errorf("failed to check conflicts: %w", err)
	}

	if len(conflicts) > 0 {
		switch act {
		case ActionRipple:
			if err := s.applyRippleShift(item.ChannelID, item.StartTime, item.EndTime, item.ID); err != nil {
				return fmt.Errorf("failed to ripple schedule: %w", err)
			}

		case ActionAdjustStart:
			report, _ := s.CheckConflicts(item.ChannelID, item.StartTime, item.EndTime, item.ID)
			if report != nil && !report.SuggestedStart.IsZero() {
				item.StartTime = report.SuggestedStart
				item.EndTime = item.StartTime.Add(time.Duration(item.DurationSeconds) * time.Second)
			}

		case ActionAdjustEnd:
			report, _ := s.CheckConflicts(item.ChannelID, item.StartTime, item.EndTime, item.ID)
			if report != nil && report.SuggestedDuration > 0 {
				item.DurationSeconds = report.SuggestedDuration
				item.EndTime = item.StartTime.Add(time.Duration(item.DurationSeconds) * time.Second)
			}

		case ActionForcePreempt, ActionForceLegacy:
			if err := s.applyForcePreempt(conflicts, item.StartTime, item.EndTime); err != nil {
				return fmt.Errorf("failed to force preempt schedule: %w", err)
			}

		default:
			return fmt.Errorf("%w: collides with '%s'", ErrScheduleConflict, conflicts[0].ProgramTitle)
		}
	}

	return s.store.UpdateScheduleItem(item)
}

// applyRippleShift shifts conflicting and subsequent programs forward so they do not overlap
func (s *Service) applyRippleShift(channelID string, newStart, newEnd time.Time, excludeID string) error {
	allItems, err := s.store.ListScheduleByChannel(channelID)
	if err != nil {
		return err
	}

	// Filter and sort items that overlap with or start after newStart
	sort.Slice(allItems, func(i, j int) bool {
		return allItems[i].StartTime.Before(allItems[j].StartTime)
	})

	currentCutoff := newEnd
	for _, it := range allItems {
		if excludeID != "" && it.ID == excludeID {
			continue
		}

		// Only ripple items starting after or during newStart
		if it.EndTime.After(newStart) {
			if it.StartTime.Before(currentCutoff) {
				shift := currentCutoff.Sub(it.StartTime)
				it.StartTime = it.StartTime.Add(shift)
				it.EndTime = it.StartTime.Add(time.Duration(it.DurationSeconds) * time.Second)
				if err := s.store.UpdateScheduleItem(&it); err != nil {
					return err
				}
				currentCutoff = it.EndTime
			} else {
				currentCutoff = it.EndTime
			}
		}
	}
	return nil
}

// applyForcePreempt trims or removes existing items that overlap with the new time slot
func (s *Service) applyForcePreempt(conflicts []models.ScheduleItem, newStart, newEnd time.Time) error {
	for _, c := range conflicts {
		if (c.StartTime.Equal(newStart) || c.StartTime.After(newStart)) && (c.EndTime.Equal(newEnd) || c.EndTime.Before(newEnd)) {
			// Fully enveloped: remove
			if err := s.store.DeleteScheduleItem(c.ID); err != nil {
				return err
			}
		} else if c.StartTime.Before(newStart) && c.EndTime.After(newStart) {
			// Starts before, overlaps into slot: trim end time to newStart
			c.EndTime = newStart
			c.DurationSeconds = int(newStart.Sub(c.StartTime).Seconds())
			if c.DurationSeconds <= 0 {
				_ = s.store.DeleteScheduleItem(c.ID)
			} else {
				if err := s.store.UpdateScheduleItem(&c); err != nil {
					return err
				}
			}
		} else if c.StartTime.Before(newEnd) && c.EndTime.After(newEnd) {
			// Starts inside slot, ends after: trim start time to newEnd
			c.StartTime = newEnd
			c.DurationSeconds = int(c.EndTime.Sub(newEnd).Seconds())
			if c.DurationSeconds <= 0 {
				_ = s.store.DeleteScheduleItem(c.ID)
			} else {
				if err := s.store.UpdateScheduleItem(&c); err != nil {
					return err
				}
			}
		}
	}
	return nil
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

// GetActiveProgram returns the scheduled item currently on-air (StartTime <= at < EndTime)
func (s *Service) GetActiveProgram(channelID string, at time.Time) (*models.ScheduleItem, error) {
	// Look in a 24-hour window around at
	windowStart := at.Add(-24 * time.Hour)
	windowEnd := at.Add(24 * time.Hour)
	items, err := s.store.ListScheduleBetween(channelID, windowStart, windowEnd)
	if err != nil {
		return nil, err
	}

	var active *models.ScheduleItem
	for _, it := range items {
		if (it.StartTime.Before(at) || it.StartTime.Equal(at)) && it.EndTime.After(at) {
			if active == nil || it.StartTime.After(active.StartTime) {
				cp := it
				active = &cp
			}
		}
	}
	return active, nil
}

// GetNextProgram returns the next upcoming scheduled item after at
func (s *Service) GetNextProgram(channelID string, at time.Time) (*models.ScheduleItem, error) {
	items, err := s.store.ListScheduleByChannel(channelID)
	if err != nil {
		return nil, err
	}

	var next *models.ScheduleItem
	for _, it := range items {
		if it.StartTime.After(at) {
			if next == nil || it.StartTime.Before(next.StartTime) {
				cp := it
				next = &cp
			}
		}
	}
	return next, nil
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
