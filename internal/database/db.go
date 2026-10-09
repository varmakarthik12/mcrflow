package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"

	"github.com/varmakarthik12/mcrflow/internal/models"
)

// DB wraps standard sql.DB with application helper methods
type DB struct {
	*sql.DB
}

// Open initializes SQLite connection with WAL mode, foreign keys, and busy timeout
func Open(dataSourceName string) (*DB, error) {
	// If path is a file path, ensure directory exists
	if dataSourceName != ":memory:" {
		dir := filepath.Dir(dataSourceName)
		if dir != "." && dir != "" {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return nil, fmt.Errorf("failed to create db directory: %w", err)
			}
		}
	}

	dsn := fmt.Sprintf("%s?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)", dataSourceName)
	if strings.Contains(dataSourceName, "?") {
		dsn = fmt.Sprintf("%s&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)", dataSourceName)
	}
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// Set connection limits
	sqlDB.SetMaxOpenConns(1) // SQLite works best with 1 writer connection
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetConnMaxLifetime(0)

	// Explicitly execute pragmas
	pragmas := []string{
		"PRAGMA journal_mode = WAL;",
		"PRAGMA foreign_keys = ON;",
		"PRAGMA busy_timeout = 5000;",
		"PRAGMA synchronous = NORMAL;",
	}
	for _, pragma := range pragmas {
		if _, err := sqlDB.Exec(pragma); err != nil {
			// On in-memory WAL might return error, but ignore if ping passes
			_ = err
		}
	}

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping sqlite database: %w", err)
	}

	db := &DB{DB: sqlDB}
	if err := db.Migrate(); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("failed to apply migrations: %w", err)
	}

	if err := db.Seed(); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("failed to seed initial data: %w", err)
	}

	return db, nil
}

// Migrate creates tables and indexes
func (db *DB) Migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		full_name TEXT NOT NULL,
		email TEXT NOT NULL,
		role TEXT NOT NULL,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS resolutions (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		width INTEGER NOT NULL,
		height INTEGER NOT NULL,
		frame_rate REAL NOT NULL,
		interlaced INTEGER NOT NULL DEFAULT 0,
		aspect_ratio TEXT NOT NULL DEFAULT '16:9',
		video_codec TEXT NOT NULL DEFAULT 'libx264',
		audio_codec TEXT NOT NULL DEFAULT 'aac',
		extra_ffmpeg_args TEXT NOT NULL DEFAULT '',
		is_preset INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS ad_templates (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		template_type TEXT NOT NULL DEFAULT 'composite',
		overlay_elements_json TEXT NOT NULL DEFAULT '[]',
		commercial_breaks_json TEXT NOT NULL DEFAULT '[]',
		is_active INTEGER NOT NULL DEFAULT 1,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS storage_mounts (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		mount_type TEXT NOT NULL,
		mount_path TEXT NOT NULL,
		smb_url TEXT NOT NULL DEFAULT '',
		is_active INTEGER NOT NULL DEFAULT 1,
		created_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS edge_agents (
		id TEXT PRIMARY KEY,
		hostname TEXT NOT NULL,
		ip_address TEXT NOT NULL,
		tailscale_ip TEXT NOT NULL DEFAULT '',
		port INTEGER NOT NULL DEFAULT 3082,
		pairing_token TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'offline',
		last_heartbeat DATETIME,
		cpu_percent REAL NOT NULL DEFAULT 0.0,
		memory_percent REAL NOT NULL DEFAULT 0.0,
		active_channels_json TEXT NOT NULL DEFAULT '[]',
		created_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS bots (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		platform TEXT NOT NULL,
		api_key TEXT NOT NULL DEFAULT '',
		webhook_url TEXT NOT NULL DEFAULT '',
		is_active INTEGER NOT NULL DEFAULT 1,
		created_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS channels (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		call_sign TEXT NOT NULL,
		resolution_id TEXT NOT NULL,
		logo_path TEXT NOT NULL DEFAULT '',
		ad_template_id TEXT NOT NULL DEFAULT '',
		primary_agent_id TEXT NOT NULL DEFAULT '',
		fallback_agent_id TEXT NOT NULL DEFAULT '',
		hls_web_token TEXT NOT NULL DEFAULT '',
		epg_web_token TEXT NOT NULL DEFAULT '',
		destinations_json TEXT NOT NULL DEFAULT '[]',
		is_active INTEGER NOT NULL DEFAULT 1,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS schedules (
		id TEXT PRIMARY KEY,
		channel_id TEXT NOT NULL,
		program_title TEXT NOT NULL,
		media_path TEXT NOT NULL,
		start_time DATETIME NOT NULL,
		duration_seconds INTEGER NOT NULL,
		end_time DATETIME NOT NULL,
		tmdb_id TEXT NOT NULL DEFAULT '',
		tmdb_poster TEXT NOT NULL DEFAULT '',
		tmdb_overview TEXT NOT NULL DEFAULT '',
		ad_template_id TEXT NOT NULL DEFAULT '',
		audio_track_index INTEGER NOT NULL DEFAULT 0,
		subtitle_track_index INTEGER NOT NULL DEFAULT -1,
		created_at DATETIME NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_schedules_channel_start ON schedules(channel_id, start_time);
	CREATE INDEX IF NOT EXISTS idx_schedules_channel_window ON schedules(channel_id, start_time, end_time);
	CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
	CREATE INDEX IF NOT EXISTS idx_channels_primary_agent ON channels(primary_agent_id);
	CREATE INDEX IF NOT EXISTS idx_edge_agents_status ON edge_agents(status);
	CREATE INDEX IF NOT EXISTS idx_storage_mounts_active ON storage_mounts(is_active);
	CREATE INDEX IF NOT EXISTS idx_ad_templates_active ON ad_templates(is_active);
	`

	_, err := db.Exec(schema)
	return err
}

// Seed initializes default presets, templates, mounts, user, and channel
func (db *DB) Seed() error {
	now := time.Now().UTC()

	// 1. Seed Resolutions (4 Indian Broadcast Presets)
	presets := []models.ResolutionPreset{
		{
			ID:              "res-in-1080i50",
			Name:            "Indian HD 1080i50 (PAL)",
			Width:           1920,
			Height:          1080,
			FrameRate:       25.0,
			Interlaced:      true,
			AspectRatio:     "16:9",
			VideoCodec:      "libx264",
			AudioCodec:      "aac",
			ExtraFFmpegArgs: "-flags +ildct+ilme -top 1 -b:v 8M -maxrate 10M -bufsize 15M",
			IsPreset:        true,
			CreatedAt:       now,
		},
		{
			ID:              "res-in-720p50",
			Name:            "Indian HD 720p50",
			Width:           1280,
			Height:          720,
			FrameRate:       50.0,
			Interlaced:      false,
			AspectRatio:     "16:9",
			VideoCodec:      "libx264",
			AudioCodec:      "aac",
			ExtraFFmpegArgs: "-b:v 5M -maxrate 7M -bufsize 10M",
			IsPreset:        true,
			CreatedAt:       now,
		},
		{
			ID:              "res-in-576i50-169",
			Name:            "Indian SD 576i50 Anamorphic (16:9)",
			Width:           720,
			Height:          576,
			FrameRate:       25.0,
			Interlaced:      true,
			AspectRatio:     "16:9",
			VideoCodec:      "libx264",
			AudioCodec:      "mp2",
			ExtraFFmpegArgs: "-aspect 16:9 -flags +ildct+ilme -top 1 -b:v 3M -maxrate 4M -bufsize 6M",
			IsPreset:        true,
			CreatedAt:       now,
		},
		{
			ID:              "res-in-4k50",
			Name:            "Indian UHD 4K50 (2160p)",
			Width:           3840,
			Height:          2160,
			FrameRate:       50.0,
			Interlaced:      false,
			AspectRatio:     "16:9",
			VideoCodec:      "libx265",
			AudioCodec:      "aac",
			ExtraFFmpegArgs: "-tag:v hvc1 -b:v 20M -maxrate 25M -bufsize 40M",
			IsPreset:        true,
			CreatedAt:       now,
		},
	}

	for _, p := range presets {
		interlacedInt := 0
		if p.Interlaced {
			interlacedInt = 1
		}
		presetInt := 0
		if p.IsPreset {
			presetInt = 1
		}
		_, err := db.Exec(`
			INSERT OR IGNORE INTO resolutions 
			(id, name, width, height, frame_rate, interlaced, aspect_ratio, video_codec, audio_codec, extra_ffmpeg_args, is_preset, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			p.ID, p.Name, p.Width, p.Height, p.FrameRate, interlacedInt, p.AspectRatio, p.VideoCodec, p.AudioCodec, p.ExtraFFmpegArgs, presetInt, p.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to seed resolution %s: %w", p.ID, err)
		}
	}

	// 2. Seed Ad Templates (3 Broadcast Templates)
	adTemplates := []struct {
		id       string
		name     string
		tplType  string
		overlays []models.OverlayElement
		breaks   []models.CommercialBreak
	}{
		{
			id:      "tmpl-news-standard",
			name:    "Standard News Lower-Third & Top-Right Bug",
			tplType: "composite",
			overlays: []models.OverlayElement{
				{
					ID:                 "elem-tr-bug",
					Type:               "logo_bug",
					X:                  1720,
					Y:                  50,
					Width:              150,
					Height:             80,
					Opacity:            0.9,
					ImagePath:          "/logos/mcr_dd1.png",
					EntranceAnimation:  "fade_in",
					StartOffsetSeconds: 0,
					DurationSeconds:    0, // persistent
				},
				{
					ID:                 "elem-news-ticker",
					Type:               "ticker",
					X:                  0,
					Y:                  1000,
					Width:              1920,
					Height:             60,
					Opacity:            1.0,
					Text:               "MCRFLOW BROADCAST AUTOMATION - 24/7 LINEAR PLAYOUT SYSTEM ACTIVE",
					EntranceAnimation:  "slide_in_left",
					StartOffsetSeconds: 5,
					DurationSeconds:    0,
				},
			},
			breaks: []models.CommercialBreak{
				{
					BreakType:       "pre_roll",
					OffsetSeconds:   0,
					DurationSeconds: 15,
					Clips:           []string{"/media/ads/sponsor_promo_1080p.mp4"},
					Scte35Cue:       true,
				},
				{
					BreakType:       "mid_roll",
					OffsetSeconds:   900,
					DurationSeconds: 30,
					Clips:           []string{"/media/ads/ad_slot_01.mp4", "/media/ads/ad_slot_02.mp4"},
					Scte35Cue:       true,
				},
			},
		},
		{
			id:      "tmpl-cinema-clean",
			name:    "Cinema Prime - Minimalist Semi-Transparent Bug",
			tplType: "overlay",
			overlays: []models.OverlayElement{
				{
					ID:                 "elem-tl-cinema-bug",
					Type:               "logo_bug",
					X:                  60,
					Y:                  50,
					Width:              140,
					Height:             60,
					Opacity:            0.65,
					ImagePath:          "/logos/cinema_watermark.png",
					EntranceAnimation:  "fade_in",
					StartOffsetSeconds: 10,
					DurationSeconds:    0,
				},
			},
			breaks: []models.CommercialBreak{
				{
					BreakType:       "pre_roll",
					OffsetSeconds:   0,
					DurationSeconds: 10,
					Clips:           []string{"/media/ads/rating_disclaimer.mp4"},
					Scte35Cue:       false,
				},
				{
					BreakType:       "post_roll",
					OffsetSeconds:   0,
					DurationSeconds: 20,
					Clips:           []string{"/media/ads/channel_credits.mp4"},
					Scte35Cue:       true,
				},
			},
		},
		{
			id:      "tmpl-sports-ticker",
			name:    "Live Sports L-Band & Score Ticker",
			tplType: "composite",
			overlays: []models.OverlayElement{
				{
					ID:                 "elem-sports-score",
					Type:               "lower_third",
					X:                  80,
					Y:                  940,
					Width:              800,
					Height:             90,
					Opacity:            0.95,
					Text:               "MATCH DAY LIVE | DD SPORTS HD",
					EntranceAnimation:  "slide_in_left",
					StartOffsetSeconds: 0,
					DurationSeconds:    0,
				},
			},
			breaks: []models.CommercialBreak{
				{
					BreakType:       "mid_roll",
					OffsetSeconds:   1800,
					DurationSeconds: 45,
					Clips:           []string{"/media/ads/drink_break_commercial.mp4"},
					Scte35Cue:       true,
				},
			},
		},
	}

	for _, t := range adTemplates {
		ovBytes, _ := json.Marshal(t.overlays)
		brkBytes, _ := json.Marshal(t.breaks)
		_, err := db.Exec(`
			INSERT OR IGNORE INTO ad_templates 
			(id, name, template_type, overlay_elements_json, commercial_breaks_json, is_active, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, 1, ?, ?)`,
			t.id, t.name, t.tplType, string(ovBytes), string(brkBytes), now, now,
		)
		if err != nil {
			return fmt.Errorf("failed to seed ad template %s: %w", t.id, err)
		}
	}

	// 3. Seed Storage Mounts (2 Mounts)
	mounts := []models.StorageMount{
		{
			ID:        "mount-local-01",
			Name:      "Local Broadcast Storage",
			MountType: "local",
			MountPath: "C:/media/storage",
			SmbURL:    "",
			IsActive:  true,
			CreatedAt: now,
		},
		{
			ID:        "mount-nas-01",
			Name:      "Primary Media NAS",
			MountType: "smb",
			MountPath: "/mnt/nas/broadcast",
			SmbURL:    "smb://nas.internal/broadcast",
			IsActive:  true,
			CreatedAt: now,
		},
	}

	for _, m := range mounts {
		act := 0
		if m.IsActive {
			act = 1
		}
		_, err := db.Exec(`
			INSERT OR IGNORE INTO storage_mounts 
			(id, name, mount_type, mount_path, smb_url, is_active, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			m.ID, m.Name, m.MountType, m.MountPath, m.SmbURL, act, m.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to seed storage mount %s: %w", m.ID, err)
		}
	}

	// 4. Seed Default Admin User
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM users WHERE username = 'admin'`).Scan(&count)
	if err == nil && count == 0 {
		hashedPass, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("failed to hash default admin password: %w", err)
		}
		_, err = db.Exec(`
			INSERT INTO users (id, username, password_hash, full_name, email, role, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			"usr-admin-01", "admin", string(hashedPass), "Master Control Administrator", "admin@mcrflow.tv", models.RoleAdmin, now, now,
		)
		if err != nil {
			return fmt.Errorf("failed to seed admin user: %w", err)
		}
	}

	// 4b. Seed Demo Operator and Scheduler Users
	var opCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM users WHERE username = 'operator'`).Scan(&opCount)
	if err == nil && opCount == 0 {
		hashedPass, _ := bcrypt.GenerateFromPassword([]byte("operator123"), bcrypt.DefaultCost)
		_, _ = db.Exec(`
			INSERT INTO users (id, username, password_hash, full_name, email, role, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			"usr-operator-01", "operator", string(hashedPass), "Rajesh Kumar (MCR Desk)", "rajesh@mcrflow.tv", models.RoleOperator, now, now,
		)
	}

	var schedCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM users WHERE username = 'scheduler'`).Scan(&schedCount)
	if err == nil && schedCount == 0 {
		hashedPass, _ := bcrypt.GenerateFromPassword([]byte("scheduler123"), bcrypt.DefaultCost)
		_, _ = db.Exec(`
			INSERT INTO users (id, username, password_hash, full_name, email, role, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			"usr-scheduler-01", "scheduler", string(hashedPass), "Rahul Sharma (Scheduler)", "rahul@mcrflow.tv", models.RoleContentScheduler, now, now,
		)
	}

	// 5. Seed Edge Agents
	agents := []models.EdgeAgent{
		{
			ID:             "agent-local-01",
			Hostname:       "delhi-dc1-primary",
			IPAddress:      "100.64.1.15",
			Port:           3082,
			PairingToken:   "agt_sec_8f43a9b2c011e749a1d2e8b409c2513f",
			Status:         "online",
			CPUPercent:     28.4,
			MemoryPercent:  42.1,
			ActiveChannels: []string{"ch-01", "ch-03"},
			LastHeartbeat:  &now,
			CreatedAt:      now,
		},
		{
			ID:             "agent-standby-01",
			Hostname:       "mumbai-dc2-hotstandby",
			IPAddress:      "100.64.2.99",
			Port:           3082,
			PairingToken:   "agt_sec_3b11ef9901ad847291bb4c0091aa38df",
			Status:         "standby",
			CPUPercent:     16.2,
			MemoryPercent:  31.0,
			ActiveChannels: []string{"ch-01", "ch-02"},
			LastHeartbeat:  &now,
			CreatedAt:      now,
		},
	}
	for _, a := range agents {
		chJSON, _ := json.Marshal(a.ActiveChannels)
		_, _ = db.Exec(`
			INSERT OR IGNORE INTO edge_agents
			(id, hostname, ip_address, tailscale_ip, port, pairing_token, status, last_heartbeat, cpu_percent, memory_percent, active_channels_json, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			a.ID, a.Hostname, a.IPAddress, a.IPAddress, a.Port, a.PairingToken, a.Status, a.LastHeartbeat, a.CPUPercent, a.MemoryPercent, string(chJSON), a.CreatedAt,
		)
	}

	// 6. Seed Bot
	_, _ = db.Exec(`
		INSERT OR IGNORE INTO bots
		(id, name, platform, api_key, webhook_url, is_active, created_at)
		VALUES (?, ?, ?, ?, ?, 1, ?)`,
		"bot-tg-01", "Telegram Playout Bot", "telegram", "7123456789:AAHq0_k9x8Z1w7e6r5t4y3u2i1o0p_xxxx", "https://api.telegram.org/bot7123456789/webhook", now,
	)

	// 7. Seed Default Channel ch-01
	destinations := []models.StreamDestination{
		{
			Type:      "udp",
			Enabled:   true,
			URL:       "udp://239.255.0.1",
			Port:      5000,
			Mode:      "",
			StreamKey: "",
			LatencyMs: 0,
		},
		{
			Type:      "srt",
			Enabled:   true,
			URL:       "srt://127.0.0.1",
			Port:      9000,
			Mode:      "caller",
			StreamKey: "live/ddnational",
			LatencyMs: 120,
		},
		{
			Type:      "hls",
			Enabled:   true,
			URL:       "/hls/ch-01/master.m3u8",
			Port:      3081,
			Mode:      "",
			StreamKey: "",
			LatencyMs: 0,
		},
	}
	destBytes, _ := json.Marshal(destinations)

	_, err = db.Exec(`
		INSERT OR IGNORE INTO channels 
		(id, name, call_sign, resolution_id, logo_path, ad_template_id, primary_agent_id, fallback_agent_id, hls_web_token, epg_web_token, destinations_json, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, ?, ?)`,
		"ch-01", "DD National HD", "MCR-DD1", "res-in-1080i50", "/logos/mcr_dd1.png", "tmpl-news-standard",
		"agent-local-01", "agent-standby-01", "live_sec_dd1_tok_2026", "epg_sec_dd1_xml_2026", string(destBytes), now, now,
	)
	if err != nil {
		return fmt.Errorf("failed to seed default channel: %w", err)
	}

	return nil
}
