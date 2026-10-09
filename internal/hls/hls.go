package hls

import (
	_ "embed"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/varmakarthik12/mcrflow/internal/models"
)

//go:embed sample.ts
var sampleTSBytes []byte

var (
	ErrUnauthorized    = errors.New("unauthorized: invalid or missing hls web token")
	ErrChannelNotFound = errors.New("channel not found")
	ErrInvalidSegment  = errors.New("invalid segment sequence requested")
)

// ChannelStore retrieves channel info for token authorization
type ChannelStore interface {
	GetChannelByID(id string) (*models.Channel, error)
}

// Manager oversees rolling HLS playlist generation and segment distribution
type Manager struct {
	mu           sync.RWMutex
	channelStore ChannelStore
	baseTime     time.Time
}

// NewManager creates a new HLS manager
func NewManager(store ChannelStore) *Manager {
	return &Manager{
		channelStore: store,
		baseTime:     time.Now().UTC(),
	}
}

// VerifyToken validates channel HLS web token
func (m *Manager) VerifyToken(channelID, token string) error {
	if m.channelStore == nil {
		return nil
	}

	ch, err := m.channelStore.GetChannelByID(channelID)
	if err != nil {
		return ErrChannelNotFound
	}

	// If channel has token set, client token must match exactly
	if ch.HlsWebToken != "" {
		token = strings.TrimSpace(token)
		if token != ch.HlsWebToken {
			return ErrUnauthorized
		}
	}

	return nil
}

// GenerateMasterManifest generates multivariant master playlist
func (m *Manager) GenerateMasterManifest(channelID, token string) (string, error) {
	if err := m.VerifyToken(channelID, token); err != nil {
		return "", err
	}

	tokenQuery := ""
	if token != "" {
		tokenQuery = "?token=" + token
	}

	manifest := fmt.Sprintf(`#EXTM3U
#EXT-X-VERSION:3
#EXT-X-INDEPENDENT-SEGMENTS
#EXT-X-STREAM-INF:BANDWIDTH=8000000,AVERAGE-BANDWIDTH=6000000,RESOLUTION=1920x1080,FRAME-RATE=25.000,CODECS="avc1.640028,mp4a.40.2"
playlist.m3u8%s
`, tokenQuery)

	return manifest, nil
}

// GenerateSlidingPlaylist generates a 10-segment rolling sliding window playlist
func (m *Manager) GenerateSlidingPlaylist(channelID, token string) (string, error) {
	if err := m.VerifyToken(channelID, token); err != nil {
		return "", err
	}

	tokenQuery := ""
	if token != "" {
		tokenQuery = "?token=" + token
	}

	// Calculate current 2-second sequence
	now := time.Now().UTC()
	currentSeq := int(now.Unix() / 2)
	windowSize := 10
	startSeq := currentSeq - windowSize + 1
	if startSeq < 0 {
		startSeq = 0
	}

	var sb strings.Builder
	sb.WriteString("#EXTM3U\n")
	sb.WriteString("#EXT-X-VERSION:3\n")
	sb.WriteString("#EXT-X-TARGETDURATION:2\n")
	sb.WriteString(fmt.Sprintf("#EXT-X-MEDIA-SEQUENCE:%d\n", startSeq))

	for i := 0; i < windowSize; i++ {
		seq := startSeq + i
		sb.WriteString("#EXTINF:2.000,\n")
		sb.WriteString(fmt.Sprintf("segment_%d.ts%s\n", seq, tokenQuery))
	}

	return sb.String(), nil
}

// GetSegmentData retrieves MPEG-TS packets for requested segment sequence
func (m *Manager) GetSegmentData(channelID string, seq int, token string) ([]byte, error) {
	if err := m.VerifyToken(channelID, token); err != nil {
		return nil, err
	}

	if len(sampleTSBytes) > 0 {
		return sampleTSBytes, nil
	}

	// Dynamic fallback: generate valid 188-byte MPEG-TS packets
	return generateFallbackTSPacket(), nil
}

// generateFallbackTSPacket constructs valid raw MPEG-TS sync packets
func generateFallbackTSPacket() []byte {
	packet := make([]byte, 188*7) // 7 TS packets
	for i := 0; i < 7; i++ {
		offset := i * 188
		packet[offset] = 0x47   // MPEG-TS Sync byte
		packet[offset+1] = 0x1F // PID 8191 (Null packet)
		packet[offset+2] = 0xFF
		packet[offset+3] = 0x10 // Payload only, CC 0
		for j := 4; j < 188; j++ {
			packet[offset+j] = 0xFF
		}
	}
	return packet
}
