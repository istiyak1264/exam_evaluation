package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/istiyak/exam-evaluation/internal/grading"
	"github.com/istiyak/exam-evaluation/internal/models"
)

// SearchResults powers the retrieval screen: any of name, matriculation
// number, academic session or script code narrows the same result set.
// studentID, when set, scopes everything to one student (portal view).
func (s *Store) SearchResults(ctx context.Context, q, sessionID, courseID, status, studentID string, limit int) ([]models.ResultRow, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.DB.QueryContext(ctx, `
		SELECT sc.id, sc.script_code, st.id, st.student_no, st.full_name, ses.name,
		       c.code, c.title, sc.status, r.final_marks, c.pass_marks, f.stat,
		       sc.registered_at, sc.resolved_at
		FROM scripts sc
		JOIN students st ON st.id = sc.student_id
		JOIN academic_sessions ses ON ses.id = st.session_id
		JOIN courses c ON c.id = sc.course_id
		LEFT JOIN results r ON r.script_id = sc.id
		LEFT JOIN LATERAL (
			SELECT gc.status AS stat FROM grade_challenges gc
			WHERE gc.script_id = sc.id ORDER BY gc.created_at DESC LIMIT 1
		) f ON TRUE
		WHERE (? = '' OR st.full_name LIKE CONCAT('%',?,'%')
		       OR st.student_no LIKE CONCAT('%',?,'%')
		       OR ses.name LIKE CONCAT('%',?,'%')
		       OR sc.script_code LIKE CONCAT('%',?,'%'))
		  AND (? = '' OR st.session_id = ?)
		  AND (? = '' OR sc.course_id = ?)
		  AND (? = '' OR sc.status = ?)
		  AND (? = '' OR sc.student_id = ?)
		ORDER BY ses.name DESC, st.full_name LIMIT ?`,
		q, q, q, q, q, sessionID, sessionID, courseID, courseID, status, status, studentID, studentID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []models.ResultRow{}
	for rows.Next() {
		var m models.ResultRow
		var final, pass sql.NullFloat64
		if err := rows.Scan(&m.ScriptID, &m.Code, &m.StudentID, &m.StudentNo, &m.StudentName, &m.Session,
			&m.CourseCode, &m.CourseTitle, &m.Status, &final, &pass, &m.Challenge,
			&m.RegisteredAt, &m.ResolvedAt); err != nil {
			return nil, err
		}
		if final.Valid {
			f := final.Float64
			m.FinalMarks = &f
			g := grading.GradeLetter(f)
			m.Grade = &g
			if pass.Valid {
				ok := f >= pass.Float64
				m.Passed = &ok
			}
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// EvaluatorQueue lists scripts assigned to one evaluator, still blind.
func (s *Store) EvaluatorQueue(ctx context.Context, evaluatorID, status string) ([]models.ScriptBlind, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT sc.id, sc.script_code, sc.status, c.code, c.title, c.max_marks,
		       sc.page_count, sa.slot, r.final_marks, sa.assigned_at,
		       EXISTS(SELECT 1 FROM marks mk WHERE mk.script_id = sc.id AND mk.slot = sa.slot) AS marked
		FROM script_assignments sa
		JOIN scripts sc ON sc.id = sa.script_id
		JOIN courses c ON c.id = sc.course_id
		LEFT JOIN results r ON r.script_id = sc.id
		WHERE sa.evaluator_id = ?
		  AND (? = ''
		       OR (? = 'PENDING' AND NOT EXISTS(SELECT 1 FROM marks m2 WHERE m2.script_id = sc.id AND m2.slot = sa.slot))
		       OR (? = 'MARKED' AND EXISTS(SELECT 1 FROM marks m3 WHERE m3.script_id = sc.id AND m3.slot = sa.slot)))
		ORDER BY sa.assigned_at DESC LIMIT 300`,
		evaluatorID, status, status, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []models.ScriptBlind{}
	for rows.Next() {
		var m models.ScriptBlind
		var pages sql.NullInt64
		var marked bool
		if err := rows.Scan(&m.ID, &m.Code, &m.Status, &m.CourseCode, &m.CourseTitle, &m.MaxMarks,
			&pages, &m.MySlot, &m.FinalMarks, &m.AssignedAt, &marked); err != nil {
			return nil, err
		}
		if pages.Valid {
			p := int(pages.Int64)
			m.PageCount = &p
		}
		m.Marks = []models.Mark{}
		if marked {
			mk, err := s.MarksForScript(ctx, m.ID)
			if err != nil {
				return nil, err
			}
			m.Marks = mk
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// MarksForScript returns the marks of one script in slot order.
func (s *Store) MarksForScript(ctx context.Context, scriptID string) ([]models.Mark, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT slot, score, comment, submitted_at FROM marks WHERE script_id = ?
		ORDER BY CASE slot WHEN 'FIRST' THEN 1 WHEN 'SECOND' THEN 2 ELSE 3 END`, scriptID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Mark{}
	for rows.Next() {
		var m models.Mark
		if err := rows.Scan(&m.Slot, &m.Score, &m.Comment, &m.SubmittedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// ---------- challenges ----------

func (s *Store) CreateChallenge(ctx context.Context, scriptID, studentID, reason string) (*models.Challenge, error) {
	var open int
	if err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM grade_challenges WHERE script_id = ? AND status = 'OPEN'`,
		scriptID).Scan(&open); err != nil {
		return nil, err
	}
	if open > 0 {
		return nil, fmt.Errorf("%w: an open challenge already exists", ErrConflict)
	}
	id := newID()
	if _, err := s.DB.ExecContext(ctx, `
		INSERT INTO grade_challenges (id, script_id, student_id, reason) VALUES (?,?,?,?)`,
		id, scriptID, studentID, reason); err != nil {
		return nil, mapErr(err)
	}
	var m models.Challenge
	err := s.DB.QueryRowContext(ctx, `
		SELECT id, script_id, reason, status, created_at FROM grade_challenges WHERE id = ?`, id).
		Scan(&m.ID, &m.ScriptID, &m.Reason, &m.Status, &m.CreatedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	return &m, nil
}

func (s *Store) ListChallenges(ctx context.Context, status string) ([]models.Challenge, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT gc.id, gc.script_id, sc.script_code, st.student_no, st.full_name, c.code,
		       gc.reason, gc.status, gc.decision_note, gc.created_at, gc.decided_at
		FROM grade_challenges gc
		JOIN scripts sc ON sc.id = gc.script_id
		JOIN students st ON st.id = gc.student_id
		JOIN courses c ON c.id = sc.course_id
		WHERE (? = '' OR gc.status = ?)
		ORDER BY CASE gc.status WHEN 'OPEN' THEN 0 ELSE 1 END, gc.created_at DESC LIMIT 300`,
		status, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Challenge{}
	for rows.Next() {
		var m models.Challenge
		if err := rows.Scan(&m.ID, &m.ScriptID, &m.Code, &m.StudentNo, &m.StudentName, &m.CourseCode,
			&m.Reason, &m.Status, &m.DecisionNote, &m.CreatedAt, &m.DecidedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) DecideChallenge(ctx context.Context, id, status, note, actorID string) error {
	res, err := s.DB.ExecContext(ctx, `
		UPDATE grade_challenges SET status = ?, decision_note = ?, decided_by = ?, decided_at = NOW(6)
		WHERE id = ? AND status = 'OPEN'`, status, note, nullIfEmpty(actorID), id)
	if err != nil {
		return mapErr(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("%w: challenge missing or already decided", ErrState)
	}
	return nil
}

// ---------- stats ----------

func (s *Store) Stats(ctx context.Context) (*models.Stats, error) {
	st := &models.Stats{Scripts: map[string]int{}, Users: map[string]int{}}
	rows, err := s.DB.QueryContext(ctx, `SELECT status, COUNT(*) FROM scripts GROUP BY status`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var k string
		var v int
		if err := rows.Scan(&k, &v); err != nil {
			rows.Close()
			return nil, err
		}
		st.Scripts[k] = v
	}
	rows.Close()

	rows, err = s.DB.QueryContext(ctx, `SELECT role, COUNT(*) FROM users GROUP BY role`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var k string
		var v int
		if err := rows.Scan(&k, &v); err != nil {
			rows.Close()
			return nil, err
		}
		st.Users[k] = v
	}
	rows.Close()

	_ = s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM grade_challenges WHERE status = 'OPEN'`).Scan(&st.Challenges)
	_ = s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM results`).Scan(&st.Resolved)
	var third, total int
	_ = s.DB.QueryRowContext(ctx, `SELECT SUM(status = 'THIRD_REVIEW'), COUNT(*) FROM scripts`).Scan(&third, &total)
	if total > 0 {
		st.ThirdRate = float64(third) / float64(total) * 100
	}
	var avg sql.NullFloat64
	_ = s.DB.QueryRowContext(ctx, `SELECT AVG(final_marks) FROM results`).Scan(&avg)
	if avg.Valid {
		st.Averages = avg.Float64
	}
	return st, nil
}
