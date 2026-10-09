package resolution

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/varmakarthik12/mcrflow/internal/models"
)

var (
	ErrPresetNotFound  = errors.New("resolution preset not found")
	ErrCannotDeleteSys = errors.New("cannot delete system resolution preset")
	ErrInvalidParams   = errors.New("invalid resolution parameters: width, height and frame rate must be positive")
)

// Store manages resolution presets.
type Store struct {
	mu      sync.RWMutex
	presets map[string]*models.ResolutionPreset
}

// NewStore initializes a resolution preset store with standard Indian cable & broadcast presets.
func NewStore() *Store {
	s := &Store{
		presets: make(map[string]*models.ResolutionPreset),
	}
	s.loadDefaultPresets()
	return s
}

// loadDefaultPresets seeds standard broadcast presets for Indian cable operators (HD, SD, OTT).
func (s *Store) loadDefaultPresets() {
	now := time.Now().UTC()
	defaults := []*models.ResolutionPreset{
		{
			ID:                   "res-1080i50-pal-hd",
			Name:                 "1080i50 PAL HD (Standard Indian Cable / DVB-C / DTH)",
			Category:             "Indian Cable HD",
			Width:                1920,
			Height:               1080,
			FrameRate:            25.0,
			AspectRatio:          "16:9",
			ScanningMode:         "interlaced",
			InterlaceFilter:      "tinterlace=mode=interleave_top",
			VideoBitrateKbps:     8500,
			AudioBitrateKbps:     192,
			AudioSampleRate:      48000,
			PixelFormat:          "yuv420p",
			ColorSpace:           "bt709",
			GopSize:              25,
			ExtraFfmpegVideoArgs: "-flags +ildct+ilme -top 1 -b:v 8500k -minrate 8500k -maxrate 8500k -bufsize 17000k",
			IsDefault:            true,
			IsSystem:             true,
			CreatedAt:            now,
			UpdatedAt:            now,
		},
		{
			ID:                   "res-720p50-hd",
			Name:                 "720p50 PAL HD (Regional Sports & News Egress)",
			Category:             "Indian Cable HD",
			Width:                1280,
			Height:               720,
			FrameRate:            50.0,
			AspectRatio:          "16:9",
			ScanningMode:         "progressive",
			VideoBitrateKbps:     5500,
			AudioBitrateKbps:     192,
			AudioSampleRate:      48000,
			PixelFormat:          "yuv420p",
			ColorSpace:           "bt709",
			GopSize:              50,
			ExtraFfmpegVideoArgs: "-b:v 5500k -minrate 5500k -maxrate 5500k -bufsize 11000k",
			IsDefault:            false,
			IsSystem:             true,
			CreatedAt:            now,
			UpdatedAt:            now,
		},
		{
			ID:                   "res-576i50-sd-4x3",
			Name:                 "576i50 PAL SD 4:3 (Legacy Analog & Standard Cable)",
			Category:             "Indian Cable SD",
			Width:                720,
			Height:               576,
			FrameRate:            25.0,
			AspectRatio:          "4:3",
			ScanningMode:         "interlaced",
			InterlaceFilter:      "tinterlace=mode=interleave_top",
			VideoBitrateKbps:     3500,
			AudioBitrateKbps:     128,
			AudioSampleRate:      48000,
			PixelFormat:          "yuv420p",
			ColorSpace:           "bt601",
			GopSize:              25,
			ExtraFfmpegVideoArgs: "-flags +ildct+ilme -top 1 -aspect 4:3 -b:v 3500k -maxrate 3500k -bufsize 7000k",
			IsDefault:            false,
			IsSystem:             true,
			CreatedAt:            now,
			UpdatedAt:            now,
		},
		{
			ID:                   "res-576i50-sd-16x9",
			Name:                 "576i50 PAL SD 16:9 Anamorphic (Digital Cable SD)",
			Category:             "Indian Cable SD",
			Width:                720,
			Height:               576,
			FrameRate:            25.0,
			AspectRatio:          "16:9 Anamorphic",
			ScanningMode:         "interlaced",
			InterlaceFilter:      "tinterlace=mode=interleave_top",
			VideoBitrateKbps:     4000,
			AudioBitrateKbps:     192,
			AudioSampleRate:      48000,
			PixelFormat:          "yuv420p",
			ColorSpace:           "bt601",
			GopSize:              25,
			ExtraFfmpegVideoArgs: "-flags +ildct+ilme -top 1 -aspect 16:9 -b:v 4000k -maxrate 4000k -bufsize 8000k",
			IsDefault:            false,
			IsSystem:             true,
			CreatedAt:            now,
			UpdatedAt:            now,
		},
		{
			ID:                   "res-1080p50-ott-hd",
			Name:                 "1080p50 Progressive HD (FAST OTT / YouTube / Web Egress)",
			Category:             "Progressive HD",
			Width:                1920,
			Height:               1080,
			FrameRate:            50.0,
			AspectRatio:          "16:9",
			ScanningMode:         "progressive",
			VideoBitrateKbps:     9000,
			AudioBitrateKbps:     256,
			AudioSampleRate:      48000,
			PixelFormat:          "yuv420p",
			ColorSpace:           "bt709",
			GopSize:              50,
			ExtraFfmpegVideoArgs: "-b:v 9000k -minrate 9000k -maxrate 9000k -bufsize 18000k",
			IsDefault:            false,
			IsSystem:             true,
			CreatedAt:            now,
			UpdatedAt:            now,
		},
		{
			ID:                   "res-4k-2160p50-uhd",
			Name:                 "4K UHD 2160p50 (Premium UHD Direct Feed)",
			Category:             "Ultra HD",
			Width:                3840,
			Height:               2160,
			FrameRate:            50.0,
			AspectRatio:          "16:9",
			ScanningMode:         "progressive",
			VideoBitrateKbps:     25000,
			AudioBitrateKbps:     384,
			AudioSampleRate:      48000,
			PixelFormat:          "yuv420p10le",
			ColorSpace:           "bt2020",
			GopSize:              50,
			ExtraFfmpegVideoArgs: "-b:v 25000k -minrate 25000k -maxrate 25000k -bufsize 50000k",
			IsDefault:            false,
			IsSystem:             true,
			CreatedAt:            now,
			UpdatedAt:            now,
		},
	}

	for _, p := range defaults {
		s.presets[p.ID] = p
	}
}

// ListPresets returns all presets.
func (s *Store) ListPresets() []*models.ResolutionPreset {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*models.ResolutionPreset, 0, len(s.presets))
	for _, p := range s.presets {
		result = append(result, p)
	}
	return result
}

// GetPreset returns a preset by ID.
func (s *Store) GetPreset(id string) (*models.ResolutionPreset, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p, exists := s.presets[id]
	if !exists {
		return nil, ErrPresetNotFound
	}
	return p, nil
}

// CreatePreset registers a new custom resolution preset.
func (s *Store) CreatePreset(p *models.ResolutionPreset) (*models.ResolutionPreset, error) {
	if p.Width <= 0 || p.Height <= 0 || p.FrameRate <= 0 {
		return nil, ErrInvalidParams
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if p.ID == "" {
		p.ID = fmt.Sprintf("res-custom-%d", time.Now().UnixNano())
	}
	if p.Category == "" {
		p.Category = "Custom"
	}
	if p.PixelFormat == "" {
		p.PixelFormat = "yuv420p"
	}
	if p.ColorSpace == "" {
		p.ColorSpace = "bt709"
	}
	if p.AudioSampleRate <= 0 {
		p.AudioSampleRate = 48000
	}
	if p.AudioBitrateKbps <= 0 {
		p.AudioBitrateKbps = 192
	}
	if p.VideoBitrateKbps <= 0 {
		p.VideoBitrateKbps = 6000
	}

	now := time.Now().UTC()
	p.CreatedAt = now
	p.UpdatedAt = now
	p.IsSystem = false

	s.presets[p.ID] = p
	return p, nil
}

// UpdatePreset updates an existing preset.
func (s *Store) UpdatePreset(id string, updated *models.ResolutionPreset) (*models.ResolutionPreset, error) {
	if updated.Width <= 0 || updated.Height <= 0 || updated.FrameRate <= 0 {
		return nil, ErrInvalidParams
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	p, exists := s.presets[id]
	if !exists {
		return nil, ErrPresetNotFound
	}

	p.Name = updated.Name
	p.Category = updated.Category
	p.Width = updated.Width
	p.Height = updated.Height
	p.FrameRate = updated.FrameRate
	p.AspectRatio = updated.AspectRatio
	p.ScanningMode = updated.ScanningMode
	p.InterlaceFilter = updated.InterlaceFilter
	p.VideoBitrateKbps = updated.VideoBitrateKbps
	p.AudioBitrateKbps = updated.AudioBitrateKbps
	p.AudioSampleRate = updated.AudioSampleRate
	p.PixelFormat = updated.PixelFormat
	p.ColorSpace = updated.ColorSpace
	p.GopSize = updated.GopSize
	p.ExtraFfmpegVideoArgs = updated.ExtraFfmpegVideoArgs
	p.UpdatedAt = time.Now().UTC()

	return p, nil
}

// DeletePreset removes a custom preset. System presets cannot be deleted.
func (s *Store) DeletePreset(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, exists := s.presets[id]
	if !exists {
		return ErrPresetNotFound
	}

	if p.IsSystem {
		return ErrCannotDeleteSys
	}

	delete(s.presets, id)
	return nil
}

// GenerateFfmpegVideoArgs creates the exact FFmpeg command arguments for this resolution.
func GenerateFfmpegVideoArgs(p *models.ResolutionPreset) []string {
	args := []string{}

	// Video filter for scaling and aspect ratio
	scaleFilter := fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2",
		p.Width, p.Height, p.Width, p.Height)

	if p.ScanningMode == "interlaced" && p.InterlaceFilter != "" {
		scaleFilter = scaleFilter + "," + p.InterlaceFilter
	}

	args = append(args, "-vf", scaleFilter)
	args = append(args, "-r", fmt.Sprintf("%.2f", p.FrameRate))
	args = append(args, "-pix_fmt", p.PixelFormat)

	if p.ColorSpace != "" {
		args = append(args, "-colorspace", p.ColorSpace)
	}

	if p.VideoBitrateKbps > 0 {
		args = append(args, "-b:v", fmt.Sprintf("%dk", p.VideoBitrateKbps))
		args = append(args, "-minrate", fmt.Sprintf("%dk", p.VideoBitrateKbps))
		args = append(args, "-maxrate", fmt.Sprintf("%dk", p.VideoBitrateKbps))
		args = append(args, "-bufsize", fmt.Sprintf("%dk", p.VideoBitrateKbps*2))
	}

	if p.GopSize > 0 {
		args = append(args, "-g", fmt.Sprintf("%d", p.GopSize))
		args = append(args, "-keyint_min", fmt.Sprintf("%d", p.GopSize))
	}

	return args
}
