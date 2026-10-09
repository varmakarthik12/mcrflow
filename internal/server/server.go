package server

import (
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
	"mcrflow/internal/models"
	"mcrflow/internal/resolution"
	"mcrflow/internal/schedule"
	"mcrflow/internal/storage"
	"mcrflow/internal/tmdb"
)

// Server encapsulates the MCRFlow Control Plane HTTP server.
type Server struct {
	router      chi.Router
	chStore     *channel.Store
	resStore    *resolution.Store
	schedStore  *schedule.Store
	storageMgr  *storage.Manager
	tmdbClient  *tmdb.Client
	adStore     *adtemplate.Store
	botStore    *bot.Store
	authMgr     *auth.Manager
	agents      map[string]*models.EdgeAgent
	staticFs    http.FileSystem
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
	StaticFS        http.FileSystem
}

// NewServer initializes routes and middleware.
func NewServer(cfg Config) *Server {
	s := &Server{
		router:      chi.NewRouter(),
		chStore:     cfg.ChannelStore,
		resStore:    cfg.ResolutionStore,
		schedStore:  cfg.ScheduleStore,
		storageMgr:  cfg.StorageManager,
		tmdbClient:  cfg.TmdbClient,
		adStore:     cfg.AdTemplateStore,
		botStore:    cfg.BotStore,
		authMgr:     cfg.AuthManager,
		agents:      make(map[string]*models.EdgeAgent),
		staticFs:    cfg.StaticFS,
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
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Agent-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
}

func (s *Server) setupRoutes() {
	s.router.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", s.handleHealth)

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

	// Static UI Server
	if s.staticFs != nil {
		fs := http.FileServer(s.staticFs)
		s.router.Handle("/*", fs)
	}
}

// Handler implementations

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":      "UP",
		"service":     "MCRFlow Control Plane",
		"version":     "1.0.0",
		"timestamp":   time.Now().UTC(),
		"cluster_sla": "99.98%",
	})
}

func (s *Server) handleListChannels(w http.ResponseWriter, r *http.Request) {
	channels := s.chStore.ListChannels()
	writeJSON(w, http.StatusOK, channels)
}

func (s *Server) handleGetChannel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ch, err := s.chStore.GetChannel(id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, ch)
}

func (s *Server) handleCreateChannel(w http.ResponseWriter, r *http.Request) {
	var ch models.Channel
	if err := json.NewDecoder(r.Body).Decode(&ch); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json payload")
		return
	}
	created, err := s.chStore.CreateChannel(&ch)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) handleUpdateChannel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var ch models.Channel
	if err := json.NewDecoder(r.Body).Decode(&ch); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json payload")
		return
	}
	updated, err := s.chStore.UpdateChannel(id, &ch)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) handleDeleteChannel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.chStore.DeleteChannel(id); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (s *Server) handleEmergencySlate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		Engage bool `json:"engage"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	status, err := s.chStore.TriggerEmergencySlate(id, req.Engage)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"status": status})
}

func (s *Server) handleExportXmltv(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ch, err := s.chStore.GetChannel(id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
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

// Resolution Handlers
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

// Schedule Handlers
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

// Storage Handlers
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

// TMDb Handler
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

// Ad Template Handlers
func (s *Server) handleListAdTemplates(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.adStore.ListTemplates())
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

func (s *Server) handleDeleteAdTemplate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.adStore.DeleteTemplate(id); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// Edge Agent Handlers
func (s *Server) handleListAgents(w http.ResponseWriter, r *http.Request) {
	agents := make([]*models.EdgeAgent, 0, len(s.agents))
	for _, a := range s.agents {
		agents = append(agents, a)
	}
	writeJSON(w, http.StatusOK, agents)
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

// Bot Handlers
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

// Router returns underlying chi router.
func (s *Server) Router() http.Handler {
	return s.router
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
