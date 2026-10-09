package bot

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/varmakarthik12/mcrflow/internal/models"
	"github.com/varmakarthik12/mcrflow/internal/tmdb"
)

// ScheduleHandler handles schedule actions from ChatOps
type ScheduleHandler interface {
	CreateItem(item *models.ScheduleItem, allowOverlap bool) error
	ListByChannel(channelID string) ([]models.ScheduleItem, error)
}

// ChannelHandler handles channel actions from ChatOps
type ChannelHandler interface {
	ListChannels() ([]models.Channel, error)
	GetChannelByID(id string) (*models.Channel, error)
}

// TMDBHandler searches movies for metadata enrichment
type TMDBHandler interface {
	Search(query string) ([]tmdb.MovieResult, error)
}

// Service processes natural language ChatOps commands
type Service struct {
	schedHandler   ScheduleHandler
	channelHandler ChannelHandler
	tmdbHandler    TMDBHandler
}

// NewService creates a new ChatOps bot service
func NewService(sched ScheduleHandler, ch ChannelHandler, tmdb TMDBHandler) *Service {
	return &Service{
		schedHandler:   sched,
		channelHandler: ch,
		tmdbHandler:    tmdb,
	}
}

// ProcessCommand parses and executes a natural language command
func (s *Service) ProcessCommand(req models.BotMessageRequest) models.BotMessageResponse {
	text := strings.TrimSpace(req.Message)
	if text == "" {
		text = strings.TrimSpace(req.Command)
	}
	if text == "" {
		text = strings.TrimSpace(req.Text)
	}
	lower := strings.ToLower(text)

	if lower == "help" || lower == "/help" {
		return models.BotMessageResponse{
			Reply:   "MCRFlow ChatOps Commands:\n- schedule <Title> at <HH:MM> on <channel_id>\n- status <channel_id>\n- channels\n- search <Title>\n- cue ad <AdTitle>",
			Action:  "help",
			Success: true,
		}
	}

	if lower == "channels" || lower == "/channels" {
		channels, err := s.channelHandler.ListChannels()
		if err != nil {
			return models.BotMessageResponse{Reply: fmt.Sprintf("Error fetching channels: %v", err), Success: false}
		}
		var sb strings.Builder
		sb.WriteString("Active Broadcast Channels:\n")
		for _, c := range channels {
			state := "ACTIVE"
			if !c.IsActive {
				state = "INACTIVE"
			}
			sb.WriteString(fmt.Sprintf("- %s (%s) [%s]\n", c.Name, c.ID, state))
		}
		return models.BotMessageResponse{
			Reply:   sb.String(),
			Action:  "list_channels",
			Success: true,
		}
	}

	// Status command: status ch-01
	if strings.HasPrefix(lower, "status") {
		parts := strings.Fields(text)
		channelID := "ch-01"
		if len(parts) >= 2 {
			channelID = parts[1]
		}
		ch, err := s.channelHandler.GetChannelByID(channelID)
		if err != nil {
			return models.BotMessageResponse{
				Reply:   fmt.Sprintf("Channel %s not found: %v", channelID, err),
				Success: false,
			}
		}
		items, _ := s.schedHandler.ListByChannel(channelID)
		return models.BotMessageResponse{
			Reply:   fmt.Sprintf("Channel: %s (%s)\nStatus: On-Air\nScheduled Events: %d", ch.Name, ch.CallSign, len(items)),
			Action:  "channel_status",
			Success: true,
		}
	}

	// Search command: search RRR
	if strings.HasPrefix(lower, "search ") {
		query := strings.TrimPrefix(text, "search ")
		query = strings.TrimSpace(query)
		if s.tmdbHandler != nil {
			hits, err := s.tmdbHandler.Search(query)
			if err != nil || len(hits) == 0 {
				return models.BotMessageResponse{
					Reply:   fmt.Sprintf("No media found for '%s'", query),
					Action:  "search_media",
					Success: false,
				}
			}
			top := hits[0]
			return models.BotMessageResponse{
				Reply:   fmt.Sprintf("Found: %s (%s)\nRating: %.1f | Runtime: %dm\nSynopsis: %s", top.Title, top.ReleaseDate, top.Rating, top.Runtime, top.Overview),
				Action:  "search_media",
				Success: true,
			}
		}
	}

	// Cue ad command: cue ad sponsor_break
	if strings.HasPrefix(lower, "cue ad") {
		adTitle := strings.TrimSpace(strings.TrimPrefix(lower, "cue ad"))
		if adTitle == "" {
			adTitle = "Default Commercial Break"
		}
		return models.BotMessageResponse{
			Reply:   fmt.Sprintf("SCTE-35 Cue triggered for '%s'. Commercial break armed on active channels.", adTitle),
			Action:  "cue_ad",
			Success: true,
		}
	}

	// Schedule command regex: schedule (?P<title>.+) at (?P<time>\d{1,2}:\d{2}) on (?P<channel>ch-[\w-]+|\w+)
	schedRegex := regexp.MustCompile(`(?i)schedule\s+(.+?)\s+at\s+(\d{1,2}:\d{2})\s+on\s+([\w-]+)`)
	matches := schedRegex.FindStringSubmatch(text)
	if len(matches) == 4 {
		title := strings.TrimSpace(matches[1])
		timeStr := strings.TrimSpace(matches[2])
		channelID := strings.TrimSpace(matches[3])

		// Parse time today
		now := time.Now().UTC()
		var hour, min int
		_, _ = fmt.Sscanf(timeStr, "%d:%d", &hour, &min)
		scheduledStart := time.Date(now.Year(), now.Month(), now.Day(), hour, min, 0, 0, time.UTC)
		if scheduledStart.Before(now.Add(-10 * time.Minute)) {
			// If time has passed today, schedule for tomorrow
			scheduledStart = scheduledStart.Add(24 * time.Hour)
		}

		duration := 7200 // default 2 hours for movies
		tmdbPoster := ""
		tmdbOverview := ""
		tmdbID := ""

		if s.tmdbHandler != nil {
			if hits, err := s.tmdbHandler.Search(title); err == nil && len(hits) > 0 {
				tmdbID = hits[0].ID
				tmdbPoster = hits[0].PosterPath
				tmdbOverview = hits[0].Overview
				if hits[0].Runtime > 0 {
					duration = hits[0].Runtime * 60
				}
			}
		}

		item := &models.ScheduleItem{
			ChannelID:       channelID,
			ProgramTitle:    title,
			MediaPath:       fmt.Sprintf("/media/features/%s.mp4", strings.ToLower(strings.ReplaceAll(title, " ", "_"))),
			StartTime:       scheduledStart,
			DurationSeconds: duration,
			TmdbID:          tmdbID,
			TmdbPoster:      tmdbPoster,
			TmdbOverview:    tmdbOverview,
		}

		if err := s.schedHandler.CreateItem(item, false); err != nil {
			return models.BotMessageResponse{
				Reply:   fmt.Sprintf("Scheduling failed: %v", err),
				Action:  "schedule_media",
				Success: false,
			}
		}

		return models.BotMessageResponse{
			Reply:   fmt.Sprintf("Successfully scheduled '%s' at %s UTC on %s (Duration: %dm).", title, scheduledStart.Format("15:04"), channelID, duration/60),
			Action:  "schedule_media",
			Success: true,
		}
	}

	return models.BotMessageResponse{
		Reply:   fmt.Sprintf("Unrecognized command: '%s'. Type 'help' for available commands.", text),
		Action:  "unknown",
		Success: false,
	}
}
