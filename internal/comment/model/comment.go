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

type SearchItem struct {
	ID        int64     `json:"id"`
	ParentID  int64     `json:"parent_id"`
	Snippet   string    `json:"snippet"`
	Rank      float64   `json:"rank"`
	CreatedAt time.Time `json:"created_at"`
}

type SearchPage struct {
	Items []SearchItem `json:"items"`
	Page  int          `json:"page"`
	Limit int          `json:"limit"`
	Total int          `json:"total"`
}