package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/istiyak/secure-exam-eval/internal/grading"
	"github.com/istiyak/secure-exam-eval/internal/models"
)

// maxCodeRetries guards the cover-sheet code generator.
const maxCodeRetries = 6

// NextScriptCode produces the cover-sheet code, e.g. CSC301-26-0007.
// The sequence is per course and year so codes leak no cross-course ordering.
func (s *Store) NextScriptCode(ctx context.Context, courseCode string) (string, error) {
	var n int
	err := s.DB.QueryRowContext(ctx, `
		SELECT COUNT(*) + 1 FROM scripts sc JOIN courses c ON c.id = sc.course_id
		WHERE c.code = ? AND YEAR(sc.registered_at) = YEAR(NOW())`, courseCode).Scan(&n)
	if err != nil {
		return "", err
	}
	prefix := strings.ToUpper(strings.ReplaceAll(courseCode, " ", ""))
	yy := time.Now().Format("06")
	for i := 0; i < maxCodeRetries; i++ {
		code := fmt.Sprintf("%s-%s-%04d", prefix, yy, n+i)
		var taken int
		if err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM scripts WHERE script_code = ?`, code).Scan(&taken); err != nil {
			return "", err
		}
		if taken == 0 {
			return code, nil
		}
	}
	return "", ErrConflict
}

type NewScript struct {
	Code      string
	CourseID  string
	StudentID string
	PDFKey    string
	SHA256    string
	Bytes     int64
	Pages     *int
	ActorID   string
}

func (s *Store) CreateScript(ctx context.Context, in NewScript) (*models.Script, error) {
	id := newID()
	var pages any
	if in.Pages != nil {
		pages = *in.Pages
	}
	if _, err := s.DB.ExecContext(ctx, `
		INSERT INTO scripts (id, script_code, course_id, student_id, pdf_key, pdf_sha256, pdf_bytes, page_count, registered_by)
		VALUES (?,?,?,?,?,?,?,?,?)`,
		id, in.Code, in.CourseID, in.StudentID, in.PDFKey, in.SHA256, in.Bytes, pages, nullIfEmpty(in.ActorID)); err != nil {
		return nil, mapErr(err)
	}
	return s.ScriptByID(ctx, id)
}

const scriptCols = `
	sc.id, sc.script_code, sc.status, sc.course_id, c.code, c.title, c.max_marks,
	st.id, st.student_no, st.full_name, ses.name, sc.page_count, sc.pdf_bytes, sc.registered_at, sc.resolved_at`

const scriptFrom = `
	FROM scripts sc
	JOIN courses c ON c.id = sc.course_id
	JOIN students st ON st.id = sc.student_id
	JOIN academic_sessions ses ON ses.id = st.session_id`

type scannable interface {
	Scan(dest ...any) error
}

func scanScript(row scannable) (*models.Script, error) {
	var m models.Script
	var pages sql.NullInt64
	err := row.Scan(&m.ID, &m.Code, &m.Status, &m.CourseID, &m.CourseCode, &m.CourseTitle, &m.MaxMarks,
		&m.StudentID, &m.StudentNo, &m.StudentName, &m.SessionName, &pages, &m.PDFBytes, &m.RegisteredAt, &m.ResolvedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	if pages.Valid {
		p := int(pages.Int64)
		m.PageCount = &p
	}
	m.Marks = []models.Mark{}
	return &m, nil
}

func (s *Store) ScriptByID(ctx context.Context, id string) (*models.Script, error) {
	m, err := scanScript(s.DB.QueryRowContext(ctx, `SELECT `+scriptCols+scriptFrom+` WHERE sc.id = ?`, id))
	if err != nil {
		return nil, err
	}
	if err := s.hydrateMarks(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

func (s *Store) hydrateMarks(ctx context.Context, m *models.Script) error {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT mk.slot, mk.score, mk.comment, u.full_name, mk.submitted_at
		FROM marks mk JOIN users u ON u.id = mk.evaluator_id
		WHERE mk.script_id = ?
		ORDER BY CASE mk.slot WHEN 'FIRST' THEN 1 WHEN 'SECOND' THEN 2 ELSE 3 END`, m.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var mk models.Mark
		if err := rows.Scan(&mk.Slot, &mk.Score, &mk.Comment, &mk.EvaluatorName, &mk.SubmittedAt); err != nil {
			return err
		}
		m.Marks = append(m.Marks, mk)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	var final, gap sql.NullFloat64
	var method sql.NullString
	err = s.DB.QueryRowContext(ctx, `SELECT final_marks, gap, method FROM results WHERE script_id = ?`, m.ID).
		Scan(&final, &gap, &method)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if final.Valid {
		m.FinalMarks = &final.Float64
	}
	if gap.Valid {
		m.Gap = &gap.Float64
	}
	if method.Valid {
		m.Method = &method.String
	}
	return nil
}

func (s *Store) ListScripts(ctx context.Context, status, courseID, q string, limit int) ([]models.Script, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT `+scriptCols+`,
			(SELECT GROUP_CONCAT(CONCAT(mk.slot, ':', mk.score) ORDER BY
			   CASE mk.slot WHEN 'FIRST' THEN 1 WHEN 'SECOND' THEN 2 ELSE 3 END)
			 FROM marks mk WHERE mk.script_id = sc.id) AS mark_pairs,
			r.final_marks, r.gap
		`+scriptFrom+`
		LEFT JOIN results r ON r.script_id = sc.id
		WHERE (? = '' OR sc.status = ?)
		  AND (? = '' OR sc.course_id = ?)
		  AND (? = '' OR sc.script_code LIKE CONCAT('%',?,'%')
		       OR st.full_name LIKE CONCAT('%',?,'%') OR st.student_no LIKE CONCAT('%',?,'%'))
		ORDER BY sc.registered_at DESC LIMIT ?`,
		status, status, courseID, courseID, q, q, q, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Script{}
	for rows.Next() {
		m, err := scanListRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *m)
	}
	return out, rows.Err()
}

// scanListRow reads a list row: the script columns plus packed marks and result.
func scanListRow(row scannable) (*models.Script, error) {
	var m models.Script
	var pages sql.NullInt64
	var pairs sql.NullString
	var final, gap sql.NullFloat64
	err := row.Scan(&m.ID, &m.Code, &m.Status, &m.CourseID, &m.CourseCode, &m.CourseTitle, &m.MaxMarks,
		&m.StudentID, &m.StudentNo, &m.StudentName, &m.SessionName, &pages, &m.PDFBytes, &m.RegisteredAt, &m.ResolvedAt,
		&pairs, &final, &gap)
	if err != nil {
		return nil, mapErr(err)
	}
	if pages.Valid {
		p := int(pages.Int64)
		m.PageCount = &p
	}
	m.Marks = []models.Mark{}
	if pairs.Valid && pairs.String != "" {
		for _, part := range strings.Split(pairs.String, ",") {
			slot, sc, ok := strings.Cut(part, ":")
			if !ok {
				continue
			}
			v, _ := strconv.ParseFloat(sc, 64)
			m.Marks = append(m.Marks, models.Mark{Slot: slot, Score: v})
		}
	}
	if final.Valid {
		f := final.Float64
		m.FinalMarks = &f
	}
	if gap.Valid {
		g := gap.Float64
		m.Gap = &g
	}
	return &m, nil
}

// AssignPair appoints the first and second evaluator. They must differ.
func (s *Store) AssignPair(ctx context.Context, scriptID, first, second, actorID string) error {
	if first == second {
		return fmt.Errorf("%w: the two evaluators must differ", ErrInvalid)
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var status string
	if err := tx.QueryRowContext(ctx, `SELECT status FROM scripts WHERE id = ? FOR UPDATE`, scriptID).Scan(&status); err != nil {
		return mapErr(err)
	}
	if status != "REGISTERED" && status != "ASSIGNED" {
		return fmt.Errorf("%w: script is %s", ErrState, status)
	}
	for slot, ev := range map[string]string{"FIRST": first, "SECOND": second} {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO script_assignments (id, script_id, evaluator_id, slot, assigned_by)
			VALUES (?,?,?,?,?)
			ON DUPLICATE KEY UPDATE evaluator_id = VALUES(evaluator_id), assigned_by = VALUES(assigned_by), assigned_at = NOW(6)`,
			newID(), scriptID, ev, slot, nullIfEmpty(actorID)); err != nil {
			return mapErr(err)
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE scripts SET status = 'ASSIGNED' WHERE id = ?`, scriptID); err != nil {
		return err
	}
	return tx.Commit()
}

// AssignThird appoints the third examiner after a gap of 13 or more.
func (s *Store) AssignThird(ctx context.Context, scriptID, evaluatorID, actorID string) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var status string
	if err := tx.QueryRowContext(ctx, `SELECT status FROM scripts WHERE id = ? FOR UPDATE`, scriptID).Scan(&status); err != nil {
		return mapErr(err)
	}
	if status != "THIRD_REVIEW" {
		return fmt.Errorf("%w: script is %s", ErrState, status)
	}
	var clash int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM script_assignments WHERE script_id = ? AND evaluator_id = ?`,
		scriptID, evaluatorID).Scan(&clash); err != nil {
		return err
	}
	if clash > 0 {
		return fmt.Errorf("%w: already marked this script", ErrInvalid)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO script_assignments (id, script_id, evaluator_id, slot, assigned_by)
		VALUES (?,?,?,'THIRD',?)
		ON DUPLICATE KEY UPDATE evaluator_id = VALUES(evaluator_id), assigned_at = NOW(6)`,
		newID(), scriptID, evaluatorID, nullIfEmpty(actorID)); err != nil {
		return mapErr(err)
	}
	return tx.Commit()
}

func nullIfEmpty(v string) any {
	if v == "" {
		return nil
	}
	return v
}

// SubmitMark records a score and applies the grading policy in one transaction.
func (s *Store) SubmitMark(ctx context.Context, scriptID, evaluatorID string, score float64, comment *string) (grading.Decision, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return grading.Decision{}, err
	}
	defer tx.Rollback()

	var status string
	var maxMarks float64
	if err := tx.QueryRowContext(ctx, `SELECT sc.status, c.max_marks FROM scripts sc
		JOIN courses c ON c.id = sc.course_id WHERE sc.id = ? FOR UPDATE`, scriptID).Scan(&status, &maxMarks); err != nil {
		return grading.Decision{}, mapErr(err)
	}
	// Assignment is checked before status so a stranger gets 403, not 409.
	var slot string
	if err := tx.QueryRowContext(ctx, `SELECT slot FROM script_assignments WHERE script_id = ? AND evaluator_id = ?`,
		scriptID, evaluatorID).Scan(&slot); err != nil {
		return grading.Decision{}, ErrForbidden
	}
	if status != "ASSIGNED" && status != "EVAL_PENDING" && status != "THIRD_REVIEW" {
		return grading.Decision{}, fmt.Errorf("%w: script is %s", ErrState, status)
	}
	var already int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM marks WHERE script_id = ? AND slot = ?`,
		scriptID, slot).Scan(&already); err != nil {
		return grading.Decision{}, err
	}
	if already > 0 {
		return grading.Decision{}, fmt.Errorf("%w: slot %s already marked", ErrConflict, slot)
	}
	switch {
	case slot == "THIRD" && status != "THIRD_REVIEW":
		return grading.Decision{}, fmt.Errorf("%w: third review not requested", ErrState)
	case slot != "THIRD" && status == "THIRD_REVIEW":
		return grading.Decision{}, fmt.Errorf("%w: third review in progress", ErrState)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO marks (id, script_id, slot, evaluator_id, score, comment) VALUES (?,?,?,?,?,?)`,
		newID(), scriptID, slot, evaluatorID, score, comment); err != nil {
		return grading.Decision{}, mapErr(err)
	}

	rows, err := tx.QueryContext(ctx, `SELECT score FROM marks WHERE script_id = ?
		ORDER BY CASE slot WHEN 'FIRST' THEN 1 WHEN 'SECOND' THEN 2 ELSE 3 END`, scriptID)
	if err != nil {
		return grading.Decision{}, err
	}
	var scores []float64
	for rows.Next() {
		var sc float64
		if err := rows.Scan(&sc); err != nil {
			rows.Close()
			return grading.Decision{}, err
		}
		scores = append(scores, sc)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return grading.Decision{}, err
	}
	rows.Close()

	decision, err := grading.Evaluate(scores, maxMarks)
	if err != nil {
		if errors.Is(err, grading.ErrInvalidScore) {
			return grading.Decision{}, err
		}
		return grading.Decision{}, fmt.Errorf("%w: %s", ErrInvalid, err)
	}

	switch {
	case decision.NeedThird:
		if _, err := tx.ExecContext(ctx, `UPDATE scripts SET status = 'THIRD_REVIEW' WHERE id = ?`, scriptID); err != nil {
			return grading.Decision{}, err
		}
	case decision.Final != nil:
		if _, err := tx.ExecContext(ctx, `UPDATE scripts SET status = 'RESOLVED', resolved_at = NOW(6) WHERE id = ?`, scriptID); err != nil {
			return grading.Decision{}, err
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO results (script_id, final_marks, method, gap) VALUES (?,?,?,?)
			ON DUPLICATE KEY UPDATE final_marks = VALUES(final_marks), method = VALUES(method),
				gap = VALUES(gap), resolved_at = NOW(6)`,
			scriptID, *decision.Final, decision.Method, decision.Gap); err != nil {
			return grading.Decision{}, err
		}
	default:
		if _, err := tx.ExecContext(ctx, `UPDATE scripts SET status = 'EVAL_PENDING' WHERE id = ? AND status = 'ASSIGNED'`, scriptID); err != nil {
			return grading.Decision{}, err
		}
	}
	return decision, tx.Commit()
}

// OverrideResult lets a registrar/admin fix a resolved mark, keeping the trail.
func (s *Store) OverrideResult(ctx context.Context, scriptID string, score float64, actorID, note string) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO results (script_id, final_marks, method, resolved_by) VALUES (?,?,'ADMIN_OVERRIDE',?)
		ON DUPLICATE KEY UPDATE final_marks = VALUES(final_marks), method = 'ADMIN_OVERRIDE',
			resolved_by = VALUES(resolved_by), resolved_at = NOW(6)`,
		scriptID, score, nullIfEmpty(actorID)); err != nil {
		return mapErr(err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE scripts SET status = 'RESOLVED', resolved_at = NOW(6) WHERE id = ?`, scriptID); err != nil {
		return err
	}
	if note != "" {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO audit_log (actor_id, action, entity, entity_id, meta)
			VALUES (?, 'RESULT_OVERRIDE', 'script', ?, JSON_OBJECT('note', ?, 'score', ?))`,
			nullIfEmpty(actorID), scriptID, note, score); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) ScriptPDFKey(ctx context.Context, scriptID string) (key, studentID string, err error) {
	err = s.DB.QueryRowContext(ctx, `SELECT pdf_key, student_id FROM scripts WHERE id = ?`, scriptID).Scan(&key, &studentID)
	if err != nil {
		return "", "", mapErr(err)
	}
	return key, studentID, nil
}

// EvaluatorSlot reports which slot an evaluator holds, if any.
func (s *Store) EvaluatorSlot(ctx context.Context, scriptID, evaluatorID string) (string, error) {
	var slot string
	err := s.DB.QueryRowContext(ctx, `SELECT slot FROM script_assignments WHERE script_id = ? AND evaluator_id = ?`,
		scriptID, evaluatorID).Scan(&slot)
	if err != nil {
		return "", mapErr(err)
	}
	return slot, nil
}

var _ = strconv.Itoa
