package tasks

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

type createTaskRequest struct {
	Title string `json:"title"`
}

type fieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type errResponse struct {
	Error   string       `json:"error"`
	Details []fieldError `json:"details,omitempty"`
}

func RegisterRoutes(r chi.Router, repo Repository) {
	r.Post("/tasks", createTask(repo))
	r.Get("/tasks", listTasks(repo))
}

func createTask(repo Repository) http.HandlerFunc {
	const maxTitleLen = 200

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var req createTaskRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, errResponse{Error: "invalid_json"})
			return
		}

		if vErrs := validateCreateTask(req.Title, maxTitleLen); len(vErrs) > 0 {
			writeJSON(w, http.StatusUnprocessableEntity, errResponse{
				Error:   "validation_error",
				Details: vErrs,
			})
			return
		}

		t, err := repo.Create(req.Title)
		if err != nil {
			if err == ErrTitleRequired {
				writeJSON(w, http.StatusUnprocessableEntity, errResponse{
					Error: "validation_error",
					Details: []fieldError{
						{Field: "title", Message: "title is required"},
					},
				})
				return
			}
			writeJSON(w, http.StatusInternalServerError, errResponse{Error: "unexpected_error"})
			return
		}

		writeJSON(w, http.StatusCreated, t)
	}
}

func listTasks(repo Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		tasks, err := repo.List()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errResponse{Error: "unexpected_error"})
			return
		}
		writeJSON(w, http.StatusOK, tasks)
	}
}

func validateCreateTask(title string, maxLen int) []fieldError {
	var errs []fieldError

	if strings.TrimSpace(title) == "" {
		errs = append(errs, fieldError{
			Field:   "title",
			Message: "title is required",
		})
	}

	if l := len(title); l > maxLen {
		errs = append(errs, fieldError{
			Field:   "title",
			Message: fmt.Sprintf("title must be at most %d characters", maxLen),
		})
	}

	return errs
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
