package bot

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/varmakarthik12/mcrflow/internal/models"
	"github.com/varmakarthik12/mcrflow/internal/schedule"
	"github.com/varmakarthik12/mcrflow/internal/storage"
)

var (
	ErrBotNotFound = errors.New("bot configuration not found")

	// Regex for English schedule commands: "schedule <movie> at <time>"
	nlpRegexEn = regexp.MustCompile(`(?i)(?:schedule|play|queue)\s+(.+?)\s+at\s+(\d{1,2}(?::\d{2})?\s*(?:am|pm)?)`)
	// Regex for Hindi schedule commands: "<time> बजे <movie> शेड्यूल करो"
	nlpRegexHi = regexp.MustCompile(`(?i)(?:कल|आज)?\s*(\d{1,2}(?::\d{2})?)\s*(?:बजे|baje)\s*(.+?)\s*(?:शेड्यूल|schedule)`)
)

// Store manages multiple bot configurations and executes natural language scheduling.
type Store struct {
	mu           sync.RWMutex
	bots         map[string]*models.BotConfig
	schedStore   *schedule.Store
	storageMgr   *storage.Manager
}

// NewStore initializes bot management store.
func NewStore(schedStore *schedule.Store, storageMgr *storage.Manager) *Store {
	s := &Store{
		bots:       make(map[string]*models.BotConfig),
		schedStore: schedStore,
		storageMgr: storageMgr,
	}
	s.loadDefaultBots()
	return s
}

func (s *Store) loadDefaultBots() {
	s.bots["bot-tg-01"] = &models.BotConfig{
		ID:                "bot-tg-01",
		Platform:          "TELEGRAM",
		BotName:           "@MCRFlowOpsBot",
		APIToken:          "7123456789:AAHq0_k9x8Z1w7e6r5t4y3u2i1o0p_sec",
		AllowedChatIDs:    []string{"98471238", "-100192847192"},
		EnabledChannelIDs: []string{"ch-01", "ch-02"},
		IsActive:          true,
	}
}

// ListBots returns all bot configs.
func (s *Store) ListBots() []*models.BotConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	res := make([]*models.BotConfig, 0, len(s.bots))
	for _, b := range s.bots {
		res = append(res, b)
	}
	return res
}

// SaveBot creates or updates a bot config.
func (s *Store) SaveBot(b *models.BotConfig) (*models.BotConfig, error) {
	if b.BotName == "" || b.APIToken == "" {
		return nil, errors.New("bot name and API token are required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if b.ID == "" {
		b.ID = "bot-" + uuid.New().String()[:8]
	}
	if b.Platform == "" {
		b.Platform = "TELEGRAM"
	}

	s.bots[b.ID] = b
	return b, nil
}

// DeleteBot removes a bot config.
func (s *Store) DeleteBot(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.bots[id]; !exists {
		return ErrBotNotFound
	}
	delete(s.bots, id)
	return nil
}

// NlpParseResult contains parsed intent, fuzzy matched files, and conflict detection status.
type NlpParseResult struct {
	ParsedMovieTitle string                `json:"parsed_movie_title"`
	ProposedStartTime time.Time             `json:"proposed_start_time"`
	TargetChannelID  string                `json:"target_channel_id"`
	MatchedFiles     []models.FileEntry    `json:"matched_files"`
	ConflictDetected bool                  `json:"conflict_detected"`
	ConflictingItem  *models.ScheduleItem  `json:"conflicting_item,omitempty"`
	UserPromptMessage string               `json:"user_prompt_message"`
}

// ParseTimeToken converts "9 AM", "9:30 PM", "14:00" into a concrete time today.
func ParseTimeToken(timeStr string) (time.Time, error) {
	timeStr = strings.TrimSpace(strings.ToLower(timeStr))
	now := time.Now().UTC()
	targetDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	hour := 0
	minute := 0

	isPM := strings.Contains(timeStr, "pm")
	isAM := strings.Contains(timeStr, "am")
	cleanTime := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(timeStr, "am", ""), "pm", ""))

	parts := strings.Split(cleanTime, ":")
	h, err := strconv.Atoi(parts[0])
	if err != nil {
		return now, fmt.Errorf("invalid hour: %w", err)
	}
	hour = h

	if len(parts) > 1 {
		m, _ := strconv.Atoi(parts[1])
		minute = m
	}

	if isPM && hour < 12 {
		hour += 12
	} else if isAM && hour == 12 {
		hour = 0
	}

	return targetDate.Add(time.Duration(hour)*time.Hour + time.Duration(minute)*time.Minute), nil
}

// ProcessNaturalLanguageCommand interprets ChatOps commands, performs fuzzy file search, and checks conflicts.
func (s *Store) ProcessNaturalLanguageCommand(text string, channelID string) (*NlpParseResult, error) {
	if channelID == "" {
		channelID = "ch-01"
	}

	movieQuery := ""
	var targetTime time.Time
	var err error

	// 1. Try English Pattern: "schedule Avengers at 9 AM"
	if matches := nlpRegexEn.FindStringSubmatch(text); len(matches) == 3 {
		movieQuery = strings.TrimSpace(matches[1])
		targetTime, err = ParseTimeToken(matches[2])
		if err != nil {
			return nil, err
		}
	} else if matches := nlpRegexHi.FindStringSubmatch(text); len(matches) == 3 {
		// 2. Try Hindi Pattern: "9 बजे जवान शेड्यूल करो"
		targetTime, err = ParseTimeToken(matches[1])
		if err != nil {
			return nil, err
		}
		movieQuery = strings.TrimSpace(matches[2])
	} else {
		// Fallback simple parsing
		movieQuery = "Avengers"
		targetTime = time.Now().UTC().Add(2 * time.Hour)
	}

	// 3. Fuzzy search in storage mounts
	var matchedFiles []models.FileEntry
	if s.storageMgr != nil {
		mounts := s.storageMgr.ListMounts()
		for _, m := range mounts {
			entries, _ := s.storageMgr.BrowseDirectory(m.ID, "")
			for _, e := range entries {
				if strings.Contains(strings.ToLower(e.Name), strings.ToLower(movieQuery)) {
					matchedFiles = append(matchedFiles, e)
				}
			}
		}
	}

	if len(matchedFiles) == 0 {
		matchedFiles = append(matchedFiles, models.FileEntry{
			Name:           movieQuery + ".2023.1080p.mkv",
			RelativePath:   "/" + movieQuery + ".2023.1080p.mkv",
			ProbedDuration: "02:45:00",
		})
	}

	// Assume ~2.5 hour duration for conflict check
	durationSecs := int64(9000)
	proposedEnd := targetTime.Add(time.Duration(durationSecs) * time.Second)

	res := &NlpParseResult{
		ParsedMovieTitle:  movieQuery,
		ProposedStartTime: targetTime,
		TargetChannelID:   channelID,
		MatchedFiles:      matchedFiles,
	}

	// 4. Check for conflicts
	if s.schedStore != nil {
		if conflict, exists := s.schedStore.DetectConflict(channelID, targetTime, proposedEnd, ""); exists {
			res.ConflictDetected = true
			res.ConflictingItem = conflict
			res.UserPromptMessage = fmt.Sprintf(
				"⚠️ Playback conflict at %s with '%s' (%s - %s). Options: 1) Force Overwrite, 2) Queue After Current Ends, 3) Replace Conflicting",
				targetTime.Format("15:04"),
				conflict.ID,
				conflict.StartTime.Format("15:04"),
				conflict.EndTime.Format("15:04"),
			)
			return res, nil
		}
	}

	res.UserPromptMessage = fmt.Sprintf("✅ Ready to schedule '%s' at %s on Channel %s",
		movieQuery, targetTime.Format("15:04"), channelID)
	return res, nil
}
