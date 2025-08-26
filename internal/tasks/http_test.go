package tasks

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func newTestServer(repo Repository) *chi.Mux {
	r := chi.NewRouter()
	RegisterRoutes(r, repo)
	return r
}

func TestPostTasks_Success(t *testing.T) {
	r := newTestServer(NewInMemoryRepo())

	body := []byte(`{"title":"learn chi"}`)
	req := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d, body=%s", rec.Code, rec.Body.String())
	}

	var got Task
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if got.ID == 0 {
		t.Errorf("expected non-zero ID")
	}
	if got.Title != "learn chi" {
		t.Errorf("expected Title=learn chi, got %q", got.Title)
	}
	if got.Done {
		t.Errorf("new tasks should default to Done=false")
	}
	if got.CreatedAt.IsZero() {
		t.Errorf("expected CreatedAt to be set")
	}
}

func TestPostTasks_TitleRequired(t *testing.T) {
	r := newTestServer(NewInMemoryRepo())

	body := []byte(`{"title":""}`)
	req := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d, body=%s", rec.Code, rec.Body.String())
	}

	var errResp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to parse error JSON: %v", err)
	}
	if errResp["error"] != "title required" {
		t.Errorf("expected error 'title required', got %q", errResp["error"])
	}
}

func TestPostTasks_InvalidJSON(t *testing.T) {
	r := newTestServer(NewInMemoryRepo())

	body := []byte(`{"title":`)
	req := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d, body=%s", rec.Code, rec.Body.String())
	}

	var errResp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to parse error JSON: %v", err)
	}
	if errResp["error"] != "invalid JSON" {
		t.Errorf("expected error 'invalid JSON', got %q", errResp["error"])
	}
}

func TestGetTasks_HappyPath(t *testing.T) {
	repo := NewInMemoryRepo()

	seed, err := repo.Create("seeded task")
	if err != nil {
		t.Fatalf("unexpected error seeding repo: %v", err)
	}
	if seed.ID == 0 {
		t.Fatalf("expected seeded task to have an ID")
	}

	r := newTestServer(repo)
	req := httptest.NewRequest(http.MethodGet, "/tasks", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", rec.Code, rec.Body.String())
	}

	var list []Task
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 task, got %d", len(list))
	}
	if list[0].Title != "seeded task" {
		t.Errorf("expected first task title 'seeded task', got %q", list[0].Title)
	}
}

type fakeRepoListError struct{}

func (f fakeRepoListError) Create(title string) (Task, error) { return Task{}, nil }
func (f fakeRepoListError) List() ([]Task, error)             { return nil, errors.New("boom") }

func TestGetTasks_RepoError(t *testing.T) {
	r := newTestServer(fakeRepoListError{})

	req := httptest.NewRequest(http.MethodGet, "/tasks", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d, body=%s", rec.Code, rec.Body.String())
	}

	var errResp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to parse error JSON: %v", err)
	}
	if errResp["error"] != "unexpected error" {
		t.Errorf("expected error 'unexpected error', got %q", errResp["error"])
	}
}
