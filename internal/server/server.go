package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"mcrflow/internal/adtemplate"
	"mcrflow/internal/auth"
	"mcrflow/internal/bot"
	"mcrflow/internal/channel"
	"mcrflow/internal/epg"
	"mcrflow/internal/hls"
	"mcrflow/internal/models"
	"mcrflow/internal/resolution"
	"mcrflow/internal/schedule"
	"mcrflow/internal/storage"
	"mcrflow/internal/tmdb"
	"mcrflow/internal/user"
)

type contextKey string

const userCtxKey contextKey = "mcrflow_user"

// Server encapsulates the MCRFlow Control Plane HTTP server.
type Server struct {
	router     chi.Router
	chStore    *channel.Store
	resStore   *resolution.Store
	schedStore *schedule.Store
	storageMgr *storage.Manager
	tmdbClient *tmdb.Client
	adStore    *adtemplate.Store
	botStore   *bot.Store
	authMgr    *auth.Manager
	userStore  *user.Store
	hlsMgr     *hls.Manager
	agents     map[string]*models.EdgeAgent
	staticFs   http.FileSystem
}

// Config bundles all dependencies for server instantiation.
type Config struct {
	ChannelStore    *channel.Store
	ResolutionStore *resolution.Store
	ScheduleStore   *schedule.Store
	StorageManager  *storage.Manager
	TmdbClient      *tmdb.Client
	AdTemplateStore *adtemplate.Store
	BotStore        *bot.Store
	AuthManager     *auth.Manager
	UserStore       *user.Store
	HlsManager      *hls.Manager
	StaticFS        http.FileSystem
}

// NewServer initializes routes and middleware.
func NewServer(cfg Config) *Server {
	userStore := cfg.UserStore
	if userStore == nil {
		userStore = user.NewStore("")
	}

	chStore := cfg.ChannelStore
	hlsMgr := cfg.HlsManager
	if hlsMgr == nil {
		hlsMgr = hls.NewManager("", func(channelID string) string {
			if chStore != nil {
				if ch, err := chStore.GetChannel(channelID); err == nil && ch != nil {
					return ch.HlsWebToken
				}
			}
			return ""
		})
	}

	s := &Server{
		router:     chi.NewRouter(),
		chStore:    chStore,
		resStore:   cfg.ResolutionStore,
		schedStore: cfg.ScheduleStore,
		storageMgr: cfg.StorageManager,
		tmdbClient: cfg.TmdbClient,
		adStore:    cfg.AdTemplateStore,
		botStore:   cfg.BotStore,
		authMgr:    cfg.AuthManager,
		userStore:  userStore,
		hlsMgr:     hlsMgr,
		agents:     make(map[string]*models.EdgeAgent),
		staticFs:   cfg.StaticFS,
	}

	s.seedDefaultAgents()
	s.setupMiddleware()
	s.setupRoutes()

	return s
}

func (s *Server) seedDefaultAgents() {
	s.agents["delhi-dc1-primary"] = &models.EdgeAgent{
		ID:                 "delhi-dc1-primary",
		Hostname:           "delhi-edge-primary-01",
		IPAddress:          "192.168.1.15",
		TailscaleIP:        "100.64.1.15",
		Status:             "ONLINE",
		CpuUsagePercent:    28.4,
		GpuUsagePercent:    42.1,
		MemoryUsagePercent: 36.0,
		ActiveChannelIDs:   []string{"ch-01"},
		LastHeartbeat:      time.Now().UTC(),
	}

	s.agents["mumbai-dc2-hotstandby"] = &models.EdgeAgent{
		ID:                 "mumbai-dc2-hotstandby",
		Hostname:           "mumbai-edge-standby-02",
		IPAddress:          "192.168.2.99",
		TailscaleIP:        "100.64.2.99",
		Status:             "STANDBY",
		CpuUsagePercent:    16.2,
		GpuUsagePercent:    31.5,
		MemoryUsagePercent: 24.5,
		ActiveChannelIDs:   []string{"ch-01-standby", "ch-02"},
		LastHeartbeat:      time.Now().UTC(),
	}
}

func (s *Server) setupMiddleware() {
	s.router.Use(middleware.RequestID)
	s.router.Use(middleware.RealIP)
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)
	s.router.Use(middleware.Timeout(60 * time.Second))

	s.router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Agent-Token", "X-HLS-Token", "X-EPG-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Whitelisted public endpoints inside api/v1
		if path == "/api/v1/health" ||
			path == "/api/v1/auth/setup-status" ||
			path == "/api/v1/auth/setup" ||
			path == "/api/v1/auth/login" {
			next.ServeHTTP(w, r)
			return
		}

		// Public EPG URL handled with its own optional EpgWebToken validation
		if strings.HasSuffix(path, "/epg.xml") {
			next.ServeHTTP(w, r)
			return
		}

		// Extract token
		authHeader := r.Header.Get("Authorization")
		token := ""
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		} else if strings.HasPrefix(authHeader, "token ") {
			token = strings.TrimPrefix(authHeader, "token ")
		} else {
			token = r.URL.Query().Get("token")
		}

		if token == "" {
			writeError(w, http.StatusUnauthorized, "unauthorized: missing authentication token")
			return
		}

		userObj, err := s.userStore.ValidateToken(token)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized: invalid or expired authentication token")
			return
		}

		// RBAC: RoleContentScheduler restrictions
		if userObj.Role == models.RoleContentScheduler {
			if strings.HasPrefix(path, "/api/v1/users") ||
				(strings.HasPrefix(path, "/api/v1/resolutions") && r.Method != http.MethodGet) ||
				strings.HasPrefix(path, "/api/v1/agents") ||
				(strings.HasPrefix(path, "/api/v1/bots") && !strings.HasPrefix(path, "/api/v1/bots/nlp-command")) ||
				(strings.HasPrefix(path, "/api/v1/channels") && r.Method != http.MethodGet) {
				writeError(w, http.StatusForbidden, "forbidden: role 'content_scheduler' is restricted to scheduling content and media library")
				return
			}
		}

		// RBAC: Users management is strictly admin-only
		if strings.HasPrefix(path, "/api/v1/users") && userObj.Role != models.RoleAdmin {
			writeError(w, http.StatusForbidden, "forbidden: administrator privileges required")
			return
		}

		ctx := context.WithValue(r.Context(), userCtxKey, userObj)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) setupRoutes() {
	// 1. Direct HLS Live Streaming Endpoint (serves /hls/{channel_id}/master.m3u8, etc.)
	s.router.HandleFunc("/hls/*", s.hlsMgr.ServeHTTP)

	// 2. REST API v1
	s.router.Route("/api/v1", func(r chi.Router) {
		r.Use(s.authMiddleware)

		// Health & Auth Setup
		r.Get("/health", s.handleHealth)
		r.Get("/auth/setup-status", s.handleSetupStatus)
		r.Post("/auth/setup", s.handleSetup)
		r.Post("/auth/login", s.handleLogin)
		r.Post("/auth/logout", s.handleLogout)
		r.Get("/auth/me", s.handleMe)

		// User Management (Admin only)
		r.Get("/users", s.handleListUsers)
		r.Post("/users", s.handleCreateUser)
		r.Get("/users/{id}", s.handleGetUser)
		r.Put("/users/{id}", s.handleUpdateUser)
		r.Delete("/users/{id}", s.handleDeleteUser)

		// Channels
		r.Get("/channels", s.handleListChannels)
		r.Post("/channels", s.handleCreateChannel)
		r.Get("/channels/{id}", s.handleGetChannel)
		r.Put("/channels/{id}", s.handleUpdateChannel)
		r.Delete("/channels/{id}", s.handleDeleteChannel)
		r.Post("/channels/{id}/slate", s.handleEmergencySlate)
		r.Get("/channels/{id}/epg.xml", s.handleExportXmltv)

		// Resolution Presets (Management UI)
		r.Get("/resolutions", s.handleListResolutions)
		r.Post("/resolutions", s.handleCreateResolution)
		r.Get("/resolutions/{id}", s.handleGetResolution)
		r.Put("/resolutions/{id}", s.handleUpdateResolution)
		r.Delete("/resolutions/{id}", s.handleDeleteResolution)

		// Scheduling
		r.Get("/schedule", s.handleListSchedule)
		r.Post("/schedule", s.handleCreateScheduleItem)
		r.Delete("/schedule/{id}", s.handleDeleteScheduleItem)

		// Storage
		r.Get("/storage/mounts", s.handleListMounts)
		r.Post("/storage/mounts", s.handleCreateMount)
		r.Delete("/storage/mounts/{id}", s.handleDeleteMount)
		r.Get("/storage/browse", s.handleBrowseStorage)
		r.Get("/storage/probe", s.handleProbeMedia)

		// TMDb
		r.Get("/tmdb/search", s.handleSearchTmdb)

		// Ad Templates
		r.Get("/ad-templates", s.handleListAdTemplates)
		r.Post("/ad-templates", s.handleCreateAdTemplate)
		r.Get("/ad-templates/{id}", s.handleGetAdTemplate)
		r.Delete("/ad-templates/{id}", s.handleDeleteAdTemplate)

		// Edge Agents & Crypto Pairing
		r.Get("/agents", s.handleListAgents)
		r.Post("/agents/pair", s.handlePairAgent)
		r.Post("/agents/{id}/heartbeat", s.handleAgentHeartbeat)

		// Bots & ChatOps
		r.Get("/bots", s.handleListBots)
		r.Post("/bots", s.handleSaveBot)
		r.Delete("/bots/{id}", s.handleDeleteBot)
		r.Post("/bots/nlp-command", s.handleNlpCommand)
	})

	// 3. Static UI Server
	if s.staticFs != nil {
		fs := http.FileServer(s.staticFs)
		s.router.Handle("/*", fs)
	}
}

// Router returns the chi router as http.Handler.
func (s *Server) Router() http.Handler {
	return s.router
}

// UserStore returns the user store.
func (s *Server) UserStore() *user.Store {
	return s.userStore
}

// HlsManager returns the HLS manager.
func (s *Server) HlsManager() *hls.Manager {
	return s.hlsMgr
}

// Helper: resolve HLS URL based on current request host
func (s *Server) resolveChannelHls(r *http.Request, ch *models.Channel) {
	baseURL := "http://" + r.Host
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		baseURL = "https://" + r.Host
	}
	hls.UpdateChannelHlsDestination(ch, baseURL)
}

// ----------------------------------------------------------------------------
// Health & Auth Handlers
// ----------------------------------------------------------------------------

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":      "UP",
		"service":     "MCRFlow Control Plane",
		"version":     "1.0.0",
		"timestamp":   time.Now().UTC(),
		"cluster_sla": "99.98%",
	})
}

func (s *Server) handleSetupStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, models.SetupStatus{
		SetupRequired: s.userStore.IsSetupRequired(),
		UserCount:     s.userStore.Count(),
	})
}

func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	var req user.SetupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid setup payload")
		return
	}

	admin, authResp, err := s.userStore.CreateInitialAdmin(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	_ = admin
	writeJSON(w, http.StatusCreated, authResp)
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req user.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid login credentials payload")
		return
	}

	authResp, err := s.userStore.Authenticate(req.Username, req.Password)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid username or password")
		return
	}

	writeJSON(w, http.StatusOK, authResp)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	token := extractToken(r)
	if token != "" {
		s.userStore.Logout(token)
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	u, ok := r.Context().Value(userCtxKey).(*models.User)
	if !ok || u == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	writeJSON(w, http.StatusOK, u)
}

// ----------------------------------------------------------------------------
// User Management Handlers (Admin Only)
// ----------------------------------------------------------------------------

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	users := s.userStore.ListUsers()
	writeJSON(w, http.StatusOK, users)
}

func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req user.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid user creation payload")
		return
	}

	created, err := s.userStore.CreateUser(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) handleGetUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	u, err := s.userStore.GetUser(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	writeJSON(w, http.StatusOK, u)
}

func (s *Server) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req user.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid user update payload")
		return
	}

	updated, err := s.userStore.UpdateUser(id, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	currentUser, _ := r.Context().Value(userCtxKey).(*models.User)
	currentUserID := ""
	if currentUser != nil {
		currentUserID = currentUser.ID
	}

	if err := s.userStore.DeleteUser(id, currentUserID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// ----------------------------------------------------------------------------
// Channel Handlers
// ----------------------------------------------------------------------------

func (s *Server) handleListChannels(w http.ResponseWriter, r *http.Request) {
	channels := s.chStore.ListChannels()
	for _, ch := range channels {
		s.resolveChannelHls(r, ch)
	}
	writeJSON(w, http.StatusOK, channels)
}

func (s *Server) handleGetChannel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ch, err := s.chStore.GetChannel(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "channel not found")
		return
	}
	s.resolveChannelHls(r, ch)
	writeJSON(w, http.StatusOK, ch)
}

func (s *Server) handleCreateChannel(w http.ResponseWriter, r *http.Request) {
	var ch models.Channel
	if err := json.NewDecoder(r.Body).Decode(&ch); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}

	created, err := s.chStore.CreateChannel(&ch)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	s.resolveChannelHls(r, created)
	// Initialize HLS rolling directory
	s.hlsMgr.GetOrCreateStream(created.ID)

	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) handleUpdateChannel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var ch models.Channel
	if err := json.NewDecoder(r.Body).Decode(&ch); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	updated, err := s.chStore.UpdateChannel(id, &ch)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.resolveChannelHls(r, updated)
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) handleDeleteChannel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.chStore.DeleteChannel(id); err != nil {
		writeError(w, http.StatusNotFound, "channel not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (s *Server) handleEmergencySlate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ch, err := s.chStore.GetChannel(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "channel not found")
		return
	}
	ch.Status = "EMERGENCY_SLATE"
	_, _ = s.chStore.UpdateChannel(id, ch)
	s.resolveChannelHls(r, ch)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":     "EMERGENCY_SLATE_TRIGGERED",
		"channel_id": id,
	})
}

func (s *Server) handleExportXmltv(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ch, err := s.chStore.GetChannel(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "channel not found")
		return
	}

	// Validate optional EpgWebToken query param
	if ch.EpgWebToken != "" {
		reqToken := r.URL.Query().Get("token")
		if reqToken == "" {
			reqToken = r.Header.Get("X-EPG-Token")
		}
		if reqToken != ch.EpgWebToken {
			writeError(w, http.StatusUnauthorized, "unauthorized: invalid or missing epg webtoken")
			return
		}
	}

	items := s.schedStore.ListByChannel(id)
	xmlBytes, err := epg.GenerateXMLTV(ch, items)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(xmlBytes)
}

// ----------------------------------------------------------------------------
// Resolution Handlers
// ----------------------------------------------------------------------------

func (s *Server) handleListResolutions(w http.ResponseWriter, r *http.Request) {
	presets := s.resStore.ListPresets()
	writeJSON(w, http.StatusOK, presets)
}

func (s *Server) handleGetResolution(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	p, err := s.resStore.GetPreset(id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) handleCreateResolution(w http.ResponseWriter, r *http.Request) {
	var p models.ResolutionPreset
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	created, err := s.resStore.CreatePreset(&p)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) handleUpdateResolution(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var p models.ResolutionPreset
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	updated, err := s.resStore.UpdatePreset(id, &p)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) handleDeleteResolution(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.resStore.DeletePreset(id); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// ----------------------------------------------------------------------------
// Schedule Handlers
// ----------------------------------------------------------------------------

func (s *Server) handleListSchedule(w http.ResponseWriter, r *http.Request) {
	channelID := r.URL.Query().Get("channel_id")
	if channelID == "" {
		channelID = "ch-01"
	}
	items := s.schedStore.ListByChannel(channelID)
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleCreateScheduleItem(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Item   models.ScheduleItem   `json:"item"`
		Action models.ConflictAction `json:"action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	created, err := s.schedStore.CreateItem(&req.Item, req.Action)
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) handleDeleteScheduleItem(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.schedStore.DeleteItem(id); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// ----------------------------------------------------------------------------
// Storage Handlers
// ----------------------------------------------------------------------------

func (s *Server) handleListMounts(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.storageMgr.ListMounts())
}

func (s *Server) handleCreateMount(w http.ResponseWriter, r *http.Request) {
	var m models.StorageMount
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	created, err := s.storageMgr.CreateMount(&m)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) handleDeleteMount(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.storageMgr.DeleteMount(id); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (s *Server) handleBrowseStorage(w http.ResponseWriter, r *http.Request) {
	mountID := r.URL.Query().Get("mount_id")
	subPath := r.URL.Query().Get("path")
	if mountID == "" {
		mounts := s.storageMgr.ListMounts()
		if len(mounts) > 0 {
			mountID = mounts[0].ID
		}
	}
	entries, err := s.storageMgr.BrowseDirectory(mountID, subPath)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, entries)
}

func (s *Server) handleProbeMedia(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	res, err := s.storageMgr.ProbeMediaFile(path)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// ----------------------------------------------------------------------------
// TMDb Handlers
// ----------------------------------------------------------------------------

func (s *Server) handleSearchTmdb(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	lang := r.URL.Query().Get("lang")
	res, err := s.tmdbClient.SearchMovie(query, lang)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// ----------------------------------------------------------------------------
// Ad Templates Handlers
// ----------------------------------------------------------------------------

func (s *Server) handleListAdTemplates(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.adStore.ListTemplates())
}

func (s *Server) handleCreateAdTemplate(w http.ResponseWriter, r *http.Request) {
	var tmpl models.AdTemplate
	if err := json.NewDecoder(r.Body).Decode(&tmpl); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	created, err := s.adStore.CreateTemplate(&tmpl)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) handleGetAdTemplate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	t, err := s.adStore.GetTemplate(id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (s *Server) handleDeleteAdTemplate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.adStore.DeleteTemplate(id); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// ----------------------------------------------------------------------------
// Edge Agent & Pairing Handlers
// ----------------------------------------------------------------------------

func (s *Server) handleListAgents(w http.ResponseWriter, r *http.Request) {
	agentsList := make([]*models.EdgeAgent, 0, len(s.agents))
	for _, a := range s.agents {
		agentsList = append(agentsList, a)
	}
	writeJSON(w, http.StatusOK, agentsList)
}

func (s *Server) handlePairAgent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AgentID   string `json:"agent_id"`
		IPAddress string `json:"ip_address"`
		Token     string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}

	if !strings.HasPrefix(req.Token, auth.TokenPrefix) {
		writeError(w, http.StatusUnauthorized, "invalid token format, must begin with "+auth.TokenPrefix)
		return
	}

	agent := &models.EdgeAgent{
		ID:                 req.AgentID,
		Hostname:           req.AgentID,
		IPAddress:          req.IPAddress,
		Status:             "ONLINE",
		CpuUsagePercent:    12.0,
		GpuUsagePercent:    20.0,
		MemoryUsagePercent: 18.0,
		LastHeartbeat:      time.Now().UTC(),
	}
	s.agents[agent.ID] = agent

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"agent":   agent,
		"message": "Edge Agent authenticated and paired successfully",
	})
}

func (s *Server) handleAgentHeartbeat(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	token := r.Header.Get("X-Agent-Token")
	if token == "" {
		writeError(w, http.StatusUnauthorized, "missing X-Agent-Token header")
		return
	}

	if agent, exists := s.agents[id]; exists {
		agent.LastHeartbeat = time.Now().UTC()
		s.chStore.RecordAgentHeartbeat(id)
	}
	writeJSON(w, http.StatusOK, map[string]bool{"acknowledged": true})
}

// ----------------------------------------------------------------------------
// Bot & ChatOps Handlers
// ----------------------------------------------------------------------------

func (s *Server) handleListBots(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.botStore.ListBots())
}

func (s *Server) handleSaveBot(w http.ResponseWriter, r *http.Request) {
	var b models.BotConfig
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	saved, err := s.botStore.SaveBot(&b)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

func (s *Server) handleDeleteBot(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.botStore.DeleteBot(id); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (s *Server) handleNlpCommand(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Text      string `json:"text"`
		ChannelID string `json:"channel_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	res, err := s.botStore.ProcessNaturalLanguageCommand(req.Text, req.ChannelID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// ----------------------------------------------------------------------------
// Utilities
// ----------------------------------------------------------------------------

func extractToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}
	return r.URL.Query().Get("token")
}

func writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, statusCode int, msg string) {
	writeJSON(w, statusCode, map[string]string{"error": msg})
}
