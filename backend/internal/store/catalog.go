package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/istiyak/secure-exam-eval/internal/models"
)

// ---------- sessions ----------

func (s *Store) CreateSession(ctx context.Context, name string) (*models.Session, error) {
	id := newID()
	if _, err := s.DB.ExecContext(ctx, `INSERT INTO academic_sessions (id, name) VALUES (?,?)`, id, name); err != nil {
		return nil, mapErr(err)
	}
	var m models.Session
	err := s.DB.QueryRowContext(ctx, `SELECT id, name, is_active, created_at FROM academic_sessions WHERE id = ?`, id).
		Scan(&m.ID, &m.Name, &m.IsActive, &m.CreatedAt)
	return &m, mapErr(err)
}

func (s *Store) ListSessions(ctx context.Context) ([]models.Session, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id, name, is_active, created_at FROM academic_sessions ORDER BY name DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Session{}
	for rows.Next() {
		var m models.Session
		if err := rows.Scan(&m.ID, &m.Name, &m.IsActive, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// DeleteSession removes a session with its courses, students and their scripts.
func (s *Store) DeleteSession(ctx context.Context, id string) error {
	res, err := s.DB.ExecContext(ctx, `DELETE FROM academic_sessions WHERE id = ?`, id)
	if err != nil {
		return mapErr(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) SetSessionActive(ctx context.Context, id string, active bool) error {
	res, err := s.DB.ExecContext(ctx, `UPDATE academic_sessions SET is_active = ? WHERE id = ?`, active, id)
	if err != nil {
		return mapErr(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		var exists int
		if err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM academic_sessions WHERE id = ?`, id).Scan(&exists); err != nil {
			return mapErr(err)
		}
		if exists == 0 {
			return ErrNotFound
		}
	}
	return nil
}

// ---------- course catalogue ----------

// termName renders a study term such as "2nd Year 1st Semester".
func termName(year, semester int) string {
	ord := func(n int) string {
		switch n {
		case 1:
			return "1st"
		case 2:
			return "2nd"
		case 3:
			return "3rd"
		}
		return "4th"
	}
	return ord(year) + " Year " + ord(semester) + " Semester"
}

// ListCourseCatalog returns the fixed curriculum in study order.
func (s *Store) ListCourseCatalog(ctx context.Context) ([]models.CatalogCourse, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT id, code, title, credit, year_no, semester_no, category
		FROM course_catalog ORDER BY year_no, semester_no, sort_order`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.CatalogCourse{}
	for rows.Next() {
		var m models.CatalogCourse
		var year, semester int
		if err := rows.Scan(&m.ID, &m.Code, &m.Title, &m.Credit, &year, &semester, &m.Category); err != nil {
			return nil, err
		}
		m.Term = termName(year, semester)
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) CatalogCourseByID(ctx context.Context, id string) (*models.CatalogCourse, error) {
	var m models.CatalogCourse
	var year, semester int
	err := s.DB.QueryRowContext(ctx, `
		SELECT id, code, title, credit, year_no, semester_no, category
		FROM course_catalog WHERE id = ?`, id).
		Scan(&m.ID, &m.Code, &m.Title, &m.Credit, &year, &semester, &m.Category)
	if err != nil {
		return nil, mapErr(err)
	}
	m.Term = termName(year, semester)
	return &m, nil
}

// ---------- students ----------

func (s *Store) CreateStudent(ctx context.Context, no, registration, name string, email *string, sessionID string) (*models.Student, error) {
	id := newID()
	var reg any
	if registration != "" {
		reg = registration
	}
	if _, err := s.DB.ExecContext(ctx, `
		INSERT INTO students (id, student_no, registration_no, full_name, email, session_id) VALUES (?,?,?,?,?,?)`,
		id, no, reg, name, email, sessionID); err != nil {
		return nil, mapErr(err)
	}
	return s.StudentByID(ctx, id)
}

const studentCols = `st.id, st.student_no, st.registration_no, st.full_name, st.email, st.session_id, ses.name, st.user_id, st.created_at`

func scanStudent(row *sql.Row) (*models.Student, error) {
	var m models.Student
	err := row.Scan(&m.ID, &m.StudentNo, &m.RegistrationNo, &m.FullName, &m.Email, &m.SessionID, &m.Session, &m.UserID, &m.CreatedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	return &m, nil
}

func (s *Store) ListStudents(ctx context.Context, q, sessionID string) ([]models.Student, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT `+studentCols+`
		FROM students st JOIN academic_sessions ses ON ses.id = st.session_id
		WHERE (? = '' OR st.full_name LIKE CONCAT('%',?,'%') OR st.student_no LIKE CONCAT('%',?,'%'))
		  AND (? = '' OR st.session_id = ?)
		ORDER BY st.full_name LIMIT 500`, q, q, q, sessionID, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Student{}
	for rows.Next() {
		var m models.Student
		if err := rows.Scan(&m.ID, &m.StudentNo, &m.RegistrationNo, &m.FullName, &m.Email, &m.SessionID, &m.Session, &m.UserID, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) StudentByID(ctx context.Context, id string) (*models.Student, error) {
	return scanStudent(s.DB.QueryRowContext(ctx, `
		SELECT `+studentCols+` FROM students st
		JOIN academic_sessions ses ON ses.id = st.session_id WHERE st.id = ?`, id))
}

func (s *Store) StudentByUserID(ctx context.Context, userID string) (*models.Student, error) {
	return scanStudent(s.DB.QueryRowContext(ctx, `
		SELECT `+studentCols+` FROM students st
		JOIN academic_sessions ses ON ses.id = st.session_id WHERE st.user_id = ?`, userID))
}

func (s *Store) LinkStudentUser(ctx context.Context, studentID, userID string) error {
	if _, err := s.DB.ExecContext(ctx, `UPDATE students SET user_id = ? WHERE id = ?`, userID, studentID); err != nil {
		return mapErr(err)
	}
	_, err := s.StudentByID(ctx, studentID)
	return err
}

// DeleteStudent removes a student record; the portal account is left alone and
// their scripts go with them.
func (s *Store) DeleteStudent(ctx context.Context, id string) error {
	res, err := s.DB.ExecContext(ctx, `DELETE FROM students WHERE id = ?`, id)
	if err != nil {
		return mapErr(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// ---------- courses ----------

// CreateCourse copies a catalogue entry into a session. Marks stay out of 100.
func (s *Store) CreateCourse(ctx context.Context, catalogID, sessionID string, instructor *string) (*models.Course, error) {
	cat, err := s.CatalogCourseByID(ctx, catalogID)
	if err != nil {
		return nil, err
	}
	id := newID()
	if _, err := s.DB.ExecContext(ctx, `
		INSERT INTO courses (id, code, title, credit, catalog_id, session_id, max_marks, pass_marks, instructor_id)
		VALUES (?,?,?,?,?,?,100,40,?)`, id, cat.Code, cat.Title, cat.Credit, cat.ID, sessionID, instructor); err != nil {
		return nil, mapErr(err)
	}
	return s.CourseByID(ctx, id)
}

func (s *Store) ListCourses(ctx context.Context, sessionID string) ([]models.Course, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT c.id, c.code, c.title, c.credit, c.session_id, ses.name, c.max_marks, c.pass_marks,
		       u.full_name, c.is_active, c.created_at,
		       (SELECT COUNT(*) FROM scripts sc WHERE sc.course_id = c.id)
		FROM courses c
		JOIN academic_sessions ses ON ses.id = c.session_id
		LEFT JOIN users u ON u.id = c.instructor_id
		WHERE (? = '' OR c.session_id = ?)
		ORDER BY c.code`, sessionID, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Course{}
	for rows.Next() {
		var m models.Course
		if err := rows.Scan(&m.ID, &m.Code, &m.Title, &m.Credit, &m.SessionID, &m.Session, &m.MaxMarks, &m.PassMarks,
			&m.Instructor, &m.IsActive, &m.CreatedAt, &m.ScriptCount); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// DeleteCourse removes a course; its scripts (and their marks/results) go too.
func (s *Store) DeleteCourse(ctx context.Context, id string) error {
	res, err := s.DB.ExecContext(ctx, `DELETE FROM courses WHERE id = ?`, id)
	if err != nil {
		return mapErr(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// CourseByID loads one course.
func (s *Store) CourseByID(ctx context.Context, id string) (*models.Course, error) {
	var m models.Course
	err := s.DB.QueryRowContext(ctx, `
		SELECT id, code, title, credit, session_id, max_marks, pass_marks, is_active, created_at
		FROM courses WHERE id = ?`, id).
		Scan(&m.ID, &m.Code, &m.Title, &m.Credit, &m.SessionID, &m.MaxMarks, &m.PassMarks, &m.IsActive, &m.CreatedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	return &m, nil
}

// UnclaimedStudent matches a student ID against a registration number for
// sign-up linking. Mismatched pairs and already-claimed records return
// ErrNotFound.
func (s *Store) UnclaimedStudent(ctx context.Context, studentNo, registrationNo string) (*models.Student, error) {
	// Exact match first, then a suffixed match so an abbreviated number such as
	// "0301" also finds "MAT/2026/0301".
	for _, unclaimedOnly := range []bool{true, false} {
		m, err := s.matchStudent(ctx, studentNo, registrationNo, unclaimedOnly)
		if err == nil {
			if unclaimedOnly {
				return m, nil
			}
			return m, ErrConflict // the record exists but already belongs to someone
		}
		if !errors.Is(err, ErrNotFound) {
			return nil, err
		}
	}
	return nil, ErrNotFound
}

func (s *Store) matchStudent(ctx context.Context, studentNo, registrationNo string, unclaimedOnly bool) (*models.Student, error) {
	noSQL, noArgs := identClause("st.student_no", studentNo)
	regSQL, regArgs := identClause("st.registration_no", registrationNo)
	q := `
		SELECT ` + studentCols + ` FROM students st
		JOIN academic_sessions ses ON ses.id = st.session_id
		WHERE ` + noSQL + ` AND ` + regSQL
	if unclaimedOnly {
		q += ` AND st.user_id IS NULL`
	}
	return scanStudent(s.DB.QueryRowContext(ctx, q, append(noArgs, regArgs...)...))
}

// identClause matches an identifier case-insensitively, and by trailing digits
// when the input is numeric only.
func identClause(col, input string) (string, []any) {
	in := strings.ToUpper(strings.TrimSpace(input))
	if in == "" {
		return col + " = ?", []any{""}
	}
	for _, r := range in {
		if r < '0' || r > '9' {
			return "UPPER(" + col + ") = ?", []any{in}
		}
	}
	return "(UPPER(" + col + ") = ? OR " + col + " LIKE ?)", []any{in, "%" + in}
}

// ClaimStudent links a portal account only if the record is still unclaimed.
func (s *Store) ClaimStudent(ctx context.Context, studentID, userID string) (bool, error) {
	res, err := s.DB.ExecContext(ctx, `UPDATE students SET user_id = ? WHERE id = ? AND user_id IS NULL`, userID, studentID)
	if err != nil {
		return false, mapErr(err)
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}
