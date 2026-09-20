package middleware

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/mailvex/mailvex/core/crypto"
	"github.com/mailvex/mailvex/core/domain"
	"github.com/mailvex/mailvex/core/repository"
)

type contextKey string

const (
	RequestIDKey   contextKey = "request_id"
	UserKey        contextKey = "user"
	TenantIDKey    contextKey = "tenant_id"
	APIKeyKey      contextKey = "api_key"
	EnvironmentKey contextKey = "environment"
)

// RequestID adds a unique req_... identifier to each request context and response header
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = crypto.NewULID(crypto.PrefixRequest)
		}
		w.Header().Set("X-Request-ID", reqID)
		ctx := context.WithValue(r.Context(), RequestIDKey, reqID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// SecurityHeaders applies production HTTP security headers
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'")
		next.ServeHTTP(w, r)
	})
}

// Logger logs each incoming HTTP request with timing and status
func Logger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			wrapped := &responseWriterWrapper{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(wrapped, r)

			reqID, _ := r.Context().Value(RequestIDKey).(string)
			logger.Info("http_request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", wrapped.statusCode,
				"duration_ms", time.Since(start).Milliseconds(),
				"request_id", reqID,
				"remote_addr", r.RemoteAddr,
			)
		})
	}
}

type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriterWrapper) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// RequireAuth authenticates requests via either API Key Bearer token or User Session
func RequireAuth(repo repository.Repository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")

			// Check for API Key Bearer token (mvx_live_... or mvx_test_...)
			if strings.HasPrefix(authHeader, "Bearer ") {
				token := strings.TrimPrefix(authHeader, "Bearer ")
				if strings.HasPrefix(token, "mvx_live_") || strings.HasPrefix(token, "mvx_test_") {
					hash := crypto.HashAPIKey(token)
					apiKey, err := repo.GetAPIKeyByHash(r.Context(), hash)
					if err != nil {
						http.Error(w, `{"error":{"code":"unauthorized","message":"Invalid, expired, or revoked API key"}}`, http.StatusUnauthorized)
						return
					}

					// Verify client IP if allowed_cidrs configured
					if len(apiKey.AllowedCIDRs) > 0 {
						clientHost, _, _ := net.SplitHostPort(r.RemoteAddr)
						if clientHost == "" {
							clientHost = r.RemoteAddr
						}
						clientIP := net.ParseIP(clientHost)
						allowed := false
						for _, cidr := range apiKey.AllowedCIDRs {
							_, ipNet, err := net.ParseCIDR(cidr)
							if err == nil && clientIP != nil && ipNet.Contains(clientIP) {
								allowed = true
								break
							}
						}
						if !allowed {
							http.Error(w, `{"error":{"code":"forbidden","message":"Client IP not allowed for this API key"}}`, http.StatusForbidden)
							return
						}
					}

					ctx := context.WithValue(r.Context(), APIKeyKey, apiKey)
					ctx = context.WithValue(ctx, TenantIDKey, apiKey.TenantID)
					ctx = context.WithValue(ctx, EnvironmentKey, apiKey.Environment)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

			// Fallback to User Session authentication
			userID := r.Header.Get("X-User-ID")
			if userID == "" {
				http.Error(w, `{"error":{"code":"unauthorized","message":"Authentication required"}}`, http.StatusUnauthorized)
				return
			}

			user, err := repo.GetUserByID(r.Context(), userID)
			if err != nil {
				http.Error(w, `{"error":{"code":"unauthorized","message":"Invalid user session"}}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), UserKey, user)

			// Tenant context resolution
			tenantID := r.Header.Get("X-Tenant-ID")
			if tenantID != "" {
				if !user.IsPlatformAdmin {
					_, err := repo.GetTenantMembership(r.Context(), tenantID, user.ID)
					if err != nil {
						http.Error(w, `{"error":{"code":"cross_tenant_forbidden","message":"Not a member of this tenant"}}`, http.StatusForbidden)
						return
					}
				}
				ctx = context.WithValue(ctx, TenantIDKey, tenantID)
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequirePermission verifies the caller has the required permission / scope
func RequirePermission(repo repository.Repository, permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. If API Key authenticated
			if apiKey, ok := r.Context().Value(APIKeyKey).(*domain.APIKey); ok && apiKey != nil {
				hasScope := false
				for _, s := range apiKey.Scopes {
					if s == "*" || s == permission || (permission == "messages.send" && s == "send.email") {
						hasScope = true
						break
					}
				}
				if !hasScope {
					http.Error(w, `{"error":{"code":"forbidden","message":"API key lacks required scope"}}`, http.StatusForbidden)
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			// 2. If User authenticated
			user, ok := r.Context().Value(UserKey).(*domain.User)
			if !ok || user == nil {
				http.Error(w, `{"error":{"code":"unauthorized","message":"Authentication required"}}`, http.StatusUnauthorized)
				return
			}

			if user.IsPlatformAdmin {
				next.ServeHTTP(w, r)
				return
			}

			tenantID, _ := r.Context().Value(TenantIDKey).(string)
			if tenantID == "" {
				http.Error(w, `{"error":{"code":"tenant_required","message":"Active tenant required for this operation"}}`, http.StatusBadRequest)
				return
			}

			membership, err := repo.GetTenantMembership(r.Context(), tenantID, user.ID)
			if err != nil {
				http.Error(w, `{"error":{"code":"forbidden","message":"No role in active tenant"}}`, http.StatusForbidden)
				return
			}

			perms, err := repo.GetRolePermissions(r.Context(), membership.RoleID)
			if err != nil {
				http.Error(w, `{"error":{"code":"forbidden","message":"Could not verify permissions"}}`, http.StatusForbidden)
				return
			}

			hasPerm := false
			for _, p := range perms {
				if p == permission {
					hasPerm = true
					break
				}
			}

			if !hasPerm {
				http.Error(w, `{"error":{"code":"forbidden","message":"Insufficient permissions"}}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
