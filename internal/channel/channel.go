package channel

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"mcrflow/internal/models"
	"mcrflow/internal/resolution"
)

var (
	ErrChannelNotFound = errors.New("channel not found")
	ErrInvalidChannel  = errors.New("channel name is required")
)

// Store manages channel state and redundancy orchestration.
type Store struct {
	mu          sync.RWMutex
	channels    map[string]*models.Channel
	resStore    *resolution.Store
	agentStatus map[string]time.Time // tracks last heartbeat of agents
}

// NewStore initializes a channel store.
func NewStore(resStore *resolution.Store) *Store {
	s := &Store{
		channels:    make(map[string]*models.Channel),
		resStore:    resStore,
		agentStatus: make(map[string]time.Time),
	}
	s.loadSampleChannels()
	return s
}

// loadSampleChannels sets up initial broadcast channels.
func (s *Store) loadSampleChannels() {
	now := time.Now().UTC()
	s.channels["ch-01"] = &models.Channel{
		ID:                   "ch-01",
		Name:                 "Star Gold HD",
		CallSign:             "STARGOLD-HD",
		LogicalChannelNumber: 101,
		LogoURL:              "/branding/stargold_hd.png",
		LogoPosition:         "TOP_RIGHT",
		LogoOpacity:          0.85,
		PrimaryAgentID:       "delhi-dc1-primary",
		FallbackAgentID:      "mumbai-dc2-hotstandby",
		RedundancyMode:       models.RedundancyActivePassiveAutoFailover,
		DefaultAdTemplateID:  "ad-tmpl-diwali",
		ResolutionPresetID:   "res-1080i50-pal-hd",
		Destinations: []models.StreamDestination{
			{
				Protocol:    models.ProtocolUDPMulticast,
				Enabled:     true,
				EndpointURL: "udp://239.255.10.1:5000?pkt_size=1316&ttl=16",
			},
			{
				Protocol:     models.ProtocolSRT,
				Enabled:      true,
				EndpointURL:  "srt://edge-delhi.star.in:9000",
				SRTMode:      "CALLER",
				SRTLatencyMs: 200,
			},
		},
		VideoCodec: "h264_nvenc",
		AudioCodec: "aac",
		Status:     "ON_AIR",
		UpdatedAt:  now,
	}

	s.channels["ch-02"] = &models.Channel{
		ID:                   "ch-02",
		Name:                 "Cinema One Tamil",
		CallSign:             "C1-TAMIL",
		LogicalChannelNumber: 204,
		LogoURL:              "/branding/cinema_one.png",
		LogoPosition:         "TOP_RIGHT",
		LogoOpacity:          0.90,
		PrimaryAgentID:       "mumbai-dc2-primary",
		FallbackAgentID:      "delhi-dc1-fallback",
		RedundancyMode:       models.RedundancyOnePlusOneMirroring,
		DefaultAdTemplateID:  "ad-tmpl-generic",
		ResolutionPresetID:   "res-1080i50-pal-hd",
		Destinations: []models.StreamDestination{
			{
				Protocol:    models.ProtocolUDPMulticast,
				Enabled:     true,
				EndpointURL: "udp://239.255.20.1:5000",
			},
		},
		VideoCodec: "h264_nvenc",
		AudioCodec: "aac",
		Status:     "ON_AIR",
		UpdatedAt:  now,
	}
}

// ListChannels returns all channels with populated resolution presets.
func (s *Store) ListChannels() []*models.Channel {
	s.mu.RLock()
	defer s.mu.RUnlock()

	channels := make([]*models.Channel, 0, len(s.channels))
	for _, ch := range s.channels {
		copyCh := *ch
		if s.resStore != nil && copyCh.ResolutionPresetID != "" {
			if preset, err := s.resStore.GetPreset(copyCh.ResolutionPresetID); err == nil {
				copyCh.ResolutionPreset = preset
			}
		}
		channels = append(channels, &copyCh)
	}
	return channels
}

// GetChannel retrieves a single channel by ID.
func (s *Store) GetChannel(id string) (*models.Channel, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ch, exists := s.channels[id]
	if !exists {
		return nil, ErrChannelNotFound
	}
	copyCh := *ch
	if s.resStore != nil && copyCh.ResolutionPresetID != "" {
		if preset, err := s.resStore.GetPreset(copyCh.ResolutionPresetID); err == nil {
			copyCh.ResolutionPreset = preset
		}
	}
	return &copyCh, nil
}

// CreateChannel creates a new channel.
func (s *Store) CreateChannel(ch *models.Channel) (*models.Channel, error) {
	if ch.Name == "" {
		return nil, ErrInvalidChannel
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if ch.ID == "" {
		ch.ID = fmt.Sprintf("ch-%02d", len(s.channels)+1)
	}
	if ch.Status == "" {
		ch.Status = "STANDBY"
	}
	if ch.RedundancyMode == "" {
		ch.RedundancyMode = models.RedundancyActivePassiveAutoFailover
	}
	if ch.ResolutionPresetID == "" {
		ch.ResolutionPresetID = "res-1080i50-pal-hd"
	}
	if ch.VideoCodec == "" {
		ch.VideoCodec = "libx264"
	}
	if ch.AudioCodec == "" {
		ch.AudioCodec = "aac"
	}

	ch.UpdatedAt = time.Now().UTC()
	s.channels[ch.ID] = ch

	return ch, nil
}

// UpdateChannel modifies an existing channel.
func (s *Store) UpdateChannel(id string, updated *models.Channel) (*models.Channel, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch, exists := s.channels[id]
	if !exists {
		return nil, ErrChannelNotFound
	}

	ch.Name = updated.Name
	ch.CallSign = updated.CallSign
	ch.LogicalChannelNumber = updated.LogicalChannelNumber
	ch.LogoURL = updated.LogoURL
	ch.LogoPosition = updated.LogoPosition
	ch.LogoOpacity = updated.LogoOpacity
	ch.PrimaryAgentID = updated.PrimaryAgentID
	ch.FallbackAgentID = updated.FallbackAgentID
	ch.RedundancyMode = updated.RedundancyMode
	ch.DefaultAdTemplateID = updated.DefaultAdTemplateID
	ch.ResolutionPresetID = updated.ResolutionPresetID
	ch.Destinations = updated.Destinations
	ch.VideoCodec = updated.VideoCodec
	ch.AudioCodec = updated.AudioCodec
	ch.UpdatedAt = time.Now().UTC()

	return ch, nil
}

// DeleteChannel removes a channel.
func (s *Store) DeleteChannel(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.channels[id]; !exists {
		return ErrChannelNotFound
	}
	delete(s.channels, id)
	return nil
}

// RecordAgentHeartbeat updates agent liveness timestamp.
func (s *Store) RecordAgentHeartbeat(agentID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.agentStatus[agentID] = time.Now().UTC()
}

// CheckAndTriggerFailovers checks for agent disconnects and executes failovers.
func (s *Store) CheckAndTriggerFailovers(heartbeatTimeout time.Duration) []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	failovers := []string{}

	for _, ch := range s.channels {
		if ch.Status != "ON_AIR" || ch.FallbackAgentID == "" {
			continue
		}

		lastHb, exists := s.agentStatus[ch.PrimaryAgentID]
		// If primary agent has not reported in heartbeatTimeout, trigger failover
		if !exists || now.Sub(lastHb) > heartbeatTimeout {
			ch.Status = "FAILOVER_STANDBY_ACTIVE"
			ch.UpdatedAt = now
			failovers = append(failovers, fmt.Sprintf("Channel %s (%s) failed over from %s to %s",
				ch.ID, ch.Name, ch.PrimaryAgentID, ch.FallbackAgentID))
		}
	}

	return failovers
}

// TriggerEmergencySlate sets a channel into emergency slate mode.
func (s *Store) TriggerEmergencySlate(id string, engage bool) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch, exists := s.channels[id]
	if !exists {
		return "", ErrChannelNotFound
	}

	if engage {
		ch.Status = "EMERGENCY_SLATE"
	} else {
		ch.Status = "ON_AIR"
	}
	ch.UpdatedAt = time.Now().UTC()
	return ch.Status, nil
}
