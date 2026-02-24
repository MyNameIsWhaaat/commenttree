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

type nodePtr struct {
	c        model.Comment
	children []*nodePtr
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

	nodes := make(map[int64]*nodePtr, 256)
	for treeRows.Next() {
		var c model.Comment
		if err := treeRows.Scan(&c.ID, &c.ParentID, &c.Text, &c.CreatedAt); err != nil {
			return model.TreePage{}, err
		}
		nodes[c.ID] = &nodePtr{c: c}
	}
	if err := treeRows.Err(); err != nil {
		return model.TreePage{}, err
	}

	for _, n := range nodes {
		if p, ok := nodes[n.c.ParentID]; ok && n.c.ParentID != 0 {
			p.children = append(p.children, n)
		}
	}

	items := make([]model.CommentNode, 0, len(roots))
	for _, rid := range roots {
		if n, ok := nodes[rid]; ok {
			items = append(items, toValueTree(n, sortMode))
		}
	}

	return model.TreePage{
		Items: items,
		Page:  page,
		Limit: limit,
		Total: total,
	}, nil
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
	var total int
	if err := r.db.QueryRowContext(ctx, `
		SELECT count(*)
		FROM comments
		WHERE search_tsv @@ plainto_tsquery('simple', $1)
	`, q).Scan(&total); err != nil {
		return model.SearchPage{}, err
	}

	if total == 0 {
		return model.SearchPage{
			Items: []model.SearchItem{},
			Page:  page,
			Limit: limit,
			Total: 0,
		}, nil
	}

	orderBy := `rank DESC, created_at DESC`
	switch sortMode {
	case "", model.SortRankDesc:
	case model.SortCreatedAtDesc:
		orderBy = `created_at DESC, rank DESC`
	case model.SortCreatedAtAsc:
		orderBy = `created_at ASC, rank DESC`
	default:
		orderBy = `rank DESC, created_at DESC`
	}

	offset := (page - 1) * limit

	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT
			id,
			parent_id,
			ts_headline('simple', text, plainto_tsquery('simple', $1),
				'StartSel=<mark>, StopSel=</mark>, MaxWords=20, MinWords=10, ShortWord=3, HighlightAll=true') AS snippet,
			ts_rank(search_tsv, plainto_tsquery('simple', $1)) AS rank,
			created_at
		FROM comments
		WHERE search_tsv @@ plainto_tsquery('simple', $1)
		ORDER BY %s
		LIMIT $2 OFFSET $3
	`, orderBy), q, limit, offset)
	if err != nil {
		return model.SearchPage{}, err
	}
	defer rows.Close()

	items := make([]model.SearchItem, 0, limit)
	for rows.Next() {
		var it model.SearchItem
		if err := rows.Scan(&it.ID, &it.ParentID, &it.Snippet, &it.Rank, &it.CreatedAt); err != nil {
			return model.SearchPage{}, err
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return model.SearchPage{}, err
	}

	return model.SearchPage{
		Items: items,
		Page:  page,
		Limit: limit,
		Total: total,
	}, nil
}

func (r *Repo) GetPath(ctx context.Context, id int64) ([]model.CommentPathItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		WITH RECURSIVE p AS (
			SELECT id, parent_id, text
			FROM comments
			WHERE id = $1
			UNION ALL
			SELECT c.id, c.parent_id, c.text
			FROM comments c
			JOIN p ON c.id = p.parent_id
			WHERE p.parent_id <> 0
		)
		SELECT id, parent_id, text
		FROM p
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.CommentPathItem
	for rows.Next() {
		var it model.CommentPathItem
		if err := rows.Scan(&it.ID, &it.ParentID, &it.Text); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, sql.ErrNoRows
	}

	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}

	return items, nil
}

func (r *Repo) GetSubtree(ctx context.Context, id int64, sortMode model.Sort) (model.CommentNode, error) {
	rows, err := r.db.QueryContext(ctx, `
		WITH RECURSIVE t AS (
			SELECT id, parent_id, text, created_at
			FROM comments
			WHERE id = $1

			UNION ALL

			SELECT c.id, c.parent_id, c.text, c.created_at
			FROM comments c
			JOIN t ON c.parent_id = t.id
		)
		SELECT id, parent_id, text, created_at
		FROM t
	`, id)
	if err != nil {
		return model.CommentNode{}, err
	}
	defer rows.Close()

	nodes := make(map[int64]*nodePtr, 256)
	for rows.Next() {
		var c model.Comment
		if err := rows.Scan(&c.ID, &c.ParentID, &c.Text, &c.CreatedAt); err != nil {
			return model.CommentNode{}, err
		}
		nodes[c.ID] = &nodePtr{c: c}
	}
	if err := rows.Err(); err != nil {
		return model.CommentNode{}, err
	}

	root, ok := nodes[id]
	if !ok {
		return model.CommentNode{}, sql.ErrNoRows
	}

	for _, n := range nodes {
		if n.c.ID == id {
			continue
		}
		if p, ok := nodes[n.c.ParentID]; ok {
			p.children = append(p.children, n)
		}
	}

	out := toValueTree(root, sortMode)
	return out, nil
}

func toValueTree(n *nodePtr, sortMode model.Sort) model.CommentNode {
	out := model.CommentNode{Comment: n.c}

	sort.Slice(n.children, func(i, j int) bool {
		a := n.children[i].c.CreatedAt
		b := n.children[j].c.CreatedAt
		if sortMode == model.SortCreatedAtAsc {
			return a.Before(b)
		}
		return a.After(b)
	})

	out.Children = make([]model.CommentNode, 0, len(n.children))
	for _, ch := range n.children {
		out.Children = append(out.Children, toValueTree(ch, sortMode))
	}
	return out
}