// Package api wires HTTP routes to the store.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/istiyak/secure-exam-eval/internal/auth"
	"github.com/istiyak/secure-exam-eval/internal/config"
	"github.com/istiyak/secure-exam-eval/internal/models"
	"github.com/istiyak/secure-exam-eval/internal/storage"
	"github.com/istiyak/secure-exam-eval/internal/store"
)

// Server carries dependencies for all handlers.
type Server struct {
	Cfg     *config.Config
	DB      *store.Store
	Tokens  *auth.Manager
	Files   storage.Store
	Log     *slog.Logger
	Limiter *loginLimiter
}

type ctxKey string

const (
	ctxUser   ctxKey = "user"
	ctxReqID  ctxKey = "reqid"
	ctxClaims ctxKey = "claims"
)

// Routes builds the mux.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("GET /api/meta", s.handleMeta)

	mux.HandleFunc("POST /api/auth/login", s.handleLogin)
	mux.HandleFunc("POST /api/auth/refresh", s.handleRefresh)
	mux.HandleFunc("POST /api/auth/logout", s.handleLogout)
	mux.Handle("GET /api/auth/me", s.authed(s.handleMe, ""))
	mux.Handle("POST /api/auth/password", s.authed(s.handleChangePassword, ""))

	// Staff: hall management team.
	mux.Handle("GET /api/sessions", s.authed(s.handleListSessions, auth.RoleAdmin, auth.RoleHallManager))
	mux.Handle("POST /api/sessions", s.authed(s.handleCreateSession, auth.RoleAdmin))
	mux.Handle("PATCH /api/sessions/{id}", s.authed(s.handlePatchSession, auth.RoleAdmin))

	mux.Handle("GET /api/students", s.authed(s.handleListStudents, auth.RoleAdmin, auth.RoleHallManager))
	mux.Handle("POST /api/students", s.authed(s.handleCreateStudent, auth.RoleAdmin, auth.RoleHallManager))
	mux.Handle("POST /api/students/{id}/link-user", s.authed(s.handleLinkStudentUser, auth.RoleAdmin))

	mux.Handle("GET /api/courses", s.authed(s.handleListCourses, auth.RoleAdmin, auth.RoleHallManager, auth.RoleEvaluator))
	mux.Handle("POST /api/courses", s.authed(s.handleCreateCourse, auth.RoleAdmin))

	mux.Handle("GET /api/users", s.authed(s.handleListUsers, auth.RoleAdmin, auth.RoleHallManager))
	mux.Handle("POST /api/users", s.authed(s.handleCreateUser, auth.RoleAdmin))
	mux.Handle("PATCH /api/users/{id}", s.authed(s.handlePatchUser, auth.RoleAdmin))

	// Script registration (hall management team only).
	mux.Handle("GET /api/scripts", s.authed(s.handleListScripts, auth.RoleAdmin, auth.RoleHallManager))
	mux.Handle("POST /api/scripts", s.authed(s.handleRegisterScript, auth.RoleAdmin, auth.RoleHallManager))
	mux.Handle("GET /api/scripts/{id}", s.authed(s.handleGetScript, auth.RoleAdmin, auth.RoleHallManager, auth.RoleEvaluator))
	mux.Handle("POST /api/scripts/{id}/assign", s.authed(s.handleAssignPair, auth.RoleAdmin, auth.RoleHallManager))
	mux.Handle("POST /api/scripts/{id}/assign-third", s.authed(s.handleAssignThird, auth.RoleAdmin, auth.RoleHallManager))
	mux.Handle("POST /api/scripts/{id}/pdf-token", s.authed(s.handlePDFToken, ""))
	mux.HandleFunc("GET /api/scripts/{id}/pdf", s.handlePDF)

	// Marks.
	mux.Handle("POST /api/scripts/{id}/marks", s.authed(s.handleSubmitMark, auth.RoleEvaluator, auth.RoleAdmin))
	mux.Handle("GET /api/queue", s.authed(s.handleQueue, auth.RoleEvaluator))
	mux.Handle("POST /api/scripts/{id}/override", s.authed(s.handleOverride, auth.RoleAdmin))

	// Retrieval.
	mux.Handle("GET /api/results", s.authed(s.handleSearch, auth.RoleAdmin, auth.RoleHallManager, auth.RoleEvaluator))
	mux.Handle("GET /api/results/mine", s.authed(s.handleMyResults, auth.RoleStudent))

	// Challenges.
	mux.Handle("GET /api/challenges", s.authed(s.handleListChallenges, auth.RoleAdmin, auth.RoleHallManager))
	mux.Handle("POST /api/challenges", s.authed(s.handleCreateChallenge, auth.RoleStudent))
	mux.Handle("PATCH /api/challenges/{id}", s.authed(s.handleDecideChallenge, auth.RoleAdmin))

	mux.Handle("GET /api/admin/stats", s.authed(s.handleStats, auth.RoleAdmin))
	mux.Handle("GET /api/admin/audit", s.authed(s.handleAudit, auth.RoleAdmin))
	mux.Handle("GET /api/export/results.csv", s.authed(s.handleExportCSV, auth.RoleAdmin, auth.RoleHallManager))

	return s.chain(mux)
}

func (s *Server) chain(h http.Handler) http.Handler {
	return s.recoverer(s.requestID(s.cors(s.securityHeaders(s.logRequests(h)))))
}

// ---------- context helpers ----------

func userFrom(ctx context.Context) *models.User {
	u, _ := ctx.Value(ctxUser).(*models.User)
	return u
}

func claimsFrom(ctx context.Context) *auth.Claims {
	c, _ := ctx.Value(ctxClaims).(*auth.Claims)
	return c
}

// ---------- responses ----------

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

type errBody struct {
	Error string `json:"error"`
	Field string `json:"field,omitempty"`
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errBody{Error: msg})
}

// mapStoreErr turns store sentinels into statuses without leaking SQL detail.
func mapStoreErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeErr(w, http.StatusNotFound, "not found")
	case errors.Is(err, store.ErrForbidden):
		writeErr(w, http.StatusForbidden, "not assigned to this script")
	case errors.Is(err, store.ErrConflict):
		writeErr(w, http.StatusConflict, "conflict: already exists")
	case errors.Is(err, store.ErrState):
		writeErr(w, http.StatusConflict, err.Error())
	case errors.Is(err, store.ErrInvalid):
		writeErr(w, http.StatusUnprocessableEntity, err.Error())
	default:
		writeErr(w, http.StatusInternalServerError, "internal error")
	}
}

func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		writeErr(w, http.StatusBadRequest, "malformed JSON body: "+err.Error())
		return false
	}
	return true
}

func intParam(r *http.Request, name string, def int) int {
	if v := r.URL.Query().Get(name); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func nowPtr() *time.Time { t := time.Now(); return &t }
