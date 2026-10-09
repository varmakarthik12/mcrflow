package playout

import (
	"fmt"
	"strings"

	"mcrflow/internal/models"
)

// PipelineBuilder creates FFmpeg command lines for broadcast playout.
type PipelineBuilder struct{}

// NewPipelineBuilder creates a new playout pipeline builder.
func NewPipelineBuilder() *PipelineBuilder {
	return &PipelineBuilder{}
}

// BuildFfmpegCommand constructs the full FFmpeg command for a channel's active broadcast item.
func (b *PipelineBuilder) BuildFfmpegCommand(
	mediaPath string,
	ch *models.Channel,
	preset *models.ResolutionPreset,
	audioSelection *models.AudioTrackSelection,
	logoBugPath string,
) []string {
	args := []string{
		"-re", // Read input in real-time at native framerate
		"-i", mediaPath,
	}

	// Complex video filter chain
	var filterChain []string

	// 1. Scale & Pad to match Resolution Preset
	if preset != nil {
		scaleFilter := fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2",
			preset.Width, preset.Height, preset.Width, preset.Height)
		if preset.ScanningMode == "interlaced" && preset.InterlaceFilter != "" {
			scaleFilter = scaleFilter + "," + preset.InterlaceFilter
		}
		filterChain = append(filterChain, scaleFilter)
	}

	// 2. Overlay Station Logo / Bug if present
	if logoBugPath != "" {
		// Second input for logo bug
		args = append(args, "-i", logoBugPath)
		// Placement coordinates
		overlayPos := "W-w-30:30" // Top-Right default
		if ch.LogoPosition == "TOP_LEFT" {
			overlayPos = "30:30"
		} else if ch.LogoPosition == "BOTTOM_RIGHT" {
			overlayPos = "W-w-30:H-h-30"
		} else if ch.LogoPosition == "BOTTOM_LEFT" {
			overlayPos = "30:H-h-30"
		}
		filterChain = append(filterChain, fmt.Sprintf("overlay=%s", overlayPos))
	}

	if len(filterChain) > 0 {
		args = append(args, "-vf", strings.Join(filterChain, ","))
	}

	// Video Codec & Profile
	vCodec := ch.VideoCodec
	if vCodec == "" {
		vCodec = "libx264"
	}
	args = append(args, "-c:v", vCodec)

	// Framerate & Bitrate from preset
	if preset != nil {
		args = append(args, "-r", fmt.Sprintf("%.2f", preset.FrameRate))
		if preset.VideoBitrateKbps > 0 {
			args = append(args, "-b:v", fmt.Sprintf("%dk", preset.VideoBitrateKbps))
			args = append(args, "-minrate", fmt.Sprintf("%dk", preset.VideoBitrateKbps))
			args = append(args, "-maxrate", fmt.Sprintf("%dk", preset.VideoBitrateKbps))
			args = append(args, "-bufsize", fmt.Sprintf("%dk", preset.VideoBitrateKbps*2))
		}
		if preset.GopSize > 0 {
			args = append(args, "-g", fmt.Sprintf("%d", preset.GopSize))
		}
		if preset.PixelFormat != "" {
			args = append(args, "-pix_fmt", preset.PixelFormat)
		}
	}

	// Audio Stream Mapping & Loudness Normalization
	aCodec := ch.AudioCodec
	if aCodec == "" {
		aCodec = "aac"
	}
	args = append(args, "-c:a", aCodec)

	if audioSelection != nil {
		if audioSelection.EbuR128Normalize {
			// EBU R128 standard loudness filter: Integrated -23 LUFS, True Peak -1.0 dBTP
			args = append(args, "-af", "loudnorm=I=-23:LRA=7:TP=-1.0")
		}
		if preset != nil && preset.AudioBitrateKbps > 0 {
			args = append(args, "-b:a", fmt.Sprintf("%dk", preset.AudioBitrateKbps))
			args = append(args, "-ar", fmt.Sprintf("%d", preset.AudioSampleRate))
		}
	}

	// Multiplex outputs for each enabled destination
	for _, dest := range ch.Destinations {
		if !dest.Enabled {
			continue
		}
		switch dest.Protocol {
		case models.ProtocolUDPMulticast:
			args = append(args, "-f", "mpegts", dest.EndpointURL)
		case models.ProtocolSRT:
			args = append(args, "-f", "mpegts", dest.EndpointURL)
		case models.ProtocolRTMP:
			args = append(args, "-f", "flv", dest.EndpointURL)
		case models.ProtocolHLS:
			args = append(args, "-f", "hls", "-hls_time", "2", "-hls_list_size", "5", dest.EndpointURL)
		}
	}

	return args
}
