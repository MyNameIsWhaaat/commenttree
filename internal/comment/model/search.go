package model

import "time"

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