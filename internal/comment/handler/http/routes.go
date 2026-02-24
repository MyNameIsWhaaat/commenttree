package http

import (
	stdhttp "net/http"
)

func (h *Handler) Routes() stdhttp.Handler {
	mux := stdhttp.NewServeMux()

	// /comments handles GET (list) and POST (create)
	mux.HandleFunc("/comments", func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		switch r.Method {
		case stdhttp.MethodGet:
			h.GetComments(w, r)
		case stdhttp.MethodPost:
			h.CreateComment(w, r)
		default:
			stdhttp.NotFound(w, r)
		}
	})

	// /comments/{id} for operations like DELETE
	mux.HandleFunc("/comments/", func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		if r.Method == stdhttp.MethodDelete {
			h.DeleteComment(w, r)
			return
		}
		stdhttp.NotFound(w, r)
	})

	mux.HandleFunc("/comments/search", h.SearchComments)
	mux.HandleFunc("/comments/path", h.GetPath)
	mux.HandleFunc("/comments/subtree", h.GetSubtree)

	mux.HandleFunc("/healthz", func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(stdhttp.StatusOK)
		_, _ = w.Write([]byte(`{"result":"ok"}`))
	})

	mux.HandleFunc("/", func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		if r.Method != stdhttp.MethodGet {
			stdhttp.NotFound(w, r)
			return
		}
		stdhttp.ServeFile(w, r, "./web/index.html")
	})

	mux.Handle("/static/", stdhttp.StripPrefix("/static/", stdhttp.FileServer(stdhttp.Dir("./web"))))

	return mux
}
