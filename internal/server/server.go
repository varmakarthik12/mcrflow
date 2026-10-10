package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/varmakarthik12/mcrflow/internal/adtemplate"
	"github.com/varmakarthik12/mcrflow/internal/auth"
	"github.com/varmakarthik12/mcrflow/internal/bot"
	"github.com/varmakarthik12/mcrflow/internal/channel"
	"github.com/varmakarthik12/mcrflow/internal/database"
	"github.com/varmakarthik12/mcrflow/internal/epg"
	"github.com/varmakarthik12/mcrflow/internal/hls"
	"github.com/varmakarthik12/mcrflow/internal/models"
	"github.com/varmakarthik12/mcrflow/internal/playout"
	"github.com/varmakarthik12/mcrflow/internal/resolution"
	"github.com/varmakarthik12/mcrflow/internal/schedule"
	"github.com/varmakarthik12/mcrflow/internal/storage"
	"github.com/varmakarthik12/mcrflow/internal/tmdb"
	"github.com/varmakarthik12/mcrflow/internal/user"
	"github.com/varmakarthik12/mcrflow/web"
)

// Config holds configuration parameters for the Control Plane server
type Config struct {
	Port     int
	DataDir  string
	MediaDir string
	TMDBKey  string
}

// Server is the Master Control Plane HTTP server
type Server struct {
	cfg        Config
	router     *chi.Mux
	db         *database.DB
	repo       *database.Repository
	tokenSvc   *auth.TokenService
	userSvc    *user.Service
	channelSvc *channel.Service
	schedSvc   *schedule.Service
	resSvc     *resolution.Service
	adTmplSvc  *adtemplate.Service
	storageMgr *storage.Manager
	tmdbClient *tmdb.Client
	epgGen     *epg.Generator
	botSvc     *bot.Service
	playoutEng *playout.Engine
	hlsMgr     *hls.Manager
}

// NewServer initializes all services and sets up Chi routes
func NewServer(cfg Config, db *database.DB) (*Server, error) {
	if cfg.Port <= 0 {
		cfg.Port = 3081
	}

	repo := database.NewRepository(db)
	tokenSvc := auth.NewTokenService("")
	userSvc := user.NewService(repo, tokenSvc)
	channelSvc := channel.NewService(repo)
	schedSvc := schedule.NewService(repo)
	resSvc := resolution.NewService(repo)
	adTmplSvc := adtemplate.NewService(repo)
	if cfg.MediaDir == "" {
		cfg.MediaDir = "./media"
	}
	storageMgr := storage.NewManager(cfg.MediaDir)
	tmdbClient := tmdb.NewClient(cfg.TMDBKey)
	epgGen := epg.NewGenerator()
	botSvc := bot.NewService(schedSvc, channelSvc, tmdbClient)
	playoutEng := playout.NewEngine()
	playoutEng.SetScheduleProvider(schedSvc)
	hlsMgr := hls.NewManager(repo)

	s := &Server{
		cfg:        cfg,
		router:     chi.NewRouter(),
		db:         db,
		repo:       repo,
		tokenSvc:   tokenSvc,
		userSvc:    userSvc,
		channelSvc: channelSvc,
		schedSvc:   schedSvc,
		resSvc:     resSvc,
		adTmplSvc:  adTmplSvc,
		storageMgr: storageMgr,
		tmdbClient: tmdbClient,
		epgGen:     epgGen,
		botSvc:     botSvc,
		playoutEng: playoutEng,
		hlsMgr:     hlsMgr,
	}

	s.setupRoutes()
	return s, nil
}

// Router returns the initialized Chi mux
func (s *Server) Router() *chi.Mux {
	return s.router
}

func (s *Server) setupRoutes() {
	r := s.router

	// Standard middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Enterprise Security Headers
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			next.ServeHTTP(w, r)
		})
	})

	// 1. Root-level Public EPG, HLS & Static Media feeds
	r.Get("/epg/{channel_id}.xml", s.handleEPGXML)
	r.Get("/hls/{channel_id}/master.m3u8", s.handleHLSMaster)
	r.Get("/hls/{channel_id}/playlist.m3u8", s.handleHLSPlaylist)
	r.Get("/hls/{channel_id}/{segment_file}", s.handleHLSSegment)
	r.Handle("/media/*", http.StripPrefix("/media", http.FileServer(http.Dir(s.cfg.MediaDir))))

	// 2. Central API v1 Router (/api/v1/*) with Enterprise AuthN & AuthZ RBAC Middleware
	r.Route("/api/v1", func(v1 chi.Router) {
		v1.Use(EnterpriseAuthMiddleware(s.tokenSvc, s.repo))
		// Public endpoints under /api/v1
		v1.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			jsonResp(w, http.StatusOK, map[string]interface{}{
				"status":    "healthy",
				"timestamp": time.Now().UTC(),
				"version":   "1.0.0",
			})
		})
		v1.Get("/epg/{channel_id}.xml", s.handleEPGXML)

		// Agent heartbeat (authenticated via PairingToken in payload)
		v1.Post("/agents/heartbeat", s.handleAgentHeartbeat)

		// Auth setup & login
		v1.Get("/auth/setup-status", s.handleSetupStatus)
		v1.Post("/auth/setup", s.handleSetup)
		v1.Post("/auth/login", s.handleLogin)

		// Protected API routes (/api/v1/*)
		v1.Group(func(api chi.Router) {
			api.Use(AuthMiddleware(s.tokenSvc))

			// Current User
			api.Get("/auth/me", s.handleMe)

		// Users (Admin only)
		api.Group(func(admin chi.Router) {
			admin.Use(RequireRole(models.RoleAdmin))
			admin.Get("/users", s.handleListUsers)
			admin.Post("/users", s.handleCreateUser)
			admin.Get("/users/{id}", s.handleGetUser)
			admin.Put("/users/{id}", s.handleUpdateUser)
			admin.Delete("/users/{id}", s.handleDeleteUser)
		})

		// Channels
		api.Get("/channels", s.handleListChannels)
		api.Get("/channels/{id}", s.handleGetChannel)
		api.Get("/channels/{id}/playout/status", s.handleGetChannelPlayoutStatus)

		api.Group(func(operator chi.Router) {
			operator.Use(RequireRole(models.RoleAdmin, models.RoleOperator))
			operator.Post("/channels/{id}/playout/start", s.handleStartChannelPlayout)
			operator.Post("/channels/{id}/playout/stop", s.handleStopChannelPlayout)
			operator.Post("/channels/{id}/slate", s.handleToggleChannelSlate)
			operator.Get("/channels/{id}/ffmpeg-cmd", s.handleGetFFmpegCommand)
			operator.Put("/channels/{id}", s.handleUpdateChannel)
			operator.Post("/channels/{id}/logo", s.handleUploadChannelLogo)
			operator.Post("/media/upload-logo", s.handleUploadLogo)
		})

		api.Group(func(admin chi.Router) {
			admin.Use(RequireRole(models.RoleAdmin))
			admin.Post("/channels", s.handleCreateChannel)
			admin.Delete("/channels/{id}", s.handleDeleteChannel)
		})

		// Schedules (Admin, Operator, Content Scheduler)
		api.Get("/schedules", s.handleListSchedules)
		api.Get("/schedules/{id}", s.handleGetSchedule)
		api.Get("/schedules/channel/{channel_id}", s.handleListScheduleByChannel)
		api.Post("/schedules", s.handleCreateSchedule)
		api.Put("/schedules/{id}", s.handleUpdateSchedule)
		api.Delete("/schedules/{id}", s.handleDeleteSchedule)
		api.Post("/schedules/check-conflicts", s.handleCheckScheduleConflicts)

		// Singular schedule aliases
		api.Get("/schedule", s.handleListSchedules)
		api.Get("/schedule/{id}", s.handleGetSchedule)
		api.Get("/schedule/channel/{channel_id}", s.handleListScheduleByChannel)
		api.Post("/schedule", s.handleCreateSchedule)
		api.Put("/schedule/{id}", s.handleUpdateSchedule)
		api.Delete("/schedule/{id}", s.handleDeleteSchedule)
		api.Post("/schedule/check-conflicts", s.handleCheckScheduleConflicts)

		// Resolutions
		api.Get("/resolutions", s.handleListResolutions)
		api.Get("/resolutions/{id}", s.handleGetResolution)
		api.Group(func(admin chi.Router) {
			admin.Use(RequireRole(models.RoleAdmin))
			admin.Post("/resolutions", s.handleCreateResolution)
			admin.Put("/resolutions/{id}", s.handleUpdateResolution)
			admin.Delete("/resolutions/{id}", s.handleDeleteResolution)
		})

		// Ad Templates
		api.Get("/ad-templates", s.handleListAdTemplates)
		api.Get("/ad-templates/{id}", s.handleGetAdTemplate)
		api.Group(func(operator chi.Router) {
			operator.Use(RequireRole(models.RoleAdmin, models.RoleOperator))
			operator.Post("/ad-templates", s.handleCreateAdTemplate)
			operator.Put("/ad-templates/{id}", s.handleUpdateAdTemplate)
			operator.Delete("/ad-templates/{id}", s.handleDeleteAdTemplate)
		})

		// Media Library & Probing
		api.Get("/media/browse", s.handleStorageBrowse)
		api.Get("/storage/browse", s.handleStorageBrowse)
		api.Get("/media/probe", s.handleStorageProbe)
		api.Post("/media/probe", s.handleStorageProbe)
		api.Get("/storage/probe", s.handleStorageProbe)
		api.Post("/storage/probe", s.handleStorageProbe)

		// Edge Agents
		api.Get("/agents", s.handleListAgents)
		api.Get("/agents/{id}", s.handleGetAgent)
		api.Post("/agents/pair", s.handlePairAgent)
		api.Post("/agents/test-connection", s.handleTestAgentConnection)
		api.Post("/agents/{id}/ping", s.handlePingAgent)
		api.Group(func(operator chi.Router) {
			operator.Use(RequireRole(models.RoleAdmin, models.RoleOperator))
			operator.Put("/agents/{id}", s.handleUpdateAgent)
			operator.Delete("/agents/{id}", s.handleDeleteAgent)
		})
		api.Group(func(admin chi.Router) {
			admin.Use(RequireRole(models.RoleAdmin))
			admin.Post("/agents", s.handleCreateAgent)
		})

		// Bots
		api.Get("/bots", s.handleListBots)
		api.Get("/bots/{id}", s.handleGetBot)
		api.Post("/bots/{id}/command", s.handleBotCommand)
		api.Post("/bot/nlp-command", s.handleNLPBotCommand)
		api.Post("/bots/command", s.handleNLPBotCommand)
		api.Group(func(admin chi.Router) {
			admin.Use(RequireRole(models.RoleAdmin))
			admin.Post("/bots", s.handleCreateBot)
			admin.Put("/bots/{id}", s.handleUpdateBot)
			admin.Delete("/bots/{id}", s.handleDeleteBot)
		})

		// TMDB
		api.Get("/tmdb/search", s.handleTMDBSearch)
		api.Get("/tmdb/details/{id}", s.handleTMDBDetails)

		// EPG DVB-EIT
		api.Get("/epg/{channel_id}/eit", s.handleEPGEIT)
		})
	})

	// 3. Embedded Web Assets & Single Page Application Fallback
	r.Mount("/", web.Handler())
}

// -------------------------------------------------------------
// Handlers Implementation
// -------------------------------------------------------------

func (s *Server) handleSetupStatus(w http.ResponseWriter, r *http.Request) {
	users, err := s.repo.ListUsers()
	if err != nil || len(users) == 0 {
		jsonResp(w, http.StatusOK, map[string]interface{}{
			"needs_setup":    true,
			"setup_required": true,
			"user_count":     0,
		})
		return
	}
	jsonResp(w, http.StatusOK, map[string]interface{}{
		"needs_setup":    false,
		"setup_required": false,
		"user_count":     len(users),
	})
}

func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	// Strict safety gate: verify zero existing users in database
	users, err := s.repo.ListUsers()
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "failed to query user registry: "+err.Error())
		return
	}
	if len(users) > 0 {
		jsonErr(w, http.StatusForbidden, "Forbidden: Initial root setup has already been completed. User creation requires authenticated administrator privileges.")
		return
	}

	var body struct {
		Username    string `json:"username"`
		Password    string `json:"password"`
		FullName    string `json:"full_name"`
		DisplayName string `json:"display_name"`
		Email       string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid setup payload")
		return
	}

	if strings.TrimSpace(body.Username) == "" || strings.TrimSpace(body.Password) == "" {
		jsonErr(w, http.StatusBadRequest, "username and password are required")
		return
	}

	fullName := body.FullName
	if fullName == "" {
		fullName = body.DisplayName
	}
	if fullName == "" {
		fullName = body.Username
	}

	u := models.User{
		ID:          "usr-root-admin",
		Username:    body.Username,
		FullName:    fullName,
		DisplayName: fullName,
		Email:       body.Email,
		Role:        models.RoleAdmin,
	}
	if err := s.userSvc.CreateUser(&u, body.Password); err != nil {
		jsonErr(w, http.StatusBadRequest, err.Error())
		return
	}

	createdUser, token, err := s.userSvc.Authenticate(body.Username, body.Password)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "failed to authenticate after setup")
		return
	}

	jsonResp(w, http.StatusOK, map[string]interface{}{
		"message": "setup complete",
		"token":   token,
		"user":    createdUser,
	})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var creds models.UserCredentials
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	u, token, err := s.userSvc.Authenticate(creds.Username, creds.Password)
	if err != nil {
		jsonErr(w, http.StatusUnauthorized, "invalid username or password")
		return
	}

	jsonResp(w, http.StatusOK, map[string]interface{}{
		"token": token,
		"user":  u,
	})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	claims := GetClaims(r.Context())
	if claims == nil {
		jsonErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	u, err := s.userSvc.GetUserByID(claims.UserID)
	if err != nil {
		jsonErr(w, http.StatusNotFound, "user not found")
		return
	}

	jsonResp(w, http.StatusOK, u)
}

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.userSvc.ListUsers()
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResp(w, http.StatusOK, users)
}

func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var body struct {
		models.User
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid user body")
		return
	}

	if body.FullName == "" && body.DisplayName != "" {
		body.FullName = body.DisplayName
	}
	if body.DisplayName == "" {
		body.DisplayName = body.FullName
	}

	if err := s.userSvc.CreateUser(&body.User, body.Password); err != nil {
		jsonErr(w, http.StatusBadRequest, err.Error())
		return
	}
	jsonResp(w, http.StatusCreated, body.User)
}

func (s *Server) handleGetUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	u, err := s.userSvc.GetUserByID(id)
	if err != nil {
		jsonErr(w, http.StatusNotFound, "user not found")
		return
	}
	jsonResp(w, http.StatusOK, u)
}

func (s *Server) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		models.User
		Password *string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid user body")
		return
	}
	body.User.ID = id
	if body.FullName == "" && body.DisplayName != "" {
		body.FullName = body.DisplayName
	}
	if body.DisplayName == "" {
		body.DisplayName = body.FullName
	}
	if err := s.userSvc.UpdateUser(&body.User, body.Password); err != nil {
		jsonErr(w, http.StatusBadRequest, err.Error())
		return
	}
	jsonResp(w, http.StatusOK, body.User)
}

func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.userSvc.DeleteUser(id); err != nil {
		jsonErr(w, http.StatusBadRequest, err.Error())
		return
	}
	jsonResp(w, http.StatusOK, map[string]string{"message": "user deleted"})
}

// Channels
func (s *Server) handleListChannels(w http.ResponseWriter, r *http.Request) {
	channels, err := s.channelSvc.ListChannels()
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResp(w, http.StatusOK, channels)
}

func (s *Server) handleGetChannel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ch, err := s.channelSvc.GetChannelByID(id)
	if err != nil {
		jsonErr(w, http.StatusNotFound, "channel not found")
		return
	}
	jsonResp(w, http.StatusOK, ch)
}

func normalizeChannel(ch *models.Channel) {
	if ch.LogoPosition == "" {
		ch.LogoPosition = "top-right"
	}
	if ch.HlsWebToken == "" {
		ch.HlsWebToken = fmt.Sprintf("live_sec_%s_tok_%d", strings.ReplaceAll(ch.ID, "-", ""), time.Now().Unix())
	}
	if ch.EpgWebToken == "" {
		ch.EpgWebToken = fmt.Sprintf("epg_sec_%s_xml_%d", strings.ReplaceAll(ch.ID, "-", ""), time.Now().Unix())
	}
	for i := range ch.Destinations {
		dst := &ch.Destinations[i]
		if dst.Type == "" && dst.Protocol != "" {
			dst.Type = strings.ToLower(dst.Protocol)
			if strings.Contains(dst.Type, "udp") {
				dst.Type = "udp"
			}
		}
		if dst.Protocol == "" && dst.Type != "" {
			dst.Protocol = dst.Type
		}
		if dst.URL == "" && dst.EndpointURL != "" {
			dst.URL = dst.EndpointURL
		}
		if dst.EndpointURL == "" && dst.URL != "" {
			dst.EndpointURL = dst.URL
		}
	}
}

func (s *Server) handleCreateChannel(w http.ResponseWriter, r *http.Request) {
	var ch models.Channel
	if err := json.NewDecoder(r.Body).Decode(&ch); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid channel body")
		return
	}
	normalizeChannel(&ch)
	if err := s.channelSvc.CreateChannel(&ch); err != nil {
		jsonErr(w, http.StatusBadRequest, err.Error())
		return
	}
	jsonResp(w, http.StatusCreated, ch)
}

func (s *Server) handleUpdateChannel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var ch models.Channel
	if err := json.NewDecoder(r.Body).Decode(&ch); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid channel body")
		return
	}
	ch.ID = id
	normalizeChannel(&ch)
	if err := s.channelSvc.UpdateChannel(&ch); err != nil {
		jsonErr(w, http.StatusBadRequest, err.Error())
		return
	}
	jsonResp(w, http.StatusOK, ch)
}

func (s *Server) handleDeleteChannel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.channelSvc.DeleteChannel(id); err != nil {
		jsonErr(w, http.StatusBadRequest, err.Error())
		return
	}
	jsonResp(w, http.StatusOK, map[string]string{"message": "channel deleted"})
}

func (s *Server) handleGetChannelPlayoutStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	status := s.playoutEng.GetStatus(id)
	jsonResp(w, http.StatusOK, status)
}

func (s *Server) handleStartChannelPlayout(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ch, err := s.channelSvc.GetChannelByID(id)
	if err != nil {
		jsonErr(w, http.StatusNotFound, "channel not found")
		return
	}

	res, err := s.resSvc.GetResolutionByID(ch.ResolutionID)
	if err != nil {
		res = &models.ResolutionPreset{Width: 1920, Height: 1080, FrameRate: 25.0}
	}

	var body struct {
		MediaPath    string `json:"media_path"`
		ProgramTitle string `json:"program_title"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body.MediaPath == "" {
		body.MediaPath = filepath.Join(s.cfg.MediaDir, "sample_movie.mp4")
	}
	if body.ProgramTitle == "" {
		body.ProgramTitle = "Master Control Live Output"
	}

	status, err := s.playoutEng.StartChannel(*ch, *res, body.MediaPath, body.ProgramTitle)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResp(w, http.StatusOK, status)
}

func (s *Server) handleStopChannelPlayout(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.playoutEng.StopChannel(id); err != nil {
		jsonErr(w, http.StatusBadRequest, err.Error())
		return
	}
	jsonResp(w, http.StatusOK, map[string]string{"status": "stopped"})
}

func (s *Server) handleToggleChannelSlate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Enabled bool `json:"enabled"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if err := s.playoutEng.SetEmergencySlate(id, body.Enabled); err != nil {
		jsonErr(w, http.StatusBadRequest, err.Error())
		return
	}
	status := s.playoutEng.GetStatus(id)
	jsonResp(w, http.StatusOK, status)
}

func (s *Server) handleGetFFmpegCommand(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ch, err := s.channelSvc.GetChannelByID(id)
	if err != nil {
		jsonErr(w, http.StatusNotFound, "channel not found")
		return
	}

	res, _ := s.resSvc.GetResolutionByID(ch.ResolutionID)
	if res == nil {
		res = &models.ResolutionPreset{Width: 1920, Height: 1080, FrameRate: 25.0}
	}

	logoPos := ch.LogoPosition
	if logoPos == "" {
		logoPos = "top-right"
	}

	cfg := playout.PlayoutConfig{
		InputMedia:        filepath.Join(s.cfg.MediaDir, "sample_movie.mp4"),
		Resolution:        *res,
		LogoPath:          ch.LogoPath,
		LogoPosition:      logoPos,
		LogoOpacity:       0.9,
		AudioTrackIndex:   0,
		NormalizeLoudness: true,
		Destinations:      ch.Destinations,
	}

	cmdStr := playout.BuildFFmpegCommand(cfg)
	jsonResp(w, http.StatusOK, map[string]string{
		"channel_id": ch.ID,
		"command":    cmdStr,
	})
}

// Schedules
func (s *Server) handleListSchedules(w http.ResponseWriter, r *http.Request) {
	channelID := r.URL.Query().Get("channel_id")
	if channelID != "" {
		items, err := s.schedSvc.ListByChannel(channelID)
		if err != nil {
			jsonErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		jsonResp(w, http.StatusOK, items)
		return
	}
	jsonErr(w, http.StatusBadRequest, "channel_id query parameter is required")
}

func (s *Server) handleListScheduleByChannel(w http.ResponseWriter, r *http.Request) {
	channelID := chi.URLParam(r, "channel_id")
	items, err := s.schedSvc.ListByChannel(channelID)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResp(w, http.StatusOK, items)
}

func (s *Server) handleGetSchedule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	it, err := s.schedSvc.GetItemByID(id)
	if err != nil {
		jsonErr(w, http.StatusNotFound, "schedule item not found")
		return
	}
	jsonResp(w, http.StatusOK, it)
}

type schedulePayload struct {
	models.ScheduleItem
	Item *struct {
		models.ScheduleItem
		TMDBMetadata *struct {
			ID       string `json:"id"`
			Title    string `json:"title"`
			Poster   string `json:"poster_path"`
			Overview string `json:"overview"`
		} `json:"tmdb_metadata,omitempty"`
	} `json:"item,omitempty"`
	TMDBMetadata *struct {
		ID       string `json:"id"`
		Title    string `json:"title"`
		Poster   string `json:"poster_path"`
		Overview string `json:"overview"`
	} `json:"tmdb_metadata,omitempty"`
	Action string `json:"action,omitempty"`
}

func normalizeScheduleItem(item *models.ScheduleItem) {
	if item.ProgramTitle == "" && item.Title != "" {
		item.ProgramTitle = item.Title
	}
	if item.Title == "" && item.ProgramTitle != "" {
		item.Title = item.ProgramTitle
	}
	if item.MediaPath == "" && item.MediaFilePath != "" {
		item.MediaPath = item.MediaFilePath
	}
	if item.MediaFilePath == "" && item.MediaPath != "" {
		item.MediaFilePath = item.MediaPath
	}
}

func (s *Server) handleCreateSchedule(w http.ResponseWriter, r *http.Request) {
	var payload schedulePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid schedule item body")
		return
	}
	item := payload.ScheduleItem
	tmdb := payload.TMDBMetadata
	if payload.Item != nil {
		item = payload.Item.ScheduleItem
		if payload.Item.TMDBMetadata != nil {
			tmdb = payload.Item.TMDBMetadata
		}
	}
	if tmdb != nil {
		if item.TmdbID == "" && tmdb.ID != "" {
			item.TmdbID = tmdb.ID
		}
		if item.TmdbPoster == "" && tmdb.Poster != "" {
			item.TmdbPoster = tmdb.Poster
		}
		if item.TmdbOverview == "" && tmdb.Overview != "" {
			item.TmdbOverview = tmdb.Overview
		}
		if item.ProgramTitle == "" && tmdb.Title != "" {
			item.ProgramTitle = tmdb.Title
			item.Title = tmdb.Title
		}
	}
	normalizeScheduleItem(&item)
	action := payload.Action
	if action == "" {
		action = r.URL.Query().Get("action")
	}
	if action == "" && (r.URL.Query().Get("allow_overlap") == "true" || payload.Action == "FORCE_OVERWRITE") {
		action = schedule.ActionForceLegacy
	}

	if err := s.schedSvc.CreateItem(&item, action); err != nil {
		if errors.Is(err, schedule.ErrScheduleConflict) {
			report, _ := s.schedSvc.CheckConflicts(item.ChannelID, item.StartTime, item.EndTime, "")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error":              err.Error(),
				"has_conflict":       true,
				"conflicts":          report.Conflicts,
				"suggested_start":    report.SuggestedStart,
				"suggested_duration": report.SuggestedDuration,
				"suggested_end":      report.SuggestedEnd,
			})
			return
		}
		jsonErr(w, http.StatusBadRequest, err.Error())
		return
	}
	jsonResp(w, http.StatusCreated, item)
}

func (s *Server) handleUpdateSchedule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var payload schedulePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid schedule item body")
		return
	}
	item := payload.ScheduleItem
	tmdb := payload.TMDBMetadata
	if payload.Item != nil {
		item = payload.Item.ScheduleItem
		if payload.Item.TMDBMetadata != nil {
			tmdb = payload.Item.TMDBMetadata
		}
	}
	item.ID = id
	if tmdb != nil {
		if item.TmdbID == "" && tmdb.ID != "" {
			item.TmdbID = tmdb.ID
		}
		if item.TmdbPoster == "" && tmdb.Poster != "" {
			item.TmdbPoster = tmdb.Poster
		}
		if item.TmdbOverview == "" && tmdb.Overview != "" {
			item.TmdbOverview = tmdb.Overview
		}
		if item.ProgramTitle == "" && tmdb.Title != "" {
			item.ProgramTitle = tmdb.Title
			item.Title = tmdb.Title
		}
	}
	normalizeScheduleItem(&item)
	action := payload.Action
	if action == "" {
		action = r.URL.Query().Get("action")
	}
	if action == "" && (r.URL.Query().Get("allow_overlap") == "true" || payload.Action == "FORCE_OVERWRITE") {
		action = schedule.ActionForceLegacy
	}

	if err := s.schedSvc.UpdateItem(&item, action); err != nil {
		if errors.Is(err, schedule.ErrScheduleConflict) {
			report, _ := s.schedSvc.CheckConflicts(item.ChannelID, item.StartTime, item.EndTime, item.ID)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error":              err.Error(),
				"has_conflict":       true,
				"conflicts":          report.Conflicts,
				"suggested_start":    report.SuggestedStart,
				"suggested_duration": report.SuggestedDuration,
				"suggested_end":      report.SuggestedEnd,
			})
			return
		}
		jsonErr(w, http.StatusBadRequest, err.Error())
		return
	}
	jsonResp(w, http.StatusOK, item)
}

func (s *Server) handleCheckScheduleConflicts(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ChannelID       string    `json:"channel_id"`
		StartTime       time.Time `json:"start_time"`
		DurationSeconds int       `json:"duration_seconds"`
		EndTime         time.Time `json:"end_time"`
		ExcludeID       string    `json:"exclude_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid conflict check request")
		return
	}
	if body.DurationSeconds <= 0 {
		body.DurationSeconds = 3600
	}
	if body.EndTime.IsZero() {
		body.EndTime = body.StartTime.Add(time.Duration(body.DurationSeconds) * time.Second)
	}

	report, err := s.schedSvc.CheckConflicts(body.ChannelID, body.StartTime, body.EndTime, body.ExcludeID)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResp(w, http.StatusOK, report)
}

func (s *Server) handleDeleteSchedule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.schedSvc.DeleteItem(id); err != nil {
		jsonErr(w, http.StatusBadRequest, err.Error())
		return
	}
	jsonResp(w, http.StatusOK, map[string]string{"message": "schedule item deleted"})
}

// Resolutions
func (s *Server) handleListResolutions(w http.ResponseWriter, r *http.Request) {
	list, err := s.resSvc.ListResolutions()
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResp(w, http.StatusOK, list)
}

func (s *Server) handleGetResolution(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	res, err := s.resSvc.GetResolutionByID(id)
	if err != nil {
		jsonErr(w, http.StatusNotFound, "resolution preset not found")
		return
	}
	jsonResp(w, http.StatusOK, res)
}

func normalizeResolution(res *models.ResolutionPreset) {
	if res.FrameRate == 0 && res.FPS > 0 {
		res.FrameRate = res.FPS
	}
	if res.FPS == 0 && res.FrameRate > 0 {
		res.FPS = res.FrameRate
	}
	if !res.Interlaced && res.ScanningMode == "interlaced" {
		res.Interlaced = true
	}
	if res.Interlaced && res.ScanningMode == "" {
		res.ScanningMode = "interlaced"
	}
	if res.ExtraFFmpegArgs == "" && res.ExtraFFmpegVideoArgs != "" {
		res.ExtraFFmpegArgs = res.ExtraFFmpegVideoArgs
	}
	if res.ExtraFFmpegVideoArgs == "" && res.ExtraFFmpegArgs != "" {
		res.ExtraFFmpegVideoArgs = res.ExtraFFmpegArgs
	}
}

func (s *Server) handleCreateResolution(w http.ResponseWriter, r *http.Request) {
	var res models.ResolutionPreset
	if err := json.NewDecoder(r.Body).Decode(&res); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid resolution body")
		return
	}
	normalizeResolution(&res)
	if err := s.resSvc.CreateResolution(&res); err != nil {
		jsonErr(w, http.StatusBadRequest, err.Error())
		return
	}
	jsonResp(w, http.StatusCreated, res)
}

func (s *Server) handleUpdateResolution(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var res models.ResolutionPreset
	if err := json.NewDecoder(r.Body).Decode(&res); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid resolution body")
		return
	}
	res.ID = id
	normalizeResolution(&res)
	if err := s.resSvc.UpdateResolution(&res); err != nil {
		jsonErr(w, http.StatusBadRequest, err.Error())
		return
	}
	jsonResp(w, http.StatusOK, res)
}

func (s *Server) handleDeleteResolution(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.resSvc.DeleteResolution(id); err != nil {
		jsonErr(w, http.StatusBadRequest, err.Error())
		return
	}
	jsonResp(w, http.StatusOK, map[string]string{"message": "resolution deleted"})
}

// Ad Templates
func (s *Server) handleListAdTemplates(w http.ResponseWriter, r *http.Request) {
	list, err := s.adTmplSvc.ListTemplates()
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResp(w, http.StatusOK, list)
}

func (s *Server) handleGetAdTemplate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	tmpl, err := s.adTmplSvc.GetTemplateByID(id)
	if err != nil {
		jsonErr(w, http.StatusNotFound, "ad template not found")
		return
	}
	jsonResp(w, http.StatusOK, tmpl)
}

func (s *Server) handleCreateAdTemplate(w http.ResponseWriter, r *http.Request) {
	var tmpl models.AdTemplate
	if err := json.NewDecoder(r.Body).Decode(&tmpl); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid template body")
		return
	}
	if err := s.adTmplSvc.CreateTemplate(&tmpl); err != nil {
		jsonErr(w, http.StatusBadRequest, err.Error())
		return
	}
	jsonResp(w, http.StatusCreated, tmpl)
}

func (s *Server) handleUpdateAdTemplate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var tmpl models.AdTemplate
	if err := json.NewDecoder(r.Body).Decode(&tmpl); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid template body")
		return
	}
	tmpl.ID = id
	if err := s.adTmplSvc.UpdateTemplate(&tmpl); err != nil {
		jsonErr(w, http.StatusBadRequest, err.Error())
		return
	}
	jsonResp(w, http.StatusOK, tmpl)
}

func (s *Server) handleDeleteAdTemplate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.adTmplSvc.DeleteTemplate(id); err != nil {
		jsonErr(w, http.StatusBadRequest, err.Error())
		return
	}
	jsonResp(w, http.StatusOK, map[string]string{"message": "ad template deleted"})
}

// Media Library & Probing
func (s *Server) handleStorageBrowse(w http.ResponseWriter, r *http.Request) {
	subPath := r.URL.Query().Get("path")
	entries, err := s.storageMgr.Browse(subPath)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResp(w, http.StatusOK, entries)
}

func (s *Server) handleUploadChannelLogo(w http.ResponseWriter, r *http.Request) {
	channelID := chi.URLParam(r, "id")
	ch, err := s.channelSvc.GetChannelByID(channelID)
	if err != nil {
		jsonErr(w, http.StatusNotFound, "channel not found")
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		jsonErr(w, http.StatusBadRequest, "failed to parse multipart form: "+err.Error())
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		file, header, err = r.FormFile("logo")
		if err != nil {
			jsonErr(w, http.StatusBadRequest, "logo file is required (field 'file' or 'logo')")
			return
		}
	}
	defer file.Close()

	logosDir := filepath.Join(s.cfg.MediaDir, "logos")
	_ = os.MkdirAll(logosDir, 0755)

	allowedLogoExts := map[string]bool{
		".png":  true,
		".jpg":  true,
		".jpeg": true,
		".webp": true,
		".svg":  true,
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext == "" || !allowedLogoExts[ext] {
		jsonErr(w, http.StatusBadRequest, "invalid image file format: allowed types are .png, .jpg, .jpeg, .webp, .svg")
		return
	}
	safeName := fmt.Sprintf("%s_logo%s", channelID, ext)
	dstPath := filepath.Join(logosDir, safeName)

	dst, err := os.Create(dstPath)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "failed to save logo on disk: "+err.Error())
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		jsonErr(w, http.StatusInternalServerError, "failed to write logo: "+err.Error())
		return
	}

	relPath := fmt.Sprintf("media/logos/%s", safeName)
	ch.LogoPath = relPath
	if pos := r.FormValue("logo_position"); pos != "" {
		ch.LogoPosition = pos
	}
	if err := s.channelSvc.UpdateChannel(ch); err != nil {
		jsonErr(w, http.StatusInternalServerError, "failed to update channel logo: "+err.Error())
		return
	}

	jsonResp(w, http.StatusOK, map[string]interface{}{
		"channel_id":    ch.ID,
		"logo_path":     ch.LogoPath,
		"logo_position": ch.LogoPosition,
		"url":           "/" + relPath,
		"message":       "Channel logo uploaded successfully",
	})
}

func (s *Server) handleUploadLogo(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		jsonErr(w, http.StatusBadRequest, "failed to parse multipart form: "+err.Error())
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		file, header, err = r.FormFile("logo")
		if err != nil {
			jsonErr(w, http.StatusBadRequest, "logo file is required (field 'file' or 'logo')")
			return
		}
	}
	defer file.Close()

	logosDir := filepath.Join(s.cfg.MediaDir, "logos")
	_ = os.MkdirAll(logosDir, 0755)

	allowedLogoExts := map[string]bool{
		".png":  true,
		".jpg":  true,
		".jpeg": true,
		".webp": true,
		".svg":  true,
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext == "" || !allowedLogoExts[ext] {
		jsonErr(w, http.StatusBadRequest, "invalid image file format: allowed types are .png, .jpg, .jpeg, .webp, .svg")
		return
	}
	safeName := fmt.Sprintf("logo_%d%s", time.Now().Unix(), ext)
	dstPath := filepath.Join(logosDir, safeName)

	dst, err := os.Create(dstPath)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "failed to save logo on disk: "+err.Error())
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		jsonErr(w, http.StatusInternalServerError, "failed to write logo: "+err.Error())
		return
	}

	relPath := fmt.Sprintf("media/logos/%s", safeName)
	jsonResp(w, http.StatusOK, map[string]interface{}{
		"logo_path": relPath,
		"url":       "/" + relPath,
		"message":   "Logo uploaded successfully",
	})
}

func (s *Server) handleStorageProbe(w http.ResponseWriter, r *http.Request) {
	targetPath := r.URL.Query().Get("path")
	if targetPath == "" && r.Method == http.MethodPost {
		var body struct {
			Path string `json:"path"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		targetPath = body.Path
	}

	if targetPath == "" {
		jsonErr(w, http.StatusBadRequest, "file path is required for probing")
		return
	}

	res, err := s.storageMgr.ProbeFile(targetPath)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResp(w, http.StatusOK, res)
}

// Edge Agents
func (s *Server) handleListAgents(w http.ResponseWriter, r *http.Request) {
	agents, err := s.repo.ListEdgeAgents()
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResp(w, http.StatusOK, agents)
}

func (s *Server) handleGetAgent(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	a, err := s.repo.GetEdgeAgentByID(id)
	if err != nil {
		jsonErr(w, http.StatusNotFound, "edge agent not found")
		return
	}
	jsonResp(w, http.StatusOK, a)
}

func normalizeEdgeAgent(a *models.EdgeAgent) {
	if a.PairingToken == "" && a.Token != "" {
		a.PairingToken = a.Token
	}
	if a.Token == "" && a.PairingToken != "" {
		a.Token = a.PairingToken
	}
	if a.CPUPercent == 0 && a.CPUUsagePercent > 0 {
		a.CPUPercent = a.CPUUsagePercent
	}
	if a.CPUUsagePercent == 0 && a.CPUPercent > 0 {
		a.CPUUsagePercent = a.CPUPercent
	}
	if a.MemoryPercent == 0 && a.MemoryUsagePercent > 0 {
		a.MemoryPercent = a.MemoryUsagePercent
	}
	if a.MemoryUsagePercent == 0 && a.MemoryPercent > 0 {
		a.MemoryUsagePercent = a.MemoryPercent
	}
}

func (s *Server) handleCreateAgent(w http.ResponseWriter, r *http.Request) {
	var a struct {
		models.EdgeAgent
		SkipVerification bool `json:"skip_verification"`
		SkipVerify       bool `json:"skip_verify"`
	}
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid agent body")
		return
	}
	normalizeEdgeAgent(&a.EdgeAgent)

	tok := a.PairingToken
	if tok == "" {
		tok = a.Token
	}
	if !strings.HasPrefix(tok, "agt_sec_") || len(tok) < 16 {
		jsonErr(w, http.StatusBadRequest, "invalid pairing token: token must begin with 'agt_sec_' and contain valid credentials")
		return
	}

	skipVerify := r.URL.Query().Get("skip_verify") == "true" || a.SkipVerification || a.SkipVerify
	if !skipVerify {
		targetIP := a.IPAddress
		if targetIP == "" {
			targetIP = "127.0.0.1"
		}
		targetPort := a.Port
		if targetPort <= 0 {
			targetPort = 3082
		}

		client := &http.Client{Timeout: 2 * time.Second}
		statusURL := fmt.Sprintf("http://%s:%d/status", targetIP, targetPort)
		req, _ := http.NewRequest(http.MethodGet, statusURL, nil)
		req.Header.Set("Authorization", "Bearer "+tok)

		resp, err := client.Do(req)
		if err != nil {
			healthURL := fmt.Sprintf("http://%s:%d/health", targetIP, targetPort)
			req2, _ := http.NewRequest(http.MethodGet, healthURL, nil)
			req2.Header.Set("Authorization", "Bearer "+tok)
			resp2, err2 := client.Do(req2)
			if err2 != nil {
				jsonErr(w, http.StatusBadRequest, fmt.Sprintf("Failed to create edge agent: node at %s:%d is unreachable (%v). Ensure mcrflow-agent is running.", targetIP, targetPort, err))
				return
			}
			_ = resp2.Body.Close()
		} else {
			if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
				jsonErr(w, http.StatusUnauthorized, "Failed to create edge agent: pairing token authentication rejected by node")
				return
			}
			_ = resp.Body.Close()
		}
	}

	if err := s.repo.CreateEdgeAgent(&a.EdgeAgent); err != nil {
		jsonErr(w, http.StatusBadRequest, err.Error())
		return
	}
	jsonResp(w, http.StatusCreated, a.EdgeAgent)
}

func (s *Server) handleUpdateAgent(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var a models.EdgeAgent
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid agent body")
		return
	}
	a.ID = id
	normalizeEdgeAgent(&a)
	if a.PairingToken != "" && (!strings.HasPrefix(a.PairingToken, "agt_sec_") || len(a.PairingToken) < 16) {
		jsonErr(w, http.StatusBadRequest, "invalid pairing token: token must begin with 'agt_sec_'")
		return
	}
	if err := s.repo.UpdateEdgeAgent(&a); err != nil {
		jsonErr(w, http.StatusBadRequest, err.Error())
		return
	}
	jsonResp(w, http.StatusOK, a)
}

func (s *Server) handlePairAgent(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Hostname         string `json:"hostname"`
		IPAddress        string `json:"ip_address"`
		TailscaleIP      string `json:"tailscale_ip"`
		Port             int    `json:"port"`
		Token            string `json:"token"`
		PairingToken     string `json:"pairing_token"`
		SkipVerification bool   `json:"skip_verification"`
		SkipVerify       bool   `json:"skip_verify"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid agent pair payload")
		return
	}
	tok := body.Token
	if tok == "" {
		tok = body.PairingToken
	}
	if tok == "" {
		jsonErr(w, http.StatusBadRequest, "pairing token is required")
		return
	}

	if !strings.HasPrefix(tok, "agt_sec_") || len(tok) < 16 {
		jsonErr(w, http.StatusBadRequest, "invalid pairing token: token must begin with 'agt_sec_' and contain valid credentials")
		return
	}

	skipVerify := r.URL.Query().Get("skip_verify") == "true" || body.SkipVerification || body.SkipVerify
	if !skipVerify {
		targetIP := body.IPAddress
		if targetIP == "" {
			targetIP = "127.0.0.1"
		}
		targetPort := body.Port
		if targetPort <= 0 {
			targetPort = 3082
		}

		client := &http.Client{Timeout: 2 * time.Second}
		statusURL := fmt.Sprintf("http://%s:%d/status", targetIP, targetPort)
		req, _ := http.NewRequest(http.MethodGet, statusURL, nil)
		req.Header.Set("Authorization", "Bearer "+tok)

		resp, err := client.Do(req)
		if err != nil {
			healthURL := fmt.Sprintf("http://%s:%d/health", targetIP, targetPort)
			req2, _ := http.NewRequest(http.MethodGet, healthURL, nil)
			req2.Header.Set("Authorization", "Bearer "+tok)
			resp2, err2 := client.Do(req2)
			if err2 != nil {
				jsonErr(w, http.StatusBadRequest, fmt.Sprintf("Failed to pair edge agent: node at %s:%d is unreachable (%v). Ensure mcrflow-agent is running.", targetIP, targetPort, err))
				return
			}
			_ = resp2.Body.Close()
		} else {
			if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
				jsonErr(w, http.StatusUnauthorized, "Failed to pair edge agent: pairing token authentication rejected by node")
				return
			}
			_ = resp.Body.Close()
		}
	}

	agents, err := s.repo.ListEdgeAgents()
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	var existing *models.EdgeAgent
	for i := range agents {
		if agents[i].PairingToken == tok || (body.Hostname != "" && agents[i].Hostname == body.Hostname) {
			existing = &agents[i]
			break
		}
	}
	if existing != nil {
		if body.Hostname != "" {
			existing.Hostname = body.Hostname
		}
		if body.IPAddress != "" {
			existing.IPAddress = body.IPAddress
		}
		if body.TailscaleIP != "" {
			existing.TailscaleIP = body.TailscaleIP
		}
		if body.Port > 0 {
			existing.Port = body.Port
		}
		now := time.Now().UTC()
		existing.PairingToken = tok
		existing.Token = tok
		existing.Status = "online"
		existing.LastHeartbeat = &now
		if err := s.repo.UpdateEdgeAgent(existing); err != nil {
			jsonErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		jsonResp(w, http.StatusOK, existing)
		return
	}

	now := time.Now().UTC()
	agent := models.EdgeAgent{
		Hostname:      body.Hostname,
		IPAddress:     body.IPAddress,
		TailscaleIP:   body.TailscaleIP,
		Port:          body.Port,
		PairingToken:  tok,
		Token:         tok,
		Status:        "online",
		LastHeartbeat: &now,
	}
	if agent.Hostname == "" {
		agent.Hostname = "edge-node-" + time.Now().Format("150405")
	}
	if agent.Port == 0 {
		agent.Port = 3082
	}
	if err := s.repo.CreateEdgeAgent(&agent); err != nil {
		jsonErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResp(w, http.StatusCreated, agent)
}

func (s *Server) handleTestAgentConnection(w http.ResponseWriter, r *http.Request) {
	var body struct {
		IPAddress string `json:"ip_address"`
		Port      int    `json:"port"`
		Token     string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid test request body")
		return
	}

	if body.IPAddress == "" {
		body.IPAddress = "127.0.0.1"
	}
	if body.Port <= 0 {
		body.Port = 3082
	}

	start := time.Now()
	client := &http.Client{Timeout: 2500 * time.Millisecond}

	targetURL := fmt.Sprintf("http://%s:%d/status", body.IPAddress, body.Port)
	req, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		jsonErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if body.Token != "" {
		req.Header.Set("Authorization", "Bearer "+body.Token)
	}

	resp, err := client.Do(req)
	latencyMs := int(time.Since(start).Milliseconds())

	if err != nil {
		healthURL := fmt.Sprintf("http://%s:%d/health", body.IPAddress, body.Port)
		req2, err2 := http.NewRequest(http.MethodGet, healthURL, nil)
		if err2 == nil {
			if body.Token != "" {
				req2.Header.Set("Authorization", "Bearer "+body.Token)
			}
			resp2, errH := client.Do(req2)
			if errH == nil {
				defer resp2.Body.Close()
				jsonResp(w, http.StatusOK, map[string]interface{}{
					"reachable":     true,
					"authenticated": true,
					"latency_ms":    int(time.Since(start).Milliseconds()),
					"status":        "online",
					"endpoint":      healthURL,
				})
				return
			}
		}

		jsonResp(w, http.StatusOK, map[string]interface{}{
			"reachable":     false,
			"authenticated": false,
			"latency_ms":    latencyMs,
			"error":         fmt.Sprintf("Connection refused to %s:%d: %v", body.IPAddress, body.Port, err),
			"status":        "offline",
		})
		return
	}
	defer resp.Body.Close()

	var agentData map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&agentData)

	jsonResp(w, http.StatusOK, map[string]interface{}{
		"reachable":      true,
		"authenticated":  resp.StatusCode == http.StatusOK,
		"status_code":    resp.StatusCode,
		"latency_ms":     latencyMs,
		"agent_data":     agentData,
		"status":         "online",
		"cpu_percent":    agentData["cpu_percent"],
		"memory_percent": agentData["memory_percent"],
	})
}

func (s *Server) handlePingAgent(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ag, err := s.repo.GetEdgeAgentByID(id)
	if err != nil {
		jsonErr(w, http.StatusNotFound, "edge agent not found")
		return
	}

	ip := ag.IPAddress
	if ip == "" {
		ip = "127.0.0.1"
	}
	port := ag.Port
	if port <= 0 {
		port = 3082
	}

	start := time.Now()
	client := &http.Client{Timeout: 2500 * time.Millisecond}
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("http://%s:%d/health", ip, port), nil)
	req.Header.Set("Authorization", "Bearer "+ag.PairingToken)

	resp, err := client.Do(req)
	latencyMs := int(time.Since(start).Milliseconds())

	if err != nil {
		jsonResp(w, http.StatusOK, map[string]interface{}{
			"id":            ag.ID,
			"reachable":     false,
			"authenticated": false,
			"latency_ms":    latencyMs,
			"error":         err.Error(),
			"status":        "offline",
		})
		return
	}
	defer resp.Body.Close()

	now := time.Now().UTC()
	ag.LastHeartbeat = &now
	ag.Status = "online"
	_ = s.repo.UpdateEdgeAgent(ag)

	jsonResp(w, http.StatusOK, map[string]interface{}{
		"id":            ag.ID,
		"reachable":     true,
		"authenticated": resp.StatusCode == http.StatusOK,
		"latency_ms":    latencyMs,
		"status":        "online",
	})
}

func (s *Server) handleDeleteAgent(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.repo.DeleteEdgeAgent(id); err != nil {
		jsonErr(w, http.StatusBadRequest, err.Error())
		return
	}
	jsonResp(w, http.StatusOK, map[string]string{"message": "edge agent deleted"})
}

func (s *Server) handleAgentHeartbeat(w http.ResponseWriter, r *http.Request) {
	var payload models.AgentHeartbeatPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid heartbeat payload")
		return
	}

	agent, err := s.repo.GetEdgeAgentByID(payload.AgentID)
	if err != nil && payload.PairingToken != "" {
		// Try resolving agent by pairing token if agent ID doesn't match
		agents, listErr := s.repo.ListEdgeAgents()
		if listErr == nil {
			for i := range agents {
				if agents[i].PairingToken == payload.PairingToken {
					agent = &agents[i]
					err = nil
					break
				}
			}
		}
	}

	if err != nil || agent == nil {
		jsonErr(w, http.StatusNotFound, "agent not registered")
		return
	}

	if payload.PairingToken == "" || agent.PairingToken == "" || agent.PairingToken != payload.PairingToken {
		jsonErr(w, http.StatusUnauthorized, "invalid or missing pairing token")
		return
	}

	channelsJSON, _ := json.Marshal(payload.ActiveChannels)
	if err := s.repo.UpdateEdgeAgentHeartbeat(agent.ID, payload.CPUPercent, payload.MemoryPercent, string(channelsJSON)); err != nil {
		jsonErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	jsonResp(w, http.StatusOK, map[string]interface{}{
		"status":    "acknowledged",
		"timestamp": time.Now().UTC(),
	})
}

// Bots
func (s *Server) handleListBots(w http.ResponseWriter, r *http.Request) {
	bots, err := s.repo.ListBots()
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResp(w, http.StatusOK, bots)
}

func (s *Server) handleGetBot(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	b, err := s.repo.GetBotByID(id)
	if err != nil {
		jsonErr(w, http.StatusNotFound, "bot not found")
		return
	}
	jsonResp(w, http.StatusOK, b)
}

func (s *Server) handleCreateBot(w http.ResponseWriter, r *http.Request) {
	var b models.Bot
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid bot body")
		return
	}
	if err := s.repo.CreateBot(&b); err != nil {
		jsonErr(w, http.StatusBadRequest, err.Error())
		return
	}
	jsonResp(w, http.StatusCreated, b)
}

func (s *Server) handleUpdateBot(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var b models.Bot
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid bot body")
		return
	}
	b.ID = id
	if err := s.repo.UpdateBot(&b); err != nil {
		jsonErr(w, http.StatusBadRequest, err.Error())
		return
	}
	jsonResp(w, http.StatusOK, b)
}

func (s *Server) handleDeleteBot(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.repo.DeleteBot(id); err != nil {
		jsonErr(w, http.StatusBadRequest, err.Error())
		return
	}
	jsonResp(w, http.StatusOK, map[string]string{"message": "bot deleted"})
}

func (s *Server) handleBotCommand(w http.ResponseWriter, r *http.Request) {
	var req models.BotMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid bot request")
		return
	}
	if req.Message == "" {
		if req.Command != "" {
			req.Message = req.Command
		} else if req.Text != "" {
			req.Message = req.Text
		}
	}
	resp := s.botSvc.ProcessCommand(req)
	jsonResp(w, http.StatusOK, resp)
}

func (s *Server) handleNLPBotCommand(w http.ResponseWriter, r *http.Request) {
	var req models.BotMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid bot request")
		return
	}
	if req.Message == "" {
		if req.Command != "" {
			req.Message = req.Command
		} else if req.Text != "" {
			req.Message = req.Text
		}
	}
	resp := s.botSvc.ProcessCommand(req)
	jsonResp(w, http.StatusOK, resp)
}

// TMDB
func (s *Server) handleTMDBSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("query")
	hits, err := s.tmdbClient.Search(q)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResp(w, http.StatusOK, hits)
}

func (s *Server) handleTMDBDetails(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	details, err := s.tmdbClient.GetDetails(id)
	if err != nil {
		jsonErr(w, http.StatusNotFound, "movie not found")
		return
	}
	jsonResp(w, http.StatusOK, details)
}

// EPG
func (s *Server) handleEPGXML(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	channelID := chi.URLParam(r, "channel_id")
	channelID = strings.TrimSuffix(channelID, ".xml")

	ch, err := s.channelSvc.GetChannelByID(channelID)
	if err != nil {
		http.Error(w, "Channel not found", http.StatusNotFound)
		return
	}

	// Verify EpgWebToken if set
	if ch.EpgWebToken != "" {
		token := r.URL.Query().Get("token")
		if token != ch.EpgWebToken {
			http.Error(w, "Unauthorized: invalid EPG token", http.StatusUnauthorized)
			return
		}
	}

	items, err := s.schedSvc.ListByChannel(channelID)
	if err != nil {
		items = []models.ScheduleItem{}
	}

	xmlBytes, err := s.epgGen.GenerateXMLTV(*ch, items)
	if err != nil {
		http.Error(w, "Failed to generate XMLTV", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	_, _ = w.Write(xmlBytes)
}

func (s *Server) handleEPGEIT(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	channelID := chi.URLParam(r, "channel_id")
	items, err := s.schedSvc.ListByChannel(channelID)
	if err != nil {
		items = []models.ScheduleItem{}
	}

	eit := s.epgGen.GenerateDVBEIT(1, items)
	jsonResp(w, http.StatusOK, eit)
}

// HLS Stream
func (s *Server) handleHLSMaster(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	channelID := chi.URLParam(r, "channel_id")
	token := r.URL.Query().Get("token")

	manifest, err := s.hlsMgr.GenerateMasterManifest(channelID, token)
	if err != nil {
		if err == hls.ErrUnauthorized {
			http.Error(w, "Unauthorized: invalid or missing hls web token", http.StatusUnauthorized)
			return
		}
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	_, _ = w.Write([]byte(manifest))
}

func (s *Server) handleHLSPlaylist(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	channelID := chi.URLParam(r, "channel_id")
	token := r.URL.Query().Get("token")

	playlist, err := s.hlsMgr.GenerateSlidingPlaylist(channelID, token)
	if err != nil {
		if err == hls.ErrUnauthorized {
			http.Error(w, "Unauthorized: invalid or missing hls web token", http.StatusUnauthorized)
			return
		}
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	_, _ = w.Write([]byte(playlist))
}

func (s *Server) handleHLSSegment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	channelID := chi.URLParam(r, "channel_id")
	segmentFile := chi.URLParam(r, "segment_file")
	token := r.URL.Query().Get("token")

	// Parse sequence from segment_123.ts
	seq := 0
	cleanName := strings.TrimSuffix(segmentFile, ".ts")
	parts := strings.Split(cleanName, "_")
	if len(parts) >= 2 {
		seq, _ = strconv.Atoi(parts[len(parts)-1])
	}

	data, err := s.hlsMgr.GetSegmentData(channelID, seq, token)
	if err != nil {
		if err == hls.ErrUnauthorized {
			http.Error(w, "Unauthorized: invalid or missing hls web token", http.StatusUnauthorized)
			return
		}
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "video/mp2t")
	w.Header().Set("Cache-Control", "public, max-age=10")
	_, _ = w.Write(data)
}

// JSON helpers
func jsonResp(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func jsonErr(w http.ResponseWriter, status int, message string) {
	jsonResp(w, status, map[string]string{"error": message})
}
