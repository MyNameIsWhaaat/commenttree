package model

type CommentPathItem struct {
	ID       int64  `json:"id"`
	ParentID int64  `json:"parent_id"`
	Text     string `json:"text"`
}
