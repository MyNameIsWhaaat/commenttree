package storage

import (
	"context"

	"github.com/MyNameIsWhaaat/commenttree/internal/comment/model"
)

type Repository interface {
	Create(ctx context.Context, parentID int64, text string) (model.Comment, error)
	GetTreePage(ctx context.Context, parentID int64, page, limit int, sort model.Sort) (model.TreePage, error)
	DeleteSubtree(ctx context.Context, id int64) (int, error)
	Search(ctx context.Context, q string, page, limit int, sort model.Sort) (model.SearchPage, error)
	Exists(ctx context.Context, id int64) (bool, error)
	GetSubtree(ctx context.Context, id int64, sort model.Sort) (model.CommentNode, error)
	GetPath(ctx context.Context, id int64) ([]model.CommentPathItem, error)
}