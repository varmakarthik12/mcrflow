package server

import (
	"encoding/json"
	"fmt"
	"net/http"
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
	storageMgr := storage.NewManager(repo)
	tmdbClient := tmdb.NewClient(cfg.TMDBKey)
	epgGen := epg.NewGenerator()
	botSvc := bot.NewService(schedSvc, channelSvc, tmdbClient)
	playoutEng := playout.NewEngine()
	hlsMgr := hls.NewManager(repo)

	// Synchronize storage mounts and channels with storageMgr & hlsMgr
	if err := storageMgr.SyncMounts(); err != nil {
		return nil, fmt.Errorf("failed to sync storage mounts: %w", err)
	}

	// If custom MediaDir provided, register or update local mount
	if cfg.MediaDir != "" {
		storageMgr.RegisterMount(models.StorageMount{
			ID:        "mount-default-media",
			Name:      "Default Media Library",
			MountType: "local",
			MountPath: cfg.MediaDir,
			IsActive:  true,
		})
	}

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

	// 1. Root-level Public EPG & HLS feeds
	r.Get("/epg/{channel_id}.xml", s.handleEPGXML)
	r.Get("/hls/{channel_id}/master.m3u8", s.handleHLSMaster)
	r.Get("/hls/{channel_id}/playlist.m3u8", s.handleHLSPlaylist)
	r.Get("/hls/{channel_id}/{segment_file}", s.handleHLSSegment)

	// 2. Central API v1 Router (/api/v1/*)
	r.Route("/api/v1", func(v1 chi.Router) {
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
			operator.Get("/channels/{id}/ffmpeg-cmd", s.handleGetFFmpegCommand)
			operator.Put("/channels/{id}", s.handleUpdateChannel)
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

		// Singular schedule aliases
		api.Get("/schedule", s.handleListSchedules)
		api.Get("/schedule/{id}", s.handleGetSchedule)
		api.Get("/schedule/channel/{channel_id}", s.handleListScheduleByChannel)
		api.Post("/schedule", s.handleCreateSchedule)
		api.Put("/schedule/{id}", s.handleUpdateSchedule)
		api.Delete("/schedule/{id}", s.handleDeleteSchedule)

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

		// Storage
		api.Get("/storage/mounts", s.handleListStorageMounts)
		api.Get("/storage/mounts/{id}", s.handleGetStorageMount)
		api.Get("/storage/browse", s.handleStorageBrowse)
		api.Get("/storage/browse/{mount_id}", s.handleStorageBrowse)
		api.Get("/storage/probe", s.handleStorageProbe)
		api.Post("/storage/probe", s.handleStorageProbe)
		api.Group(func(admin chi.Router) {
			admin.Use(RequireRole(models.RoleAdmin))
			admin.Post("/storage/mounts", s.handleCreateStorageMount)
			admin.Put("/storage/mounts/{id}", s.handleUpdateStorageMount)
			admin.Delete("/storage/mounts/{id}", s.handleDeleteStorageMount)
		})

		// Edge Agents
		api.Get("/agents", s.handleListAgents)
		api.Get("/agents/{id}", s.handleGetAgent)
		api.Post("/agents/pair", s.handlePairAgent)
		api.Group(func(admin chi.Router) {
			admin.Use(RequireRole(models.RoleAdmin))
			admin.Post("/agents", s.handleCreateAgent)
			admin.Put("/agents/{id}", s.handleUpdateAgent)
			admin.Delete("/agents/{id}", s.handleDeleteAgent)
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
	users, err := s.userSvc.ListUsers()
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

	if body.Username == "" || body.Password == "" {
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

	existing, _ := s.userSvc.ListUsers()
	if len(existing) > 0 {
		jsonErr(w, http.StatusConflict, "setup already completed")
		return
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
		body.MediaPath = "/media/storage/sample.mp4"
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

	cfg := playout.PlayoutConfig{
		InputMedia:        "/media/storage/sample.mp4",
		Resolution:        *res,
		LogoPath:          ch.LogoPath,
		LogoPosition:      "top-right",
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
	allowOverlap := r.URL.Query().Get("allow_overlap") == "true" || payload.Action == "FORCE_OVERWRITE"
	if err := s.schedSvc.CreateItem(&item, allowOverlap); err != nil {
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
	allowOverlap := r.URL.Query().Get("allow_overlap") == "true" || payload.Action == "FORCE_OVERWRITE"
	if err := s.schedSvc.UpdateItem(&item, allowOverlap); err != nil {
		jsonErr(w, http.StatusBadRequest, err.Error())
		return
	}
	jsonResp(w, http.StatusOK, item)
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

// Storage
func (s *Server) handleListStorageMounts(w http.ResponseWriter, r *http.Request) {
	mounts, err := s.repo.ListStorageMounts()
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResp(w, http.StatusOK, mounts)
}

func (s *Server) handleGetStorageMount(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	m, err := s.repo.GetStorageMountByID(id)
	if err != nil {
		jsonErr(w, http.StatusNotFound, "storage mount not found")
		return
	}
	jsonResp(w, http.StatusOK, m)
}

func normalizeStorageMount(m *models.StorageMount) {
	if m.MountType == "" && m.Type != "" {
		m.MountType = m.Type
	}
	if m.Type == "" && m.MountType != "" {
		m.Type = m.MountType
	}
	if m.MountPath == "" && m.TargetPath != "" {
		m.MountPath = m.TargetPath
	}
	if m.TargetPath == "" && m.MountPath != "" {
		m.TargetPath = m.MountPath
	}
	if m.SmbURL == "" && m.ServerHost != "" {
		m.SmbURL = m.ServerHost
	}
	if m.ServerHost == "" && m.SmbURL != "" {
		m.ServerHost = m.SmbURL
	}
}

func (s *Server) handleCreateStorageMount(w http.ResponseWriter, r *http.Request) {
	var m models.StorageMount
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid mount body")
		return
	}
	normalizeStorageMount(&m)
	if err := s.repo.CreateStorageMount(&m); err != nil {
		jsonErr(w, http.StatusBadRequest, err.Error())
		return
	}
	s.storageMgr.RegisterMount(m)
	jsonResp(w, http.StatusCreated, m)
}

func (s *Server) handleUpdateStorageMount(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var m models.StorageMount
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid mount body")
		return
	}
	m.ID = id
	normalizeStorageMount(&m)
	if err := s.repo.UpdateStorageMount(&m); err != nil {
		jsonErr(w, http.StatusBadRequest, err.Error())
		return
	}
	s.storageMgr.RegisterMount(m)
	jsonResp(w, http.StatusOK, m)
}

func (s *Server) handleDeleteStorageMount(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.repo.DeleteStorageMount(id); err != nil {
		jsonErr(w, http.StatusBadRequest, err.Error())
		return
	}
	_ = s.storageMgr.SyncMounts()
	jsonResp(w, http.StatusOK, map[string]string{"message": "storage mount deleted"})
}

func (s *Server) handleStorageBrowse(w http.ResponseWriter, r *http.Request) {
	mountID := r.URL.Query().Get("mount_id")
	if mountID == "" {
		mountID = chi.URLParam(r, "mount_id")
	}
	subPath := r.URL.Query().Get("path")

	if mountID == "" {
		mounts := s.storageMgr.ListActiveMounts()
		if len(mounts) == 0 {
			jsonResp(w, http.StatusOK, []models.FileEntry{})
			return
		}
		mountID = mounts[0].ID
	}

	entries, err := s.storageMgr.Browse(mountID, subPath)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResp(w, http.StatusOK, entries)
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
	var a models.EdgeAgent
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid agent body")
		return
	}
	normalizeEdgeAgent(&a)
	if err := s.repo.CreateEdgeAgent(&a); err != nil {
		jsonErr(w, http.StatusBadRequest, err.Error())
		return
	}
	jsonResp(w, http.StatusCreated, a)
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
	if err := s.repo.UpdateEdgeAgent(&a); err != nil {
		jsonErr(w, http.StatusBadRequest, err.Error())
		return
	}
	jsonResp(w, http.StatusOK, a)
}

func (s *Server) handlePairAgent(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Hostname     string `json:"hostname"`
		IPAddress    string `json:"ip_address"`
		TailscaleIP  string `json:"tailscale_ip"`
		Port         int    `json:"port"`
		Token        string `json:"token"`
		PairingToken string `json:"pairing_token"`
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
		agent.Port = 8080
	}
	if err := s.repo.CreateEdgeAgent(&agent); err != nil {
		jsonErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResp(w, http.StatusCreated, agent)
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

	if agent.PairingToken != "" && agent.PairingToken != payload.PairingToken {
		jsonErr(w, http.StatusUnauthorized, "invalid pairing token")
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
