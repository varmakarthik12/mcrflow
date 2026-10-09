package channel

import (
	"errors"
	"strings"

	"github.com/varmakarthik12/mcrflow/internal/models"
)

var (
	ErrNameRequired     = errors.New("channel name is required")
	ErrCallSignRequired = errors.New("channel call sign is required")
)

// Store defines persistence operations for channels
type Store interface {
	CreateChannel(ch *models.Channel) error
	GetChannelByID(id string) (*models.Channel, error)
	ListChannels() ([]models.Channel, error)
	UpdateChannel(ch *models.Channel) error
	DeleteChannel(id string) error
}

// Service handles business logic for broadcast channels
type Service struct {
	store Store
}

// NewService creates a new channel service
func NewService(store Store) *Service {
	return &Service{store: store}
}

// CreateChannel validates and saves a new channel
func (s *Service) CreateChannel(ch *models.Channel) error {
	ch.Name = strings.TrimSpace(ch.Name)
	if ch.Name == "" {
		return ErrNameRequired
	}
	ch.CallSign = strings.TrimSpace(ch.CallSign)
	if ch.CallSign == "" {
		return ErrCallSignRequired
	}
	if ch.ResolutionID == "" {
		ch.ResolutionID = "res-in-1080i50"
	}
	return s.store.CreateChannel(ch)
}

// GetChannelByID retrieves a channel
func (s *Service) GetChannelByID(id string) (*models.Channel, error) {
	return s.store.GetChannelByID(id)
}

// ListChannels returns all channels
func (s *Service) ListChannels() ([]models.Channel, error) {
	return s.store.ListChannels()
}

// UpdateChannel validates and updates a channel
func (s *Service) UpdateChannel(ch *models.Channel) error {
	ch.Name = strings.TrimSpace(ch.Name)
	if ch.Name == "" {
		return ErrNameRequired
	}
	ch.CallSign = strings.TrimSpace(ch.CallSign)
	if ch.CallSign == "" {
		return ErrCallSignRequired
	}
	return s.store.UpdateChannel(ch)
}

// DeleteChannel removes a channel
func (s *Service) DeleteChannel(id string) error {
	return s.store.DeleteChannel(id)
}
