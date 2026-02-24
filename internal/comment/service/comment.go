package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/MyNameIsWhaaat/commenttree/internal/comment/model"
	"github.com/MyNameIsWhaaat/commenttree/internal/comment/storage"
	"github.com/redis/go-redis/v9"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrInvalidInput = errors.New("invalid input")
)

type commentService struct {
	repo storage.Repository
	rdb  *redis.Client
}

func New(repo storage.Repository, rdb *redis.Client) CommentService {
	return &commentService{repo: repo, rdb: rdb}
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
	c, err := s.repo.Create(ctx, parentID, text)
	if err != nil {
		return model.Comment{}, err
	}

	if s.rdb != nil {
		_ = s.invalidateTreeCache(ctx, 0)

		_ = s.invalidateTreeCache(ctx, parentID)

		if parentID != 0 {
			path, err := s.repo.GetPath(ctx, parentID)
			if err == nil && len(path) > 0 {
				rootID := path[0].ID
				_ = s.invalidateSubtreeCache(ctx, rootID)
			}
		}
	}
	return c, nil
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

	if s.rdb != nil {
		key := s.treeCacheKey(parentID, page, limit, sortMode)
		data, err := s.rdb.Get(ctx, key).Bytes()
		if err == nil {
			var tp model.TreePage
			if jerr := json.Unmarshal(data, &tp); jerr == nil {
				return tp, nil
			}
		}

		tp, err := s.repo.GetTreePage(ctx, parentID, page, limit, sortMode)
		if err != nil {
			return model.TreePage{}, err
		}
		if b, jerr := json.Marshal(tp); jerr == nil {
			_ = s.rdb.Set(ctx, key, b, 2*time.Minute).Err()
		}
		return tp, nil
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

	deleted, err := s.repo.DeleteSubtree(ctx, id)
	if err != nil {
		return 0, err
	}
	if s.rdb != nil {
		_ = s.invalidateAllTreeCache(ctx)
		_ = s.invalidateAllSubtreeCache(ctx)
	}
	return deleted, nil
}

func (s *commentService) treeCacheKey(parentID int64, page, limit int, sort model.Sort) string {
	return fmt.Sprintf("tree:parent:%d:page:%d:limit:%d:sort:%s", parentID, page, limit, string(sort))
}

func (s *commentService) subtreeCacheKey(id int64, sort model.Sort, ver int64) string {
	return fmt.Sprintf("subtree:v:%d:root:%d:sort:%s", ver, id, string(sort))
}

func (s *commentService) invalidateTreeCache(ctx context.Context, parentID int64) error {
	pattern := fmt.Sprintf("tree:parent:%d:*", parentID)
	keys, err := s.rdb.Keys(ctx, pattern).Result()
	if err != nil {
		return err
	}
	if len(keys) == 0 {
		return nil
	}
	return s.rdb.Del(ctx, keys...).Err()
}

func (s *commentService) invalidateAllTreeCache(ctx context.Context) error {
	keys, err := s.rdb.Keys(ctx, "tree:parent:*").Result()
	if err != nil {
		return err
	}
	if len(keys) == 0 {
		return nil
	}
	return s.rdb.Del(ctx, keys...).Err()
}

func (s *commentService) invalidateSubtreeCache(ctx context.Context, id int64) error {
	pattern := fmt.Sprintf("subtree:root:%d:*", id)
	keys, err := s.rdb.Keys(ctx, pattern).Result()
	if err != nil {
		return err
	}
	if len(keys) == 0 {
		return nil
	}
	return s.rdb.Del(ctx, keys...).Err()
}

func (s *commentService) invalidateAllSubtreeCache(ctx context.Context) error {
	keys, err := s.rdb.Keys(ctx, "subtree:root:*").Result()
	if err != nil {
		return err
	}
	if len(keys) == 0 {
		return nil
	}
	return s.rdb.Del(ctx, keys...).Err()
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

func (s *commentService) GetPath(ctx context.Context, id int64) ([]model.CommentPathItem, error) {
	if id <= 0 {
		return nil, ErrInvalidInput
	}
	items, err := s.repo.GetPath(ctx, id)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (s *commentService) GetSubtree(ctx context.Context, id int64, sortMode model.Sort) (model.CommentNode, error) {
	if id <= 0 {
		return model.CommentNode{}, ErrInvalidInput
	}
	if sortMode == "" {
		sortMode = model.SortCreatedAtDesc
	}
	if sortMode != model.SortCreatedAtAsc && sortMode != model.SortCreatedAtDesc {
		return model.CommentNode{}, ErrInvalidInput
	}

	if s.rdb != nil {
		ver := s.getVer(ctx, s.subtreeVerKey(id))
		key := s.subtreeCacheKey(id, sortMode, ver)

		if b, err := s.rdb.Get(ctx, key).Bytes(); err == nil {
			var node model.CommentNode
			if jerr := json.Unmarshal(b, &node); jerr == nil {
				return node, nil
			}
		}

		node, err := s.repo.GetSubtree(ctx, id, sortMode)
		if err == sql.ErrNoRows {
			return model.CommentNode{}, ErrNotFound
		}
		if err != nil {
			return model.CommentNode{}, err
		}

		if b, jerr := json.Marshal(node); jerr == nil {
			_ = s.rdb.Set(ctx, key, b, 2*time.Minute).Err()
		}
		return node, nil
	}

	n, err := s.repo.GetSubtree(ctx, id, sortMode)
	if err == sql.ErrNoRows {
		return model.CommentNode{}, ErrNotFound
	}
	if err != nil {
		return model.CommentNode{}, err
	}
	return n, nil
}

func (s *commentService) subtreeVerKey(rootID int64) string {
	return fmt.Sprintf("ver:subtree:root:%d", rootID)
}

func (s *commentService) getVer(ctx context.Context, key string) int64 {
	if s.rdb == nil {
		return 1
	}
	v, err := s.rdb.Get(ctx, key).Int64()
	if err == nil && v > 0 {
		return v
	}
	return 1
}

func (s *commentService) bumpVer(ctx context.Context, key string) {
	if s.rdb == nil {
		return
	}

	_ = s.rdb.Incr(ctx, key).Err()
	_ = s.rdb.Expire(ctx, key, 24*time.Hour).Err()
}

func validateText(text string) error {
	t := strings.TrimSpace(text)
	if t == "" || len(t) > 2000 {
		return ErrInvalidInput
	}
	return nil
}
