package resolution

import (
	"errors"
	"strings"

	"github.com/varmakarthik12/mcrflow/internal/models"
)

var (
	ErrNameRequired       = errors.New("resolution preset name is required")
	ErrDimensionInvalid   = errors.New("width and height must be greater than zero")
	ErrCannotDeletePreset = errors.New("cannot delete standard broadcast preset")
)

// Store defines persistence operations for resolution presets
type Store interface {
	CreateResolution(res *models.ResolutionPreset) error
	GetResolutionByID(id string) (*models.ResolutionPreset, error)
	ListResolutions() ([]models.ResolutionPreset, error)
	UpdateResolution(res *models.ResolutionPreset) error
	DeleteResolution(id string) error
}

// Service manages resolution presets
type Service struct {
	store Store
}

// NewService creates a new resolution service
func NewService(store Store) *Service {
	return &Service{store: store}
}

// CreateResolution validates and creates a new custom resolution preset
func (s *Service) CreateResolution(res *models.ResolutionPreset) error {
	res.Name = strings.TrimSpace(res.Name)
	if res.Name == "" {
		return ErrNameRequired
	}
	if res.Width <= 0 || res.Height <= 0 {
		return ErrDimensionInvalid
	}
	if res.AspectRatio == "" {
		res.AspectRatio = "16:9"
	}
	if res.VideoCodec == "" {
		res.VideoCodec = "libx264"
	}
	if res.AudioCodec == "" {
		res.AudioCodec = "aac"
	}
	return s.store.CreateResolution(res)
}

// GetResolutionByID returns preset by ID
func (s *Service) GetResolutionByID(id string) (*models.ResolutionPreset, error) {
	return s.store.GetResolutionByID(id)
}

// ListResolutions returns all resolution presets
func (s *Service) ListResolutions() ([]models.ResolutionPreset, error) {
	return s.store.ListResolutions()
}

// UpdateResolution updates a resolution preset
func (s *Service) UpdateResolution(res *models.ResolutionPreset) error {
	if res.Width <= 0 || res.Height <= 0 {
		return ErrDimensionInvalid
	}
	return s.store.UpdateResolution(res)
}

// DeleteResolution deletes a non-preset resolution
func (s *Service) DeleteResolution(id string) error {
	preset, err := s.store.GetResolutionByID(id)
	if err != nil {
		return err
	}
	if preset.IsPreset {
		return ErrCannotDeletePreset
	}
	return s.store.DeleteResolution(id)
}
