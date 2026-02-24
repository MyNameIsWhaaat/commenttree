package service

import (
	"context"

	"github.com/MyNameIsWhaaat/commenttree/internal/comment/model"
)

type CommentService interface {
	Create(ctx context.Context, parentID int64, text string) (model.Comment, error)
	GetTreePage(ctx context.Context, parentID int64, page, limit int, sort model.Sort) (model.TreePage, error)
	DeleteSubtree(ctx context.Context, id int64) (deleted int, err error)
	// Search(ctx context.Context, q string, page, limit int, sort model.Sort) (model.SearchPage, error)
}