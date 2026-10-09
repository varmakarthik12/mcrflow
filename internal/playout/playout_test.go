package playout_test

import (
	"strings"
	"testing"
	"time"

	"github.com/varmakarthik12/mcrflow/internal/models"
	"github.com/varmakarthik12/mcrflow/internal/playout"
)

func TestBuildFFmpegArgs(t *testing.T) {
	cfg := playout.PlayoutConfig{
		InputMedia: "/media/movies/rrr_1080p.mp4",
		Resolution: models.ResolutionPreset{
			Width:           1920,
			Height:          1080,
			FrameRate:       25.0,
			Interlaced:      true,
			VideoCodec:      "libx264",
			AudioCodec:      "aac",
			ExtraFFmpegArgs: "-b:v 8M -maxrate 10M",
		},
		LogoPath:          "/logos/mcr_bug.png",
		LogoPosition:      "top-right",
		LogoOpacity:       0.85,
		AudioTrackIndex:   1, // Select second audio stream (e.g. English)
		NormalizeLoudness: true,
		Destinations: []models.StreamDestination{
			{Type: "udp", Enabled: true, URL: "udp://239.255.0.1", Port: 5000},
			{Type: "srt", Enabled: true, URL: "srt://127.0.0.1", Port: 9000, Mode: "caller", LatencyMs: 120},
			{Type: "rtmp", Enabled: true, URL: "rtmp://live.twitch.tv/app", StreamKey: "live_secret_key"},
			{Type: "hls", Enabled: true, URL: "/data/hls/ch-01/playlist.m3u8"},
		},
	}

	cmdStr := playout.BuildFFmpegCommand(cfg)

	// Verify key requirements
	if !strings.Contains(cmdStr, "-i /media/movies/rrr_1080p.mp4") {
		t.Errorf("missing input media in ffmpeg command")
	}
	if !strings.Contains(cmdStr, "-i /logos/mcr_bug.png") {
		t.Errorf("missing logo bug input in ffmpeg command")
	}
	if !strings.Contains(cmdStr, "-filter_complex") {
		t.Errorf("missing -filter_complex in ffmpeg command")
	}
	if !strings.Contains(cmdStr, "yadif=0:-1:1") {
		t.Errorf("missing yadif deinterlacer for interlaced preset")
	}
	if !strings.Contains(cmdStr, "colorchannelmixer=aa=0.85") {
		t.Errorf("missing logo opacity mixer")
	}
	if !strings.Contains(cmdStr, "overlay=") {
		t.Errorf("missing overlay filter")
	}
	if !strings.Contains(cmdStr, "-map 0:a:1") {
		t.Errorf("missing mapped audio track 0:a:1")
	}
	if !strings.Contains(cmdStr, "-af loudnorm=I=-23:LRA=7:TP=-1.0") {
		t.Errorf("missing EBU R128 loudness normalization filter")
	}
	if !strings.Contains(cmdStr, "udp://239.255.0.1:5000?pkt_size=1316") {
		t.Errorf("missing UDP destination")
	}
	if !strings.Contains(cmdStr, "srt://127.0.0.1:9000?mode=caller&latency=120") {
		t.Errorf("missing SRT destination")
	}
	if !strings.Contains(cmdStr, "rtmp://live.twitch.tv/app/live_secret_key") {
		t.Errorf("missing RTMP destination")
	}
	if !strings.Contains(cmdStr, "/data/hls/ch-01/playlist.m3u8") {
		t.Errorf("missing HLS destination")
	}
}

func TestEngineLifecycle(t *testing.T) {
	engine := playout.NewEngine()

	ch := models.Channel{
		ID:       "ch-01",
		Name:     "DD National",
		LogoPath: "/logos/logo.png",
	}
	res := models.ResolutionPreset{
		Width:     1920,
		Height:    1080,
		FrameRate: 25.0,
	}

	status, err := engine.StartChannel(ch, res, "/media/movie.mp4", "Feature Presentation")
	if err != nil {
		t.Fatalf("failed to start channel: %v", err)
	}
	if status.State != "ON-AIR" {
		t.Errorf("expected ON-AIR, got %s", status.State)
	}

	time.Sleep(100 * time.Millisecond)

	current := engine.GetStatus("ch-01")
	if current.State != "ON-AIR" {
		t.Errorf("expected ON-AIR status, got %s", current.State)
	}

	if err := engine.StopChannel("ch-01"); err != nil {
		t.Fatalf("failed to stop channel: %v", err)
	}

	stopped := engine.GetStatus("ch-01")
	if stopped.State != "STANDBY" {
		t.Errorf("expected STANDBY after stop, got %s", stopped.State)
	}
}
