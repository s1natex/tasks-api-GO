package tasks

import (
	"errors"
	"sync"
	"time"
)

var ErrTitleRequired = errors.New("title required")

type Repository interface {
	Create(title string) (Task, error)
	List() []Task
}

type InMemoryRepo struct {
	mu    sync.Mutex
	seq   int64
	store map[int64]Task
}

func NewInMemoryRepo() *InMemoryRepo {
	return &InMemoryRepo{
		store: make(map[int64]Task),
	}
}

func (r *InMemoryRepo) Create(title string) (Task, error) {
	if title == "" {
		return Task{}, ErrTitleRequired
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.seq++
	t := Task{
		ID:        r.seq,
		Title:     title,
		Done:      false,
		CreatedAt: time.Now().UTC(),
	}
	r.store[t.ID] = t
	return t, nil
}

func (r *InMemoryRepo) List() []Task {
	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]Task, 0, len(r.store))
	for _, t := range r.store {
		out = append(out, t)
	}
	return out
}
