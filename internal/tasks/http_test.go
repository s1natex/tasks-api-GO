package tasks

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestPostTasks_TitleRequired_ValidationError(t *testing.T) {
	r := newTestServer(NewInMemoryRepo())

	body := []byte(`{"title":"   "}`)
	req := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422, got %d, body=%s", rec.Code, rec.Body.String())
	}

	var resp errResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse error JSON: %v", err)
	}
	if resp.Error != "validation_error" {
		t.Fatalf("expected error 'validation_error', got %q", resp.Error)
	}
	if len(resp.Details) == 0 {
		t.Fatalf("expected at least 1 field error")
	}
	found := false
	for _, d := range resp.Details {
		if d.Field == "title" && strings.Contains(d.Message, "required") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected title 'required' validation error, details=%v", resp.Details)
	}
}

func TestPostTasks_TitleTooLong_ValidationError(t *testing.T) {
	r := newTestServer(NewInMemoryRepo())

	long := strings.Repeat("x", 201) // > 200
	body := []byte(`{"title":"` + long + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422, got %d, body=%s", rec.Code, rec.Body.String())
	}

	var resp errResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse error JSON: %v", err)
	}
	if resp.Error != "validation_error" {
		t.Fatalf("expected error 'validation_error', got %q", resp.Error)
	}
	found := false
	for _, d := range resp.Details {
		if d.Field == "title" && strings.Contains(d.Message, "at most 200") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected title 'max length' validation error, details=%v", resp.Details)
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

	var resp errResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse error JSON: %v", err)
	}
	if resp.Error != "invalid_json" {
		t.Errorf("expected error 'invalid_json', got %q", resp.Error)
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

	var resp errResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse error JSON: %v", err)
	}
	if resp.Error != "unexpected_error" {
		t.Errorf("expected error 'unexpected_error', got %q", resp.Error)
	}
}
