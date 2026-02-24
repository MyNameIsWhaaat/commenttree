package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	commenthttp "github.com/MyNameIsWhaaat/commenttree/internal/comment/handler/http"

	"github.com/MyNameIsWhaaat/commenttree/internal/comment/service"
	"github.com/MyNameIsWhaaat/commenttree/internal/comment/storage/postgres"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is required")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	repo := postgres.New(db)
	svc := service.New(repo)
	h := commenthttp.New(svc)

	addr := ":" + port
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, h.Routes()))
}