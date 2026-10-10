package server

import (
	"context"
	"net/http"
	"strings"

	"github.com/varmakarthik12/mcrflow/internal/auth"
	"github.com/varmakarthik12/mcrflow/internal/database"
	"github.com/varmakarthik12/mcrflow/internal/models"
)

// RouteRule defines access permissions for an HTTP route
type RouteRule struct {
	Method       string   // HTTP method: GET, POST, PUT, DELETE, or "*"
	PathPattern  string   // URL path pattern, e.g. "/api/v1/users/{id}"
	AllowedRoles []string // Roles permitted: admin, operator, content_scheduler
	IsPublic     bool     // If true, route bypasses JWT auth
}

// EnterpriseRBACPolicy is the master enterprise role-to-endpoint access mapping
var EnterpriseRBACPolicy = []RouteRule{
	// -------------------------------------------------------------
	// 1. Public Unauthenticated Endpoints
	// -------------------------------------------------------------
	{Method: "GET", PathPattern: "/api/v1/health", IsPublic: true},
	{Method: "GET", PathPattern: "/api/v1/auth/setup-status", IsPublic: true},
	{Method: "POST", PathPattern: "/api/v1/auth/setup", IsPublic: true}, // Gated: allowed ONLY when user_count == 0
	{Method: "POST", PathPattern: "/api/v1/auth/login", IsPublic: true},
	{Method: "POST", PathPattern: "/api/v1/agents/heartbeat", IsPublic: true}, // Authenticated via PairingToken in payload
	{Method: "GET", PathPattern: "/api/v1/epg/{channel_id}.xml", IsPublic: true}, // Authenticated via EpgWebToken query param

	// -------------------------------------------------------------
	// 2. Authentication & Current User
	// -------------------------------------------------------------
	{
		Method:       "GET",
		PathPattern:  "/api/v1/auth/me",
		AllowedRoles: []string{models.RoleAdmin, models.RoleOperator, models.RoleContentScheduler},
	},

	// -------------------------------------------------------------
	// 3. User Administration (Admin Only)
	// -------------------------------------------------------------
	{Method: "GET", PathPattern: "/api/v1/users", AllowedRoles: []string{models.RoleAdmin}},
	{Method: "POST", PathPattern: "/api/v1/users", AllowedRoles: []string{models.RoleAdmin}},
	{Method: "GET", PathPattern: "/api/v1/users/{id}", AllowedRoles: []string{models.RoleAdmin}},
	{Method: "PUT", PathPattern: "/api/v1/users/{id}", AllowedRoles: []string{models.RoleAdmin}},
	{Method: "DELETE", PathPattern: "/api/v1/users/{id}", AllowedRoles: []string{models.RoleAdmin}},

	// -------------------------------------------------------------
	// 4. Channel Management & Playout Operations
	// -------------------------------------------------------------
	{
		Method:       "GET",
		PathPattern:  "/api/v1/channels",
		AllowedRoles: []string{models.RoleAdmin, models.RoleOperator, models.RoleContentScheduler},
	},
	{
		Method:       "GET",
		PathPattern:  "/api/v1/channels/{id}",
		AllowedRoles: []string{models.RoleAdmin, models.RoleOperator, models.RoleContentScheduler},
	},
	{
		Method:       "GET",
		PathPattern:  "/api/v1/channels/{id}/playout/status",
		AllowedRoles: []string{models.RoleAdmin, models.RoleOperator, models.RoleContentScheduler},
	},
	{
		Method:       "POST",
		PathPattern:  "/api/v1/channels",
		AllowedRoles: []string{models.RoleAdmin},
	},
	{
		Method:       "PUT",
		PathPattern:  "/api/v1/channels/{id}",
		AllowedRoles: []string{models.RoleAdmin, models.RoleOperator},
	},
	{
		Method:       "DELETE",
		PathPattern:  "/api/v1/channels/{id}",
		AllowedRoles: []string{models.RoleAdmin},
	},
	{
		Method:       "POST",
		PathPattern:  "/api/v1/channels/{id}/playout/start",
		AllowedRoles: []string{models.RoleAdmin, models.RoleOperator},
	},
	{
		Method:       "POST",
		PathPattern:  "/api/v1/channels/{id}/playout/stop",
		AllowedRoles: []string{models.RoleAdmin, models.RoleOperator},
	},
	{
		Method:       "POST",
		PathPattern:  "/api/v1/channels/{id}/slate",
		AllowedRoles: []string{models.RoleAdmin, models.RoleOperator},
	},
	{
		Method:       "GET",
		PathPattern:  "/api/v1/channels/{id}/ffmpeg-cmd",
		AllowedRoles: []string{models.RoleAdmin, models.RoleOperator},
	},
	{
		Method:       "POST",
		PathPattern:  "/api/v1/channels/{id}/logo",
		AllowedRoles: []string{models.RoleAdmin, models.RoleOperator},
	},
	{
		Method:       "POST",
		PathPattern:  "/api/v1/media/upload-logo",
		AllowedRoles: []string{models.RoleAdmin, models.RoleOperator},
	},

	// -------------------------------------------------------------
	// 5. Broadcast Schedule Management
	// -------------------------------------------------------------
	{
		Method:       "GET",
		PathPattern:  "/api/v1/schedules",
		AllowedRoles: []string{models.RoleAdmin, models.RoleOperator, models.RoleContentScheduler},
	},
	{
		Method:       "GET",
		PathPattern:  "/api/v1/schedules/{id}",
		AllowedRoles: []string{models.RoleAdmin, models.RoleOperator, models.RoleContentScheduler},
	},
	{
		Method:       "GET",
		PathPattern:  "/api/v1/schedules/channel/{channel_id}",
		AllowedRoles: []string{models.RoleAdmin, models.RoleOperator, models.RoleContentScheduler},
	},
	{
		Method:       "POST",
		PathPattern:  "/api/v1/schedules",
		AllowedRoles: []string{models.RoleAdmin, models.RoleOperator, models.RoleContentScheduler},
	},
	{
		Method:       "PUT",
		PathPattern:  "/api/v1/schedules/{id}",
		AllowedRoles: []string{models.RoleAdmin, models.RoleOperator, models.RoleContentScheduler},
	},
	{
		Method:       "DELETE",
		PathPattern:  "/api/v1/schedules/{id}",
		AllowedRoles: []string{models.RoleAdmin, models.RoleOperator, models.RoleContentScheduler},
	},
	{
		Method:       "POST",
		PathPattern:  "/api/v1/schedules/check-conflicts",
		AllowedRoles: []string{models.RoleAdmin, models.RoleOperator, models.RoleContentScheduler},
	},

	// Schedule aliases (singular /api/v1/schedule)
	{
		Method:       "GET",
		PathPattern:  "/api/v1/schedule",
		AllowedRoles: []string{models.RoleAdmin, models.RoleOperator, models.RoleContentScheduler},
	},
	{
		Method:       "GET",
		PathPattern:  "/api/v1/schedule/{id}",
		AllowedRoles: []string{models.RoleAdmin, models.RoleOperator, models.RoleContentScheduler},
	},
	{
		Method:       "GET",
		PathPattern:  "/api/v1/schedule/channel/{channel_id}",
		AllowedRoles: []string{models.RoleAdmin, models.RoleOperator, models.RoleContentScheduler},
	},
	{
		Method:       "POST",
		PathPattern:  "/api/v1/schedule",
		AllowedRoles: []string{models.RoleAdmin, models.RoleOperator, models.RoleContentScheduler},
	},
	{
		Method:       "PUT",
		PathPattern:  "/api/v1/schedule/{id}",
		AllowedRoles: []string{models.RoleAdmin, models.RoleOperator, models.RoleContentScheduler},
	},
	{
		Method:       "DELETE",
		PathPattern:  "/api/v1/schedule/{id}",
		AllowedRoles: []string{models.RoleAdmin, models.RoleOperator, models.RoleContentScheduler},
	},
	{
		Method:       "POST",
		PathPattern:  "/api/v1/schedule/check-conflicts",
		AllowedRoles: []string{models.RoleAdmin, models.RoleOperator, models.RoleContentScheduler},
	},

	// -------------------------------------------------------------
	// 6. Resolution Presets
	// -------------------------------------------------------------
	{
		Method:       "GET",
		PathPattern:  "/api/v1/resolutions",
		AllowedRoles: []string{models.RoleAdmin, models.RoleOperator, models.RoleContentScheduler},
	},
	{
		Method:       "GET",
		PathPattern:  "/api/v1/resolutions/{id}",
		AllowedRoles: []string{models.RoleAdmin, models.RoleOperator, models.RoleContentScheduler},
	},
	{
		Method:       "POST",
		PathPattern:  "/api/v1/resolutions",
		AllowedRoles: []string{models.RoleAdmin},
	},
	{
		Method:       "PUT",
		PathPattern:  "/api/v1/resolutions/{id}",
		AllowedRoles: []string{models.RoleAdmin},
	},
	{
		Method:       "DELETE",
		PathPattern:  "/api/v1/resolutions/{id}",
		AllowedRoles: []string{models.RoleAdmin},
	},

	// -------------------------------------------------------------
	// 7. Ad & Graphics Templates (Ad Studio)
	// -------------------------------------------------------------
	{
		Method:       "GET",
		PathPattern:  "/api/v1/ad-templates",
		AllowedRoles: []string{models.RoleAdmin, models.RoleOperator, models.RoleContentScheduler},
	},
	{
		Method:       "GET",
		PathPattern:  "/api/v1/ad-templates/{id}",
		AllowedRoles: []string{models.RoleAdmin, models.RoleOperator, models.RoleContentScheduler},
	},
	{
		Method:       "POST",
		PathPattern:  "/api/v1/ad-templates",
		AllowedRoles: []string{models.RoleAdmin, models.RoleOperator},
	},
	{
		Method:       "PUT",
		PathPattern:  "/api/v1/ad-templates/{id}",
		AllowedRoles: []string{models.RoleAdmin, models.RoleOperator},
	},
	{
		Method:       "DELETE",
		PathPattern:  "/api/v1/ad-templates/{id}",
		AllowedRoles: []string{models.RoleAdmin, models.RoleOperator},
	},

	// -------------------------------------------------------------
	// 8. Storage & Media Library Inspection
	// -------------------------------------------------------------
	{
		Method:       "GET",
		PathPattern:  "/api/v1/storage/browse",
		AllowedRoles: []string{models.RoleAdmin, models.RoleOperator, models.RoleContentScheduler},
	},
	{
		Method:       "GET",
		PathPattern:  "/api/v1/media/browse",
		AllowedRoles: []string{models.RoleAdmin, models.RoleOperator, models.RoleContentScheduler},
	},
	{
		Method:       "GET",
		PathPattern:  "/api/v1/storage/probe",
		AllowedRoles: []string{models.RoleAdmin, models.RoleOperator, models.RoleContentScheduler},
	},
	{
		Method:       "POST",
		PathPattern:  "/api/v1/storage/probe",
		AllowedRoles: []string{models.RoleAdmin, models.RoleOperator, models.RoleContentScheduler},
	},
	{
		Method:       "GET",
		PathPattern:  "/api/v1/media/probe",
		AllowedRoles: []string{models.RoleAdmin, models.RoleOperator, models.RoleContentScheduler},
	},
	{
		Method:       "POST",
		PathPattern:  "/api/v1/media/probe",
		AllowedRoles: []string{models.RoleAdmin, models.RoleOperator, models.RoleContentScheduler},
	},

	// -------------------------------------------------------------
	// 9. Edge Playout Agents
	// -------------------------------------------------------------
	{
		Method:       "GET",
		PathPattern:  "/api/v1/agents",
		AllowedRoles: []string{models.RoleAdmin, models.RoleOperator},
	},
	{
		Method:       "GET",
		PathPattern:  "/api/v1/agents/{id}",
		AllowedRoles: []string{models.RoleAdmin, models.RoleOperator},
	},
	{
		Method:       "POST",
		PathPattern:  "/api/v1/agents",
		AllowedRoles: []string{models.RoleAdmin},
	},
	{
		Method:       "PUT",
		PathPattern:  "/api/v1/agents/{id}",
		AllowedRoles: []string{models.RoleAdmin, models.RoleOperator},
	},
	{
		Method:       "DELETE",
		PathPattern:  "/api/v1/agents/{id}",
		AllowedRoles: []string{models.RoleAdmin},
	},
	{
		Method:       "POST",
		PathPattern:  "/api/v1/agents/pair",
		AllowedRoles: []string{models.RoleAdmin},
	},
	{
		Method:       "POST",
		PathPattern:  "/api/v1/agents/test-connection",
		AllowedRoles: []string{models.RoleAdmin, models.RoleOperator},
	},
	{
		Method:       "POST",
		PathPattern:  "/api/v1/agents/{id}/ping",
		AllowedRoles: []string{models.RoleAdmin, models.RoleOperator},
	},

	// -------------------------------------------------------------
	// 10. ChatOps Bots
	// -------------------------------------------------------------
	{
		Method:       "GET",
		PathPattern:  "/api/v1/bots",
		AllowedRoles: []string{models.RoleAdmin, models.RoleOperator},
	},
	{
		Method:       "GET",
		PathPattern:  "/api/v1/bots/{id}",
		AllowedRoles: []string{models.RoleAdmin, models.RoleOperator},
	},
	{
		Method:       "POST",
		PathPattern:  "/api/v1/bots",
		AllowedRoles: []string{models.RoleAdmin},
	},
	{
		Method:       "PUT",
		PathPattern:  "/api/v1/bots/{id}",
		AllowedRoles: []string{models.RoleAdmin},
	},
	{
		Method:       "DELETE",
		PathPattern:  "/api/v1/bots/{id}",
		AllowedRoles: []string{models.RoleAdmin},
	},
	{
		Method:       "POST",
		PathPattern:  "/api/v1/bots/{id}/command",
		AllowedRoles: []string{models.RoleAdmin, models.RoleOperator},
	},
	{
		Method:       "POST",
		PathPattern:  "/api/v1/bot/nlp-command",
		AllowedRoles: []string{models.RoleAdmin, models.RoleOperator},
	},
	{
		Method:       "POST",
		PathPattern:  "/api/v1/bots/command",
		AllowedRoles: []string{models.RoleAdmin, models.RoleOperator},
	},

	// -------------------------------------------------------------
	// 11. TMDb & EPG Metadata
	// -------------------------------------------------------------
	{
		Method:       "GET",
		PathPattern:  "/api/v1/tmdb/search",
		AllowedRoles: []string{models.RoleAdmin, models.RoleOperator, models.RoleContentScheduler},
	},
	{
		Method:       "GET",
		PathPattern:  "/api/v1/tmdb/details/{id}",
		AllowedRoles: []string{models.RoleAdmin, models.RoleOperator, models.RoleContentScheduler},
	},
	{
		Method:       "GET",
		PathPattern:  "/api/v1/epg/{channel_id}/eit",
		AllowedRoles: []string{models.RoleAdmin, models.RoleOperator, models.RoleContentScheduler},
	},
}

// matchPath checks if a request path matches a route pattern with {param} or wildcard *
func matchPath(pattern, path string) bool {
	// Strip trailing slashes for comparison
	pattern = strings.TrimRight(pattern, "/")
	path = strings.TrimRight(path, "/")

	if pattern == path {
		return true
	}

	pSegs := strings.Split(pattern, "/")
	rSegs := strings.Split(path, "/")

	if len(pSegs) != len(rSegs) {
		return false
	}

	for i := range pSegs {
		if strings.HasPrefix(pSegs[i], "{") && strings.HasSuffix(pSegs[i], "}") {
			// URL parameter placeholder (e.g. {id}, {channel_id})
			continue
		}
		if pSegs[i] == "*" {
			continue
		}
		if !strings.EqualFold(pSegs[i], rSegs[i]) {
			return false
		}
	}
	return true
}

// FindMatchingRule locates the RBAC rule for the given method and path
func FindMatchingRule(method, path string) *RouteRule {
	for i := range EnterpriseRBACPolicy {
		rule := &EnterpriseRBACPolicy[i]
		if (rule.Method == "*" || strings.EqualFold(rule.Method, method)) && matchPath(rule.PathPattern, path) {
			return rule
		}
	}
	return nil
}

// IsPublicEndpoint checks if the request path is explicitly unauthenticated
func IsPublicEndpoint(method, path string) bool {
	// Root HLS & EPG feeds are public from JWT perspective (authenticated via stream tokens)
	if strings.HasPrefix(path, "/hls/") || strings.HasPrefix(path, "/epg/") {
		return true
	}
	if !strings.HasPrefix(path, "/api/v1") {
		return true
	}

	rule := FindMatchingRule(method, path)
	if rule != nil && rule.IsPublic {
		return true
	}
	return false
}

const userModelContextKey = contextKey("mcrflow_user_model")

// GetAuthenticatedUser extracts the full database user from request context
func GetAuthenticatedUser(ctx context.Context) *models.User {
	if val := ctx.Value(userModelContextKey); val != nil {
		if u, ok := val.(*models.User); ok {
			return u
		}
	}
	return nil
}

// EnterpriseAuthMiddleware provides unified Authentication and Authorization (RBAC) enforcement
func EnterpriseAuthMiddleware(tokenService *auth.TokenService, repo *database.Repository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path

			// 1. Bypass check for public endpoints
			if IsPublicEndpoint(r.Method, path) {
				// Special security gate for /api/v1/auth/setup:
				// Strictly forbidden if users already exist in the database
				if r.Method == "POST" && (path == "/api/v1/auth/setup" || path == "/api/v1/auth/setup/") {
					users, err := repo.ListUsers()
					if err == nil && len(users) > 0 {
						w.Header().Set("Content-Type", "application/json; charset=utf-8")
						w.WriteHeader(http.StatusForbidden)
						_, _ = w.Write([]byte(`{"error":"forbidden: root administrator setup is already finalized"}`))
						return
					}
				}
				next.ServeHTTP(w, r)
				return
			}

			// 2. Extract Bearer token
			tokenStr := ""
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" {
				parts := strings.Split(authHeader, " ")
				if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
					tokenStr = parts[1]
				}
			}
			if tokenStr == "" {
				tokenStr = r.URL.Query().Get("token")
			}

			if tokenStr == "" {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"unauthorized: authentication token required"}`))
				return
			}

			// 3. Cryptographic Token Validation
			claims, err := tokenService.ValidateToken(tokenStr)
			if err != nil {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"unauthorized: invalid or expired session token"}`))
				return
			}

			// 4. Verify user exists in the SQLite registry
			dbUser, err := repo.GetUserByID(claims.UserID)
			if err != nil || dbUser == nil {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"unauthorized: user account is deactivated or deleted"}`))
				return
			}

			// 5. RBAC Authorization Evaluation against EnterpriseRBACPolicy
			rule := FindMatchingRule(r.Method, path)
			if rule == nil {
				// Fail-closed default: any unmapped route under /api/v1 is forbidden
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"error":"forbidden: no RBAC authorization rule defined for endpoint"}`))
				return
			}

			// Check role permissions: Admin has universal access
			authorized := false
			if dbUser.Role == models.RoleAdmin {
				authorized = true
			} else {
				for _, allowedRole := range rule.AllowedRoles {
					if strings.EqualFold(allowedRole, dbUser.Role) {
						authorized = true
						break
					}
				}
			}

			if !authorized {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"error":"forbidden: role '` + dbUser.Role + `' is not authorized to access this resource"}`))
				return
			}

			// 6. Context Injection
			ctx := context.WithValue(r.Context(), userContextKey, claims)
			ctx = context.WithValue(ctx, userModelContextKey, dbUser)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
