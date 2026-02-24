package main

import (
	"log"
	"net/http"
	"os"

	commenthttp "github.com/MyNameIsWhaaat/commenttree/internal/comment/handler/http"

	"github.com/MyNameIsWhaaat/commenttree/internal/comment/service"
	"github.com/MyNameIsWhaaat/commenttree/internal/comment/storage/inmemory"
)

func main() {
	repo := inmemory.New()
	svc := service.New(repo)
	h := commenthttp.New(svc)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	addr := ":" + port
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, h.Routes()))
}