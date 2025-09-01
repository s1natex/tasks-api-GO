package tasks

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

//go:generate echo "(no codegen)"

type SQLiteRepo struct {
	db *sql.DB
}

func NewSQLiteRepo(dsn string) (*SQLiteRepo, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	// Reasonable pragmas for an app server
	if _, err := db.Exec(`
		PRAGMA journal_mode=WAL;
		PRAGMA synchronous=NORMAL;
		PRAGMA foreign_keys=ON;
	`); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &SQLiteRepo{db: db}, nil
}

func (r *SQLiteRepo) Close() error { return r.db.Close() }

// Create implements Repository.Create with basic validation
func (r *SQLiteRepo) Create(title string) (Task, error) {
	if strings.TrimSpace(title) == "" {
		return Task{}, ErrTitleRequired
	}
	now := time.Now().UTC()
	res, err := r.db.Exec(`
		INSERT INTO tasks (title, done, created_at)
		VALUES (?, 0, ?)
	`, title, now.Format(time.RFC3339Nano))
	if err != nil {
		return Task{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Task{}, err
	}
	return Task{
		ID:        id,
		Title:     title,
		Done:      false,
		CreatedAt: now,
	}, nil
}

// List implements Repository.List
func (r *SQLiteRepo) List() ([]Task, error) {
	rows, err := r.db.Query(`
		SELECT id, title, done, created_at
		FROM tasks
		ORDER BY id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Task
	for rows.Next() {
		var t Task
		var created string
		if err := rows.Scan(&t.ID, &t.Title, &t.Done, &created); err != nil {
			return nil, err
		}
		if ts, err := time.Parse(time.RFC3339Nano, created); err == nil {
			t.CreatedAt = ts
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// ApplyMigrations ensures schema exists
func (r *SQLiteRepo) ApplyMigrations(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS tasks (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	title TEXT NOT NULL,
	done INTEGER NOT NULL DEFAULT 0,
	created_at TEXT NOT NULL
);
	`)
	return err
}

// Helper to build DSN like: file:/absolute/path?_pragma=busy_timeout(5000)
func SQLiteFileDSN(path string) (string, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return "file:" + filepath.ToSlash(abs) + "?_pragma=busy_timeout(5000)", nil
}
