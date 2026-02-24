package http

import (
	"encoding/json"
	"errors"
	stdhttp "net/http"
	"strconv"

	"github.com/MyNameIsWhaaat/commenttree/internal/comment/model"
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

func (h *Handler) GetComments(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	q := r.URL.Query()

	parentID := int64(0)
	if v := q.Get("parent"); v != "" {
		parsed, err := parseInt64(v)
		if err != nil {
			writeJSON(w, stdhttp.StatusBadRequest, map[string]any{"error": "invalid parent"})
			return
		}
		parentID = parsed
	}

	page := 1
	if v := q.Get("page"); v != "" {
		parsed, err := parseInt(v)
		if err != nil {
			writeJSON(w, stdhttp.StatusBadRequest, map[string]any{"error": "invalid page"})
			return
		}
		page = parsed
	}

	limit := 20
	if v := q.Get("limit"); v != "" {
		parsed, err := parseInt(v)
		if err != nil {
			writeJSON(w, stdhttp.StatusBadRequest, map[string]any{"error": "invalid limit"})
			return
		}
		limit = parsed
	}

	sortMode := model.Sort(q.Get("sort"))

	res, err := h.svc.GetTreePage(r.Context(), parentID, page, limit, sortMode)
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

	writeJSON(w, stdhttp.StatusOK, res)
}

func (h *Handler) DeleteComment(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	idStr := r.PathValue("id")
	id, err := parseInt64(idStr)
	if err != nil || id <= 0 {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]any{"error": "invalid id"})
		return
	}

	deleted, err := h.svc.DeleteSubtree(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidInput):
			writeJSON(w, stdhttp.StatusBadRequest, map[string]any{"error": "invalid input"})
		case errors.Is(err, service.ErrNotFound):
			writeJSON(w, stdhttp.StatusNotFound, map[string]any{"error": "not found"})
		default:
			writeJSON(w, stdhttp.StatusInternalServerError, map[string]any{"error": "internal error"})
		}
		return
	}

	writeJSON(w, stdhttp.StatusOK, map[string]any{"deleted": deleted})
}

func writeJSON(w stdhttp.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func parseInt(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func parseInt64(s string) (int64, error) {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return n, nil
}