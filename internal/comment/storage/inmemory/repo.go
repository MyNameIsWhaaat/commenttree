package inmemory

import (
	"context"
	"sync"
	"time"

	"github.com/MyNameIsWhaaat/commenttree/internal/comment/model"
	// "github.com/MyNameIsWhaaat/commenttree/internal/comment/storage"
)

type Repo struct {
	mu sync.RWMutex

	nextID int64
	byID   map[int64]model.Comment
}

// var _ storage.Repository = (*Repo)(nil)

func New() *Repo {
	return &Repo{
		nextID: 1,
		byID:   make(map[int64]model.Comment),
	}
}

func (r *Repo) Exists(ctx context.Context, id int64) (bool, error) {
	_ = ctx
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.byID[id]
	return ok, nil
}

func (r *Repo) Create(ctx context.Context, parentID int64, text string) (model.Comment, error) {
	_ = ctx

	r.mu.Lock()
	defer r.mu.Unlock()

	c := model.Comment{
		ID:        r.nextID,
		ParentID:  parentID,
		Text:      text,
		CreatedAt: time.Now().UTC(),
	}
	r.nextID++
	r.byID[c.ID] = c

	return c, nil
}