package service

import (
	"context"
	"database/sql"
	"testing"

	"github.com/MyNameIsWhaaat/commenttree/internal/comment/model"
	inm "github.com/MyNameIsWhaaat/commenttree/internal/comment/storage/inmemory"
)

// fakeRepo wraps inmemory.Repo and implements remaining methods of storage.Repository
type fakeRepo struct {
	*inm.Repo
}

func (f *fakeRepo) Search(ctx context.Context, q string, page, limit int, sort model.Sort) (model.SearchPage, error) {
	return model.SearchPage{}, nil
}

func (f *fakeRepo) GetSubtree(ctx context.Context, id int64, sort model.Sort) (model.CommentNode, error) {
	return model.CommentNode{}, sql.ErrNoRows
}

func (f *fakeRepo) GetPath(ctx context.Context, id int64) ([]model.CommentPathItem, error) {
	return nil, sql.ErrNoRows
}

func TestCreateValidation(t *testing.T) {
	repo := &fakeRepo{inm.New()}
	svc := New(repo)

	_, err := svc.Create(context.Background(), 0, "   ")
	if err == nil {
		t.Fatalf("expected error for empty text, got nil")
	}
}

func TestCreateParentNotFound(t *testing.T) {
	repo := &fakeRepo{inm.New()}
	svc := New(repo)

	_, err := svc.Create(context.Background(), 9999, "hello")
	if err == nil {
		t.Fatalf("expected ErrNotFound for missing parent, got nil")
	}
}

func TestCreateAndDeleteSubtree(t *testing.T) {
	ctx := context.Background()
	repo := &fakeRepo{inm.New()}
	svc := New(repo)

	root, err := svc.Create(ctx, 0, "root")
	if err != nil {
		t.Fatalf("create root: %v", err)
	}

	_, err = svc.Create(ctx, root.ID, "child1")
	if err != nil {
		t.Fatalf("create child1: %v", err)
	}
	_, err = svc.Create(ctx, root.ID, "child2")
	if err != nil {
		t.Fatalf("create child2: %v", err)
	}

	// ensure GetTreePage reports total 2 children for parent root
	tp, err := svc.GetTreePage(ctx, root.ID, 1, 10, model.SortCreatedAtDesc)
	if err != nil {
		t.Fatalf("GetTreePage: %v", err)
	}
	if tp.Total != 2 {
		t.Fatalf("expected total 2 children, got %d", tp.Total)
	}

	deleted, err := svc.DeleteSubtree(ctx, root.ID)
	if err != nil {
		t.Fatalf("DeleteSubtree: %v", err)
	}
	if deleted != 3 {
		t.Fatalf("expected deleted 3 (root+2 children), got %d", deleted)
	}
}

func TestGetTreePagePagination(t *testing.T) {
	ctx := context.Background()
	repo := &fakeRepo{inm.New()}
	svc := New(repo)

	// create 5 top-level comments
	for i := 0; i < 5; i++ {
		_, err := svc.Create(ctx, 0, "c")
		if err != nil {
			t.Fatalf("create: %v", err)
		}
	}

	tp, err := svc.GetTreePage(ctx, 0, 1, 2, model.SortCreatedAtDesc)
	if err != nil {
		t.Fatalf("GetTreePage: %v", err)
	}
	if tp.Total != 5 {
		t.Fatalf("expected total 5, got %d", tp.Total)
	}
	if len(tp.Items) != 2 {
		t.Fatalf("expected page size 2, got %d", len(tp.Items))
	}
}
