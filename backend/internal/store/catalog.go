package store

import (
	"context"
	"database/sql"

	"github.com/istiyak/secure-exam-eval/internal/models"
)

// ---------- sessions ----------

func (s *Store) CreateSession(ctx context.Context, name, code string) (*models.Session, error) {
	id := newID()
	if _, err := s.DB.ExecContext(ctx, `INSERT INTO academic_sessions (id, name, code) VALUES (?,?,?)`, id, name, code); err != nil {
		return nil, mapErr(err)
	}
	var m models.Session
	err := s.DB.QueryRowContext(ctx, `SELECT id, name, code, is_active, created_at FROM academic_sessions WHERE id = ?`, id).
		Scan(&m.ID, &m.Name, &m.Code, &m.IsActive, &m.CreatedAt)
	return &m, mapErr(err)
}

func (s *Store) ListSessions(ctx context.Context) ([]models.Session, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id, name, code, is_active, created_at FROM academic_sessions ORDER BY name DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Session{}
	for rows.Next() {
		var m models.Session
		if err := rows.Scan(&m.ID, &m.Name, &m.Code, &m.IsActive, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
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

// ---------- students ----------

func (s *Store) CreateStudent(ctx context.Context, no, name string, email *string, sessionID string) (*models.Student, error) {
	id := newID()
	if _, err := s.DB.ExecContext(ctx, `
		INSERT INTO students (id, student_no, full_name, email, session_id) VALUES (?,?,?,?,?)`,
		id, no, name, email, sessionID); err != nil {
		return nil, mapErr(err)
	}
	return s.StudentByID(ctx, id)
}

const studentCols = `st.id, st.student_no, st.full_name, st.email, st.session_id, ses.name, st.user_id, st.created_at`

func scanStudent(row *sql.Row) (*models.Student, error) {
	var m models.Student
	err := row.Scan(&m.ID, &m.StudentNo, &m.FullName, &m.Email, &m.SessionID, &m.Session, &m.UserID, &m.CreatedAt)
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
		if err := rows.Scan(&m.ID, &m.StudentNo, &m.FullName, &m.Email, &m.SessionID, &m.Session, &m.UserID, &m.CreatedAt); err != nil {
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

// ---------- courses ----------

func (s *Store) CreateCourse(ctx context.Context, code, title, sessionID string, maxMarks, passMarks float64, instructor *string) (*models.Course, error) {
	id := newID()
	if _, err := s.DB.ExecContext(ctx, `
		INSERT INTO courses (id, code, title, session_id, max_marks, pass_marks, instructor_id)
		VALUES (?,?,?,?,?,?,?)`, id, code, title, sessionID, maxMarks, passMarks, instructor); err != nil {
		return nil, mapErr(err)
	}
	return s.CourseByID(ctx, id)
}

func (s *Store) ListCourses(ctx context.Context, sessionID string) ([]models.Course, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT c.id, c.code, c.title, c.session_id, ses.name, c.max_marks, c.pass_marks,
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
		if err := rows.Scan(&m.ID, &m.Code, &m.Title, &m.SessionID, &m.Session, &m.MaxMarks, &m.PassMarks,
			&m.Instructor, &m.IsActive, &m.CreatedAt, &m.ScriptCount); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// CourseByID loads one course.
func (s *Store) CourseByID(ctx context.Context, id string) (*models.Course, error) {
	var m models.Course
	err := s.DB.QueryRowContext(ctx, `
		SELECT id, code, title, session_id, max_marks, pass_marks, is_active, created_at
		FROM courses WHERE id = ?`, id).
		Scan(&m.ID, &m.Code, &m.Title, &m.SessionID, &m.MaxMarks, &m.PassMarks, &m.IsActive, &m.CreatedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	return &m, nil
}

// UnclaimedStudent looks a candidate up by matriculation number for sign-up
// linking. Returns ErrNotFound when the record does not exist or is taken.
func (s *Store) UnclaimedStudent(ctx context.Context, studentNo string) (*models.Student, error) {
	return scanStudent(s.DB.QueryRowContext(ctx, `
		SELECT `+studentCols+` FROM students st
		JOIN academic_sessions ses ON ses.id = st.session_id
		WHERE st.student_no = ? AND st.user_id IS NULL`, studentNo))
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
