package tasks

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func newTempDB(t *testing.T) *SQLiteRepo {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	dsn, err := SQLiteFileDSN(dbPath)
	if err != nil {
		t.Fatalf("dsn error: %v", err)
	}
	repo, err := NewSQLiteRepo(dsn)
	if err != nil {
		t.Fatalf("open error: %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
		_ = os.RemoveAll(dir)
	})
	if err := repo.ApplyMigrations(context.Background()); err != nil {
		t.Fatalf("migrate error: %v", err)
	}
	return repo
}

func TestSQLiteRepo_CreateAndList(t *testing.T) {
	repo := newTempDB(t)

	_, err := repo.Create("") // validation
	if err == nil {
		t.Fatalf("expected ErrTitleRequired")
	}
	if err != ErrTitleRequired {
		t.Fatalf("expected ErrTitleRequired, got %v", err)
	}

	a, err := repo.Create("first")
	if err != nil {
		t.Fatalf("create first: %v", err)
	}
	if a.ID == 0 || a.Title != "first" || a.Done {
		t.Fatalf("bad first task: %+v", a)
	}

	b, err := repo.Create("second")
	if err != nil {
		t.Fatalf("create second: %v", err)
	}
	if b.ID <= a.ID {
		t.Fatalf("expected monotonic IDs: a=%d b=%d", a.ID, b.ID)
	}

	list, err := repo.List()
	if err != nil {
		t.Fatalf("list error: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(list))
	}
	if list[0].Title != "first" || list[1].Title != "second" {
		t.Fatalf("unexpected order: %+v", list)
	}
}
