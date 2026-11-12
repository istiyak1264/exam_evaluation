// Package store wraps every SQL statement the API needs (MySQL dialect).
package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"

	"github.com/istiyak/secure-exam-eval/internal/models"
)

// Sentinel errors mapped to HTTP statuses by the API layer.
var (
	ErrNotFound  = errors.New("not found")
	ErrConflict  = errors.New("conflict")
	ErrForbidden = errors.New("forbidden")
	ErrState     = errors.New("invalid state for this operation")
	ErrInvalid   = errors.New("invalid input")
	ErrInUse     = errors.New("still referenced by other records")
)

// Store is the persistence gateway.
type Store struct{ DB *sql.DB }

func New(db *sql.DB) *Store { return &Store{DB: db} }

func newID() string { return uuid.NewString() }

// mysqlErrNo extracts the server error number, or 0.
func mysqlErrNo(err error) uint16 {
	var me *mysql.MySQLError
	if errors.As(err, &me) {
		return me.Number
	}
	return 0
}

func mapErr(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, sql.ErrNoRows):
		return ErrNotFound
	case mysqlErrNo(err) == 1062:
		return ErrConflict
	case mysqlErrNo(err) == 1452:
		return fmt.Errorf("%w: referenced row missing", ErrInvalid)
	case mysqlErrNo(err) == 1451:
		return fmt.Errorf("%w: row still referenced", ErrConflict)
	}
	return err
}

// ---------- users ----------

type UserRow struct {
	models.User
	PasswordHash string
}

func (s *Store) CreateUser(ctx context.Context, email, hash, name, role string) (*models.User, error) {
	id := newID()
	if _, err := s.DB.ExecContext(ctx, `
		INSERT INTO users (id, email, password_hash, full_name, role) VALUES (?,?,?,?,?)`,
		id, email, hash, name, role); err != nil {
		return nil, mapErr(err)
	}
	return s.UserByID(ctx, id)
}

func (s *Store) UserByEmail(ctx context.Context, email string) (*UserRow, error) {
	var u UserRow
	err := s.DB.QueryRowContext(ctx, `
		SELECT id, email, full_name, role, is_active, created_at, password_hash
		FROM users WHERE email = ?`, email).
		Scan(&u.ID, &u.Email, &u.FullName, &u.Role, &u.IsActive, &u.CreatedAt, &u.PasswordHash)
	if err != nil {
		return nil, mapErr(err)
	}
	return &u, nil
}

func (s *Store) UserByID(ctx context.Context, id string) (*models.User, error) {
	var u models.User
	err := s.DB.QueryRowContext(ctx, `
		SELECT id, email, full_name, role, is_active, created_at FROM users WHERE id = ?`, id).
		Scan(&u.ID, &u.Email, &u.FullName, &u.Role, &u.IsActive, &u.CreatedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	return &u, nil
}

func (s *Store) ListUsers(ctx context.Context, role, q string) ([]models.User, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT id, email, full_name, role, is_active, created_at FROM users
		WHERE (? = '' OR role = ?)
		  AND (? = '' OR full_name LIKE CONCAT('%',?,'%') OR email LIKE CONCAT('%',?,'%'))
		ORDER BY role, full_name LIMIT 500`, role, role, q, q, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.User{}
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Email, &u.FullName, &u.Role, &u.IsActive, &u.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// DeleteUser removes an account outright, but only when nothing else depends on
// it: assignments, marks and similar rows keep a NOT NULL reference to the user
// and would otherwise have to lose data.
func (s *Store) DeleteUser(ctx context.Context, id string) error {
	var refs int
	if err := s.DB.QueryRowContext(ctx, `
		SELECT (SELECT COUNT(*) FROM marks WHERE evaluator_id = ?)
		     + (SELECT COUNT(*) FROM script_assignments WHERE evaluator_id = ?)
		     + (SELECT COUNT(*) FROM scripts WHERE registered_by = ?)
		     + (SELECT COUNT(*) FROM results WHERE resolved_by = ?)
		     + (SELECT COUNT(*) FROM grade_challenges WHERE decided_by = ?)
		     + (SELECT COUNT(*) FROM courses WHERE instructor_id = ?)`,
		id, id, id, id, id, id).Scan(&refs); err != nil {
		return mapErr(err)
	}
	if refs > 0 {
		return ErrInUse
	}
	_, err := s.DB.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id)
	return mapErr(err)
}

func (s *Store) SetUserRole(ctx context.Context, id, role string) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE users SET role = ? WHERE id = ?`, role, id)
	return mapErr(err)
}

// CountActiveAdmins counts administrators, used to stop the last one being
// demoted.
func (s *Store) CountActiveAdmins(ctx context.Context, excludeID string) (int, error) {
	var n int
	err := s.DB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM users WHERE role = 'ADMIN' AND is_active = 1 AND id <> ?`, excludeID).Scan(&n)
	return n, err
}

func (s *Store) SetUserActive(ctx context.Context, id string, active bool) error {
	if _, err := s.DB.ExecContext(ctx, `UPDATE users SET is_active = ? WHERE id = ?`, active, id); err != nil {
		return mapErr(err)
	}
	_, err := s.UserByID(ctx, id)
	return err
}

func (s *Store) SetPassword(ctx context.Context, id, hash string) error {
	if _, err := s.DB.ExecContext(ctx, `UPDATE users SET password_hash = ? WHERE id = ?`, hash, id); err != nil {
		return mapErr(err)
	}
	_, err := s.UserByID(ctx, id)
	return err
}

// ---------- refresh tokens ----------

func (s *Store) SaveRefresh(ctx context.Context, userID, hash string, exp time.Time) error {
	_, err := s.DB.ExecContext(ctx, `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at) VALUES (?,?,?,?)`,
		newID(), userID, hash, exp.UTC())
	return err
}

// RotateRefresh consumes a refresh token (single use) and returns its owner.
func (s *Store) RotateRefresh(ctx context.Context, oldHash string) (*models.User, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var userID string
	err = tx.QueryRowContext(ctx, `
		SELECT user_id FROM refresh_tokens
		WHERE token_hash = ? AND revoked_at IS NULL AND expires_at > NOW(6) FOR UPDATE`, oldHash).Scan(&userID)
	if err != nil {
		return nil, mapErr(err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE refresh_tokens SET revoked_at = NOW(6) WHERE token_hash = ?`, oldHash); err != nil {
		return nil, err
	}
	var u models.User
	if err := tx.QueryRowContext(ctx, `SELECT id, email, full_name, role, is_active FROM users WHERE id = ?`, userID).
		Scan(&u.ID, &u.Email, &u.FullName, &u.Role, &u.IsActive); err != nil {
		return nil, mapErr(err)
	}
	if !u.IsActive {
		return nil, ErrForbidden
	}
	return &u, tx.Commit()
}

func (s *Store) RevokeRefresh(ctx context.Context, hash string) {
	_, _ = s.DB.ExecContext(ctx, `UPDATE refresh_tokens SET revoked_at = NOW(6) WHERE token_hash = ?`, hash)
}

// ---------- audit ----------

func (s *Store) Audit(ctx context.Context, actorID, actorRole, action, entity, entityID string, meta map[string]any) {
	if meta == nil {
		meta = map[string]any{}
	}
	payload, err := json.Marshal(meta)
	if err != nil {
		payload = []byte("{}")
	}
	// audit_log.id is AUTO_INCREMENT, so it is never supplied.
	cols := "action, entity, meta"
	vals := "?,?,?"
	args := []any{action, entity, payload}

	if actorID != "" {
		cols += ", actor_id"
		vals += ",?"
		args = append(args, actorID)
	}
	if actorRole != "" {
		cols += ", actor_role"
		vals += ",?"
		args = append(args, actorRole)
	}
	if entityID != "" {
		cols += ", entity_id"
		vals += ",?"
		args = append(args, entityID)
	}
	_, _ = s.DB.ExecContext(ctx, `INSERT INTO audit_log (`+cols+`) VALUES (`+vals+`)`, args...)
}

// ClearAudit empties the trail and reports how many rows were removed.
func (s *Store) ClearAudit(ctx context.Context) (int64, error) {
	res, err := s.DB.ExecContext(ctx, `DELETE FROM audit_log`)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}

func (s *Store) ListAudit(ctx context.Context, entity, entityID string, limit int) ([]models.AuditEntry, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	rows, err := s.DB.QueryContext(ctx, `
		SELECT a.id, u.full_name, a.actor_role, a.action, a.entity, a.entity_id, a.meta, a.created_at
		FROM audit_log a LEFT JOIN users u ON u.id = a.actor_id
		WHERE (? = '' OR a.entity = ?) AND (? = '' OR a.entity_id = ?)
		ORDER BY a.id DESC LIMIT ?`, entity, entity, entityID, entityID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.AuditEntry{}
	for rows.Next() {
		var a models.AuditEntry
		var role sql.NullString
		var meta []byte
		if err := rows.Scan(&a.ID, &a.ActorName, &role, &a.Action, &a.Entity, &a.EntityID, &meta, &a.CreatedAt); err != nil {
			return nil, err
		}
		if role.Valid {
			a.ActorRole = &role.String
		}
		a.Meta = json.RawMessage(meta)
		out = append(out, a)
	}
	return out, rows.Err()
}
