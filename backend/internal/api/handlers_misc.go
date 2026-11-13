package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/istiyak/exam-evaluation/internal/auth"
)

func trimZero(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

func (s *Server) handleListChallenges(w http.ResponseWriter, r *http.Request) {
	out, err := s.DB.ListChallenges(r.Context(), r.URL.Query().Get("status"))
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"challenges": out})
}

type challengeReq struct {
	ScriptID string `json:"script_id"`
	Reason   string `json:"reason"`
}

// handleCreateChallenge lets a student contest a published script.
func (s *Server) handleCreateChallenge(w http.ResponseWriter, r *http.Request) {
	var req challengeReq
	if !decode(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Reason) == "" || len(req.Reason) < 10 {
		writeErr(w, http.StatusUnprocessableEntity, "reason must be at least 10 characters")
		return
	}
	st, err := s.mustStudentUser(r)
	if err != nil {
		writeErr(w, http.StatusNotFound, "no student record linked to this account")
		return
	}
	_, owner, err := s.DB.ScriptPDFKey(r.Context(), req.ScriptID)
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	if owner != st.ID {
		writeErr(w, http.StatusForbidden, "not your script")
		return
	}
	// Only a published (resolved) mark can be challenged.
	sc, err := s.DB.ScriptByID(r.Context(), req.ScriptID)
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	if sc.Status != "RESOLVED" {
		writeErr(w, http.StatusUnprocessableEntity, "this script has no published result to challenge yet")
		return
	}
	ch, err := s.DB.CreateChallenge(r.Context(), req.ScriptID, st.ID, strings.TrimSpace(req.Reason))
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	s.audit(r, "CHALLENGE_CREATED", "challenge", ch.ID, map[string]any{"script": req.ScriptID})
	writeJSON(w, http.StatusCreated, ch)
}

type decideChallengeReq struct {
	Status string `json:"status"`
	Note   string `json:"note"`
}

func (s *Server) handleDecideChallenge(w http.ResponseWriter, r *http.Request) {
	var req decideChallengeReq
	if !decode(w, r, &req) {
		return
	}
	req.Status = strings.ToUpper(req.Status)
	if req.Status != "UPHELD" && req.Status != "DISMISSED" {
		writeErr(w, http.StatusUnprocessableEntity, "status must be UPHELD or DISMISSED")
		return
	}
	if err := s.DB.DecideChallenge(r.Context(), r.PathValue("id"), req.Status, req.Note, userFrom(r.Context()).ID); err != nil {
		mapStoreErr(w, err)
		return
	}
	s.audit(r, "CHALLENGE_DECIDED", "challenge", r.PathValue("id"), map[string]any{"status": req.Status})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	out, err := s.DB.Stats(r.Context())
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleAudit(w http.ResponseWriter, r *http.Request) {
	out, err := s.DB.ListAudit(r.Context(), r.URL.Query().Get("entity"), r.URL.Query().Get("entity_id"), intParam(r, "limit", 200))
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"entries": out})
}

func (s *Server) handleClearAudit(w http.ResponseWriter, r *http.Request) {
	n, err := s.DB.ClearAudit(r.Context())
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	s.audit(r, "AUDIT_CLEARED", "audit", "", map[string]any{"entries": n})
	writeJSON(w, http.StatusOK, map[string]any{"cleared": n})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	dbOK := s.DB.DB.PingContext(r.Context()) == nil
	status := http.StatusOK
	if !dbOK {
		status = http.StatusServiceUnavailable
	}
	writeJSON(w, status, map[string]any{"status": map[bool]string{true: "ok", false: "degraded"}[dbOK],
		"db": dbOK, "env": s.Cfg.Env})
}

func (s *Server) handleMeta(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"third_review_threshold": 13,
		"roles":                  []auth.Role{auth.RoleAdmin, auth.RoleEvaluator, auth.RoleStudent},
		"version":                "1.0.0",
	})
}
