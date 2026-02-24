package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	commenthttp "github.com/MyNameIsWhaaat/commenttree/internal/comment/handler/http"

	"github.com/MyNameIsWhaaat/commenttree/internal/comment/service"
	"github.com/MyNameIsWhaaat/commenttree/internal/comment/storage/postgres"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/redis/go-redis/v9"
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

	var rdb *redis.Client
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "redis:6379"
	}

	if os.Getenv("REDIS_DISABLED") == "" {
		rdb = redis.NewClient(&redis.Options{Addr: redisAddr})
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := rdb.Ping(ctx).Err(); err != nil {
			log.Printf("warning: redis not available: %v", err)
			rdb = nil
		}
	}

	svc := service.New(repo, rdb)
	h := commenthttp.New(svc)

	addr := ":" + port
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, h.Routes()))
}