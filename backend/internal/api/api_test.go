package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/istiyak/secure-exam-eval/internal/auth"
	"github.com/istiyak/secure-exam-eval/internal/config"
	"github.com/istiyak/secure-exam-eval/internal/db"
	"github.com/istiyak/secure-exam-eval/internal/storage"
	"github.com/istiyak/secure-exam-eval/internal/store"
)

// harness is a running API over a real MySQL database.
type harness struct {
	t   *testing.T
	srv *httptest.Server
	db  *sql.DB
	api *Server
	dir string
	sfx string
}

const testPDF = "%PDF-1.7\n1 0 obj\n<< /Type /Pages /Count 1 >>\nendobj\n/Type /Page\ntrailer\n<<>>\n%%EOF\n"

func newHarness(t *testing.T) *harness {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		url = "exam:examtest@tcp(127.0.0.1:53306)/exam_eval_test?parseTime=true&multiStatements=true"
	}
	ctx := context.Background()
	pool, err := db.Open(ctx, url)
	if err != nil {
		t.Skipf("no test database: %v", err)
	}
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.ExecContext(ctx, `SET FOREIGN_KEY_CHECKS = 0`); err != nil {
		t.Fatalf("fk off: %v", err)
	}
	for _, tbl := range []string{"refresh_tokens", "audit_log", "grade_challenges", "results", "marks",
		"script_assignments", "scripts", "courses", "students", "academic_sessions", "users"} {
		if _, err := pool.ExecContext(ctx, "DELETE FROM "+tbl); err != nil {
			t.Fatalf("truncate %s: %v", tbl, err)
		}
	}
	if _, err := pool.ExecContext(ctx, `SET FOREIGN_KEY_CHECKS = 1`); err != nil {
		t.Fatalf("fk on: %v", err)
	}
	dir := t.TempDir()
	files, err := storage.New("local", dir, "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{
		Env: "test", DatabaseURL: url, JWTSecret: []byte(strings.Repeat("k", 40)),
		AccessTTL: 15 * time.Minute, RefreshTTL: time.Hour, StorageDriver: "local", StorageDir: dir,
		MaxUploadBytes: 5 << 20, AllowedOrigins: []string{"http://localhost:5173"},
	}
	s := &Server{Cfg: cfg, DB: store.New(pool), Tokens: auth.NewManager(cfg.JWTSecret, cfg.AccessTTL, cfg.RefreshTTL),
		Files: files, Log: slog.New(slog.NewTextHandler(os.Stderr, nil)), Limiter: NewLimiter(100, time.Minute)}

	srv := httptest.NewServer(s.Routes())
	t.Cleanup(func() { srv.Close(); pool.Close() })
	return &harness{t: t, srv: srv, db: pool, api: s, sfx: uuid.NewString()[:6]}
}

func (h *harness) do(method, path, token string, body any) (int, map[string]any) {
	h.t.Helper()
	var rdr io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, h.srv.URL+path, rdr)
	if err != nil {
		h.t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		h.t.Fatal(err)
	}
	defer res.Body.Close()
	out := map[string]any{}
	raw, _ := io.ReadAll(res.Body)
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &out)
	}
	return res.StatusCode, out
}

// user creates an account through the store and returns (id, token).
func (h *harness) user(role auth.Role) (string, string) {
	h.t.Helper()
	email := fmt.Sprintf("%s-%s@uni.test", role, uuid.NewString()[:8])
	hash, err := auth.HashPassword("correct-horse-battery")
	if err != nil {
		h.t.Fatal(err)
	}
	u, err := h.api.DB.CreateUser(context.Background(), email, hash, string(role)+" User", string(role))
	if err != nil {
		h.t.Fatalf("create %s: %v", role, err)
	}
	tok, err := h.api.Tokens.Access(u.ID, role)
	if err != nil {
		h.t.Fatal(err)
	}
	return u.ID, tok
}

// fixtures builds a session, student, course and returns their ids.
func (h *harness) fixtures(sessionToken string) (studentID, courseID string) {
	h.t.Helper()
	st, body := h.do("POST", "/api/sessions", sessionToken, map[string]any{"name": "2026/2027 " + h.sfx, "code": "T" + h.sfx})
	if st != 201 {
		h.t.Fatalf("session: %d %v", st, body)
	}
	ses := body["id"].(string)

	stBody, b1 := h.do("POST", "/api/students", sessionToken, map[string]any{
		"student_no": "MAT" + h.sfx, "full_name": "Ada Lovelace " + h.sfx, "session_id": ses})
	if st != 201 {
		h.t.Fatalf("student: %d %v", stBody, b1)
	}
	crsBody, b2 := h.do("POST", "/api/courses", sessionToken, map[string]any{
		"code": "CSC301" + h.sfx, "title": "Compilers", "session_id": ses})
	if crsBody != 201 {
		h.t.Fatalf("course: %d %v", crsBody, b2)
	}
	return b1["id"].(string), b2["id"].(string)
}

func (h *harness) uploadScript(token, courseID, studentID, code string) (int, map[string]any) {
	h.t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("course_id", courseID)
	_ = mw.WriteField("student_id", studentID)
	if code != "" {
		_ = mw.WriteField("script_code", code)
	}
	fw, _ := mw.CreateFormFile("file", "script.pdf")
	_, _ = fw.Write([]byte(testPDF))
	mw.Close()

	req, _ := http.NewRequest("POST", h.srv.URL+"/api/scripts", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		h.t.Fatal(err)
	}
	defer res.Body.Close()
	out := map[string]any{}
	raw, _ := io.ReadAll(res.Body)
	_ = json.Unmarshal(raw, &out)
	return res.StatusCode, out
}

func (h *harness) login(email, password string) (int, map[string]any) {
	return h.do("POST", "/api/auth/login", "", map[string]any{"email": email, "password": password})
}

func TestLoginAndRefresh(t *testing.T) {
	h := newHarness(t)
	email := "admin-" + h.sfx + "@uni.test"
	hash, _ := auth.HashPassword("correct-horse-battery")
	_, err := h.api.DB.CreateUser(context.Background(), email, hash, "Admin", "ADMIN")
	if err != nil {
		t.Fatal(err)
	}

	if st, _ := h.login(email, "wrong-password-here"); st != 401 {
		t.Fatalf("wrong password: %d", st)
	}
	st, body := h.login(email, "correct-horse-battery")
	if st != 200 {
		t.Fatalf("login: %d %v", st, body)
	}
	refresh := body["refresh_token"].(string)
	if refresh == "" || body["access_token"] == "" {
		t.Fatal("tokens missing")
	}
	if st, _ := h.do("POST", "/api/auth/refresh", "", map[string]any{"refresh_token": refresh}); st != 200 {
		t.Fatalf("refresh: %d", st)
	}
	// Rotation invalidates the old refresh token.
	if st, _ := h.do("POST", "/api/auth/refresh", "", map[string]any{"refresh_token": refresh}); st != 401 {
		t.Fatalf("reused refresh token accepted: %d", st)
	}
}

func TestAuthorizationBoundaries(t *testing.T) {
	h := newHarness(t)
	_, adminTok := h.user(auth.RoleAdmin)
	_, evalTok := h.user(auth.RoleEvaluator)
	_, studentTok := h.user(auth.RoleStudent)
	studentID, _ := h.fixtures(adminTok)

	if st, _ := h.do("GET", "/api/scripts", "", nil); st != 401 {
		t.Fatalf("anonymous: %d", st)
	}
	if st, _ := h.do("GET", "/api/scripts", evalTok, nil); st != 403 {
		t.Fatalf("evaluator listing scripts: %d", st)
	}
	if st, _ := h.do("GET", "/api/admin/stats", studentTok, nil); st != 403 {
		t.Fatalf("student reading stats: %d", st)
	}
	if st, _ := h.do("POST", "/api/users", evalTok, map[string]any{"email": "x@y.z", "full_name": "X", "role": "ADMIN", "password": "longenough1"}); st != 403 {
		t.Fatalf("evaluator creating user: %d", st)
	}
	_ = studentID
}

func TestFullMarkingWorkflow(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	_, adminTok := h.user(auth.RoleAdmin)
	ev1, ev1Tok := h.user(auth.RoleEvaluator)
	ev2, ev2Tok := h.user(auth.RoleEvaluator)
	ev3, ev3Tok := h.user(auth.RoleEvaluator)
	stuUser, studentTok := h.user(auth.RoleStudent)
	studentID, courseID := h.fixtures(adminTok)
	if st, _ := h.do("POST", "/api/students/"+studentID+"/link-user", adminTok, map[string]any{"user_id": stuUser}); st != 200 {
		t.Fatalf("link: %d", st)
	}

	// --- close-gap script: average
	st, body := h.uploadScript(adminTok, courseID, studentID, "CSC301-26-0001")
	if st != 201 {
		t.Fatalf("upload: %d %v", st, body)
	}
	s1 := body["script"].(map[string]any)["id"].(string)

	if st, _ := h.do("POST", "/api/scripts/"+s1+"/assign", adminTok, map[string]any{
		"first_evaluator_id": ev1, "second_evaluator_id": ev1}); st != 422 {
		t.Fatalf("same evaluator twice: %d", st)
	}
	if st, _ := h.do("POST", "/api/scripts/"+s1+"/assign", adminTok, map[string]any{
		"first_evaluator_id": ev1, "second_evaluator_id": ev2}); st != 200 {
		t.Fatalf("assign: %d", st)
	}

	// Blind projection: no student identity leaks to the examiner.
	if _, got := h.do("GET", "/api/scripts/"+s1, ev1Tok, nil); got["student_name"] != nil || got["student_no"] != nil {
		t.Fatalf("evaluator saw student identity: %v", got)
	}
	if st, _ := h.do("POST", "/api/scripts/"+s1+"/marks", ev1Tok, map[string]any{"score": 70}); st != 201 {
		t.Fatalf("mark 1: %d", st)
	}
	if st, _ := h.do("POST", "/api/scripts/"+s1+"/marks", ev1Tok, map[string]any{"score": 88}); st != 409 {
		t.Fatalf("duplicate mark: %d", st)
	}
	st, body = h.do("POST", "/api/scripts/"+s1+"/marks", ev2Tok, map[string]any{"score": 80, "comment": "solid"})
	if st != 201 || body["status"] != "RESOLVED" {
		t.Fatalf("mark 2: %d %v", st, body)
	}
	if body["final_marks"].(float64) != 75 {
		t.Fatalf("final should be 75, got %v", body["final_marks"])
	}
	// Unassigned evaluator cannot mark.
	if st, _ := h.do("POST", "/api/scripts/"+s1+"/marks", ev3Tok, map[string]any{"score": 50}); st != 403 {
		t.Fatalf("stranger marking: %d", st)
	}

	// --- wide-gap script: third review, on a second student of the same course
	stu2User, stu2Tok := h.user(auth.RoleStudent)
	stu2Resp, stu2Body := h.do("POST", "/api/students", adminTok, map[string]any{
		"student_no": "MAT2" + h.sfx, "full_name": "Grace Hopper " + h.sfx, "session_id": bodySessionID(t, h, adminTok)})
	if stu2Resp != 201 {
		t.Fatalf("student 2: %d %v", stu2Resp, stu2Body)
	}
	stu2 := stu2Body["id"].(string)
	h.do("POST", "/api/students/"+stu2+"/link-user", adminTok, map[string]any{"user_id": stu2User})

	st, body = h.uploadScript(adminTok, courseID, stu2, "CSC301-26-0002")
	if st != 201 {
		t.Fatalf("upload 2: %d %v", st, body)
	}
	s2 := body["script"].(map[string]any)["id"].(string)
	h.do("POST", "/api/scripts/"+s2+"/assign", adminTok, map[string]any{"first_evaluator_id": ev1, "second_evaluator_id": ev2})

	h.do("POST", "/api/scripts/"+s2+"/marks", ev1Tok, map[string]any{"score": 40})
	st, body = h.do("POST", "/api/scripts/"+s2+"/marks", ev2Tok, map[string]any{"score": 90})
	if body["status"] != "THIRD_REVIEW_REQUIRED" {
		t.Fatalf("expected third review, got %v", body)
	}
	if body["gap"].(float64) != 50 {
		t.Fatalf("gap: %v", body["gap"])
	}
	if st, _ := h.do("POST", "/api/scripts/"+s2+"/assign-third", adminTok, map[string]any{"evaluator_id": ev1}); st != 422 {
		t.Fatalf("third = existing examiner: %d", st)
	}
	if st, _ := h.do("POST", "/api/scripts/"+s2+"/assign-third", adminTok, map[string]any{"evaluator_id": ev3}); st != 200 {
		t.Fatalf("assign third: %d", st)
	}
	st, body = h.do("POST", "/api/scripts/"+s2+"/marks", ev3Tok, map[string]any{"score": 60})
	if st != 201 || body["status"] != "RESOLVED" {
		t.Fatalf("third mark: %d %v", st, body)
	}
	if body["final_marks"].(float64) != 60 || body["method"] != "MEDIAN_OF_THREE" {
		t.Fatalf("median expected 60, got %v %v", body["final_marks"], body["method"])
	}

	// --- retrieval by every key
	for _, q := range []string{"Ada Lovelace", "MAT" + h.sfx, "2026/2027 " + h.sfx, "CSC301-26-0002"} {
		st, body := h.do("GET", "/api/results?q="+urlq(q), adminTok, nil)
		if st != 200 || int(body["count"].(float64)) == 0 {
			t.Fatalf("search %q returned nothing: %d %v", q, st, body)
		}
	}

	// --- PDF access control
	st, body = h.do("POST", "/api/scripts/"+s1+"/pdf-token", studentTok, nil)
	if st != 200 {
		t.Fatalf("student pdf token: %d %v", st, body)
	}
	res, err := http.Get(h.srv.URL + body["url"].(string))
	if err != nil {
		t.Fatal(err)
	}
	pdf, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 200 || !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		t.Fatalf("pdf fetch: %d", res.StatusCode)
	}
	// Another student's script is off limits.
	if st, _ := h.do("POST", "/api/scripts/"+s1+"/pdf-token", stu2Tok, nil); st != 403 {
		t.Fatalf("foreign pdf token: %d", st)
	}
	// Unsigned PDF access fails.
	req, _ := http.NewRequest("GET", h.srv.URL+"/api/scripts/"+s1+"/pdf", nil)
	r2, _ := http.DefaultClient.Do(req)
	if r2.StatusCode != 401 {
		t.Fatalf("unsigned pdf: %d", r2.StatusCode)
	}
	r2.Body.Close()

	// --- student portal + challenge
	st, body = h.do("GET", "/api/results/mine", stu2Tok, nil)
	if st != 200 || len(body["results"].([]any)) != 1 {
		t.Fatalf("portal results: %d %v", st, body)
	}
	st, body = h.do("POST", "/api/challenges", stu2Tok, map[string]any{"script_id": s2, "reason": "The second script was marked too harshly."})
	if st != 201 {
		t.Fatalf("challenge: %d %v", st, body)
	}
	chID := body["id"].(string)
	if st, _ := h.do("POST", "/api/challenges", stu2Tok, map[string]any{"script_id": s2, "reason": "duplicate attempt"}); st != 409 {
		t.Fatalf("duplicate challenge: %d", st)
	}
	if st, _ := h.do("PATCH", "/api/challenges/"+chID, adminTok, map[string]any{"status": "UPHELD", "note": "remarked"}); st != 200 {
		t.Fatalf("decide: %d", st)
	}

	// --- admin override and audit trail
	if st, _ := h.do("POST", "/api/scripts/"+s2+"/override", adminTok, map[string]any{"score": 62, "note": "board decision"}); st != 200 {
		t.Fatalf("override: %d", st)
	}
	st, body = h.do("GET", "/api/results?q=CSC301-26-0002", adminTok, nil)
	row := body["results"].([]any)[0].(map[string]any)
	if row["final_marks"].(float64) != 62 {
		t.Fatalf("override not applied: %v", row["final_marks"])
	}

	_, auditBody := h.do("GET", "/api/admin/audit?entity=script&entity_id="+s1, adminTok, nil)
	if len(auditBody["entries"].([]any)) == 0 {
		t.Fatal("no audit entries recorded")
	}
	_ = ctx
	_ = ev1
}

func bodySessionID(t *testing.T, h *harness, adminTok string) string {
	t.Helper()
	_, body := h.do("GET", "/api/sessions", adminTok, nil)
	list := body["sessions"].([]any)
	return list[0].(map[string]any)["id"].(string)
}

func urlq(s string) string {
	r := strings.NewReplacer(" ", "+", "/", "%2F", "@", "%40")
	return r.Replace(s)
}

func TestUploadValidation(t *testing.T) {
	h := newHarness(t)
	_, adminTok := h.user(auth.RoleAdmin)
	studentID, courseID := h.fixtures(adminTok)

	// Non-PDF payload is rejected.
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	mw.WriteField("course_id", courseID)
	mw.WriteField("student_id", studentID)
	fw, _ := mw.CreateFormFile("file", "notes.txt")
	fw.Write([]byte("just text"))
	mw.Close()
	req, _ := http.NewRequest("POST", h.srv.URL+"/api/scripts", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+adminTok)
	res, _ := http.DefaultClient.Do(req)
	if res.StatusCode != 422 {
		t.Fatalf("non-pdf accepted: %d", res.StatusCode)
	}
	res.Body.Close()

	// Same student twice on one course is a conflict.
	if st, _ := h.uploadScript(adminTok, courseID, studentID, "DUP-1"); st != 201 {
		t.Fatalf("first upload: %d", st)
	}
	if st, _ := h.uploadScript(adminTok, courseID, studentID, "DUP-2"); st != 409 {
		t.Fatalf("duplicate student: %d", st)
	}
}
