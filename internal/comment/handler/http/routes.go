package http

import stdhttp "net/http"

func (h *Handler) Routes() stdhttp.Handler {
	mux := stdhttp.NewServeMux()

	mux.HandleFunc("POST /comments", h.CreateComment)
	mux.HandleFunc("GET /healthz", func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(stdhttp.StatusOK)
		_, _ = w.Write([]byte(`{"result":"ok"}`))
	})
	mux.HandleFunc("GET /comments", h.GetComments)
	mux.HandleFunc("DELETE /comments/{id}", h.DeleteComment)
	mux.HandleFunc("GET /comments/search", h.SearchComments)

	return mux
}