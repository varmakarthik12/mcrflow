package models

import (
	"time"
)

// RedundancyMode defines the failover strategy for channels.
type RedundancyMode string

const (
	RedundancyActivePassiveAutoFailover RedundancyMode = "ACTIVE_PASSIVE_AUTO_FAILOVER"
	RedundancyOnePlusOneMirroring       RedundancyMode = "ONE_PLUS_ONE_MIRRORING"
	RedundancyStandalone                RedundancyMode = "STANDALONE_NO_FALLBACK"
)

// StreamProtocol defines egress protocol types.
type StreamProtocol string

const (
	ProtocolUDPMulticast StreamProtocol = "UDP_MULTICAST"
	ProtocolSRT          StreamProtocol = "SRT"
	ProtocolRTMP         StreamProtocol = "RTMP"
	ProtocolHLS          StreamProtocol = "HLS"
	ProtocolNDI          StreamProtocol = "NDI"
)

// StreamDestination configures an egress stream target.
type StreamDestination struct {
	Protocol      StreamProtocol `json:"protocol"`
	Enabled       bool           `json:"enabled"`
	EndpointURL   string         `json:"endpoint_url"`
	SRTMode       string         `json:"srt_mode,omitempty"` // CALLER, LISTENER, RENDEZVOUS
	SRTLatencyMs  int            `json:"srt_latency_ms,omitempty"`
	SRTPassphrase string         `json:"srt_passphrase,omitempty"`
	StreamKey     string         `json:"stream_key,omitempty"`
}

// ResolutionPreset represents a broadcast resolution & encoding standard.
type ResolutionPreset struct {
	ID                   string    `json:"id"`
	Name                 string    `json:"name"`
	Category             string    `json:"category"` // Indian Cable HD, Indian Cable SD, Progressive HD, UHD, Custom
	Width                int       `json:"width"`
	Height               int       `json:"height"`
	FrameRate            float64   `json:"frame_rate"`
	AspectRatio          string    `json:"aspect_ratio"` // 16:9, 4:3, 16:9 Anamorphic
	ScanningMode         string    `json:"scanning_mode"` // progressive, interlaced
	InterlaceFilter      string    `json:"interlace_filter,omitempty"` // tinterlace, yadif
	VideoBitrateKbps     int       `json:"video_bitrate_kbps"`
	AudioBitrateKbps     int       `json:"audio_bitrate_kbps"`
	AudioSampleRate      int       `json:"audio_sample_rate"` // 48000
	PixelFormat          string    `json:"pixel_format"`      // yuv420p
	ColorSpace           string    `json:"color_space"`       // bt709, bt601, bt2020
	GopSize              int       `json:"gop_size"`
	ExtraFfmpegVideoArgs string    `json:"extra_ffmpeg_video_args,omitempty"`
	IsDefault            bool      `json:"is_default"`
	IsSystem             bool      `json:"is_system"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// Channel represents an active broadcast TV channel in MCRFlow.
type Channel struct {
	ID                   string              `json:"id"`
	Name                 string              `json:"name"`
	CallSign             string              `json:"call_sign"`
	LogicalChannelNumber int                 `json:"logical_channel_number"`
	LogoURL              string              `json:"logo_url"`
	LogoPosition         string              `json:"logo_position"` // TOP_RIGHT, TOP_LEFT, BOTTOM_RIGHT, BOTTOM_LEFT
	LogoOpacity          float64             `json:"logo_opacity"`
	PrimaryAgentID       string              `json:"primary_agent_id"`
	FallbackAgentID      string              `json:"fallback_agent_id"`
	RedundancyMode       RedundancyMode      `json:"redundancy_mode"`
	DefaultAdTemplateID  string              `json:"default_ad_template_id"`
	ResolutionPresetID   string              `json:"resolution_preset_id"`
	ResolutionPreset     *ResolutionPreset   `json:"resolution_preset,omitempty"`
	Destinations         []StreamDestination `json:"destinations"`
	VideoCodec           string              `json:"video_codec"` // h264_nvenc, libx264, hevc_nvenc, mpeg2video
	AudioCodec           string              `json:"audio_codec"` // aac, ac3, mp2
	Status               string              `json:"status"`      // ON_AIR, STANDBY, FAILOVER, ERROR
	NowPlayingID         string              `json:"now_playing_id,omitempty"`
	HlsWebToken          string              `json:"hls_web_token,omitempty"` // Optional query param token for HLS stream
	EpgWebToken          string              `json:"epg_web_token,omitempty"` // Optional query param token for EPG XML
	HlsStreamURL         string              `json:"hls_stream_url,omitempty"` // Resolved direct HLS streaming URL
	UpdatedAt            time.Time           `json:"updated_at"`
}

// UserRole defines user authorization levels in MCRFlow.
type UserRole string

const (
	RoleAdmin            UserRole = "admin"
	RoleOperator         UserRole = "operator"
	RoleContentScheduler UserRole = "content_scheduler"
)

// User represents an authorized user in MCRFlow.
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	DisplayName  string    `json:"display_name"`
	Email        string    `json:"email"`
	Role         UserRole  `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// UserAuthResponse is returned upon successful authentication.
type UserAuthResponse struct {
	Token     string    `json:"token"`
	User      *User     `json:"user"`
	ExpiresAt time.Time `json:"expires_at"`
}

// SetupStatus indicates whether initial administrator setup is needed.
type SetupStatus struct {
	SetupRequired bool `json:"setup_required"`
	UserCount     int  `json:"user_count"`
}

// ConflictAction determines how schedule collisions are handled.
type ConflictAction string

const (
	ConflictForceOverwrite   ConflictAction = "FORCE_OVERWRITE"
	ConflictQueueAfter       ConflictAction = "QUEUE_AFTER"
	ConflictReplaceConflict  ConflictAction = "REPLACE_CONFLICT"
)

// TmdbMetadata holds enriched movie/show metadata for EPG and on-air display.
type TmdbMetadata struct {
	TmdbID          int64    `json:"tmdb_id"`
	Title           string   `json:"title"`
	LocalizedTitle  string   `json:"localized_title,omitempty"`
	Overview        string   `json:"overview"`
	PosterURL       string   `json:"poster_url"`
	BackdropURL     string   `json:"backdrop_url,omitempty"`
	ReleaseYear     int      `json:"release_year"`
	Genres          []string `json:"genres"`
	ContentRating   string   `json:"content_rating"` // U, U/A 13+, U/A 16+, A
	RuntimeMinutes  int      `json:"runtime_minutes"`
	Director        string   `json:"director,omitempty"`
	Cast            []string `json:"cast,omitempty"`
}

// AudioTrackSelection specifies which audio PID to play.
type AudioTrackSelection struct {
	StreamIndex       int    `json:"stream_index"`
	LanguageCode      string `json:"language_code"` // hin, tam, tel, eng
	Channels          string `json:"channels"`      // 2.0, 5.1
	EbuR128Normalize  bool   `json:"ebu_r128_normalize"`
}

// SubtitleTrackSelection specifies subtitle behavior.
type SubtitleTrackSelection struct {
	StreamIndex  int    `json:"stream_index"`
	LanguageCode string `json:"language_code"`
	BurnIn       bool   `json:"burn_in"` // true = burn-in overlay, false = CEA-608/708 VANC
}

// AdvancedFfmpegConfig stores low-level broadcast overrides.
type AdvancedFfmpegConfig struct {
	CustomVideoFilters string `json:"custom_video_filters,omitempty"`
	CustomAudioFilters string `json:"custom_audio_filters,omitempty"`
	CustomMuxerArgs    string `json:"custom_muxer_args,omitempty"`
	ServicePID         int    `json:"service_pid,omitempty"`
	PmtPID             int    `json:"pmt_pid,omitempty"`
}

// ScheduleItem represents a single scheduled program event.
type ScheduleItem struct {
	ID                   string                 `json:"id"`
	ChannelID            string                 `json:"channel_id"`
	StorageMountID       string                 `json:"storage_mount_id"`
	MediaFilePath        string                 `json:"media_file_path"`
	StartTime            time.Time              `json:"start_time"`
	EndTime              time.Time              `json:"end_time"`
	DurationSeconds      int64                  `json:"duration_seconds"`
	TmdbMetadata         *TmdbMetadata          `json:"tmdb_metadata,omitempty"`
	ContentAdTemplateID  string                 `json:"content_ad_template_id,omitempty"` // overrides channel global
	AudioSelection       AudioTrackSelection    `json:"audio_selection"`
	SubtitleSelection    SubtitleTrackSelection `json:"subtitle_selection"`
	AdvancedFfmpeg       AdvancedFfmpegConfig   `json:"advanced_ffmpeg"`
	Status               string                 `json:"status"` // PLAYED, ON_AIR, CUED, PENDING
}

// StorageType represents the storage mounting technology.
type StorageType string

const (
	StorageLocalAlias StorageType = "LOCAL_ALIAS"
	StorageNFS        StorageType = "NFS"
	StorageSMB        StorageType = "SMB"
)

// StorageMount represents a connected storage volume.
type StorageMount struct {
	ID             string      `json:"id"`
	Name           string      `json:"name"`
	Type           StorageType `json:"type"`
	TargetPath     string      `json:"target_path"`
	ServerHost     string      `json:"server_host,omitempty"`
	ShareName      string      `json:"share_name,omitempty"`
	Username       string      `json:"username,omitempty"`
	IsActive       bool        `json:"is_active"`
	TotalBytes     int64       `json:"total_bytes"`
	AvailableBytes int64       `json:"available_bytes"`
}

// FileEntry represents a probed file or directory in storage.
type FileEntry struct {
	Name           string    `json:"name"`
	RelativePath   string    `json:"relative_path"`
	IsDirectory    bool      `json:"is_directory"`
	SizeBytes      int64     `json:"size_bytes"`
	ModTime        time.Time `json:"mod_time"`
	ProbedDuration string    `json:"probed_duration,omitempty"` // "02:49:12"
}

// AudioStreamInfo contains probed audio details from ffprobe.
type AudioStreamInfo struct {
	Index      int    `json:"index"`
	Codec      string `json:"codec"`
	Language   string `json:"language"`
	Channels   int    `json:"channels"`
	SampleRate int    `json:"sample_rate"`
}

// SubtitleStreamInfo contains probed subtitle details from ffprobe.
type SubtitleStreamInfo struct {
	Index    int    `json:"index"`
	Codec    string `json:"codec"`
	Language string `json:"language"`
	Title    string `json:"title,omitempty"`
}

// MediaProbeResult contains ffprobe container and stream analysis.
type MediaProbeResult struct {
	DurationSeconds int64                `json:"duration_seconds"`
	DurationString  string               `json:"duration_string"` // HH:MM:SS
	Resolution      string               `json:"resolution"`
	Framerate       float64              `json:"framerate"`
	VideoCodec      string               `json:"video_codec"`
	AudioStreams    []AudioStreamInfo    `json:"audio_streams"`
	SubtitleStreams []SubtitleStreamInfo `json:"subtitle_streams"`
	FileSizeBytes   int64                `json:"file_size_bytes"`
}

// BannerTransition defines entrance/exit animations.
type BannerTransition string

const (
	TransitionFade        BannerTransition = "FADE"
	TransitionSlideLeft   BannerTransition = "SLIDE_LEFT"
	TransitionSlideRight  BannerTransition = "SLIDE_RIGHT"
	TransitionSlideBottom BannerTransition = "SLIDE_BOTTOM"
	TransitionZoom        BannerTransition = "ZOOM"
	TransitionBounce      BannerTransition = "BOUNCE"
)

// BannerOverlayElement defines on-screen graphics overlay elements.
type BannerOverlayElement struct {
	ID                     string           `json:"id"`
	ElementType            string           `json:"element_type"` // IMAGE, GIF, VIDEO, TICKER_TEXT, LOWER_THIRD
	AssetURL               string           `json:"asset_url"`
	TextContent            string           `json:"text_content,omitempty"`
	PosXPercent            float64          `json:"pos_x_percent"` // 0.0 to 100.0%
	PosYPercent            float64          `json:"pos_y_percent"` // 0.0 to 100.0%
	WidthPercent           float64          `json:"width_percent"`
	HeightPercent          float64          `json:"height_percent"`
	Opacity                float64          `json:"opacity"`
	EntranceAnimation      BannerTransition `json:"entrance_animation"`
	ExitAnimation          BannerTransition `json:"exit_animation"`
	AnimationDurationMs    int              `json:"animation_duration_ms"`
	StartOffsetSeconds     int              `json:"start_offset_seconds"`
	DisplayDurationSeconds int              `json:"display_duration_seconds"`
	RepeatIntervalSeconds  int              `json:"repeat_interval_seconds"`
}

// AdRollClip defines a commercial video clip.
type AdRollClip struct {
	ID              string `json:"id"`
	MediaPath       string `json:"media_path"`
	DurationSeconds int    `json:"duration_seconds"`
	Title           string `json:"title"`
}

// AdRollSlot defines linear ad breaks (pre-roll, mid-roll, post-roll).
type AdRollSlot struct {
	SlotType            string       `json:"slot_type"` // PRE_ROLL, MID_ROLL, POST_ROLL
	MidRollOffsetSeconds int          `json:"mid_roll_offset_seconds,omitempty"`
	Clips               []AdRollClip `json:"clips"`
	EmitScte35          bool         `json:"emit_scte35"`
	Scte35EventID       int          `json:"scte35_event_id,omitempty"`
}

// AdTemplate bundles on-screen banners and commercial rolls.
type AdTemplate struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	BannerOverlays []BannerOverlayElement `json:"banner_overlays"`
	AdRolls        []AdRollSlot           `json:"ad_rolls"`
}

// EdgeAgent represents a distributed playout node.
type EdgeAgent struct {
	ID                 string    `json:"id"`
	Hostname           string    `json:"hostname"`
	IPAddress          string    `json:"ip_address"`
	TailscaleIP        string    `json:"tailscale_ip,omitempty"`
	Status             string    `json:"status"` // ONLINE, STREAMING, STANDBY, OFFLINE
	CpuUsagePercent    float64   `json:"cpu_usage_percent"`
	GpuUsagePercent    float64   `json:"gpu_usage_percent"`
	MemoryUsagePercent float64   `json:"memory_usage_percent"`
	ActiveChannelIDs   []string  `json:"active_channel_ids"`
	LastHeartbeat      time.Time `json:"last_heartbeat"`
}

// BotConfig configures a ChatOps bot instance.
type BotConfig struct {
	ID                string   `json:"id"`
	Platform          string   `json:"platform"` // TELEGRAM, SLACK, WHATSAPP
	BotName           string   `json:"bot_name"`
	APIToken          string   `json:"api_token"`
	AllowedChatIDs    []string `json:"allowed_chat_ids"`
	EnabledChannelIDs []string `json:"enabled_channel_ids"`
	IsActive          bool     `json:"is_active"`
	WebhookURL        string   `json:"webhook_url,omitempty"`
}

// TelemetryMessage is emitted for real-time WebSocket monitoring.
type TelemetryMessage struct {
	Type      string      `json:"type"` // CHANNEL_UPDATE, HEARTBEAT, ALERT, LOG
	Timestamp time.Time   `json:"timestamp"`
	Payload   interface{} `json:"payload"`
}
