package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/istiyak/exam-evaluation/internal/auth"
	"github.com/istiyak/exam-evaluation/internal/models"
)

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type tokenResp struct {
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
	ExpiresIn    int         `json:"expires_in"`
	User         models.User `json:"user"`
}

func (s *Server) issue(ctx context.Context, w http.ResponseWriter, u *models.User) {
	access, err := s.Tokens.Access(u.ID, auth.Role(u.Role))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "could not issue token")
		return
	}
	refresh, hash, exp, err := s.Tokens.Refresh(u.ID, auth.Role(u.Role))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "could not issue token")
		return
	}
	if err := s.DB.SaveRefresh(ctx, u.ID, hash, exp); err != nil {
		writeErr(w, http.StatusInternalServerError, "could not persist session")
		return
	}
	writeJSON(w, http.StatusOK, tokenResp{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    int(s.Cfg.AccessTTL.Seconds()),
		User:         *u,
	})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if !decode(w, r, &req) {
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	key := req.Email + "|" + clientIP(r)
	if !s.Limiter.allow(key) {
		writeErr(w, http.StatusTooManyRequests, "too many attempts, wait a minute")
		return
	}
	row, err := s.DB.UserByEmail(r.Context(), req.Email)
	if err != nil || !auth.CheckPassword(row.PasswordHash, req.Password) || !row.IsActive {
		// Same message either way so accounts cannot be enumerated.
		writeErr(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	s.DB.Audit(r.Context(), row.ID, row.Role, "LOGIN", "user", row.ID, nil)
	s.issue(r.Context(), w, &row.User)
}

type refreshReq struct {
	RefreshToken string `json:"refresh_token"`
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	var req refreshReq
	if !decode(w, r, &req) {
		return
	}
	u, err := s.DB.RotateRefresh(r.Context(), auth.HashToken(req.RefreshToken))
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "session expired, sign in again")
		return
	}
	s.issue(r.Context(), w, u)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	var req refreshReq
	if err := decodeQuiet(r, &req); err == nil && req.RefreshToken != "" {
		s.DB.RevokeRefresh(r.Context(), auth.HashToken(req.RefreshToken))
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "signed out"})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r.Context())
	out := map[string]any{"user": u}
	if u.Role == string(auth.RoleStudent) {
		if st, err := s.mustStudentUser(r); err == nil {
			out["student"] = st
		}
	}
	writeJSON(w, http.StatusOK, out)
}

type changePasswordReq struct {
	Current string `json:"current_password"`
	New     string `json:"new_password"`
}

func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r.Context())
	var req changePasswordReq
	if !decode(w, r, &req) {
		return
	}
	row, err := s.DB.UserByEmail(r.Context(), u.Email)
	if err != nil || !auth.CheckPassword(row.PasswordHash, req.Current) {
		writeErr(w, http.StatusUnauthorized, "current password incorrect")
		return
	}
	hash, err := auth.HashPassword(req.New)
	if err != nil {
		writeErr(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if err := s.DB.SetPassword(r.Context(), u.ID, hash); err != nil {
		mapStoreErr(w, err)
		return
	}
	s.audit(r, "PASSWORD_CHANGED", "user", u.ID, nil)
	writeJSON(w, http.StatusOK, map[string]string{"status": "password updated"})
}
