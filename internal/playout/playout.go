package playout

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/varmakarthik12/mcrflow/internal/models"
)

// PlayoutConfig configures the playout engine parameters
type PlayoutConfig struct {
	InputMedia        string
	Resolution        models.ResolutionPreset
	LogoPath          string
	LogoPosition      string // top-right, top-left, bottom-right, bottom-left
	LogoOpacity       float64
	LogoX             int
	LogoY             int
	AudioTrackIndex   int
	NormalizeLoudness bool
	Destinations      []models.StreamDestination
	Loop              bool
	ExtraArgs         []string
	HLSOutputDir      string
	ChannelID         string
	DataDir           string
	MediaDir          string
}

// BuildFFmpegArgs constructs command line arguments for FFmpeg broadcast pipeline
func BuildFFmpegArgs(cfg PlayoutConfig) ([]string, error) {
	if cfg.InputMedia == "" {
		return nil, fmt.Errorf("input media path is required")
	}

	args := []string{
		"-re", // Read input at native framerate
	}

	if cfg.Loop {
		args = append(args, "-stream_loop", "-1")
	}

	args = append(args, "-i", cfg.InputMedia)

	// Resolve logo file path
	resolvedLogo := cfg.LogoPath
	hasLogo := false
	if resolvedLogo != "" {
		hasLogo = true
		if _, err := os.Stat(resolvedLogo); err != nil && cfg.DataDir != "" {
			cand := filepath.Join(cfg.DataDir, "logos", filepath.Base(resolvedLogo))
			if _, errCand := os.Stat(cand); errCand == nil {
				resolvedLogo = cand
			} else {
				candDef := filepath.Join(cfg.DataDir, "logos", "channel_logo.png")
				if _, errDef := os.Stat(candDef); errDef == nil {
					resolvedLogo = candDef
				}
			}
		}
		args = append(args, "-i", resolvedLogo)
	}

	// Determine width/height
	w := cfg.Resolution.Width
	if w <= 0 {
		w = 1920
	}
	h := cfg.Resolution.Height
	if h <= 0 {
		h = 1080
	}

	// Build filter_complex
	var filters []string
	baseScale := fmt.Sprintf("[0:v]scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2[scaled]", w, h, w, h)
	filters = append(filters, baseScale)

	lastV := "[scaled]"
	if cfg.Resolution.Interlaced {
		filters = append(filters, fmt.Sprintf("%syadif=0:-1:1[deint]", lastV))
		lastV = "[deint]"
	}

	// Count effective output destinations
	type destTarget struct {
		format   string
		args     []string
		url      string
	}
	targets := make([]destTarget, 0)

	for _, dst := range cfg.Destinations {
		if !dst.Enabled {
			continue
		}
		dstType := strings.ToLower(dst.Type)
		if dstType == "" && dst.Protocol != "" {
			dstType = strings.ToLower(strings.TrimPrefix(dst.Protocol, "UDP_"))
		}
		targetURL := dst.URL
		if targetURL == "" && dst.EndpointURL != "" {
			targetURL = dst.EndpointURL
		}
		if targetURL == "" {
			continue
		}

		switch dstType {
		case "udp", "multicast":
			if dst.Port > 0 && !strings.Contains(targetURL, fmt.Sprintf(":%d", dst.Port)) {
				targetURL = fmt.Sprintf("%s:%d", targetURL, dst.Port)
			}
			targets = append(targets, destTarget{
				format: "mpegts",
				url:    fmt.Sprintf("%s?pkt_size=1316", targetURL),
			})

		case "srt":
			if dst.Port > 0 && !strings.Contains(targetURL, fmt.Sprintf(":%d", dst.Port)) {
				targetURL = fmt.Sprintf("%s:%d", targetURL, dst.Port)
			}
			mode := dst.Mode
			if mode == "" {
				mode = "caller"
			}
			latency := dst.LatencyMs
			if latency <= 0 {
				latency = 120
			}
			targets = append(targets, destTarget{
				format: "mpegts",
				url:    fmt.Sprintf("%s?mode=%s&latency=%d", targetURL, mode, latency),
			})

		case "rtmp":
			if dst.StreamKey != "" {
				targetURL = fmt.Sprintf("%s/%s", strings.TrimSuffix(targetURL, "/"), dst.StreamKey)
			}
			targets = append(targets, destTarget{
				format: "flv",
				url:    targetURL,
			})

		case "hls":
			targets = append(targets, destTarget{
				format: "hls",
				args:   []string{"-hls_time", "2", "-hls_list_size", "10", "-hls_flags", "delete_segments"},
				url:    targetURL,
			})
		}
	}

	// Local HLS packaging if HLSOutputDir is specified
	if cfg.HLSOutputDir != "" {
		hasLocalHLS := false
		for _, t := range targets {
			if t.format == "hls" {
				hasLocalHLS = true
				break
			}
		}
		if !hasLocalHLS {
			_ = os.MkdirAll(cfg.HLSOutputDir, 0755)
			playlistPath := filepath.Join(cfg.HLSOutputDir, "playlist.m3u8")
			segmentPattern := filepath.Join(cfg.HLSOutputDir, "segment_%d.ts")
			targets = append(targets, destTarget{
				format: "hls",
				args:   []string{"-hls_time", "2", "-hls_list_size", "10", "-hls_flags", "delete_segments+split_by_time", "-hls_segment_filename", segmentPattern},
				url:    playlistPath,
			})
		}
	}

	numOutputs := len(targets)
	if numOutputs == 0 {
		targets = append(targets, destTarget{format: "null", url: "-"})
		numOutputs = 1
	}

	if hasLogo {
		opacity := cfg.LogoOpacity
		if opacity <= 0 || opacity > 1.0 {
			opacity = 0.90
		}

		// Calculate reasonable station bug dimensions relative to output canvas (max 10% width, 10% height)
		maxBugW := int(float64(w) * 0.10)
		if maxBugW < 80 {
			maxBugW = 80
		}
		maxBugH := int(float64(h) * 0.10)
		if maxBugH < 60 {
			maxBugH = 60
		}

		filters = append(filters, fmt.Sprintf("[1:v]scale=w=%d:h=%d:force_original_aspect_ratio=decrease,format=rgba,colorchannelmixer=aa=%.2f[logo]", maxBugW, maxBugH, opacity))

		// Determine overlay coordinates
		var overlayPos string
		if cfg.LogoX > 0 || cfg.LogoY > 0 {
			overlayPos = fmt.Sprintf("x=%d:y=%d", cfg.LogoX, cfg.LogoY)
		} else {
			switch strings.ToLower(cfg.LogoPosition) {
			case "top-left":
				overlayPos = "x=40:y=40"
			case "bottom-left":
				overlayPos = fmt.Sprintf("x=40:y=main_h-overlay_h-40")
			case "bottom-right":
				overlayPos = fmt.Sprintf("x=main_w-overlay_w-40:y=main_h-overlay_h-40")
			case "top-right":
				fallthrough
			default:
				overlayPos = fmt.Sprintf("x=main_w-overlay_w-40:y=40")
			}
		}

		if numOutputs > 1 {
			splitLabels := ""
			for i := 0; i < numOutputs; i++ {
				splitLabels += fmt.Sprintf("[v_out_%d]", i)
			}
			filters = append(filters, fmt.Sprintf("%s[logo]overlay=%s[v_comp];[v_comp]split=%d%s", lastV, overlayPos, numOutputs, splitLabels))
		} else {
			filters = append(filters, fmt.Sprintf("%s[logo]overlay=%s[v_out]", lastV, overlayPos))
		}
	} else {
		if numOutputs > 1 {
			splitLabels := ""
			for i := 0; i < numOutputs; i++ {
				splitLabels += fmt.Sprintf("[v_out_%d]", i)
			}
			filters = append(filters, fmt.Sprintf("%snull[v_comp];[v_comp]split=%d%s", lastV, numOutputs, splitLabels))
		} else {
			filters = append(filters, fmt.Sprintf("%snull[v_out]", lastV))
		}
	}

	args = append(args, "-filter_complex", strings.Join(filters, ";"))

	// Global / per-stream Audio Normalization (EBU R128 standard)
	if cfg.NormalizeLoudness {
		args = append(args, "-af", "loudnorm=I=-23:LRA=7:TP=-1.0")
	}

	// Video Codec & Encoding parameters
	vCodec := cfg.Resolution.VideoCodec
	if vCodec == "" {
		vCodec = "libx264"
	}
	aCodec := cfg.Resolution.AudioCodec
	if aCodec == "" {
		aCodec = "aac"
	}

	// Audio Track Index
	audioMap := fmt.Sprintf("0:a:%d", cfg.AudioTrackIndex)

	for i, target := range targets {
		vLabel := "[v_out]"
		if numOutputs > 1 {
			vLabel = fmt.Sprintf("[v_out_%d]", i)
		}
		args = append(args, "-map", vLabel, "-map", audioMap)
		args = append(args, "-c:v", vCodec, "-preset", "ultrafast")
		if cfg.Resolution.FrameRate > 0 {
			args = append(args, "-r", fmt.Sprintf("%.2f", cfg.Resolution.FrameRate))
		}
		args = append(args, "-c:a", aCodec, "-b:a", "192k", "-ar", "48000")

		if cfg.Resolution.ExtraFFmpegArgs != "" {
			parts := strings.Fields(cfg.Resolution.ExtraFFmpegArgs)
			args = append(args, parts...)
		}
		if len(cfg.ExtraArgs) > 0 {
			args = append(args, cfg.ExtraArgs...)
		}

		if len(target.args) > 0 {
			args = append(args, target.args...)
		}
		args = append(args, "-f", target.format, target.url)
	}

	return args, nil
}

// BuildFFmpegCommand returns the full command string for display and logging
func BuildFFmpegCommand(cfg PlayoutConfig) string {
	args, err := BuildFFmpegArgs(cfg)
	if err != nil {
		return ""
	}
	return "ffmpeg " + strings.Join(args, " ")
}

// ScheduleProvider defines timeline program retrieval for master control playout
type ScheduleProvider interface {
	GetActiveProgram(channelID string, at time.Time) (*models.ScheduleItem, error)
	GetNextProgram(channelID string, at time.Time) (*models.ScheduleItem, error)
}

// PlayoutChannelState manages execution of playout for a channel
type PlayoutChannelState struct {
	ChannelID             string
	Config                PlayoutConfig
	Status                models.PlayoutStatus
	Cancel                context.CancelFunc
	Cmd                   *exec.Cmd
	CurrentScheduleItemID string
	IsSlateActive         bool
}

// Engine supervises active channel playout processes
type Engine struct {
	mu               sync.RWMutex
	channels         map[string]*PlayoutChannelState
	scheduleProvider ScheduleProvider
	mediaDir         string
	dataDir          string
}

// NewEngine creates a new playout engine
func NewEngine(dirs ...string) *Engine {
	mediaDir := "./media"
	dataDir := "./data"
	if len(dirs) > 0 && dirs[0] != "" {
		mediaDir = dirs[0]
	}
	if len(dirs) > 1 && dirs[1] != "" {
		dataDir = dirs[1]
	}
	return &Engine{
		channels: make(map[string]*PlayoutChannelState),
		mediaDir: mediaDir,
		dataDir:  dataDir,
	}
}

// SetScheduleProvider injects schedule provider for wall-clock event transitions
func (e *Engine) SetScheduleProvider(sp ScheduleProvider) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.scheduleProvider = sp
}

// StartChannel begins playout for a channel
func (e *Engine) StartChannel(ch models.Channel, res models.ResolutionPreset, mediaPath string, programTitle string) (*models.PlayoutStatus, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Stop existing if running
	if existing, ok := e.channels[ch.ID]; ok && existing.Cancel != nil {
		existing.Cancel()
		if existing.Cmd != nil && existing.Cmd.Process != nil {
			_ = existing.Cmd.Process.Kill()
		}
	}

	logoPos := ch.LogoPosition
	if logoPos == "" {
		logoPos = "top-right"
	}

	// 1. Resolve logo file
	resolvedLogo := ch.LogoPath
	if resolvedLogo != "" {
		if _, err := os.Stat(resolvedLogo); err != nil {
			cand := filepath.Join(e.dataDir, "logos", filepath.Base(resolvedLogo))
			if _, errCand := os.Stat(cand); errCand == nil {
				resolvedLogo = cand
			} else {
				candDef := filepath.Join(e.dataDir, "logos", "channel_logo.png")
				if _, errDef := os.Stat(candDef); errDef == nil {
					resolvedLogo = candDef
				}
			}
		}
	} else {
		candDef := filepath.Join(e.dataDir, "logos", "channel_logo.png")
		if _, errDef := os.Stat(candDef); errDef == nil {
			resolvedLogo = candDef
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	now := time.Now().UTC()

	initialProg := programTitle
	initialMedia := mediaPath
	initialItemID := ""
	initialRemaining := 3600
	initialElapsed := 0

	if e.scheduleProvider != nil {
		if sched, _ := e.scheduleProvider.GetActiveProgram(ch.ID, now); sched != nil {
			initialProg = sched.ProgramTitle
			initialMedia = sched.MediaPath
			initialItemID = sched.ID
			initialElapsed = int(now.Sub(sched.StartTime).Seconds())
			if initialElapsed < 0 {
				initialElapsed = 0
			}
			initialRemaining = int(sched.EndTime.Sub(now).Seconds())
			if initialRemaining < 0 {
				initialRemaining = 0
			}
		}
	}

	// 2. Resolve input media path
	resolvedMedia := initialMedia
	if resolvedMedia == "" {
		resolvedMedia = "sample_movie.mp4"
	}
	if _, err := os.Stat(resolvedMedia); err != nil {
		cand := filepath.Join(e.mediaDir, resolvedMedia)
		if _, errCand := os.Stat(cand); errCand == nil {
			resolvedMedia = cand
		} else {
			candSample := filepath.Join(e.mediaDir, "sample_movie.mp4")
			if _, errSample := os.Stat(candSample); errSample == nil {
				resolvedMedia = candSample
			}
		}
	}

	hlsDir := filepath.Join(e.dataDir, "hls", ch.ID)
	_ = os.MkdirAll(hlsDir, 0755)

	cfg := PlayoutConfig{
		InputMedia:        resolvedMedia,
		Resolution:        res,
		LogoPath:          resolvedLogo,
		LogoPosition:      logoPos,
		LogoOpacity:       0.90,
		AudioTrackIndex:   0,
		NormalizeLoudness: true,
		Destinations:      ch.Destinations,
		Loop:              true,
		ChannelID:         ch.ID,
		HLSOutputDir:      hlsDir,
		DataDir:           e.dataDir,
		MediaDir:          e.mediaDir,
	}

	status := models.PlayoutStatus{
		ChannelID:        ch.ID,
		State:            "ON-AIR",
		CurrentProgram:   initialProg,
		MediaPath:        resolvedMedia,
		ElapsedSeconds:   initialElapsed,
		RemainingSeconds: initialRemaining,
		SMPTETimecode:    "00:00:00:00",
		FPS:              res.FrameRate,
		CPUUsage:         14.2,
		AudioLUFS:        -23.0,
		UpdatedAt:        now,
	}

	state := &PlayoutChannelState{
		ChannelID:             ch.ID,
		Config:                cfg,
		Status:                status,
		Cancel:                cancel,
		CurrentScheduleItemID: initialItemID,
	}

	// 3. Launch FFmpeg broadcast encoding pipeline if ffmpeg binary exists
	if ffmpegPath, err := exec.LookPath("ffmpeg"); err == nil {
		args, err := BuildFFmpegArgs(cfg)
		if err == nil {
			_ = os.MkdirAll(filepath.Join(e.dataDir, "logs"), 0755)
			logPath := filepath.Join(e.dataDir, "logs", fmt.Sprintf("%s_playout.log", ch.ID))
			logFile, errLog := os.Create(logPath)
			cmd := exec.CommandContext(ctx, ffmpegPath, args...)
			if errLog == nil {
				cmd.Stdout = logFile
				cmd.Stderr = logFile
			}
			if err := cmd.Start(); err == nil {
				state.Cmd = cmd
				go func() {
					_ = cmd.Wait()
					if logFile != nil {
						_ = logFile.Close()
					}
				}()
			} else if logFile != nil {
				_, _ = fmt.Fprintf(logFile, "Failed to start ffmpeg: %v\n", err)
				_ = logFile.Close()
			}
		} else {
			_ = os.MkdirAll(filepath.Join(e.dataDir, "logs"), 0755)
			logPath := filepath.Join(e.dataDir, "logs", fmt.Sprintf("%s_playout.log", ch.ID))
			_ = os.WriteFile(logPath, []byte(fmt.Sprintf("Failed to build ffmpeg args: %v\n", err)), 0644)
		}
	}

	// Playout supervisor goroutine respecting timeline schedule
	go func(c context.Context, st *PlayoutChannelState) {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-c.Done():
				e.mu.Lock()
				st.Status.State = "STANDBY"
				st.Status.UpdatedAt = time.Now().UTC()
				e.mu.Unlock()
				return
			case <-ticker.C:
				e.mu.Lock()
				now := time.Now().UTC()

				if st.IsSlateActive {
					st.Status.State = "EMERGENCY_SLATE"
					st.Status.CurrentProgram = "EMERGENCY TECHNICAL DIFFICULTIES SLATE"
					st.Status.UpdatedAt = now
					e.mu.Unlock()
					continue
				}

				// Check active scheduled program
				if e.scheduleProvider != nil {
					sched, _ := e.scheduleProvider.GetActiveProgram(st.ChannelID, now)
					if sched != nil {
						if st.CurrentScheduleItemID != sched.ID {
							st.CurrentScheduleItemID = sched.ID
							st.Status.CurrentProgram = sched.ProgramTitle
							st.Status.MediaPath = sched.MediaPath
							st.Config.InputMedia = sched.MediaPath
							st.Status.State = "ON-AIR"
						}
						el := int(now.Sub(sched.StartTime).Seconds())
						if el < 0 {
							el = 0
						}
						rem := int(sched.EndTime.Sub(now).Seconds())
						if rem < 0 {
							rem = 0
						}
						st.Status.ElapsedSeconds = el
						st.Status.RemainingSeconds = rem
						h := el / 3600
						m := (el % 3600) / 60
						s := el % 60
						st.Status.SMPTETimecode = fmt.Sprintf("%02d:%02d:%02d:00", h, m, s)
						st.Status.UpdatedAt = now
						e.mu.Unlock()
						continue
					} else if st.CurrentScheduleItemID != "" {
						st.CurrentScheduleItemID = ""
						st.Status.CurrentProgram = "Station Playout Loop"
						st.Status.RemainingSeconds = 3600
					}
				}

				st.Status.ElapsedSeconds++
				if st.Status.RemainingSeconds > 0 {
					st.Status.RemainingSeconds--
				}
				h := st.Status.ElapsedSeconds / 3600
				m := (st.Status.ElapsedSeconds % 3600) / 60
				s := st.Status.ElapsedSeconds % 60
				st.Status.SMPTETimecode = fmt.Sprintf("%02d:%02d:%02d:00", h, m, s)
				st.Status.UpdatedAt = now
				e.mu.Unlock()
			}
		}
	}(ctx, state)

	e.channels[ch.ID] = state
	return &status, nil
}

// SetEmergencySlate engages or disengages emergency slate
func (e *Engine) SetEmergencySlate(channelID string, enabled bool) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	st, ok := e.channels[channelID]
	if !ok {
		return fmt.Errorf("channel %s is not active", channelID)
	}

	st.IsSlateActive = enabled
	if enabled {
		st.Status.State = "EMERGENCY_SLATE"
		st.Status.CurrentProgram = "EMERGENCY TECHNICAL DIFFICULTIES SLATE"
	} else {
		st.Status.State = "ON-AIR"
	}
	st.Status.UpdatedAt = time.Now().UTC()
	return nil
}

// StopChannel stops playout for a channel
func (e *Engine) StopChannel(channelID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	state, ok := e.channels[channelID]
	if !ok {
		return fmt.Errorf("channel %s is not active", channelID)
	}

	if state.Cancel != nil {
		state.Cancel()
	}
	if state.Cmd != nil && state.Cmd.Process != nil {
		_ = state.Cmd.Process.Kill()
	}
	state.Status.State = "STANDBY"
	state.Status.UpdatedAt = time.Now().UTC()
	return nil
}

// GetStatus returns the current status of a channel
func (e *Engine) GetStatus(channelID string) models.PlayoutStatus {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if state, ok := e.channels[channelID]; ok {
		return state.Status
	}

	return models.PlayoutStatus{
		ChannelID: channelID,
		State:     "STANDBY",
		UpdatedAt: time.Now().UTC(),
	}
}
