package schedule

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"mcrflow/internal/models"
)

var (
	ErrScheduleNotFound = errors.New("schedule item not found")
	ErrConflictDetected = errors.New("schedule conflict detected with existing playback slot")
	ErrInvalidTimes     = errors.New("invalid schedule item: end time must be after start time")
)

// Store manages schedule items and conflict resolution.
type Store struct {
	mu    sync.RWMutex
	items map[string]*models.ScheduleItem
}

// NewStore initializes a schedule store.
func NewStore() *Store {
	s := &Store{
		items: make(map[string]*models.ScheduleItem),
	}
	s.loadSampleSchedule()
	return s
}

// loadSampleSchedule populates realistic broadcast schedule items.
func (s *Store) loadSampleSchedule() {
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	// Item 1: On-Air Movie
	start1 := today.Add(16*time.Hour + 15*time.Minute)
	dur1 := int64(2*3600 + 49*60 + 12) // 02:49:12
	end1 := start1.Add(time.Duration(dur1) * time.Second)

	s.items["sched-01"] = &models.ScheduleItem{
		ID:              "sched-01",
		ChannelID:       "ch-01",
		StorageMountID:  "mount-nas-01",
		MediaFilePath:   "/movies/bollywood/2023/Jawan.2023.1080p.Hindi.Atmos.mkv",
		StartTime:       start1,
		EndTime:         end1,
		DurationSeconds: dur1,
		TmdbMetadata: &models.TmdbMetadata{
			TmdbID:         872585,
			Title:          "Jawan",
			LocalizedTitle: "जवान",
			Overview:       "A high-octane action thriller which outlines the emotional journey of a man set to rectify wrongs in society.",
			PosterURL:      "https://image.tmdb.org/t/p/w500/jFpA4D7X2LbgD8nE1v01yT.jpg",
			ReleaseYear:    2023,
			Genres:         []string{"Action", "Thriller"},
			ContentRating:  "U/A 16+",
			RuntimeMinutes: 169,
			Director:       "Atlee",
			Cast:           []string{"Shah Rukh Khan", "Nayanthara", "Vijay Sethupathi"},
		},
		ContentAdTemplateID: "ad-tmpl-diwali", // Overrides channel global
		AudioSelection: models.AudioTrackSelection{
			StreamIndex:      1,
			LanguageCode:     "hin",
			Channels:         "5.1",
			EbuR128Normalize: true,
		},
		SubtitleSelection: models.SubtitleTrackSelection{
			StreamIndex:  2,
			LanguageCode: "eng",
			BurnIn:       false,
		},
		Status: "ON_AIR",
	}

	// Item 2: Cued Next Movie
	start2 := end1
	dur2 := int64(2*3600 + 40*60 + 5) // 02:40:05
	end2 := start2.Add(time.Duration(dur2) * time.Second)

	s.items["sched-02"] = &models.ScheduleItem{
		ID:              "sched-02",
		ChannelID:       "ch-01",
		StorageMountID:  "mount-nas-01",
		MediaFilePath:   "/movies/bollywood/2023/Dunki.2023.1080p.mkv",
		StartTime:       start2,
		EndTime:         end2,
		DurationSeconds: dur2,
		TmdbMetadata: &models.TmdbMetadata{
			TmdbID:         906221,
			Title:          "Dunki",
			Overview:       "An exhilarating, heartwarming tale of four friends embarking on a journey towards the UK.",
			PosterURL:      "https://image.tmdb.org/t/p/w500/dunki_poster.jpg",
			ReleaseYear:    2023,
			Genres:         []string{"Comedy", "Drama"},
			ContentRating:  "U/A",
			RuntimeMinutes: 160,
			Director:       "Rajkumar Hirani",
		},
		ContentAdTemplateID: "", // falls back to channel default
		AudioSelection: models.AudioTrackSelection{
			StreamIndex:      1,
			LanguageCode:     "hin",
			Channels:         "5.1",
			EbuR128Normalize: true,
		},
		Status: "CUED",
	}
}

// ListByChannel returns schedule items for a given channel sorted by start time.
func (s *Store) ListByChannel(channelID string) []*models.ScheduleItem {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := []*models.ScheduleItem{}
	for _, item := range s.items {
		if item.ChannelID == channelID {
			items = append(items, item)
		}
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].StartTime.Before(items[j].StartTime)
	})

	return items
}

// DetectConflict checks if the proposed time interval collides with existing items on the channel.
func (s *Store) DetectConflict(channelID string, start, end time.Time, excludeID string) (*models.ScheduleItem, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, item := range s.items {
		if item.ChannelID != channelID || item.ID == excludeID {
			continue
		}

		// Overlap condition: start < item.EndTime AND end > item.StartTime
		if start.Before(item.EndTime) && end.After(item.StartTime) {
			return item, true
		}
	}

	return nil, false
}

// CreateItem adds a new item, detecting conflicts and applying conflict resolution if needed.
func (s *Store) CreateItem(item *models.ScheduleItem, action models.ConflictAction) (*models.ScheduleItem, error) {
	if item.DurationSeconds <= 0 {
		item.DurationSeconds = 3600 // 1 hour default
	}

	// Calculate EndTime if not provided or inconsistent
	if item.EndTime.IsZero() || item.EndTime.Before(item.StartTime) {
		item.EndTime = item.StartTime.Add(time.Duration(item.DurationSeconds) * time.Second)
	}

	if item.EndTime.Before(item.StartTime) {
		return nil, ErrInvalidTimes
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if item.ID == "" {
		item.ID = "sched-" + uuid.New().String()[:8]
	}
	if item.Status == "" {
		item.Status = "PENDING"
	}

	// Check conflict
	var conflictingItem *models.ScheduleItem
	for _, existing := range s.items {
		if existing.ChannelID == item.ChannelID && item.StartTime.Before(existing.EndTime) && item.EndTime.After(existing.StartTime) {
			conflictingItem = existing
			break
		}
	}

	if conflictingItem != nil {
		switch action {
		case models.ConflictForceOverwrite:
			// Remove or trim colliding item
			delete(s.items, conflictingItem.ID)
		case models.ConflictQueueAfter:
			// Queue new item immediately after conflicting item ends
			item.StartTime = conflictingItem.EndTime
			item.EndTime = item.StartTime.Add(time.Duration(item.DurationSeconds) * time.Second)
		case models.ConflictReplaceConflict:
			// Replace conflicting item with this one
			delete(s.items, conflictingItem.ID)
		default:
			// Reject with conflict error
			return nil, fmt.Errorf("%w: collides with '%s' (%s - %s)",
				ErrConflictDetected,
				conflictingItem.ID,
				conflictingItem.StartTime.Format("15:04"),
				conflictingItem.EndTime.Format("15:04"),
			)
		}
	}

	s.items[item.ID] = item
	return item, nil
}

// DeleteItem removes a schedule item.
func (s *Store) DeleteItem(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.items[id]; !exists {
		return ErrScheduleNotFound
	}
	delete(s.items, id)
	return nil
}

// GetItem retrieves a schedule item by ID.
func (s *Store) GetItem(id string) (*models.ScheduleItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, exists := s.items[id]
	if !exists {
		return nil, ErrScheduleNotFound
	}
	return item, nil
}
