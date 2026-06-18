package httpapi

import (
	"context"
	"net/http"
	"strings"
	"time"
)

// requireAuth authenticates via the Authorization: Bearer header and places the
// authUser in the request context. A missing/invalid/expired token yields 401.
func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		if token == "" {
			writeError(w, http.StatusUnauthorized, codeUnauthorized, "authentication required")
			return
		}
		s.withClaims(w, r, next, token)
	})
}

// requireAuthQuery is requireAuth for the one SSE endpoint, which also accepts
// the token in the ?token= query string because EventSource cannot set headers.
func (s *Server) requireAuthQuery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		if token == "" {
			token = r.URL.Query().Get("token")
		}
		if token == "" {
			writeError(w, http.StatusUnauthorized, codeUnauthorized, "authentication required")
			return
		}
		s.withClaims(w, r, next, token)
	})
}

// requireRole authenticates and then enforces that the user holds one of the
// allowed roles, returning 403 otherwise.
func (s *Server) requireRole(next http.Handler, roles ...string) http.Handler {
	return s.requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, _ := userFrom(r.Context())
		for _, role := range roles {
			if u.Role == role {
				next.ServeHTTP(w, r)
				return
			}
		}
		writeError(w, http.StatusForbidden, codeForbidden, "insufficient permissions")
	}))
}

// withClaims validates a token and, on success, calls next with the user in
// context.
func (s *Server) withClaims(w http.ResponseWriter, r *http.Request, next http.Handler, token string) {
	claims, err := s.tokens.ParseAccessToken(token)
	if err != nil {
		writeError(w, http.StatusUnauthorized, codeUnauthorized, "invalid or expired token")
		return
	}
	ctx := context.WithValue(r.Context(), userCtxKey, authUser{ID: claims.UID, Role: claims.Role})
	next.ServeHTTP(w, r.WithContext(ctx))
}

// bearerToken extracts the token from an "Authorization: Bearer <token>"
// header, or "" if absent/malformed.
func bearerToken(r *http.Request) string {
	const prefix = "Bearer "
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, prefix) {
		return strings.TrimSpace(h[len(prefix):])
	}
	return ""
}

// rateLimit applies the per-IP limiter to the credential endpoints.
func (s *Server) rateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.limiter.allow(clientIP(r)) {
			writeError(w, http.StatusTooManyRequests, codeRateLimited, "too many attempts, please wait a moment")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// cors allows the single configured frontend origin and the methods/headers the
// API uses. Credentials are allowed so the refresh cookie can travel.
func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && origin == s.cfg.FrontendOrigin {
			h := w.Header()
			h.Set("Access-Control-Allow-Origin", origin)
			h.Set("Vary", "Origin")
			h.Set("Access-Control-Allow-Credentials", "true")
			h.Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
			h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// statusRecorder captures the response status for logging.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (sr *statusRecorder) WriteHeader(code int) {
	sr.status = code
	sr.ResponseWriter.WriteHeader(code)
}

// Flush forwards to the underlying writer so SSE streaming keeps working
// through the logging wrapper.
func (sr *statusRecorder) Flush() {
	if f, ok := sr.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// logRequests logs one structured line per request with method, path, status,
// and duration.
func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		s.logger.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

// recoverPanic turns a handler panic into a 500 instead of crashing the server,
// logging the failure with the request path.
func (s *Server) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				s.logger.Error("panic recovered", "path", r.URL.Path, "panic", rec)
				writeError(w, http.StatusInternalServerError, codeInternal, "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// clientIP returns the best-effort client IP for rate limiting: the first
// X-Forwarded-For entry if present (we sit behind a proxy in prod), else the
// remote address.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.IndexByte(xff, ','); i >= 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	host := r.RemoteAddr
	if i := strings.LastIndexByte(host, ':'); i >= 0 {
		return host[:i]
	}
	return host
}
