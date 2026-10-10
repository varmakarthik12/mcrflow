package database

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/varmakarthik12/mcrflow/internal/models"
)

var (
	ErrNotFound              = errors.New("resource not found")
	ErrConflict              = errors.New("resource conflict")
	ErrCannotDeleteLastAdmin = errors.New("cannot delete the last remaining administrator")
)

// Repository provides unified access to all database operations
type Repository struct {
	db *DB
}

// NewRepository creates a new database repository
func NewRepository(db *DB) *Repository {
	return &Repository{db: db}
}

// ==========================================
// USER REPOSITORY
// ==========================================

func (r *Repository) CreateUser(u *models.User, plainPassword string) error {
	if u.ID == "" {
		u.ID = "usr-" + uuid.New().String()[:8]
	}
	if u.FullName == "" && u.DisplayName != "" {
		u.FullName = u.DisplayName
	}
	if u.DisplayName == "" {
		u.DisplayName = u.FullName
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	u.PasswordHash = string(hash)
	now := time.Now().UTC()
	u.CreatedAt = now
	u.UpdatedAt = now

	_, err = r.db.Exec(`
		INSERT INTO users (id, username, password_hash, full_name, email, role, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		u.ID, u.Username, u.PasswordHash, u.FullName, u.Email, u.Role, u.CreatedAt, u.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert user: %w", err)
	}
	return nil
}

func (r *Repository) GetUserByID(id string) (*models.User, error) {
	row := r.db.QueryRow(`
		SELECT id, username, password_hash, full_name, email, role, created_at, updated_at
		FROM users WHERE id = ?`, id)

	u := &models.User{}
	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.FullName, &u.Email, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to query user by id: %w", err)
	}
	u.DisplayName = u.FullName
	return u, nil
}

func (r *Repository) GetUserByUsername(username string) (*models.User, error) {
	row := r.db.QueryRow(`
		SELECT id, username, password_hash, full_name, email, role, created_at, updated_at
		FROM users WHERE username = ?`, username)

	u := &models.User{}
	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.FullName, &u.Email, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to query user by username: %w", err)
	}
	u.DisplayName = u.FullName
	return u, nil
}

func (r *Repository) ListUsers() ([]models.User, error) {
	rows, err := r.db.Query(`
		SELECT id, username, password_hash, full_name, email, role, created_at, updated_at
		FROM users ORDER BY created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	users := make([]models.User, 0)
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.FullName, &u.Email, &u.Role, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		u.DisplayName = u.FullName
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error in ListUsers: %w", err)
	}
	return users, nil
}

func (r *Repository) UpdateUser(u *models.User, newPassword *string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Guard against demoting the last administrator
	var currentRole string
	err = tx.QueryRow(`SELECT role FROM users WHERE id = ?`, u.ID).Scan(&currentRole)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("failed to query existing user: %w", err)
	}

	if currentRole == models.RoleAdmin && u.Role != models.RoleAdmin {
		var adminCount int
		err := tx.QueryRow(`SELECT COUNT(*) FROM users WHERE role = ?`, models.RoleAdmin).Scan(&adminCount)
		if err != nil {
			return fmt.Errorf("failed to check admin count: %w", err)
		}
		if adminCount <= 1 {
			return ErrCannotDeleteLastAdmin
		}
	}

	if u.FullName == "" && u.DisplayName != "" {
		u.FullName = u.DisplayName
	}
	if u.DisplayName == "" {
		u.DisplayName = u.FullName
	}
	u.UpdatedAt = time.Now().UTC()

	if newPassword != nil && *newPassword != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(*newPassword), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("failed to hash new password: %w", err)
		}
		u.PasswordHash = string(hash)
		res, err := tx.Exec(`
			UPDATE users 
			SET username = ?, password_hash = ?, full_name = ?, email = ?, role = ?, updated_at = ?
			WHERE id = ?`,
			u.Username, u.PasswordHash, u.FullName, u.Email, u.Role, u.UpdatedAt, u.ID,
		)
		if err != nil {
			return fmt.Errorf("failed to update user with password: %w", err)
		}
		rowsAffected, _ := res.RowsAffected()
		if rowsAffected == 0 {
			return ErrNotFound
		}
		return tx.Commit()
	}

	res, err := tx.Exec(`
		UPDATE users 
		SET username = ?, full_name = ?, email = ?, role = ?, updated_at = ?
		WHERE id = ?`,
		u.Username, u.FullName, u.Email, u.Role, u.UpdatedAt, u.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return tx.Commit()
}

func (r *Repository) DeleteUser(id string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Guard against deleting last admin
	var userRole string
	err = tx.QueryRow(`SELECT role FROM users WHERE id = ?`, id).Scan(&userRole)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("failed to query user for deletion: %w", err)
	}

	if userRole == models.RoleAdmin {
		var adminCount int
		err := tx.QueryRow(`SELECT COUNT(*) FROM users WHERE role = ?`, models.RoleAdmin).Scan(&adminCount)
		if err != nil {
			return fmt.Errorf("failed to check admin count: %w", err)
		}
		if adminCount <= 1 {
			return ErrCannotDeleteLastAdmin
		}
	}

	res, err := tx.Exec(`DELETE FROM users WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return tx.Commit()
}

// ==========================================
// CHANNEL REPOSITORY
// ==========================================

func (r *Repository) CreateChannel(ch *models.Channel) error {
	if ch.ID == "" {
		ch.ID = "ch-" + uuid.New().String()[:8]
	}
	if ch.LogoPosition == "" {
		ch.LogoPosition = "top-right"
	}
	if ch.LogoOpacity <= 0 {
		ch.LogoOpacity = 0.90
	}
	if ch.LogoFit == "" {
		ch.LogoFit = "contain"
	}
	for i := range ch.Destinations {
		if ch.Destinations[i].Type == "" && ch.Destinations[i].Protocol != "" {
			ch.Destinations[i].Type = strings.ToLower(strings.TrimPrefix(ch.Destinations[i].Protocol, "UDP_"))
		}
		if ch.Destinations[i].URL == "" && ch.Destinations[i].EndpointURL != "" {
			ch.Destinations[i].URL = ch.Destinations[i].EndpointURL
		}
		if ch.Destinations[i].Protocol == "" {
			ch.Destinations[i].Protocol = strings.ToUpper(ch.Destinations[i].Type)
		}
		if ch.Destinations[i].EndpointURL == "" {
			ch.Destinations[i].EndpointURL = ch.Destinations[i].URL
		}
	}
	ch.PackDestinations()
	ch.PackOverlays()
	now := time.Now().UTC()
	ch.CreatedAt = now
	ch.UpdatedAt = now

	isActiveInt := 0
	if ch.IsActive {
		isActiveInt = 1
	}

	_, err := r.db.Exec(`
		INSERT INTO channels 
		(id, name, call_sign, resolution_id, logo_path, logo_position, logo_x, logo_y, logo_width, logo_height, logo_opacity, logo_fit, overlays_json, ad_template_id, primary_agent_id, fallback_agent_id, hls_web_token, epg_web_token, destinations_json, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		ch.ID, ch.Name, ch.CallSign, ch.ResolutionID, ch.LogoPath, ch.LogoPosition,
		ch.LogoX, ch.LogoY, ch.LogoWidth, ch.LogoHeight, ch.LogoOpacity, ch.LogoFit, ch.OverlaysJSON,
		ch.AdTemplateID, ch.PrimaryAgentID, ch.FallbackAgentID, ch.HlsWebToken, ch.EpgWebToken,
		ch.DestinationsJSON, isActiveInt, ch.CreatedAt, ch.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert channel: %w", err)
	}
	return nil
}

func (r *Repository) GetChannelByID(id string) (*models.Channel, error) {
	row := r.db.QueryRow(`
		SELECT id, name, call_sign, resolution_id, logo_path, logo_position, logo_x, logo_y, logo_width, logo_height, logo_opacity, logo_fit, overlays_json, ad_template_id, primary_agent_id, fallback_agent_id, hls_web_token, epg_web_token, destinations_json, is_active, created_at, updated_at
		FROM channels WHERE id = ?`, id)

	ch := &models.Channel{}
	var isActiveInt int
	err := row.Scan(
		&ch.ID, &ch.Name, &ch.CallSign, &ch.ResolutionID, &ch.LogoPath, &ch.LogoPosition,
		&ch.LogoX, &ch.LogoY, &ch.LogoWidth, &ch.LogoHeight, &ch.LogoOpacity, &ch.LogoFit, &ch.OverlaysJSON,
		&ch.AdTemplateID, &ch.PrimaryAgentID, &ch.FallbackAgentID,
		&ch.HlsWebToken, &ch.EpgWebToken, &ch.DestinationsJSON,
		&isActiveInt, &ch.CreatedAt, &ch.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to query channel: %w", err)
	}
	ch.IsActive = (isActiveInt == 1)
	ch.ParseDestinations()
	ch.ParseOverlays()
	for i := range ch.Destinations {
		if ch.Destinations[i].Protocol == "" {
			ch.Destinations[i].Protocol = strings.ToUpper(ch.Destinations[i].Type)
		}
		if ch.Destinations[i].EndpointURL == "" {
			ch.Destinations[i].EndpointURL = ch.Destinations[i].URL
		}
		if ch.Destinations[i].Type == "" {
			ch.Destinations[i].Type = strings.ToLower(strings.TrimPrefix(ch.Destinations[i].Protocol, "UDP_"))
		}
		if ch.Destinations[i].URL == "" {
			ch.Destinations[i].URL = ch.Destinations[i].EndpointURL
		}
	}
	return ch, nil
}

func (r *Repository) ListChannels() ([]models.Channel, error) {
	rows, err := r.db.Query(`
		SELECT id, name, call_sign, resolution_id, logo_path, logo_position, logo_x, logo_y, logo_width, logo_height, logo_opacity, logo_fit, overlays_json, ad_template_id, primary_agent_id, fallback_agent_id, hls_web_token, epg_web_token, destinations_json, is_active, created_at, updated_at
		FROM channels ORDER BY created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("failed to query channels: %w", err)
	}
	defer rows.Close()

	channels := make([]models.Channel, 0)
	for rows.Next() {
		var ch models.Channel
		var isActiveInt int
		err := rows.Scan(
			&ch.ID, &ch.Name, &ch.CallSign, &ch.ResolutionID, &ch.LogoPath, &ch.LogoPosition,
			&ch.LogoX, &ch.LogoY, &ch.LogoWidth, &ch.LogoHeight, &ch.LogoOpacity, &ch.LogoFit, &ch.OverlaysJSON,
			&ch.AdTemplateID, &ch.PrimaryAgentID, &ch.FallbackAgentID,
			&ch.HlsWebToken, &ch.EpgWebToken, &ch.DestinationsJSON,
			&isActiveInt, &ch.CreatedAt, &ch.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan channel: %w", err)
		}
		ch.IsActive = (isActiveInt == 1)
		ch.ParseDestinations()
		ch.ParseOverlays()
		for i := range ch.Destinations {
			if ch.Destinations[i].Protocol == "" {
				ch.Destinations[i].Protocol = strings.ToUpper(ch.Destinations[i].Type)
			}
			if ch.Destinations[i].EndpointURL == "" {
				ch.Destinations[i].EndpointURL = ch.Destinations[i].URL
			}
			if ch.Destinations[i].Type == "" {
				ch.Destinations[i].Type = strings.ToLower(strings.TrimPrefix(ch.Destinations[i].Protocol, "UDP_"))
			}
			if ch.Destinations[i].URL == "" {
				ch.Destinations[i].URL = ch.Destinations[i].EndpointURL
			}
		}
		channels = append(channels, ch)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error in ListChannels: %w", err)
	}
	return channels, nil
}

func (r *Repository) UpdateChannel(ch *models.Channel) error {
	if ch.LogoPosition == "" {
		ch.LogoPosition = "top-right"
	}
	if ch.LogoOpacity <= 0 {
		ch.LogoOpacity = 0.90
	}
	if ch.LogoFit == "" {
		ch.LogoFit = "contain"
	}
	for i := range ch.Destinations {
		if ch.Destinations[i].Type == "" && ch.Destinations[i].Protocol != "" {
			ch.Destinations[i].Type = strings.ToLower(strings.TrimPrefix(ch.Destinations[i].Protocol, "UDP_"))
		}
		if ch.Destinations[i].URL == "" && ch.Destinations[i].EndpointURL != "" {
			ch.Destinations[i].URL = ch.Destinations[i].EndpointURL
		}
		if ch.Destinations[i].Protocol == "" {
			ch.Destinations[i].Protocol = strings.ToUpper(ch.Destinations[i].Type)
		}
		if ch.Destinations[i].EndpointURL == "" {
			ch.Destinations[i].EndpointURL = ch.Destinations[i].URL
		}
	}
	ch.PackDestinations()
	ch.PackOverlays()
	ch.UpdatedAt = time.Now().UTC()
	isActiveInt := 0
	if ch.IsActive {
		isActiveInt = 1
	}

	res, err := r.db.Exec(`
		UPDATE channels 
		SET name = ?, call_sign = ?, resolution_id = ?, logo_path = ?, logo_position = ?,
		    logo_x = ?, logo_y = ?, logo_width = ?, logo_height = ?, logo_opacity = ?, logo_fit = ?,
		    overlays_json = ?, ad_template_id = ?,
		    primary_agent_id = ?, fallback_agent_id = ?, hls_web_token = ?, epg_web_token = ?,
		    destinations_json = ?, is_active = ?, updated_at = ?
		WHERE id = ?`,
		ch.Name, ch.CallSign, ch.ResolutionID, ch.LogoPath, ch.LogoPosition,
		ch.LogoX, ch.LogoY, ch.LogoWidth, ch.LogoHeight, ch.LogoOpacity, ch.LogoFit,
		ch.OverlaysJSON, ch.AdTemplateID,
		ch.PrimaryAgentID, ch.FallbackAgentID, ch.HlsWebToken, ch.EpgWebToken,
		ch.DestinationsJSON, isActiveInt, ch.UpdatedAt, ch.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update channel: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) DeleteChannel(id string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Cascade delete associated schedules
	if _, err := tx.Exec(`DELETE FROM schedules WHERE channel_id = ?`, id); err != nil {
		return fmt.Errorf("failed to cascade delete schedules: %w", err)
	}

	res, err := tx.Exec(`DELETE FROM channels WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete channel: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return tx.Commit()
}

// ==========================================
// SCHEDULE REPOSITORY
// ==========================================

func (r *Repository) CreateScheduleItem(s *models.ScheduleItem) error {
	if s.ID == "" {
		s.ID = "sch-" + uuid.New().String()[:8]
	}
	if s.ProgramTitle == "" && s.Title != "" {
		s.ProgramTitle = s.Title
	}
	if s.Title == "" {
		s.Title = s.ProgramTitle
	}
	if s.MediaPath == "" && s.MediaFilePath != "" {
		s.MediaPath = s.MediaFilePath
	}
	if s.MediaFilePath == "" {
		s.MediaFilePath = s.MediaPath
	}
	if s.DurationSeconds <= 0 {
		s.DurationSeconds = 3600 // default 1 hour
	}
	if s.EndTime.IsZero() {
		s.EndTime = s.StartTime.Add(time.Duration(s.DurationSeconds) * time.Second)
	}
	now := time.Now().UTC()
	s.CreatedAt = now

	_, err := r.db.Exec(`
		INSERT INTO schedules 
		(id, channel_id, program_title, media_path, start_time, duration_seconds, end_time, tmdb_id, tmdb_poster, tmdb_overview, ad_template_id, audio_track_index, subtitle_track_index, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.ChannelID, s.ProgramTitle, s.MediaPath, s.StartTime,
		s.DurationSeconds, s.EndTime, s.TmdbID, s.TmdbPoster, s.TmdbOverview,
		s.AdTemplateID, s.AudioTrackIndex, s.SubtitleTrackIndex, s.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert schedule item: %w", err)
	}
	return nil
}

func (r *Repository) GetScheduleItemByID(id string) (*models.ScheduleItem, error) {
	row := r.db.QueryRow(`
		SELECT id, channel_id, program_title, media_path, start_time, duration_seconds, end_time, tmdb_id, tmdb_poster, tmdb_overview, ad_template_id, audio_track_index, subtitle_track_index, created_at
		FROM schedules WHERE id = ?`, id)

	s := &models.ScheduleItem{}
	err := row.Scan(
		&s.ID, &s.ChannelID, &s.ProgramTitle, &s.MediaPath, &s.StartTime,
		&s.DurationSeconds, &s.EndTime, &s.TmdbID, &s.TmdbPoster, &s.TmdbOverview,
		&s.AdTemplateID, &s.AudioTrackIndex, &s.SubtitleTrackIndex, &s.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to query schedule item: %w", err)
	}
	s.Title = s.ProgramTitle
	s.MediaFilePath = s.MediaPath
	return s, nil
}

func (r *Repository) ListScheduleByChannel(channelID string) ([]models.ScheduleItem, error) {
	rows, err := r.db.Query(`
		SELECT id, channel_id, program_title, media_path, start_time, duration_seconds, end_time, tmdb_id, tmdb_poster, tmdb_overview, ad_template_id, audio_track_index, subtitle_track_index, created_at
		FROM schedules WHERE channel_id = ? ORDER BY start_time ASC`, channelID)
	if err != nil {
		return nil, fmt.Errorf("failed to query schedule by channel: %w", err)
	}
	defer rows.Close()

	items := make([]models.ScheduleItem, 0)
	for rows.Next() {
		var s models.ScheduleItem
		err := rows.Scan(
			&s.ID, &s.ChannelID, &s.ProgramTitle, &s.MediaPath, &s.StartTime,
			&s.DurationSeconds, &s.EndTime, &s.TmdbID, &s.TmdbPoster, &s.TmdbOverview,
			&s.AdTemplateID, &s.AudioTrackIndex, &s.SubtitleTrackIndex, &s.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan schedule item: %w", err)
		}
		s.Title = s.ProgramTitle
		s.MediaFilePath = s.MediaPath
		items = append(items, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error in ListScheduleByChannel: %w", err)
	}
	return items, nil
}

func (r *Repository) ListScheduleBetween(channelID string, start, end time.Time) ([]models.ScheduleItem, error) {
	rows, err := r.db.Query(`
		SELECT id, channel_id, program_title, media_path, start_time, duration_seconds, end_time, tmdb_id, tmdb_poster, tmdb_overview, ad_template_id, audio_track_index, subtitle_track_index, created_at
		FROM schedules 
		WHERE channel_id = ? AND start_time < ? AND end_time > ? 
		ORDER BY start_time ASC`, channelID, end, start)
	if err != nil {
		return nil, fmt.Errorf("failed to query schedule between: %w", err)
	}
	defer rows.Close()

	items := make([]models.ScheduleItem, 0)
	for rows.Next() {
		var s models.ScheduleItem
		err := rows.Scan(
			&s.ID, &s.ChannelID, &s.ProgramTitle, &s.MediaPath, &s.StartTime,
			&s.DurationSeconds, &s.EndTime, &s.TmdbID, &s.TmdbPoster, &s.TmdbOverview,
			&s.AdTemplateID, &s.AudioTrackIndex, &s.SubtitleTrackIndex, &s.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan schedule item: %w", err)
		}
		s.Title = s.ProgramTitle
		s.MediaFilePath = s.MediaPath
		items = append(items, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error in ListScheduleBetween: %w", err)
	}
	return items, nil
}

func (r *Repository) UpdateScheduleItem(s *models.ScheduleItem) error {
	if s.ProgramTitle == "" && s.Title != "" {
		s.ProgramTitle = s.Title
	}
	if s.Title == "" {
		s.Title = s.ProgramTitle
	}
	if s.MediaPath == "" && s.MediaFilePath != "" {
		s.MediaPath = s.MediaFilePath
	}
	if s.MediaFilePath == "" {
		s.MediaFilePath = s.MediaPath
	}
	if s.DurationSeconds <= 0 {
		s.DurationSeconds = 3600
	}
	if s.EndTime.IsZero() {
		s.EndTime = s.StartTime.Add(time.Duration(s.DurationSeconds) * time.Second)
	}

	res, err := r.db.Exec(`
		UPDATE schedules 
		SET program_title = ?, media_path = ?, start_time = ?, duration_seconds = ?, end_time = ?,
		    tmdb_id = ?, tmdb_poster = ?, tmdb_overview = ?, ad_template_id = ?, audio_track_index = ?,
		    subtitle_track_index = ?
		WHERE id = ?`,
		s.ProgramTitle, s.MediaPath, s.StartTime, s.DurationSeconds, s.EndTime,
		s.TmdbID, s.TmdbPoster, s.TmdbOverview, s.AdTemplateID, s.AudioTrackIndex,
		s.SubtitleTrackIndex, s.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update schedule item: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) DeleteScheduleItem(id string) error {
	res, err := r.db.Exec(`DELETE FROM schedules WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete schedule item: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// ==========================================
// RESOLUTION REPOSITORY
// ==========================================

func (r *Repository) CreateResolution(res *models.ResolutionPreset) error {
	if res.ID == "" {
		res.ID = "res-" + uuid.New().String()[:8]
	}
	if res.FrameRate == 0 && res.FPS > 0 {
		res.FrameRate = res.FPS
	}
	if !res.Interlaced && res.ScanningMode == "interlaced" {
		res.Interlaced = true
	}
	if res.ExtraFFmpegArgs == "" && res.ExtraFFmpegVideoArgs != "" {
		res.ExtraFFmpegArgs = res.ExtraFFmpegVideoArgs
	}
	now := time.Now().UTC()
	res.CreatedAt = now
	interlacedInt := 0
	if res.Interlaced {
		interlacedInt = 1
	}
	presetInt := 0
	if res.IsPreset {
		presetInt = 1
	}

	_, err := r.db.Exec(`
		INSERT INTO resolutions 
		(id, name, width, height, frame_rate, interlaced, aspect_ratio, video_codec, audio_codec, extra_ffmpeg_args, is_preset, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		res.ID, res.Name, res.Width, res.Height, res.FrameRate, interlacedInt,
		res.AspectRatio, res.VideoCodec, res.AudioCodec, res.ExtraFFmpegArgs, presetInt, res.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert resolution: %w", err)
	}
	return nil
}

func (r *Repository) GetResolutionByID(id string) (*models.ResolutionPreset, error) {
	row := r.db.QueryRow(`
		SELECT id, name, width, height, frame_rate, interlaced, aspect_ratio, video_codec, audio_codec, extra_ffmpeg_args, is_preset, created_at
		FROM resolutions WHERE id = ?`, id)

	res := &models.ResolutionPreset{}
	var interlacedInt, presetInt int
	err := row.Scan(
		&res.ID, &res.Name, &res.Width, &res.Height, &res.FrameRate,
		&interlacedInt, &res.AspectRatio, &res.VideoCodec, &res.AudioCodec,
		&res.ExtraFFmpegArgs, &presetInt, &res.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to query resolution: %w", err)
	}
	res.Interlaced = (interlacedInt == 1)
	res.IsPreset = (presetInt == 1)
	res.FPS = res.FrameRate
	res.ExtraFFmpegVideoArgs = res.ExtraFFmpegArgs
	if res.Interlaced {
		res.ScanningMode = "interlaced"
	} else {
		res.ScanningMode = "progressive"
	}
	res.VideoBitrateKbps = 6500
	res.AudioBitrateKbps = 192
	return res, nil
}

func (r *Repository) ListResolutions() ([]models.ResolutionPreset, error) {
	rows, err := r.db.Query(`
		SELECT id, name, width, height, frame_rate, interlaced, aspect_ratio, video_codec, audio_codec, extra_ffmpeg_args, is_preset, created_at
		FROM resolutions ORDER BY width DESC, frame_rate DESC`)
	if err != nil {
		return nil, fmt.Errorf("failed to query resolutions: %w", err)
	}
	defer rows.Close()

	list := make([]models.ResolutionPreset, 0)
	for rows.Next() {
		var res models.ResolutionPreset
		var interlacedInt, presetInt int
		err := rows.Scan(
			&res.ID, &res.Name, &res.Width, &res.Height, &res.FrameRate,
			&interlacedInt, &res.AspectRatio, &res.VideoCodec, &res.AudioCodec,
			&res.ExtraFFmpegArgs, &presetInt, &res.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan resolution: %w", err)
		}
		res.Interlaced = (interlacedInt == 1)
		res.IsPreset = (presetInt == 1)
		res.FPS = res.FrameRate
		res.ExtraFFmpegVideoArgs = res.ExtraFFmpegArgs
		if res.Interlaced {
			res.ScanningMode = "interlaced"
		} else {
			res.ScanningMode = "progressive"
		}
		res.VideoBitrateKbps = 6500
		res.AudioBitrateKbps = 192
		list = append(list, res)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error in ListResolutions: %w", err)
	}
	return list, nil
}

func (r *Repository) UpdateResolution(res *models.ResolutionPreset) error {
	if res.FrameRate == 0 && res.FPS > 0 {
		res.FrameRate = res.FPS
	}
	if !res.Interlaced && res.ScanningMode == "interlaced" {
		res.Interlaced = true
	}
	if res.ExtraFFmpegArgs == "" && res.ExtraFFmpegVideoArgs != "" {
		res.ExtraFFmpegArgs = res.ExtraFFmpegVideoArgs
	}
	interlacedInt := 0
	if res.Interlaced {
		interlacedInt = 1
	}
	presetInt := 0
	if res.IsPreset {
		presetInt = 1
	}

	result, err := r.db.Exec(`
		UPDATE resolutions 
		SET name = ?, width = ?, height = ?, frame_rate = ?, interlaced = ?,
		    aspect_ratio = ?, video_codec = ?, audio_codec = ?, extra_ffmpeg_args = ?, is_preset = ?
		WHERE id = ?`,
		res.Name, res.Width, res.Height, res.FrameRate, interlacedInt,
		res.AspectRatio, res.VideoCodec, res.AudioCodec, res.ExtraFFmpegArgs, presetInt, res.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update resolution: %w", err)
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) DeleteResolution(id string) error {
	res, err := r.db.Exec(`DELETE FROM resolutions WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete resolution: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// ==========================================
// AD TEMPLATE REPOSITORY
// ==========================================

func (r *Repository) CreateAdTemplate(tmpl *models.AdTemplate) error {
	if tmpl.ID == "" {
		tmpl.ID = "tmpl-" + uuid.New().String()[:8]
	}
	tmpl.PackJSON()
	now := time.Now().UTC()
	tmpl.CreatedAt = now
	tmpl.UpdatedAt = now

	isActiveInt := 0
	if tmpl.IsActive {
		isActiveInt = 1
	}

	_, err := r.db.Exec(`
		INSERT INTO ad_templates 
		(id, name, template_type, overlay_elements_json, commercial_breaks_json, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		tmpl.ID, tmpl.Name, tmpl.TemplateType, tmpl.OverlayElementsJSON, tmpl.CommercialBreaksJSON, isActiveInt, tmpl.CreatedAt, tmpl.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert ad template: %w", err)
	}
	return nil
}

func (r *Repository) GetAdTemplateByID(id string) (*models.AdTemplate, error) {
	row := r.db.QueryRow(`
		SELECT id, name, template_type, overlay_elements_json, commercial_breaks_json, is_active, created_at, updated_at
		FROM ad_templates WHERE id = ?`, id)

	tmpl := &models.AdTemplate{}
	var isActiveInt int
	err := row.Scan(
		&tmpl.ID, &tmpl.Name, &tmpl.TemplateType, &tmpl.OverlayElementsJSON,
		&tmpl.CommercialBreaksJSON, &isActiveInt, &tmpl.CreatedAt, &tmpl.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to query ad template: %w", err)
	}
	tmpl.IsActive = (isActiveInt == 1)
	tmpl.ParseJSON()
	return tmpl, nil
}

func (r *Repository) ListAdTemplates() ([]models.AdTemplate, error) {
	rows, err := r.db.Query(`
		SELECT id, name, template_type, overlay_elements_json, commercial_breaks_json, is_active, created_at, updated_at
		FROM ad_templates ORDER BY created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("failed to query ad templates: %w", err)
	}
	defer rows.Close()

	list := make([]models.AdTemplate, 0)
	for rows.Next() {
		var tmpl models.AdTemplate
		var isActiveInt int
		err := rows.Scan(
			&tmpl.ID, &tmpl.Name, &tmpl.TemplateType, &tmpl.OverlayElementsJSON,
			&tmpl.CommercialBreaksJSON, &isActiveInt, &tmpl.CreatedAt, &tmpl.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan ad template: %w", err)
		}
		tmpl.IsActive = (isActiveInt == 1)
		tmpl.ParseJSON()
		list = append(list, tmpl)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error in ListAdTemplates: %w", err)
	}
	return list, nil
}

func (r *Repository) UpdateAdTemplate(tmpl *models.AdTemplate) error {
	tmpl.PackJSON()
	tmpl.UpdatedAt = time.Now().UTC()
	isActiveInt := 0
	if tmpl.IsActive {
		isActiveInt = 1
	}

	res, err := r.db.Exec(`
		UPDATE ad_templates 
		SET name = ?, template_type = ?, overlay_elements_json = ?, commercial_breaks_json = ?, is_active = ?, updated_at = ?
		WHERE id = ?`,
		tmpl.Name, tmpl.TemplateType, tmpl.OverlayElementsJSON, tmpl.CommercialBreaksJSON, isActiveInt, tmpl.UpdatedAt, tmpl.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update ad template: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) DeleteAdTemplate(id string) error {
	res, err := r.db.Exec(`DELETE FROM ad_templates WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete ad template: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// ==========================================
// EDGE AGENT REPOSITORY
// ==========================================

func (r *Repository) CreateEdgeAgent(a *models.EdgeAgent) error {
	if a.ID == "" {
		a.ID = "agt-" + uuid.New().String()[:8]
	}
	if a.PairingToken == "" && a.Token != "" {
		a.PairingToken = a.Token
	}
	a.PackChannels()
	now := time.Now().UTC()
	a.CreatedAt = now

	_, err := r.db.Exec(`
		INSERT INTO edge_agents 
		(id, hostname, ip_address, tailscale_ip, port, pairing_token, status, last_heartbeat, cpu_percent, memory_percent, active_channels_json, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.Hostname, a.IPAddress, a.TailscaleIP, a.Port, a.PairingToken,
		a.Status, a.LastHeartbeat, a.CPUPercent, a.MemoryPercent, a.ActiveChannelsJSON, a.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert edge agent: %w", err)
	}
	return nil
}

func (r *Repository) GetEdgeAgentByID(id string) (*models.EdgeAgent, error) {
	row := r.db.QueryRow(`
		SELECT id, hostname, ip_address, tailscale_ip, port, pairing_token, status, last_heartbeat, cpu_percent, memory_percent, active_channels_json, created_at
		FROM edge_agents WHERE id = ?`, id)

	a := &models.EdgeAgent{}
	err := row.Scan(
		&a.ID, &a.Hostname, &a.IPAddress, &a.TailscaleIP, &a.Port, &a.PairingToken,
		&a.Status, &a.LastHeartbeat, &a.CPUPercent, &a.MemoryPercent, &a.ActiveChannelsJSON, &a.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to query edge agent: %w", err)
	}
	a.ParseChannels()
	a.Token = a.PairingToken
	a.CPUUsagePercent = a.CPUPercent
	a.MemoryUsagePercent = a.MemoryPercent
	return a, nil
}

func (r *Repository) ListEdgeAgents() ([]models.EdgeAgent, error) {
	rows, err := r.db.Query(`
		SELECT id, hostname, ip_address, tailscale_ip, port, pairing_token, status, last_heartbeat, cpu_percent, memory_percent, active_channels_json, created_at
		FROM edge_agents ORDER BY hostname ASC`)
	if err != nil {
		return nil, fmt.Errorf("failed to query edge agents: %w", err)
	}
	defer rows.Close()

	list := make([]models.EdgeAgent, 0)
	for rows.Next() {
		var a models.EdgeAgent
		err := rows.Scan(
			&a.ID, &a.Hostname, &a.IPAddress, &a.TailscaleIP, &a.Port, &a.PairingToken,
			&a.Status, &a.LastHeartbeat, &a.CPUPercent, &a.MemoryPercent, &a.ActiveChannelsJSON, &a.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan edge agent: %w", err)
		}
		a.ParseChannels()
		a.Token = a.PairingToken
		a.CPUUsagePercent = a.CPUPercent
		a.MemoryUsagePercent = a.MemoryPercent
		list = append(list, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error in ListEdgeAgents: %w", err)
	}
	return list, nil
}

func (r *Repository) UpdateEdgeAgent(a *models.EdgeAgent) error {
	if a.PairingToken == "" && a.Token != "" {
		a.PairingToken = a.Token
	}
	a.PackChannels()
	res, err := r.db.Exec(`
		UPDATE edge_agents 
		SET hostname = ?, ip_address = ?, tailscale_ip = ?, port = ?, pairing_token = ?,
		    status = ?, last_heartbeat = ?, cpu_percent = ?, memory_percent = ?, active_channels_json = ?
		WHERE id = ?`,
		a.Hostname, a.IPAddress, a.TailscaleIP, a.Port, a.PairingToken,
		a.Status, a.LastHeartbeat, a.CPUPercent, a.MemoryPercent, a.ActiveChannelsJSON, a.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update edge agent: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) UpdateEdgeAgentHeartbeat(agentID string, cpuPercent, memoryPercent float64, activeChannelsJSON string) error {
	now := time.Now().UTC()
	res, err := r.db.Exec(`
		UPDATE edge_agents 
		SET status = 'online', last_heartbeat = ?, cpu_percent = ?, memory_percent = ?, active_channels_json = ?
		WHERE id = ?`,
		now, cpuPercent, memoryPercent, activeChannelsJSON, agentID,
	)
	if err != nil {
		return fmt.Errorf("failed to update heartbeat: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) DeleteEdgeAgent(id string) error {
	res, err := r.db.Exec(`DELETE FROM edge_agents WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete edge agent: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// ==========================================
// BOT REPOSITORY
// ==========================================

func (r *Repository) CreateBot(b *models.Bot) error {
	if b.ID == "" {
		b.ID = "bot-" + uuid.New().String()[:8]
	}
	now := time.Now().UTC()
	b.CreatedAt = now
	isActiveInt := 0
	if b.IsActive {
		isActiveInt = 1
	}

	_, err := r.db.Exec(`
		INSERT INTO bots 
		(id, name, platform, api_key, webhook_url, is_active, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		b.ID, b.Name, b.Platform, b.APIKey, b.WebhookURL, isActiveInt, b.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert bot: %w", err)
	}
	return nil
}

func (r *Repository) GetBotByID(id string) (*models.Bot, error) {
	row := r.db.QueryRow(`
		SELECT id, name, platform, api_key, webhook_url, is_active, created_at
		FROM bots WHERE id = ?`, id)

	b := &models.Bot{}
	var isActiveInt int
	err := row.Scan(&b.ID, &b.Name, &b.Platform, &b.APIKey, &b.WebhookURL, &isActiveInt, &b.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to query bot: %w", err)
	}
	b.IsActive = (isActiveInt == 1)
	return b, nil
}

func (r *Repository) ListBots() ([]models.Bot, error) {
	rows, err := r.db.Query(`
		SELECT id, name, platform, api_key, webhook_url, is_active, created_at
		FROM bots ORDER BY created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("failed to query bots: %w", err)
	}
	defer rows.Close()

	list := make([]models.Bot, 0)
	for rows.Next() {
		var b models.Bot
		var isActiveInt int
		err := rows.Scan(&b.ID, &b.Name, &b.Platform, &b.APIKey, &b.WebhookURL, &isActiveInt, &b.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan bot: %w", err)
		}
		b.IsActive = (isActiveInt == 1)
		list = append(list, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error in ListBots: %w", err)
	}
	return list, nil
}

func (r *Repository) UpdateBot(b *models.Bot) error {
	isActiveInt := 0
	if b.IsActive {
		isActiveInt = 1
	}

	res, err := r.db.Exec(`
		UPDATE bots 
		SET name = ?, platform = ?, api_key = ?, webhook_url = ?, is_active = ?
		WHERE id = ?`,
		b.Name, b.Platform, b.APIKey, b.WebhookURL, isActiveInt, b.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update bot: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) DeleteBot(id string) error {
	res, err := r.db.Exec(`DELETE FROM bots WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete bot: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
