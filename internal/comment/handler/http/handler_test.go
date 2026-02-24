package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"database/sql"

	handler "github.com/MyNameIsWhaaat/commenttree/internal/comment/handler/http"
	"github.com/MyNameIsWhaaat/commenttree/internal/comment/model"
	"github.com/MyNameIsWhaaat/commenttree/internal/comment/service"
	inm "github.com/MyNameIsWhaaat/commenttree/internal/comment/storage/inmemory"
)

// fakeRepo wraps inmemory.Repo and implements missing methods required by storage.Repository
type fakeRepo struct{ *inm.Repo }

func (f *fakeRepo) Search(ctx context.Context, q string, page, limit int, sort model.Sort) (model.SearchPage, error) {
	return model.SearchPage{}, nil
}

func (f *fakeRepo) GetSubtree(ctx context.Context, id int64, sort model.Sort) (model.CommentNode, error) {
	return model.CommentNode{}, sql.ErrNoRows
}

func (f *fakeRepo) GetPath(ctx context.Context, id int64) ([]model.CommentPathItem, error) {
	return nil, sql.ErrNoRows
}

func newServer() (*httptest.Server, *fakeRepo) {
	repo := &fakeRepo{inm.New()}
	svc := service.New(repo, nil)
	h := handler.New(svc)
	srv := httptest.NewServer(h.Routes())
	return srv, repo
}

func TestCreateGetDeleteComments(t *testing.T) {
	srv, _ := newServer()
	defer srv.Close()

	// create root
	reqBody := map[string]any{"parent_id": 0, "text": "root"}
	b, _ := json.Marshal(reqBody)
	res, err := http.Post(srv.URL+"/comments", "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("post create: %v", err)
	}
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 created, got %d", res.StatusCode)
	}
	var created model.Comment
	if err := json.NewDecoder(res.Body).Decode(&created); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	_ = res.Body.Close()

	// create child
	reqBody = map[string]any{"parent_id": created.ID, "text": "child"}
	b, _ = json.Marshal(reqBody)
	res, err = http.Post(srv.URL+"/comments", "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("post create child: %v", err)
	}
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 created for child, got %d", res.StatusCode)
	}
	_ = res.Body.Close()

	// list children for parent
	res, err = http.Get(srv.URL + "/comments?parent=" + strconv.FormatInt(created.ID, 10))
	if err != nil {
		t.Fatalf("get comments: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 ok, got %d", res.StatusCode)
	}
	var tp model.TreePage
	if err := json.NewDecoder(res.Body).Decode(&tp); err != nil {
		t.Fatalf("decode tree page: %v", err)
	}
	_ = res.Body.Close()
	if tp.Total != 1 {
		t.Fatalf("expected total 1 child, got %d", tp.Total)
	}

	// delete subtree (root)
	client := &http.Client{}
	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/comments/"+strconv.FormatInt(created.ID, 10), nil)
	res, err = client.Do(req)
	if err != nil {
		t.Fatalf("delete request: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("expected 200 on delete, got %d: %s", res.StatusCode, string(body))
	}
}

func TestHandlerValidationErrors(t *testing.T) {
	srv, _ := newServer()
	defer srv.Close()

	// invalid JSON
	res, err := http.Post(srv.URL+"/comments", "application/json", bytes.NewReader([]byte("{bad json")))
	if err != nil {
		t.Fatalf("post bad json: %v", err)
	}
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad json, got %d", res.StatusCode)
	}
	_ = res.Body.Close()

	// invalid page param
	res, err = http.Get(srv.URL + "/comments?page=notanint")
	if err != nil {
		t.Fatalf("get invalid page: %v", err)
	}
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid page, got %d", res.StatusCode)
	}
	_ = res.Body.Close()

	// invalid limit param
	res, err = http.Get(srv.URL + "/comments?limit=notanint")
	if err != nil {
		t.Fatalf("get invalid limit: %v", err)
	}
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid limit, got %d", res.StatusCode)
	}
	_ = res.Body.Close()

	// delete with invalid id in path
	client := &http.Client{}
	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/comments/abc", nil)
	res, err = client.Do(req)
	if err != nil {
		t.Fatalf("delete invalid id: %v", err)
	}
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid id, got %d", res.StatusCode)
	}
	_ = res.Body.Close()

	// get path with invalid id query
	res, err = http.Get(srv.URL + "/comments/path?id=xyz")
	if err != nil {
		t.Fatalf("get path invalid id: %v", err)
	}
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid path id, got %d", res.StatusCode)
	}
	_ = res.Body.Close()

	// search with empty query (should be 400)
	res, err = http.Get(srv.URL + "/comments/search?q=   ")
	if err != nil {
		t.Fatalf("search empty q: %v", err)
	}
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty search q, got %d", res.StatusCode)
	}
	_ = res.Body.Close()
}
