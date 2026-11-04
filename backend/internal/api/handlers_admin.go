package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/istiyak/secure-exam-eval/internal/auth"
)

// ---------- sessions ----------

func (s *Server) handleListSessions(w http.ResponseWriter, r *http.Request) {
	out, err := s.DB.ListSessions(r.Context())
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sessions": out})
}

type sessionReq struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

func (s *Server) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	var req sessionReq
	if !decode(w, r, &req) {
		return
	}
	req.Name, req.Code = strings.TrimSpace(req.Name), strings.ToUpper(strings.TrimSpace(req.Code))
	if req.Name == "" || req.Code == "" {
		writeErr(w, http.StatusUnprocessableEntity, "name and code are required")
		return
	}
	m, err := s.DB.CreateSession(r.Context(), req.Name, req.Code)
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	s.audit(r, "SESSION_CREATED", "session", m.ID, map[string]any{"name": m.Name})
	writeJSON(w, http.StatusCreated, m)
}

type activeReq struct {
	IsActive *bool `json:"is_active"`
}

func (s *Server) handlePatchSession(w http.ResponseWriter, r *http.Request) {
	var req activeReq
	if !decode(w, r, &req) {
		return
	}
	if req.IsActive == nil {
		writeErr(w, http.StatusUnprocessableEntity, "is_active is required")
		return
	}
	if err := s.DB.SetSessionActive(r.Context(), r.PathValue("id"), *req.IsActive); err != nil {
		mapStoreErr(w, err)
		return
	}
	s.audit(r, "SESSION_UPDATED", "session", r.PathValue("id"), map[string]any{"active": *req.IsActive})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ---------- students ----------

func (s *Server) handleListStudents(w http.ResponseWriter, r *http.Request) {
	out, err := s.DB.ListStudents(r.Context(), r.URL.Query().Get("q"), r.URL.Query().Get("session_id"))
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"students": out})
}

type studentReq struct {
	StudentNo      string `json:"student_no"`
	RegistrationNo string `json:"registration_no"`
	FullName       string `json:"full_name"`
	Email          string `json:"email"`
	SessionID      string `json:"session_id"`
}

func (s *Server) handleCreateStudent(w http.ResponseWriter, r *http.Request) {
	var req studentReq
	if !decode(w, r, &req) {
		return
	}
	if req.StudentNo == "" || req.FullName == "" || req.SessionID == "" {
		writeErr(w, http.StatusUnprocessableEntity, "student_no, full_name and session_id are required")
		return
	}
	var email *string
	if e := strings.TrimSpace(req.Email); e != "" {
		email = &e
	}
	m, err := s.DB.CreateStudent(r.Context(), strings.TrimSpace(req.StudentNo), strings.TrimSpace(req.RegistrationNo), strings.TrimSpace(req.FullName), email, req.SessionID)
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	s.audit(r, "STUDENT_CREATED", "student", m.ID, map[string]any{"student_no": m.StudentNo})
	writeJSON(w, http.StatusCreated, m)
}

type linkReq struct {
	UserID string `json:"user_id"`
}

func (s *Server) handleLinkStudentUser(w http.ResponseWriter, r *http.Request) {
	var req linkReq
	if !decode(w, r, &req) {
		return
	}
	if err := s.DB.LinkStudentUser(r.Context(), r.PathValue("id"), req.UserID); err != nil {
		mapStoreErr(w, err)
		return
	}
	s.audit(r, "STUDENT_LINKED", "student", r.PathValue("id"), map[string]any{"user_id": req.UserID})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ---------- courses ----------

func (s *Server) handleListCourses(w http.ResponseWriter, r *http.Request) {
	out, err := s.DB.ListCourses(r.Context(), r.URL.Query().Get("session_id"))
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"courses": out})
}

type courseReq struct {
	Code         string   `json:"code"`
	Title        string   `json:"title"`
	SessionID    string   `json:"session_id"`
	MaxMarks     *float64 `json:"max_marks"`
	PassMarks    *float64 `json:"pass_marks"`
	InstructorID string   `json:"instructor_id"`
}

func (s *Server) handleCreateCourse(w http.ResponseWriter, r *http.Request) {
	var req courseReq
	if !decode(w, r, &req) {
		return
	}
	if req.Code == "" || req.Title == "" || req.SessionID == "" {
		writeErr(w, http.StatusUnprocessableEntity, "code, title and session_id are required")
		return
	}
	max, pass := 100.0, 40.0
	if req.MaxMarks != nil {
		max = *req.MaxMarks
	}
	if req.PassMarks != nil {
		pass = *req.PassMarks
	}
	if max <= 0 || pass < 0 || pass > max {
		writeErr(w, http.StatusUnprocessableEntity, "max_marks/pass_marks out of range")
		return
	}
	var instr *string
	if req.InstructorID != "" {
		instr = &req.InstructorID
	}
	m, err := s.DB.CreateCourse(r.Context(), strings.ToUpper(strings.TrimSpace(req.Code)), strings.TrimSpace(req.Title),
		req.SessionID, max, pass, instr)
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	s.audit(r, "COURSE_CREATED", "course", m.ID, map[string]any{"code": m.Code})
	writeJSON(w, http.StatusCreated, m)
}

// ---------- users ----------

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	out, err := s.DB.ListUsers(r.Context(), r.URL.Query().Get("role"), r.URL.Query().Get("q"))
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"users": out})
}

type createUserReq struct {
	Email    string `json:"email"`
	FullName string `json:"full_name"`
	Role     string `json:"role"`
	Password string `json:"password"`
}

func validRole(r string) bool {
	switch auth.Role(r) {
	case auth.RoleAdmin, auth.RoleEvaluator, auth.RoleStudent:
		return true
	}
	return false
}

func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req createUserReq
	if !decode(w, r, &req) {
		return
	}
	req.Role = strings.ToUpper(strings.TrimSpace(req.Role))
	if !validRole(req.Role) {
		writeErr(w, http.StatusUnprocessableEntity, "role must be ADMIN, EVALUATOR or STUDENT")
		return
	}
	if !strings.Contains(req.Email, "@") || req.FullName == "" {
		writeErr(w, http.StatusUnprocessableEntity, "valid email and full_name are required")
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeErr(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	u, err := s.DB.CreateUser(r.Context(), req.Email, hash, req.FullName, req.Role)
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	s.audit(r, "USER_CREATED", "user", u.ID, map[string]any{"role": u.Role})
	writeJSON(w, http.StatusCreated, u)
}

type patchUserReq struct {
	IsActive *bool   `json:"is_active"`
	Password *string `json:"password"`
}

func (s *Server) handlePatchUser(w http.ResponseWriter, r *http.Request) {
	var req patchUserReq
	if !decode(w, r, &req) {
		return
	}
	id := r.PathValue("id")
	if req.IsActive != nil {
		if err := s.DB.SetUserActive(r.Context(), id, *req.IsActive); err != nil {
			mapStoreErr(w, err)
			return
		}
		s.audit(r, "USER_ACTIVE_SET", "user", id, map[string]any{"active": *req.IsActive})
	}
	if req.Password != nil {
		hash, err := auth.HashPassword(*req.Password)
		if err != nil {
			writeErr(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		if err := s.DB.SetPassword(r.Context(), id, hash); err != nil {
			mapStoreErr(w, err)
			return
		}
		s.audit(r, "USER_PASSWORD_RESET", "user", id, nil)
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

var _ = strconv.Itoa
