package tasks

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type createTaskRequest struct {
	Title string `json:"title"`
}

type errResponse struct {
	Error string `json:"error"`
}

func RegisterRoutes(r chi.Router, repo Repository) {
	r.Post("/tasks", createTask(repo))
	r.Get("/tasks", listTasks(repo))
}

func createTask(repo Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var req createTaskRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(errResponse{Error: "invalid JSON"})
			return
		}

		t, err := repo.Create(req.Title)
		if err != nil {
			if err == ErrTitleRequired {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(errResponse{Error: "title required"})
				return
			}
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(errResponse{Error: "unexpected error"})
			return
		}

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(t)
	}
}

func listTasks(repo Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		tasks, err := repo.List()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(errResponse{Error: "unexpected error"})
			return
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(tasks)
	}
}
