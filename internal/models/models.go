package models

import (
	"encoding/json"
	"time"
)

// User roles in Master Control Playout
const (
	RoleAdmin            = "admin"
	RoleOperator         = "operator"
	RoleContentScheduler = "content_scheduler"
)

// User represents an authorized operator or administrator
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	FullName     string    `json:"full_name"`
	DisplayName  string    `json:"display_name,omitempty"` // alias for full_name
	Email        string    `json:"email"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// UserCredentials for login
type UserCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// StreamDestination configures an egress stream target (UDP, RTMP, HLS)
type StreamDestination struct {
	Type        string `json:"type"`                   // udp, rtmp, hls
	Protocol    string `json:"protocol,omitempty"`     // UDP_MULTICAST, RTMP, HLS alias
	Enabled     bool   `json:"enabled"`                // active egress switch
	URL         string `json:"url"`                    // destination URL / address
	EndpointURL string `json:"endpoint_url,omitempty"` // alias for url
	Port        int    `json:"port"`                   // network port
	StreamKey   string `json:"stream_key"`             // RTMP key
}

// Channel represents a linear broadcast television channel
type Channel struct {
	ID               string              `json:"id"`
	Name             string              `json:"name"`
	CallSign         string              `json:"call_sign"`
	ResolutionID     string              `json:"resolution_id"`
	LogoPath         string              `json:"logo_path"`
	LogoPosition     string              `json:"logo_position"` // top-right, top-left, bottom-right, bottom-left
	LogoX            int                 `json:"logo_x"`
	LogoY            int                 `json:"logo_y"`
	LogoWidth        int                 `json:"logo_width"`
	LogoHeight       int                 `json:"logo_height"`
	LogoOpacity      float64             `json:"logo_opacity"`
	LogoFit          string              `json:"logo_fit"` // contain, cover
	OverlaysJSON     string              `json:"-"`
	Overlays         []OverlayElement    `json:"overlays"`
	AdTemplateID     string              `json:"ad_template_id"`
	PrimaryAgentID   string              `json:"primary_agent_id"`
	FallbackAgentID  string              `json:"fallback_agent_id"`
	HlsWebToken      string              `json:"hls_web_token"`
	EpgWebToken      string              `json:"epg_web_token"`
	DestinationsJSON string              `json:"-"`
	Destinations     []StreamDestination `json:"destinations"`
	IsActive         bool                `json:"is_active"`
	CreatedAt        time.Time           `json:"created_at"`
	UpdatedAt        time.Time           `json:"updated_at"`
}

// AudioTrackSelection represents an audio PID selection
type AudioTrackSelection struct {
	Index    int    `json:"index"`
	Language string `json:"language"`
	Codec    string `json:"codec"`
	Channels int    `json:"channels"`
	Title    string `json:"title"`
}

// ScheduleItem represents a programmed broadcast event on the timeline
type ScheduleItem struct {
	ID                 string    `json:"id"`
	ChannelID          string    `json:"channel_id"`
	ProgramTitle       string    `json:"program_title"`
	Title              string    `json:"title,omitempty"`           // alias for program_title
	MediaPath          string    `json:"media_path"`
	MediaFilePath      string    `json:"media_file_path,omitempty"` // alias for media_path
	StartTime          time.Time `json:"start_time"`
	DurationSeconds    int       `json:"duration_seconds"`
	EndTime            time.Time `json:"end_time"`
	TmdbID             string    `json:"tmdb_id"`
	TmdbPoster         string    `json:"tmdb_poster"`
	TmdbOverview        string    `json:"tmdb_overview"`
	AdTemplateID        string    `json:"ad_template_id"`
	SpecialPromoTitle   string    `json:"special_promo_title,omitempty"`
	SpecialPromoSubtext string    `json:"special_promo_subtext,omitempty"`
	AudioTrackIndex     int       `json:"audio_track_index"`
	SubtitleTrackIndex  int       `json:"subtitle_track_index"`
	CreatedAt           time.Time `json:"created_at"`
}

// ResolutionPreset represents a broadcast raster format standard (e.g. 1080i50, 720p50)
type ResolutionPreset struct {
	ID                   string    `json:"id"`
	Name                 string    `json:"name"`
	Width                int       `json:"width"`
	Height               int       `json:"height"`
	FrameRate            float64   `json:"frame_rate"`
	FPS                  float64   `json:"fps,omitempty"`
	Interlaced           bool      `json:"interlaced"`
	ScanningMode         string    `json:"scanning_mode,omitempty"`
	AspectRatio          string    `json:"aspect_ratio"`
	VideoCodec           string    `json:"video_codec"`
	AudioCodec           string    `json:"audio_codec"`
	ExtraFFmpegArgs      string    `json:"extra_ffmpeg_args"`
	ExtraFFmpegVideoArgs string    `json:"extra_ffmpeg_video_args,omitempty"`
	VideoBitrateKbps     int       `json:"video_bitrate_kbps,omitempty"`
	AudioBitrateKbps     int       `json:"audio_bitrate_kbps,omitempty"`
	IsPreset             bool      `json:"is_preset"`
	CreatedAt            time.Time `json:"created_at"`
}

// OverlayElement represents an on-screen graphics bug, ticker, or banner
type OverlayElement struct {
	ID                 string  `json:"id"`
	Type               string  `json:"type"` // logo_bug, lower_third, ticker, dve_squeeze, header_banner, footer_banner, now_playing, up_next, promo
	X                  int     `json:"x"`
	Y                  int     `json:"y"`
	Width              int     `json:"width"`
	Height             int     `json:"height"`
	Opacity            float64 `json:"opacity"`
	ImagePath          string  `json:"image_path"`
	Text               string  `json:"text"`
	SubText            string  `json:"sub_text,omitempty"`
	BackgroundColor    string  `json:"background_color,omitempty"`
	TextColor          string  `json:"text_color,omitempty"`
	FontSize           int     `json:"font_size,omitempty"`
	EntranceAnimation  string  `json:"entrance_animation"` // fade_in, slide_in_left, slide_in_bottom, scroll_left, zoom_in
	ExitAnimation      string  `json:"exit_animation"`     // fade_out, slide_out, zoom_out
	StartOffsetSeconds int     `json:"start_offset_seconds"`
	DurationSeconds    int     `json:"duration_seconds"`
	IsActive           bool    `json:"is_active"`
}

// CommercialBreak represents an ad insertion slot
type CommercialBreak struct {
	BreakType       string   `json:"break_type"` // pre_roll, mid_roll, post_roll
	OffsetSeconds   int      `json:"offset_seconds"`
	DurationSeconds int      `json:"duration_seconds"`
	Clips           []string `json:"clips"`
	Scte35Cue       bool     `json:"scte35_cue"`
}

// AdTemplate represents a reusable graphics, branding, and commercial layout template
type AdTemplate struct {
	ID                       string            `json:"id"`
	Name                     string            `json:"name"`
	TemplateType             string            `json:"template_type"` // overlay, commercial_break, composite
	LogoPath                 string            `json:"logo_path,omitempty"`
	LogoPosition             string            `json:"logo_position,omitempty"` // top-right, top-left, bottom-right, bottom-left, custom
	LogoX                    int               `json:"logo_x,omitempty"`
	LogoY                    int               `json:"logo_y,omitempty"`
	LogoWidth                int               `json:"logo_width,omitempty"`
	LogoHeight               int               `json:"logo_height,omitempty"`
	LogoOpacity              float64           `json:"logo_opacity,omitempty"`
	LogoFit                  string            `json:"logo_fit,omitempty"` // contain, cover
	OverlayElementsJSON      string            `json:"-"`
	OverlayElements          []OverlayElement  `json:"overlay_elements"`
	CommercialBreaksJSON     string            `json:"-"`
	CommercialBreaks         []CommercialBreak `json:"commercial_breaks"`
	CollisionBehavior        string            `json:"collision_behavior,omitempty"` // alternate, priority
	AlternateDurationSeconds int               `json:"alternate_duration_seconds,omitempty"`
	PriorityOrderJSON        string            `json:"-"`
	PriorityOrder            []string          `json:"priority_order,omitempty"`
	GeneralLayoutJSON        string            `json:"-"`
	GeneralLayout            map[string]any    `json:"general_layout,omitempty"`
	IsActive                 bool              `json:"is_active"`
	CreatedAt                time.Time         `json:"created_at"`
	UpdatedAt                time.Time         `json:"updated_at"`
}

// FileEntry represents a file or directory discovered in storage
type FileEntry struct {
	Name            string                `json:"name"`
	Path            string                `json:"path"`
	IsDir           bool                  `json:"is_dir"`
	Size            int64                 `json:"size"`
	Extension       string                `json:"extension"`
	DurationSeconds int                   `json:"duration_seconds,omitempty"`
	VideoCodec      string                `json:"video_codec,omitempty"`
	AudioTracks     []AudioTrackSelection `json:"audio_tracks,omitempty"`
	ModTime         time.Time             `json:"mod_time"`
}

// EdgeAgent represents a distributed playout daemon node
type EdgeAgent struct {
	ID                 string     `json:"id"`
	Hostname           string     `json:"hostname"`
	IPAddress          string     `json:"ip_address"`
	TailscaleIP        string     `json:"tailscale_ip"`
	Port               int        `json:"port"`
	PairingToken       string     `json:"pairing_token"`
	Token              string     `json:"token,omitempty"` // alias for pairing_token
	Status             string     `json:"status"`          // online, offline, standby, error
	LastHeartbeat      *time.Time `json:"last_heartbeat"`
	CPUPercent         float64    `json:"cpu_percent"`
	CPUUsagePercent    float64    `json:"cpu_usage_percent,omitempty"` // alias for cpu_percent
	MemoryPercent      float64    `json:"memory_percent"`
	MemoryUsagePercent float64    `json:"memory_usage_percent,omitempty"` // alias for memory_percent
	ActiveChannelsJSON string     `json:"-"`
	ActiveChannels     []string   `json:"active_channels"`
	CreatedAt          time.Time  `json:"created_at"`
}

// AgentHeartbeatPayload received from edge agents
type AgentHeartbeatPayload struct {
	AgentID        string   `json:"agent_id"`
	PairingToken   string   `json:"pairing_token"`
	CPUPercent     float64  `json:"cpu_percent"`
	MemoryPercent  float64  `json:"memory_percent"`
	ActiveChannels []string `json:"active_channels"`
}

// Bot represents a ChatOps Telegram, Slack, or Discord integration
type Bot struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Platform   string    `json:"platform"` // telegram, slack, discord
	APIKey     string    `json:"api_key"`
	WebhookURL string    `json:"webhook_url"`
	IsActive   bool      `json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
}

// BotMessageRequest represents an incoming command to ChatOps
type BotMessageRequest struct {
	Message   string `json:"message"`
	Sender    string `json:"sender,omitempty"`
	Command   string `json:"command,omitempty"`
	Text      string `json:"text,omitempty"`
	Platform  string `json:"platform,omitempty"`
	ChannelID string `json:"channel_id,omitempty"`
	UserID    string `json:"user_id,omitempty"`
}

// BotMessageResponse represents response from ChatOps
type BotMessageResponse struct {
	Reply   string `json:"reply"`
	Action  string `json:"action"`
	Success bool   `json:"success"`
}

// MediaProbeResult contains probed technical properties of video
type MediaProbeResult struct {
	Path            string                `json:"path"`
	Format          string                `json:"format"`
	DurationSeconds int                   `json:"duration_seconds"`
	Width           int                   `json:"width"`
	Height          int                   `json:"height"`
	VideoCodec      string                `json:"video_codec"`
	FrameRate       float64               `json:"frame_rate"`
	AudioTracks     []AudioTrackSelection `json:"audio_tracks"`
	Bitrate         int64                 `json:"bitrate"`
}

// PlayoutStatus holds real-time telemetry of a channel playout
type PlayoutStatus struct {
	ChannelID        string    `json:"channel_id"`
	State            string    `json:"state"` // ON-AIR, STANDBY, HOLD, ERROR
	CurrentProgram   string    `json:"current_program"`
	MediaPath        string    `json:"media_path"`
	ElapsedSeconds   int       `json:"elapsed_seconds"`
	RemainingSeconds int       `json:"remaining_seconds"`
	SMPTETimecode    string    `json:"smpte_timecode"`
	FPS              float64   `json:"fps"`
	CPUUsage         float64   `json:"cpu_usage"`
	AudioLUFS        float64   `json:"audio_lufs"`
	NextProgram      string    `json:"next_program"`
	NextStartTime    time.Time `json:"next_start_time"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// UnmarshalJSONHelpers
func (c *Channel) ParseDestinations() {
	if c.DestinationsJSON != "" {
		_ = json.Unmarshal([]byte(c.DestinationsJSON), &c.Destinations)
	}
	if c.Destinations == nil {
		c.Destinations = []StreamDestination{}
	}
}

func (c *Channel) PackDestinations() {
	if c.Destinations != nil {
		bytes, _ := json.Marshal(c.Destinations)
		c.DestinationsJSON = string(bytes)
	} else {
		c.DestinationsJSON = "[]"
	}
}

func (c *Channel) ParseOverlays() {
	if c.OverlaysJSON != "" {
		_ = json.Unmarshal([]byte(c.OverlaysJSON), &c.Overlays)
	}
	if c.Overlays == nil {
		c.Overlays = []OverlayElement{}
	}
}

func (c *Channel) PackOverlays() {
	if c.Overlays != nil {
		bytes, _ := json.Marshal(c.Overlays)
		c.OverlaysJSON = string(bytes)
	} else {
		c.OverlaysJSON = "[]"
	}
}

func (a *AdTemplate) ParseJSON() {
	if a.OverlayElementsJSON != "" {
		_ = json.Unmarshal([]byte(a.OverlayElementsJSON), &a.OverlayElements)
	}
	if a.OverlayElements == nil {
		a.OverlayElements = []OverlayElement{}
	}
	if a.CommercialBreaksJSON != "" {
		_ = json.Unmarshal([]byte(a.CommercialBreaksJSON), &a.CommercialBreaks)
	}
	if a.CommercialBreaks == nil {
		a.CommercialBreaks = []CommercialBreak{}
	}
	if a.PriorityOrderJSON != "" {
		_ = json.Unmarshal([]byte(a.PriorityOrderJSON), &a.PriorityOrder)
	}
	if a.PriorityOrder == nil {
		a.PriorityOrder = []string{}
	}
	if a.GeneralLayoutJSON != "" {
		_ = json.Unmarshal([]byte(a.GeneralLayoutJSON), &a.GeneralLayout)
	}
	if a.GeneralLayout == nil {
		a.GeneralLayout = make(map[string]any)
	}
}

func (a *AdTemplate) PackJSON() {
	if a.OverlayElements != nil {
		bytes, _ := json.Marshal(a.OverlayElements)
		a.OverlayElementsJSON = string(bytes)
	} else {
		a.OverlayElementsJSON = "[]"
	}
	if a.CommercialBreaks != nil {
		bytes, _ := json.Marshal(a.CommercialBreaks)
		a.CommercialBreaksJSON = string(bytes)
	} else {
		a.CommercialBreaksJSON = "[]"
	}
	if a.PriorityOrder != nil {
		bytes, _ := json.Marshal(a.PriorityOrder)
		a.PriorityOrderJSON = string(bytes)
	} else {
		a.PriorityOrderJSON = "[]"
	}
	if a.GeneralLayout != nil {
		bytes, _ := json.Marshal(a.GeneralLayout)
		a.GeneralLayoutJSON = string(bytes)
	} else {
		a.GeneralLayoutJSON = "{}"
	}
}

func (e *EdgeAgent) ParseChannels() {
	if e.ActiveChannelsJSON != "" {
		_ = json.Unmarshal([]byte(e.ActiveChannelsJSON), &e.ActiveChannels)
	}
	if e.ActiveChannels == nil {
		e.ActiveChannels = []string{}
	}
}

func (e *EdgeAgent) PackChannels() {
	if e.ActiveChannels != nil {
		bytes, _ := json.Marshal(e.ActiveChannels)
		e.ActiveChannelsJSON = string(bytes)
	} else {
		e.ActiveChannelsJSON = "[]"
	}
}
