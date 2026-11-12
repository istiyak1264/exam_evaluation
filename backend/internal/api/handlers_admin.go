package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/istiyak/secure-exam-eval/internal/auth"
	"github.com/istiyak/secure-exam-eval/internal/models"
	"github.com/istiyak/secure-exam-eval/internal/store"
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
}

func (s *Server) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	var req sessionReq
	if !decode(w, r, &req) {
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeErr(w, http.StatusUnprocessableEntity, "name is required")
		return
	}
	m, err := s.DB.CreateSession(r.Context(), req.Name)
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	s.audit(r, "SESSION_CREATED", "session", m.ID, map[string]any{"name": m.Name})
	writeJSON(w, http.StatusCreated, m)
}

func (s *Server) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.DB.DeleteSession(r.Context(), id); err != nil {
		mapStoreErr(w, err)
		return
	}
	s.audit(r, "SESSION_DELETED", "session", id, nil)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
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

func (s *Server) handleDeleteStudent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.DB.DeleteStudent(r.Context(), id); err != nil {
		mapStoreErr(w, err)
		return
	}
	s.audit(r, "STUDENT_DELETED", "student", id, nil)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ---------- course catalogue ----------

func (s *Server) handleListCourseCatalog(w http.ResponseWriter, r *http.Request) {
	out, err := s.DB.ListCourseCatalog(r.Context())
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"courses": out})
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
	CatalogID    string `json:"catalog_id"`
	SessionID    string `json:"session_id"`
	InstructorID string `json:"instructor_id"`
}

func (s *Server) handleCreateCourse(w http.ResponseWriter, r *http.Request) {
	var req courseReq
	if !decode(w, r, &req) {
		return
	}
	if req.CatalogID == "" || req.SessionID == "" {
		writeErr(w, http.StatusUnprocessableEntity, "catalog_id and session_id are required")
		return
	}
	var instr *string
	if req.InstructorID != "" {
		instr = &req.InstructorID
	}
	m, err := s.DB.CreateCourse(r.Context(), req.CatalogID, req.SessionID, instr)
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	s.audit(r, "COURSE_CREATED", "course", m.ID, map[string]any{"code": m.Code})
	writeJSON(w, http.StatusCreated, m)
}

func (s *Server) handleDeleteCourse(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.DB.DeleteCourse(r.Context(), id); err != nil {
		mapStoreErr(w, err)
		return
	}
	s.audit(r, "COURSE_DELETED", "course", id, nil)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
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
	Email     string `json:"email"`
	FullName  string `json:"full_name"`
	Role      string `json:"role"`
	Password  string `json:"password"`
	StudentNo string `json:"student_no"`
	RegNo     string `json:"registration_no"`
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
	// A student account is created against a candidate record; the office
	// supplies the identifiers it issued.
	var candidate *models.Student
	if req.Role == string(auth.RoleStudent) && (req.StudentNo != "" || req.RegNo != "") {
		st, lerr := s.DB.UnclaimedStudent(r.Context(), req.StudentNo, req.RegNo)
		if lerr != nil {
			if errors.Is(lerr, store.ErrConflict) {
				writeErr(w, http.StatusConflict, "that student record already has an account")
				return
			}
			writeErr(w, http.StatusUnprocessableEntity,
				"no student record matches that student ID and registration number")
			return
		}
		candidate = st
	}

	u, err := s.DB.CreateUser(r.Context(), req.Email, hash, req.FullName, req.Role)
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	if candidate != nil {
		if ok, cerr := s.DB.ClaimStudent(r.Context(), candidate.ID, u.ID); cerr == nil && !ok {
			_ = s.DB.SetUserActive(r.Context(), u.ID, false)
			writeErr(w, http.StatusConflict, "that student record was claimed in the meantime")
			return
		}
	}
	s.audit(r, "USER_CREATED", "user", u.ID, map[string]any{"role": u.Role, "linked": candidate != nil})
	writeJSON(w, http.StatusCreated, u)
}

type patchUserReq struct {
	IsActive *bool   `json:"is_active"`
	Password *string `json:"password"`
	Role     *string `json:"role"`
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
	if req.Role != nil {
		role := strings.ToUpper(strings.TrimSpace(*req.Role))
		if !validRole(role) {
			writeErr(w, http.StatusUnprocessableEntity, "role must be ADMIN, EVALUATOR or STUDENT")
			return
		}
		cur, err := s.DB.UserByID(r.Context(), id)
		if err != nil {
			mapStoreErr(w, err)
			return
		}
		if cur.Role == role {
			// no-op
		} else if cur.Role == string(auth.RoleAdmin) {
			n, err := s.DB.CountActiveAdmins(r.Context(), id)
			if err != nil {
				mapStoreErr(w, err)
				return
			}
			if n == 0 {
				writeErr(w, http.StatusUnprocessableEntity, "this is the last administrator — promote another first")
				return
			}
			if err := s.DB.SetUserRole(r.Context(), id, role); err != nil {
				mapStoreErr(w, err)
				return
			}
			s.audit(r, "USER_ROLE_SET", "user", id, map[string]any{"role": role})
		} else {
			if err := s.DB.SetUserRole(r.Context(), id, role); err != nil {
				mapStoreErr(w, err)
				return
			}
			s.audit(r, "USER_ROLE_SET", "user", id, map[string]any{"role": role})
		}
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

func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if me := userFrom(r.Context()); me != nil && me.ID == id {
		writeErr(w, http.StatusUnprocessableEntity, "you cannot delete the account you are signed in with")
		return
	}
	u, err := s.DB.UserByID(r.Context(), id)
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	if u.Role == string(auth.RoleAdmin) {
		n, err := s.DB.CountActiveAdmins(r.Context(), id)
		if err != nil {
			mapStoreErr(w, err)
			return
		}
		if n == 0 {
			writeErr(w, http.StatusUnprocessableEntity, "this is the last administrator")
			return
		}
	}
	if err := s.DB.DeleteUser(r.Context(), id); err != nil {
		if errors.Is(err, store.ErrInUse) {
			writeErr(w, http.StatusConflict,
				"this account has marking history, so it can be disabled but not deleted")
			return
		}
		mapStoreErr(w, err)
		return
	}
	s.audit(r, "USER_DELETED", "user", id, map[string]any{"email": u.Email, "role": u.Role})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
