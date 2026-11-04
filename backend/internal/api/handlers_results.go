package api

import (
	"encoding/csv"
	"net/http"
	"strings"
)

// handleSearch is the registrar/hall-team retrieval screen.
func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	rows, err := s.DB.SearchResults(r.Context(), strings.TrimSpace(q.Get("q")),
		q.Get("session_id"), q.Get("course_id"), q.Get("status"), "", intParam(r, "limit", 100))
	if err != nil {
		s.log().Error("search failed", "err", err)
		mapStoreErr(w, err)
		return
	}
	s.audit(r, "RESULT_SEARCH", "result", "", map[string]any{"q": q.Get("q")})
	writeJSON(w, http.StatusOK, map[string]any{"results": rows, "count": len(rows)})
}

func (s *Server) handleMyResults(w http.ResponseWriter, r *http.Request) {
	st, err := s.mustStudentUser(r)
	if err != nil {
		writeErr(w, http.StatusNotFound, "no student record linked to this account")
		return
	}
	rows, err := s.DB.SearchResults(r.Context(), "", "", "", "", st.ID, 200)
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	// Marks stay hidden until the board upholds a challenge on that script.
	for i := range rows {
		upheld := rows[i].Challenge != nil && *rows[i].Challenge == "UPHELD"
		rows[i].Revealed = upheld
		if !upheld {
			rows[i].FinalMarks, rows[i].Grade, rows[i].Passed = nil, nil, nil
			continue
		}
		marks, err := s.DB.MarksForScript(r.Context(), rows[i].ScriptID)
		if err != nil {
			mapStoreErr(w, err)
			return
		}
		rows[i].Marks = marks
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": rows, "student": st})
}

func (s *Server) handleExportCSV(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	rows, err := s.DB.SearchResults(r.Context(), q.Get("q"), q.Get("session_id"), q.Get("course_id"),
		q.Get("status"), "", 500)
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=results.csv")
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"script_code", "student_no", "student_name", "session", "course_code", "status", "final_marks", "grade", "passed"})
	for _, row := range rows {
		grade, passed, final := "", "", ""
		if row.Grade != nil {
			grade = *row.Grade
		}
		if row.Passed != nil {
			passed = map[bool]string{true: "yes", false: "no"}[*row.Passed]
		}
		if row.FinalMarks != nil {
			final = ftoa(*row.FinalMarks)
		}
		_ = cw.Write([]string{row.Code, row.StudentNo, row.StudentName, row.Session, row.CourseCode,
			row.Status, final, grade, passed})
	}
	cw.Flush()
	s.audit(r, "RESULTS_EXPORTED", "result", "", map[string]any{"q": q.Get("q")})
}

func ftoa(f float64) string {
	s := trimZero(f)
	return s
}
