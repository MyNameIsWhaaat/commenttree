package http

import (
	"encoding/json"
	"errors"
	stdhttp "net/http"

	"github.com/MyNameIsWhaaat/commenttree/internal/comment/service"
)

type Handler struct {
	svc service.CommentService
}

func New(svc service.CommentService) *Handler {
	return &Handler{svc: svc}
}

type createCommentRequest struct {
	ParentID int64  `json:"parent_id"`
	Text     string `json:"text"`
}

func (h *Handler) CreateComment(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	var req createCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]any{"error": "bad json"})
		return
	}

	c, err := h.svc.Create(r.Context(), req.ParentID, req.Text)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidInput):
			writeJSON(w, stdhttp.StatusBadRequest, map[string]any{"error": "invalid input"})
		case errors.Is(err, service.ErrNotFound):
			writeJSON(w, stdhttp.StatusNotFound, map[string]any{"error": "parent not found"})
		default:
			writeJSON(w, stdhttp.StatusInternalServerError, map[string]any{"error": "internal error"})
		}
		return
	}

	writeJSON(w, stdhttp.StatusCreated, c)
}

func writeJSON(w stdhttp.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}