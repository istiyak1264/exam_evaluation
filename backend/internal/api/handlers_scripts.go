package api

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/istiyak/exam-evaluation/internal/auth"
	"github.com/istiyak/exam-evaluation/internal/grading"
	"github.com/istiyak/exam-evaluation/internal/store"
)

func (s *Server) handleListScripts(w http.ResponseWriter, r *http.Request) {
	out, err := s.DB.ListScripts(r.Context(), r.URL.Query().Get("status"), r.URL.Query().Get("course_id"),
		r.URL.Query().Get("q"), intParam(r, "limit", 100))
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"scripts": out})
}

func (s *Server) handleGetScript(w http.ResponseWriter, r *http.Request) {
	sc, err := s.DB.ScriptByID(r.Context(), r.PathValue("id"))
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	u := userFrom(r.Context())
	if u.Role == string(auth.RoleEvaluator) {
		// Evaluators get the blinded projection: no student identity.
		slot, err := s.DB.EvaluatorSlot(r.Context(), sc.ID, u.ID)
		if err != nil {
			mapStoreErr(w, err)
			return
		}
		out := map[string]any{
			"id": sc.ID, "script_code": sc.Code, "status": sc.Status,
			"course_code": sc.CourseCode, "course_title": sc.CourseTitle, "max_marks": sc.MaxMarks,
			"page_count": sc.PageCount, "marks": sc.Marks, "final_marks": sc.FinalMarks, "my_slot": slot, "method": sc.Method,
		}
		writeJSON(w, http.StatusOK, out)
		return
	}
	writeJSON(w, http.StatusOK, sc)
}

// handleRegisterScript ingests an anonymised PDF and binds it to a student.
// multipart: file (PDF), course_id, student_id, optional script_code.
func (s *Server) handleRegisterScript(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, s.Cfg.MaxUploadBytes+(1<<20))
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		writeErr(w, http.StatusBadRequest, "multipart upload too large or malformed")
		return
	}
	courseID := strings.TrimSpace(r.FormValue("course_id"))
	studentID := strings.TrimSpace(r.FormValue("student_id"))
	if courseID == "" || studentID == "" {
		writeErr(w, http.StatusUnprocessableEntity, "course_id and student_id are required")
		return
	}
	course, err := s.DB.CourseByID(r.Context(), courseID)
	if err != nil {
		mapStoreErr(w, err)
		return
	}

	file, hdr, err := r.FormFile("file")
	if err != nil {
		writeErr(w, http.StatusUnprocessableEntity, "a PDF file is required in field 'file'")
		return
	}
	defer file.Close()

	head := make([]byte, 1024)
	n, _ := io.ReadFull(file, head)
	head = head[:n]
	if !bytes.HasPrefix(head, []byte("%PDF-")) {
		writeErr(w, http.StatusUnprocessableEntity, "file is not a PDF (missing %PDF- header)")
		return
	}
	body := io.MultiReader(bytes.NewReader(head), file)

	code := strings.TrimSpace(r.FormValue("script_code"))
	if code == "" {
		code, err = s.DB.NextScriptCode(r.Context(), course.Code)
		if err != nil {
			mapStoreErr(w, err)
			return
		}
	}

	key := path.Join("scripts", strings.ToLower(course.Code), code+".pdf")
	// Hash + page estimate on a tee so nothing is buffered twice.
	hasher := sha256.New()
	var counter countingWriter
	pages := &pageCounter{}
	mw := io.MultiWriter(hasher, &counter, pages)

	if err := s.Files.Put(r.Context(), key, io.TeeReader(body, mw), hdr.Size, "application/pdf"); err != nil {
		s.log().Error("storage put failed", "err", err)
		writeErr(w, http.StatusInternalServerError, "could not store the PDF")
		return
	}

	pc := pages.count()
	sc, err := s.DB.CreateScript(r.Context(), store.NewScript{
		Code: code, CourseID: courseID, StudentID: studentID, PDFKey: key,
		SHA256: hex.EncodeToString(hasher.Sum(nil)), Bytes: counter.n,
		Pages: &pc, ActorID: userFrom(r.Context()).ID,
	})
	if err != nil {
		_ = s.Files.Delete(r.Context(), key)
		if errors.Is(err, store.ErrConflict) {
			writeErr(w, http.StatusConflict, "that script code or (course, student) pair is already registered")
			return
		}
		mapStoreErr(w, err)
		return
	}
	s.audit(r, "SCRIPT_REGISTERED", "script", sc.ID, map[string]any{
		"code": sc.Code, "sha256": hex.EncodeToString(hasher.Sum(nil)), "bytes": counter.n,
	})
	writeJSON(w, http.StatusCreated, map[string]any{"script": sc, "script_code": sc.Code})
}

type countingWriter struct{ n int64 }

func (c *countingWriter) Write(p []byte) (int, error) { c.n += int64(len(p)); return len(p), nil }

// pageCounter estimates page count from "/Type /Page" tokens without a parser.
type pageCounter struct{ seen, pages int }

func (p *pageCounter) Write(b []byte) (int, error) {
	for _, tok := range bytes.Split(b, []byte("/Type")) {
		p.seen++
		if bytes.Contains(tok, []byte("/Page")) && !bytes.Contains(tok, []byte("/Pages")) {
			p.pages++
		}
	}
	return len(b), nil
}

func (p *pageCounter) count() int { return p.pages }

type assignReq struct {
	FirstEvaluatorID  string `json:"first_evaluator_id"`
	SecondEvaluatorID string `json:"second_evaluator_id"`
}

func (s *Server) handleAssignPair(w http.ResponseWriter, r *http.Request) {
	var req assignReq
	if !decode(w, r, &req) {
		return
	}
	if req.FirstEvaluatorID == "" || req.SecondEvaluatorID == "" {
		writeErr(w, http.StatusUnprocessableEntity, "both evaluator ids are required")
		return
	}
	if err := s.DB.AssignPair(r.Context(), r.PathValue("id"), req.FirstEvaluatorID, req.SecondEvaluatorID, userFrom(r.Context()).ID); err != nil {
		mapStoreErr(w, err)
		return
	}
	s.audit(r, "SCRIPT_ASSIGNED", "script", r.PathValue("id"), map[string]any{
		"first": req.FirstEvaluatorID, "second": req.SecondEvaluatorID,
	})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

type assignThirdReq struct {
	EvaluatorID string `json:"evaluator_id"`
}

func (s *Server) handleAssignThird(w http.ResponseWriter, r *http.Request) {
	var req assignThirdReq
	if !decode(w, r, &req) {
		return
	}
	if req.EvaluatorID == "" {
		writeErr(w, http.StatusUnprocessableEntity, "evaluator_id is required")
		return
	}
	if err := s.DB.AssignThird(r.Context(), r.PathValue("id"), req.EvaluatorID, userFrom(r.Context()).ID); err != nil {
		mapStoreErr(w, err)
		return
	}
	s.audit(r, "THIRD_EXAMINER_ASSIGNED", "script", r.PathValue("id"), map[string]any{"evaluator": req.EvaluatorID})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

type markReq struct {
	Score   *float64 `json:"score"`
	Comment string   `json:"comment"`
}

func (s *Server) handleSubmitMark(w http.ResponseWriter, r *http.Request) {
	var req markReq
	if !decode(w, r, &req) {
		return
	}
	if req.Score == nil {
		writeErr(w, http.StatusUnprocessableEntity, "score is required")
		return
	}
	u := userFrom(r.Context())
	var comment *string
	if c := strings.TrimSpace(req.Comment); c != "" {
		comment = &c
	}
	decision, err := s.DB.SubmitMark(r.Context(), r.PathValue("id"), u.ID, *req.Score, comment)
	if err != nil {
		if errors.Is(err, grading.ErrInvalidScore) {
			writeErr(w, http.StatusUnprocessableEntity, "score out of range for this paper")
			return
		}
		mapStoreErr(w, err)
		return
	}
	sc, _ := s.DB.ScriptByID(r.Context(), r.PathValue("id"))
	meta := map[string]any{"score": *req.Score}
	out := map[string]any{"ok": true, "status": "PENDING_SECOND"}
	switch {
	case decision.NeedThird:
		out["status"] = "THIRD_REVIEW_REQUIRED"
		out["gap"] = decision.Gap
		meta["third_review"] = true
	case decision.Final != nil:
		out["status"] = "RESOLVED"
		out["final_marks"] = decision.Final
		out["method"] = decision.Method
	}
	s.audit(r, "MARK_SUBMITTED", "script", r.PathValue("id"), meta)
	if sc != nil {
		out["script_status"] = sc.Status
	}
	writeJSON(w, http.StatusCreated, out)
}

func (s *Server) handleQueue(w http.ResponseWriter, r *http.Request) {
	out, err := s.DB.EvaluatorQueue(r.Context(), userFrom(r.Context()).ID, r.URL.Query().Get("status"))
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"queue": out})
}

type overrideReq struct {
	Score float64 `json:"score"`
	Note  string  `json:"note"`
}

func (s *Server) handleOverride(w http.ResponseWriter, r *http.Request) {
	var req overrideReq
	if !decode(w, r, &req) {
		return
	}
	if req.Score < 0 || req.Score > 100 {
		writeErr(w, http.StatusUnprocessableEntity, "score must be 0-100")
		return
	}
	if err := s.DB.OverrideResult(r.Context(), r.PathValue("id"), req.Score, userFrom(r.Context()).ID, req.Note); err != nil {
		s.log().Error("override failed", "err", err)
		mapStoreErr(w, err)
		return
	}
	s.audit(r, "RESULT_OVERRIDE", "script", r.PathValue("id"), map[string]any{"score": req.Score, "note": req.Note})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handlePDFToken mints a short-lived URL the browser can load in an iframe.
func (s *Server) handlePDFToken(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	u := userFrom(r.Context())
	if err := s.authorizePDF(r, id); err != nil {
		mapStoreErr(w, err)
		return
	}
	tok, err := s.Tokens.PDFToken(u.ID, id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "could not sign token")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"url":        fmt.Sprintf("/api/scripts/%s/pdf?t=%s", id, tok),
		"expires_in": 600,
	})
}

// handlePDF streams the PDF, authorised by ?t= token.
func (s *Server) handlePDF(w http.ResponseWriter, r *http.Request) {
	raw := r.URL.Query().Get("t")
	if raw == "" {
		raw = bearer(r)
	}
	claims, err := s.Tokens.Parse(raw, auth.PurposePDF)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "invalid or expired document token")
		return
	}
	id := r.PathValue("id")
	if claims.ScriptID != id {
		writeErr(w, http.StatusForbidden, "token is for another script")
		return
	}
	u, err := s.DB.UserByID(r.Context(), claims.UserID)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "unknown user")
		return
	}
	if u.Role == string(auth.RoleStudent) {
		st, err := s.DB.StudentByUserID(r.Context(), u.ID)
		if err != nil {
			mapStoreErr(w, err)
			return
		}
		var owner string
		_, owner, err = s.DB.ScriptPDFKey(r.Context(), id)
		if err != nil {
			mapStoreErr(w, err)
			return
		}
		if owner != st.ID {
			writeErr(w, http.StatusForbidden, "not your script")
			return
		}
	} else if u.Role == string(auth.RoleEvaluator) {
		if _, err := s.DB.EvaluatorSlot(r.Context(), id, u.ID); err != nil {
			mapStoreErr(w, err)
			return
		}
	}

	key, _, err := s.DB.ScriptPDFKey(r.Context(), id)
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	f, err := s.Files.Open(r.Context(), key)
	if err != nil {
		if os.IsNotExist(err) {
			writeErr(w, http.StatusNotFound, "PDF object missing")
			return
		}
		writeErr(w, http.StatusInternalServerError, "could not read PDF")
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "inline; filename=\""+id+".pdf\"")
	w.Header().Set("Cache-Control", "private, max-age=300")
	w.Header().Set("Content-Security-Policy", "frame-ancestors 'self'")
	// http.ServeContent handles ranges so the browser viewer can page.
	seeker, ok := f.(io.ReadSeeker)
	if ok {
		http.ServeContent(w, r, "", time.Time{}, seeker)
		return
	}
	_, _ = io.Copy(w, f)
}

func (s *Server) authorizePDF(r *http.Request, scriptID string) error {
	u := userFrom(r.Context())
	switch auth.Role(u.Role) {
	case auth.RoleAdmin:
		_, _, err := s.DB.ScriptPDFKey(r.Context(), scriptID)
		return err
	case auth.RoleEvaluator:
		_, err := s.DB.EvaluatorSlot(r.Context(), scriptID, u.ID)
		return err
	case auth.RoleStudent:
		st, err := s.DB.StudentByUserID(r.Context(), u.ID)
		if err != nil {
			return err
		}
		_, owner, err := s.DB.ScriptPDFKey(r.Context(), scriptID)
		if err != nil {
			return err
		}
		if owner != st.ID {
			return store.ErrForbidden
		}
		return nil
	}
	return store.ErrForbidden
}

var _ = strconv.Atoi
