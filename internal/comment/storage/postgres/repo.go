package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"sort"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/MyNameIsWhaaat/commenttree/internal/comment/model"
)

type Repo struct {
	db *sql.DB
}

func New(db *sql.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) Exists(ctx context.Context, id int64) (bool, error) {
	var one int
	err := r.db.QueryRowContext(ctx, `SELECT 1 FROM comments WHERE id=$1`, id).Scan(&one)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

func (r *Repo) Create(ctx context.Context, parentID int64, text string) (model.Comment, error) {
	var c model.Comment
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO comments(parent_id, text)
		VALUES ($1, $2)
		RETURNING id, parent_id, text, created_at
	`, parentID, text).Scan(&c.ID, &c.ParentID, &c.Text, &c.CreatedAt)
	if err != nil {
		return model.Comment{}, err
	}
	return c, nil
}

func (r *Repo) GetTreePage(ctx context.Context, parentID int64, page, limit int, sortMode model.Sort) (model.TreePage, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT count(*) FROM comments WHERE parent_id=$1`, parentID).Scan(&total); err != nil {
		return model.TreePage{}, err
	}

	order := "DESC"
	if sortMode == model.SortCreatedAtAsc {
		order = "ASC"
	}

	offset := (page - 1) * limit

	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT id
		FROM comments
		WHERE parent_id=$1
		ORDER BY created_at %s
		LIMIT $2 OFFSET $3
	`, order), parentID, limit, offset)
	if err != nil {
		return model.TreePage{}, err
	}
	defer rows.Close()

	var roots []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return model.TreePage{}, err
		}
		roots = append(roots, id)
	}
	if err := rows.Err(); err != nil {
		return model.TreePage{}, err
	}

	if len(roots) == 0 {
		return model.TreePage{
			Items: []model.CommentNode{},
			Page:  page,
			Limit: limit,
			Total: total,
		}, nil
	}

	treeRows, err := r.db.QueryContext(ctx, `
		WITH RECURSIVE t AS (
			SELECT id, parent_id, text, created_at
			FROM comments
			WHERE id = ANY($1)

			UNION ALL

			SELECT c.id, c.parent_id, c.text, c.created_at
			FROM comments c
			JOIN t ON c.parent_id = t.id
		)
		SELECT id, parent_id, text, created_at
		FROM t
	`, roots)
	if err != nil {
		return model.TreePage{}, err
	}
	defer treeRows.Close()

	nodes := make(map[int64]*model.CommentNode, 256)
	for treeRows.Next() {
		var c model.Comment
		if err := treeRows.Scan(&c.ID, &c.ParentID, &c.Text, &c.CreatedAt); err != nil {
			return model.TreePage{}, err
		}
		n := model.CommentNode{Comment: c}
		nodes[c.ID] = &n
	}
	if err := treeRows.Err(); err != nil {
		return model.TreePage{}, err
	}

	for _, n := range nodes {
		if n.ParentID == 0 {
			continue
		}
		if p, ok := nodes[n.ParentID]; ok {
			p.Children = append(p.Children, *n)
		}
	}

	items := make([]model.CommentNode, 0, len(roots))
	for _, rid := range roots {
		if n, ok := nodes[rid]; ok {
			items = append(items, *n)
		}
	}

	sortChildren(items, sortMode)

	return model.TreePage{
		Items: items,
		Page:  page,
		Limit: limit,
		Total: total,
	}, nil
}

func sortChildren(nodes []model.CommentNode, sortMode model.Sort) {
	less := func(a, b model.CommentNode) bool {
		if sortMode == model.SortCreatedAtAsc {
			return a.CreatedAt.Before(b.CreatedAt)
		}
		return a.CreatedAt.After(b.CreatedAt)
	}

	for i := range nodes {
		if len(nodes[i].Children) > 0 {
			sort.Slice(nodes[i].Children, func(a, b int) bool {
				return less(nodes[i].Children[a], nodes[i].Children[b])
			})
			sortChildren(nodes[i].Children, sortMode)
		}
	}
}

func (r *Repo) DeleteSubtree(ctx context.Context, id int64) (int, error) {
	var deleted int
	err := r.db.QueryRowContext(ctx, `
		WITH RECURSIVE t AS (
			SELECT id FROM comments WHERE id=$1
			UNION ALL
			SELECT c.id FROM comments c JOIN t ON c.parent_id = t.id
		)
		DELETE FROM comments
		WHERE id IN (SELECT id FROM t)
		RETURNING 1
	`, id).Scan(new(int))
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		rows, qerr := r.db.QueryContext(ctx, `
			WITH RECURSIVE t AS (
				SELECT id FROM comments WHERE id=$1
				UNION ALL
				SELECT c.id FROM comments c JOIN t ON c.parent_id = t.id
			)
			DELETE FROM comments
			WHERE id IN (SELECT id FROM t)
			RETURNING id
		`, id)
		if qerr != nil {
			return 0, qerr
		}
		defer rows.Close()

		for rows.Next() {
			deleted++
		}
		if err := rows.Err(); err != nil {
			return 0, err
		}
		return deleted, nil
	}

	return 1, nil
}

func (r *Repo) Search(ctx context.Context, q string, page, limit int, sortMode model.Sort) (model.SearchPage, error) {
	return model.SearchPage{}, fmt.Errorf("not implemented")
}