// Package db owns the MySQL pool and embedded schema migrations.
package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"time"

	"github.com/go-sql-driver/mysql"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// Open dials MySQL, forcing the DSN options the app depends on.
func Open(ctx context.Context, dsn string) (*sql.DB, error) {
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse mysql dsn: %w", err)
	}
	// parseTime keeps DATETIME columns as time.Time; multiStatements lets a
	// migration file run as one statement batch.
	cfg.ParseTime = true
	cfg.MultiStatements = true
	cfg.Loc = time.UTC
	cfg.Collation = "utf8mb4_general_ci"
	if cfg.Timeout == 0 {
		cfg.Timeout = 10 * time.Second
	}

	connector, err := mysql.NewConnector(cfg)
	if err != nil {
		return nil, err
	}
	pool := sql.OpenDB(connector)
	pool.SetMaxOpenConns(20)
	pool.SetMaxIdleConns(10)
	pool.SetConnMaxLifetime(30 * time.Minute)

	pingCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if err := pool.PingContext(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping mysql: %w", err)
	}
	return pool, nil
}

// Migrate applies each embedded migration once, guarded by a named lock so
// replicas starting together cannot race.
func Migrate(ctx context.Context, pool *sql.DB) error {
	conn, err := pool.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	var got sql.NullInt64
	if err := conn.QueryRowContext(ctx, `SELECT GET_LOCK('secure_exam_migrate', 30)`).Scan(&got); err != nil {
		return err
	}
	if !got.Valid || got.Int64 != 1 {
		return fmt.Errorf("could not acquire migration lock")
	}
	defer conn.ExecContext(context.Background(), `SELECT RELEASE_LOCK('secure_exam_migrate')`)

	if _, err := conn.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version VARCHAR(160) PRIMARY KEY,
		applied_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`); err != nil {
		return err
	}

	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)

	for _, name := range names {
		var exists int
		if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE version = ?`, name).Scan(&exists); err != nil {
			return err
		}
		if exists > 0 {
			continue
		}
		body, err := migrationFS.ReadFile("migrations/" + name)
		if err != nil {
			return err
		}
		if _, err := conn.ExecContext(ctx, string(body)); err != nil {
			// DDL is not transactional in MySQL; record nothing so it retries.
			return fmt.Errorf("migration %s: %w", name, err)
		}
		if _, err := conn.ExecContext(ctx, `INSERT INTO schema_migrations (version) VALUES (?)`, name); err != nil {
			return err
		}
	}
	return nil
}
