package inmemory

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/MyNameIsWhaaat/commenttree/internal/comment/model"
)

type Repo struct {
	mu sync.RWMutex

	nextID int64
	byID   map[int64]model.Comment
	children map[int64][]int64
}

func New() *Repo {
	return &Repo{
		nextID:   1,
		byID:     make(map[int64]model.Comment),
		children: make(map[int64][]int64),
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
	r.children[parentID] = append(r.children[parentID], c.ID)

	return c, nil
}

func (r *Repo) GetTreePage(ctx context.Context, parentID int64, page, limit int, sortMode model.Sort) (model.TreePage, error) {
	_ = ctx

	r.mu.RLock()
	defer r.mu.RUnlock()

	childIDs := append([]int64(nil), r.children[parentID]...)
	total := len(childIDs)

	sort.Slice(childIDs, func(i, j int) bool {
		a := r.byID[childIDs[i]].CreatedAt
		b := r.byID[childIDs[j]].CreatedAt
		if sortMode == model.SortCreatedAtAsc {
			return a.Before(b)
		}
		return a.After(b)
	})

	start := (page - 1) * limit
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}
	pageIDs := childIDs[start:end]

	items := make([]model.CommentNode, 0, len(pageIDs))
	for _, id := range pageIDs {
		items = append(items, r.buildNodeLocked(id, sortMode))
	}

	return model.TreePage{
		Items: items,
		Page:  page,
		Limit: limit,
		Total: total,
	}, nil
}

func (r *Repo) buildNodeLocked(id int64, sortMode model.Sort) model.CommentNode {
	c := r.byID[id]
	childIDs := append([]int64(nil), r.children[id]...)

	sort.Slice(childIDs, func(i, j int) bool {
		a := r.byID[childIDs[i]].CreatedAt
		b := r.byID[childIDs[j]].CreatedAt
		if sortMode == model.SortCreatedAtAsc {
			return a.Before(b)
		}
		return a.After(b)
	})

	children := make([]model.CommentNode, 0, len(childIDs))
	for _, cid := range childIDs {
		children = append(children, r.buildNodeLocked(cid, sortMode))
	}

	return model.CommentNode{
		Comment:  c,
		Children: children,
	}
}

func (r *Repo) DeleteSubtree(ctx context.Context, id int64) (int, error) {
	_ = ctx

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.byID[id]; !ok {
		return 0, nil
	}

	toDelete := make([]int64, 0, 16)
	stack := []int64{id}

	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		toDelete = append(toDelete, n)

		for _, ch := range r.children[n] {
			stack = append(stack, ch)
		}
	}

	for _, cid := range toDelete {
		parent := r.byID[cid].ParentID
		r.children[parent] = removeID(r.children[parent], cid)

		delete(r.byID, cid)
		delete(r.children, cid)
	}

	return len(toDelete), nil
}

func removeID(ids []int64, target int64) []int64 {
	out := ids[:0]
	for _, v := range ids {
		if v != target {
			out = append(out, v)
		}
	}
	return append([]int64(nil), out...)
}