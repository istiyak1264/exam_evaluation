package api

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/istiyak/secure-exam-eval/internal/auth"
	"github.com/istiyak/secure-exam-eval/internal/models"
	"github.com/istiyak/secure-exam-eval/internal/store"
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

type registerReq struct {
	FullName  string `json:"full_name"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	Role      string `json:"role"`
	StudentNo string `json:"student_no"`
}

// handleRegister is the public sign-up. Only candidates and examiners may
// self-register; administrators are created by an administrator.
func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerReq
	if !decode(w, r, &req) {
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.FullName = strings.TrimSpace(req.FullName)
	req.Role = strings.ToUpper(strings.TrimSpace(req.Role))
	req.StudentNo = strings.TrimSpace(req.StudentNo)

	if req.Role != string(auth.RoleStudent) && req.Role != string(auth.RoleEvaluator) {
		writeErr(w, http.StatusUnprocessableEntity, "role must be STUDENT or EVALUATOR")
		return
	}
	if !strings.Contains(req.Email, "@") || len(req.FullName) < 3 {
		writeErr(w, http.StatusUnprocessableEntity, "a full name and valid email are required")
		return
	}
	key := "signup|" + clientIP(r)
	if !s.Limiter.allow(key) {
		writeErr(w, http.StatusTooManyRequests, "too many attempts, wait a minute")
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeErr(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	u, err := s.DB.CreateUser(r.Context(), req.Email, hash, req.FullName, req.Role)
	if err != nil {
		if errors.Is(err, store.ErrConflict) {
			writeErr(w, http.StatusConflict, "an account with that email already exists")
			return
		}
		mapStoreErr(w, err)
		return
	}

	// A candidate who gives their matriculation number is linked to the
	// record the office already holds, so their scripts appear immediately.
	linked := false
	if req.Role == string(auth.RoleStudent) && req.StudentNo != "" {
		if st, err := s.DB.UnclaimedStudent(r.Context(), req.StudentNo); err == nil {
			if ok, err := s.DB.ClaimStudent(r.Context(), st.ID, u.ID); err == nil && ok {
				linked = true
			}
		}
	}
	s.DB.Audit(r.Context(), u.ID, u.Role, "SIGNED_UP", "user", u.ID, map[string]any{"linked_student": linked})
	s.issue(r.Context(), w, u)
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
