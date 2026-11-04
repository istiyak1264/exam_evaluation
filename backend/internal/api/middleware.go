package api

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/istiyak/secure-exam-eval/internal/auth"
	"github.com/istiyak/secure-exam-eval/internal/models"
	"github.com/istiyak/secure-exam-eval/internal/store"
)

// authed wraps a handler with authentication and optional role checks.
// An empty role list means "any signed-in user".
func (s *Server) authed(h http.HandlerFunc, roles ...auth.Role) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := bearer(r)
		if raw == "" {
			writeErr(w, http.StatusUnauthorized, "missing bearer token")
			return
		}
		claims, err := s.Tokens.Parse(raw, auth.PurposeAccess)
		if err != nil {
			writeErr(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}
		u, err := s.DB.UserByID(r.Context(), claims.UserID)
		if err != nil || !u.IsActive {
			writeErr(w, http.StatusUnauthorized, "account inactive")
			return
		}
		allowed := roles[:0:0]
		for _, role := range roles {
			if role != "" {
				allowed = append(allowed, role)
			}
		}
		if len(allowed) > 0 {
			ok := false
			for _, role := range allowed {
				if string(role) == u.Role {
					ok = true
					break
				}
			}
			if !ok {
				writeErr(w, http.StatusForbidden, "insufficient role")
				return
			}
		}
		ctx := r.Context()
		ctx = contextWith(ctx, ctxUser, u)
		ctx = contextWith(ctx, ctxClaims, claims)
		h(w, r.WithContext(ctx))
	})
}

func bearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return ""
}

func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; frame-ancestors 'self'; object-src 'self'; img-src 'self' data:")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) cors(next http.Handler) http.Handler {
	allowed := map[string]bool{}
	for _, o := range s.Cfg.AllowedOrigins {
		allowed[o] = true
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && allowed[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				s.Log.Error("panic", "err", rec, "stack", string(debug.Stack()), "path", r.URL.Path)
				writeErr(w, http.StatusInternalServerError, "internal error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(c int) { w.status = c; w.ResponseWriter.WriteHeader(c) }

func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(sw, r)
		// PDF streams are long; keep the path only.
		s.Log.Info("http", "method", r.Method, "path", r.URL.Path, "status", sw.status,
			"ms", time.Since(start).Milliseconds(), "rid", r.Context().Value(ctxReqID))
	})
}

func (s *Server) requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-Id")
		if id == "" || len(id) > 64 {
			id = uuid.NewString()[:8]
		}
		w.Header().Set("X-Request-Id", id)
		next.ServeHTTP(w, r.WithContext(contextWith(r.Context(), ctxReqID, id)))
	})
}

// loginLimiter is a small in-memory token bucket per email+IP.
type loginLimiter struct {
	mu     sync.Mutex
	hits   map[string][]time.Time
	limit  int
	window time.Duration
}

// NewLimiter builds the login throttle.
func NewLimiter(limit int, window time.Duration) *loginLimiter {
	return &loginLimiter{hits: map[string][]time.Time{}, limit: limit, window: window}
}

func (l *loginLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	cut := time.Now().Add(-l.window)
	kept := l.hits[key][:0]
	for _, t := range l.hits[key] {
		if t.After(cut) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= l.limit {
		l.hits[key] = kept
		return false
	}
	l.hits[key] = append(kept, time.Now())
	return true
}

// audit is a thin wrapper so handlers stay readable.
func (s *Server) audit(r *http.Request, action, entity, entityID string, meta map[string]any) {
	u := userFrom(r.Context())
	s.DB.Audit(r.Context(), u.ID, u.Role, action, entity, entityID, meta)
}

func (s *Server) log() *slog.Logger { return s.Log }

// mustStudentUser resolves the student record behind a STUDENT account.
func (s *Server) mustStudentUser(r *http.Request) (*models.Student, error) {
	u := userFrom(r.Context())
	return s.DB.StudentByUserID(r.Context(), u.ID)
}

var errNoStudent = store.ErrNotFound
