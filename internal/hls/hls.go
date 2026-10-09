package hls

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"mcrflow/internal/models"
)

const (
	MaxActiveSegments = 10
	DefaultSegmentSec = 6.0
)

// SegmentInfo represents an active media segment in the rolling playlist.
type SegmentInfo struct {
	SequenceNum int64
	Filename    string
	DurationSec float64
	CreatedAt   time.Time
}

// ChannelStream tracks the 10-segment sliding window for a channel.
type ChannelStream struct {
	mu            sync.RWMutex
	ChannelID     string
	OutputDir     string
	MediaSequence int64
	Segments      []SegmentInfo
	WebToken      string
}

// Manager coordinates live sliding-window HLS streaming across channels.
type Manager struct {
	mu         sync.RWMutex
	baseDir    string
	streams    map[string]*ChannelStream
	tokenCheck func(channelID string) string // returns HlsWebToken for channel
}

// NewManager initializes the HLS streaming manager.
func NewManager(baseDir string, tokenCheck func(channelID string) string) *Manager {
	if baseDir == "" {
		baseDir = filepath.Join(os.TempDir(), "mcrflow-hls")
	}
	_ = os.MkdirAll(baseDir, 0755)

	return &Manager{
		baseDir:    baseDir,
		streams:    make(map[string]*ChannelStream),
		tokenCheck: tokenCheck,
	}
}

// GetOrCreateStream returns or initializes the 10-segment sliding window for a channel.
func (m *Manager) GetOrCreateStream(channelID string) *ChannelStream {
	m.mu.Lock()
	defer m.mu.Unlock()

	if s, exists := m.streams[channelID]; exists {
		return s
	}

	channelDir := filepath.Join(m.baseDir, channelID)
	_ = os.MkdirAll(channelDir, 0755)

	s := &ChannelStream{
		ChannelID:     channelID,
		OutputDir:     channelDir,
		MediaSequence: 100,
		Segments:      make([]SegmentInfo, 0, MaxActiveSegments),
	}

	// Initialize 10 rolling segments
	for i := 0; i < MaxActiveSegments; i++ {
		s.appendSegmentLocked(DefaultSegmentSec)
	}
	s.writePlaylistLocked()

	m.streams[channelID] = s
	return s
}

// AppendSegment advances the live sliding window by 1 segment, deleting segments > 10.
func (s *ChannelStream) AppendSegment(durationSec float64) SegmentInfo {
	s.mu.Lock()
	defer s.mu.Unlock()

	seg := s.appendSegmentLocked(durationSec)
	s.writePlaylistLocked()
	return seg
}

func (s *ChannelStream) appendSegmentLocked(durationSec float64) SegmentInfo {
	s.MediaSequence++
	filename := fmt.Sprintf("segment_%06d.ts", s.MediaSequence)
	segPath := filepath.Join(s.OutputDir, filename)

	// Create valid MPEG-TS mock packet (188 bytes beginning with sync byte 0x47)
	tsPacket := make([]byte, 188*10)
	for i := 0; i < len(tsPacket); i += 188 {
		tsPacket[i] = 0x47 // MPEG-TS Sync Byte
	}
	_ = os.WriteFile(segPath, tsPacket, 0644)

	seg := SegmentInfo{
		SequenceNum: s.MediaSequence,
		Filename:    filename,
		DurationSec: durationSec,
		CreatedAt:   time.Now().UTC(),
	}

	s.Segments = append(s.Segments, seg)

	// Enforce strictly 10 active segments in the sliding window
	for len(s.Segments) > MaxActiveSegments {
		oldSeg := s.Segments[0]
		s.Segments = s.Segments[1:]
		oldPath := filepath.Join(s.OutputDir, oldSeg.Filename)
		_ = os.Remove(oldPath)
	}

	return seg
}

// writePlaylistLocked generates the #EXTM3U media playlist with exactly 10 active segments.
func (s *ChannelStream) writePlaylistLocked() {
	if len(s.Segments) == 0 {
		return
	}

	firstSeq := s.Segments[0].SequenceNum

	var buf bytes.Buffer
	buf.WriteString("#EXTM3U\n")
	buf.WriteString("#EXT-X-VERSION:3\n")
	buf.WriteString(fmt.Sprintf("#EXT-X-TARGETDURATION:%d\n", int(DefaultSegmentSec)))
	buf.WriteString(fmt.Sprintf("#EXT-X-MEDIA-SEQUENCE:%d\n", firstSeq))

	for _, seg := range s.Segments {
		buf.WriteString(fmt.Sprintf("#EXTINF:%.3f,\n", seg.DurationSec))
		buf.WriteString(fmt.Sprintf("%s\n", seg.Filename))
	}

	playlistPath := filepath.Join(s.OutputDir, "master.m3u8")
	_ = os.WriteFile(playlistPath, buf.Bytes(), 0644)

	// Alias index.m3u8 as well
	indexPath := filepath.Join(s.OutputDir, "index.m3u8")
	_ = os.WriteFile(indexPath, buf.Bytes(), 0644)
}

// GetActiveSegments returns the current segment list for inspection.
func (s *ChannelStream) GetActiveSegments() []SegmentInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]SegmentInfo, len(s.Segments))
	copy(out, s.Segments)
	return out
}

// ServeHTTP handles HLS requests: /hls/{channel_id}/master.m3u8 or segment_*.ts
func (m *Manager) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Path format: /hls/{channel_id}/{file}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 || parts[0] != "hls" {
		http.NotFound(w, r)
		return
	}

	channelID := parts[1]
	requestedFile := parts[2]

	// Check WebToken requirement if configured
	if m.tokenCheck != nil {
		configuredToken := m.tokenCheck(channelID)
		if configuredToken != "" {
			reqToken := r.URL.Query().Get("token")
			if reqToken == "" {
				reqToken = r.Header.Get("X-HLS-Token")
			}
			if reqToken != configuredToken {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error": "unauthorized: invalid or missing hls webtoken"}`))
				return
			}
		}
	}

	stream := m.GetOrCreateStream(channelID)
	filePath := filepath.Join(stream.OutputDir, requestedFile)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	}

	if strings.HasSuffix(requestedFile, ".m3u8") {
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	} else if strings.HasSuffix(requestedFile, ".ts") {
		w.Header().Set("Content-Type", "video/mp2t")
		w.Header().Set("Cache-Control", "max-age=60")
	}

	http.ServeFile(w, r, filePath)
}

// ResolveHlsStreamURL returns the direct web URL for the channel HLS stream.
func ResolveHlsStreamURL(baseURL, channelID, webToken string) string {
	base := strings.TrimRight(baseURL, "/")
	if base == "" {
		base = "http://localhost:8080"
	}
	url := fmt.Sprintf("%s/hls/%s/master.m3u8", base, channelID)
	if webToken != "" {
		url += "?token=" + webToken
	}
	return url
}

// UpdateChannelHlsDestination updates or appends the direct HLS stream destination in a channel.
func UpdateChannelHlsDestination(ch *models.Channel, baseURL string) {
	resolvedURL := ResolveHlsStreamURL(baseURL, ch.ID, ch.HlsWebToken)
	ch.HlsStreamURL = resolvedURL

	hasHLS := false
	for i, dest := range ch.Destinations {
		if dest.Protocol == models.ProtocolHLS {
			ch.Destinations[i].EndpointURL = resolvedURL
			ch.Destinations[i].Enabled = true
			hasHLS = true
			break
		}
	}
	if !hasHLS {
		ch.Destinations = append(ch.Destinations, models.StreamDestination{
			Protocol:    models.ProtocolHLS,
			Enabled:     true,
			EndpointURL: resolvedURL,
		})
	}
}
