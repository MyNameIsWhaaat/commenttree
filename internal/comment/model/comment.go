package model

import "time"

type Comment struct {
	ID        int64     `json:"id"`
	ParentID  int64     `json:"parent_id"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

type CommentNode struct {
	Comment
	Children []CommentNode `json:"children"`
}

type TreePage struct {
	Items []CommentNode `json:"items"`
	Page  int           `json:"page"`
	Limit int           `json:"limit"`
	Total int           `json:"total"`
}
