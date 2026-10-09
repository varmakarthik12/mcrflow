package adtemplate

import (
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/varmakarthik12/mcrflow/internal/models"
)

var (
	ErrTemplateNotFound = errors.New("ad template not found")
	ErrInvalidTemplate  = errors.New("template name is required")
)

// Store manages graphics and ad break templates.
type Store struct {
	mu        sync.RWMutex
	templates map[string]*models.AdTemplate
}

// NewStore initializes ad template store.
func NewStore() *Store {
	s := &Store{
		templates: make(map[string]*models.AdTemplate),
	}
	s.loadDefaultTemplates()
	return s
}

func (s *Store) loadDefaultTemplates() {
	s.templates["ad-tmpl-diwali"] = &models.AdTemplate{
		ID:          "ad-tmpl-diwali",
		Name:        "Diwali Prime Festive Bug & Rolls",
		Description: "Festive sponsor bug, animated lower-third, breaking news ticker, and mid-roll breaks with SCTE-35",
		BannerOverlays: []models.BannerOverlayElement{
			{
				ID:                     "elem-sponsor-bug",
				ElementType:            "IMAGE",
				AssetURL:               "/branding/vivo_festive.png",
				PosXPercent:            85.0,
				PosYPercent:            10.0,
				WidthPercent:           10.0,
				HeightPercent:          6.0,
				Opacity:                0.95,
				EntranceAnimation:      models.TransitionSlideLeft,
				ExitAnimation:          models.TransitionFade,
				AnimationDurationMs:    600,
				StartOffsetSeconds:     5,
				DisplayDurationSeconds: 15,
				RepeatIntervalSeconds:  900,
			},
			{
				ID:                     "elem-lower-third",
				ElementType:            "LOWER_THIRD",
				TextContent:            "COMING UP NEXT: DUNKI (2023) AT 19:30 IST",
				PosXPercent:            10.0,
				PosYPercent:            82.0,
				WidthPercent:           45.0,
				HeightPercent:          8.0,
				Opacity:                0.90,
				EntranceAnimation:      models.TransitionSlideBottom,
				ExitAnimation:          models.TransitionFade,
				AnimationDurationMs:    500,
				StartOffsetSeconds:     60,
				DisplayDurationSeconds: 12,
				RepeatIntervalSeconds:  1200,
			},
			{
				ID:                     "elem-ticker",
				ElementType:            "TICKER_TEXT",
				TextContent:            "SPECIAL FESTIVE BROADCAST: TATA MOTORS PRESENTS BLOCKBUSTER MOVIE MARATHON",
				PosXPercent:            0.0,
				PosYPercent:            94.0,
				WidthPercent:           100.0,
				HeightPercent:          6.0,
				Opacity:                0.95,
				EntranceAnimation:      models.TransitionFade,
				ExitAnimation:          models.TransitionFade,
				AnimationDurationMs:    300,
				StartOffsetSeconds:     0,
				DisplayDurationSeconds: 3600,
				RepeatIntervalSeconds:  0,
			},
		},
		AdRolls: []models.AdRollSlot{
			{
				SlotType: "PRE_ROLL",
				Clips: []models.AdRollClip{
					{ID: "clip-01", MediaPath: "/ads/StarFestive_Promo_30s.mp4", DurationSeconds: 30, Title: "Festive Promo"},
				},
				EmitScte35:    true,
				Scte35EventID: 101,
			},
			{
				SlotType:             "MID_ROLL",
				MidRollOffsetSeconds: 2700, // 45 mins
				Clips: []models.AdRollClip{
					{ID: "clip-02", MediaPath: "/ads/Vivo_Diwali_Commercial.mp4", DurationSeconds: 30, Title: "Vivo Spot"},
					{ID: "clip-03", MediaPath: "/ads/CocaCola_Festive_Spot.mp4", DurationSeconds: 30, Title: "Coke Spot"},
				},
				EmitScte35:    true,
				Scte35EventID: 102,
			},
		},
	}

	s.templates["ad-tmpl-generic"] = &models.AdTemplate{
		ID:          "ad-tmpl-generic",
		Name:        "Standard Station Identification Template",
		Description: "Minimalist corner bug and station ident promo",
		BannerOverlays: []models.BannerOverlayElement{
			{
				ID:                     "elem-station-ident",
				ElementType:            "IMAGE",
				AssetURL:               "/branding/station_ident.png",
				PosXPercent:            88.0,
				PosYPercent:            8.0,
				WidthPercent:           8.0,
				HeightPercent:          5.0,
				Opacity:                0.85,
				EntranceAnimation:      models.TransitionFade,
				ExitAnimation:          models.TransitionFade,
				AnimationDurationMs:    400,
				StartOffsetSeconds:     0,
				DisplayDurationSeconds: 3600,
				RepeatIntervalSeconds:  0,
			},
		},
	}
}

// ListTemplates returns all ad templates.
func (s *Store) ListTemplates() []*models.AdTemplate {
	s.mu.RLock()
	defer s.mu.RUnlock()

	res := make([]*models.AdTemplate, 0, len(s.templates))
	for _, t := range s.templates {
		res = append(res, t)
	}
	return res
}

// GetTemplate retrieves a template by ID.
func (s *Store) GetTemplate(id string) (*models.AdTemplate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	t, exists := s.templates[id]
	if !exists {
		return nil, ErrTemplateNotFound
	}
	return t, nil
}

// CreateTemplate creates a new template.
func (s *Store) CreateTemplate(tmpl *models.AdTemplate) (*models.AdTemplate, error) {
	if tmpl.Name == "" {
		return nil, ErrInvalidTemplate
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if tmpl.ID == "" {
		tmpl.ID = fmt.Sprintf("ad-tmpl-%s", uuid.New().String()[:8])
	}

	s.templates[tmpl.ID] = tmpl
	return tmpl, nil
}

// UpdateTemplate updates an existing template.
func (s *Store) UpdateTemplate(id string, tmpl *models.AdTemplate) (*models.AdTemplate, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.templates[id]; !exists {
		return nil, ErrTemplateNotFound
	}

	tmpl.ID = id
	s.templates[id] = tmpl
	return tmpl, nil
}

// DeleteTemplate removes a template.
func (s *Store) DeleteTemplate(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.templates[id]; !exists {
		return ErrTemplateNotFound
	}
	delete(s.templates, id)
	return nil
}

// ResolveEffectiveTemplate implements the hierarchical precedence rule:
// Content-level template takes precedence; if empty, falls back to Channel-level template.
func ResolveEffectiveTemplate(s *Store, contentTmplID, channelTmplID string) (*models.AdTemplate, string) {
	if contentTmplID != "" {
		if tmpl, err := s.GetTemplate(contentTmplID); err == nil {
			return tmpl, "CONTENT_OVERRIDE"
		}
	}

	if channelTmplID != "" {
		if tmpl, err := s.GetTemplate(channelTmplID); err == nil {
			return tmpl, "CHANNEL_GLOBAL_DEFAULT"
		}
	}

	return nil, "NONE"
}
