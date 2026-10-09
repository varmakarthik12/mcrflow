package playout

import (
	"strings"
	"testing"

	"mcrflow/internal/models"
	"mcrflow/internal/resolution"
)

func TestBuildFfmpegCommand(t *testing.T) {
	builder := NewPipelineBuilder()
	resStore := resolution.NewStore()
	preset, _ := resStore.GetPreset("res-1080i50-pal-hd")

	ch := &models.Channel{
		ID:           "ch-01",
		Name:         "Star Gold HD",
		LogoPosition: "TOP_RIGHT",
		VideoCodec:   "h264_nvenc",
		AudioCodec:   "aac",
		Destinations: []models.StreamDestination{
			{
				Protocol:    models.ProtocolUDPMulticast,
				Enabled:     true,
				EndpointURL: "udp://239.255.10.1:5000",
			},
			{
				Protocol:    models.ProtocolSRT,
				Enabled:     true,
				EndpointURL: "srt://edge.star.in:9000",
			},
		},
	}

	audioSel := &models.AudioTrackSelection{
		StreamIndex:      1,
		LanguageCode:     "hin",
		Channels:         "5.1",
		EbuR128Normalize: true,
	}

	args := builder.BuildFfmpegCommand(
		"/media/movies/Jawan.mkv",
		ch,
		preset,
		audioSel,
		"/branding/logo.png",
	)

	cmdStr := strings.Join(args, " ")

	if !strings.Contains(cmdStr, "-re -i /media/movies/Jawan.mkv") {
		t.Errorf("expected real-time input flag in command: %s", cmdStr)
	}

	if !strings.Contains(cmdStr, "scale=1920:1080") {
		t.Errorf("expected 1080p scale filter: %s", cmdStr)
	}

	if !strings.Contains(cmdStr, "overlay=W-w-30:30") {
		t.Errorf("expected top-right logo overlay: %s", cmdStr)
	}

	if !strings.Contains(cmdStr, "loudnorm=I=-23") {
		t.Errorf("expected EBU R128 loudness filter: %s", cmdStr)
	}

	if !strings.Contains(cmdStr, "-f mpegts udp://239.255.10.1:5000") {
		t.Errorf("expected UDP multicast egress: %s", cmdStr)
	}

	if !strings.Contains(cmdStr, "-f mpegts srt://edge.star.in:9000") {
		t.Errorf("expected SRT egress: %s", cmdStr)
	}
}
