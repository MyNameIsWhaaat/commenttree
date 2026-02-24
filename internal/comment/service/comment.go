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

func validateText(text string) error {
	t := strings.TrimSpace(text)
	if t == "" || len(t) > 2000 {
		return ErrInvalidInput
	}
	return nil
}