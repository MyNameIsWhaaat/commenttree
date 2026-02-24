package model

type Sort string

const (
	SortCreatedAtAsc  Sort = "created_at_asc"
	SortCreatedAtDesc Sort = "created_at_desc"
	SortRankDesc      Sort = "rank_desc"
)