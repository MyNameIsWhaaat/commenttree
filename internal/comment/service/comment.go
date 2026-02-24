package service

import (
	"context"
	"errors"
	"strings"

	"github.com/MyNameIsWhaaat/commenttree/internal/comment/model"
	"github.com/MyNameIsWhaaat/commenttree/internal/comment/storage"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrInvalidInput = errors.New("invalid input")
)

type commentService struct {
	repo storage.Repository
}

func New(repo storage.Repository) CommentService {
	return &commentService{repo: repo}
}

func (s *commentService) Create(ctx context.Context, parentID int64, text string) (model.Comment, error) {
	if err := validateText(text); err != nil {
		return model.Comment{}, err
	}
	if parentID < 0 {
		return model.Comment{}, ErrInvalidInput
	}
	if parentID != 0 {
		ok, err := s.repo.Exists(ctx, parentID)
		if err != nil {
			return model.Comment{}, err
		}
		if !ok {
			return model.Comment{}, ErrNotFound
		}
	}
	return s.repo.Create(ctx, parentID, text)
}

func (s *commentService) GetTreePage(ctx context.Context, parentID int64, page, limit int, sortMode model.Sort) (model.TreePage, error) {
	if parentID < 0 {
		return model.TreePage{}, ErrInvalidInput
	}
	if page <= 0 || limit <= 0 || limit > 100 {
		return model.TreePage{}, ErrInvalidInput
	}
	if sortMode == "" {
		sortMode = model.SortCreatedAtDesc
	}
	if sortMode != model.SortCreatedAtAsc && sortMode != model.SortCreatedAtDesc {
		return model.TreePage{}, ErrInvalidInput
	}

	if parentID != 0 {
		ok, err := s.repo.Exists(ctx, parentID)
		if err != nil {
			return model.TreePage{}, err
		}
		if !ok {
			return model.TreePage{}, ErrNotFound
		}
	}

	return s.repo.GetTreePage(ctx, parentID, page, limit, sortMode)
}

func (s *commentService) DeleteSubtree(ctx context.Context, id int64) (int, error) {
	if id <= 0 {
		return 0, ErrInvalidInput
	}

	ok, err := s.repo.Exists(ctx, id)
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, ErrNotFound
	}

	return s.repo.DeleteSubtree(ctx, id)
}

func (s *commentService) Search(ctx context.Context, q string, page, limit int, sortMode model.Sort) (model.SearchPage, error) {
	if strings.TrimSpace(q) == "" {
		return model.SearchPage{}, ErrInvalidInput
	}
	if page <= 0 || limit <= 0 || limit > 100 {
		return model.SearchPage{}, ErrInvalidInput
	}

	switch sortMode {
	case "", model.SortRankDesc, model.SortCreatedAtAsc, model.SortCreatedAtDesc:
	default:
		return model.SearchPage{}, ErrInvalidInput
	}

	return s.repo.Search(ctx, q, page, limit, sortMode)
}

func validateText(text string) error {
	t := strings.TrimSpace(text)
	if t == "" || len(t) > 2000 {
		return ErrInvalidInput
	}
	return nil
}