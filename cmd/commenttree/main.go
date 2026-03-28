package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	commenthttp "github.com/MyNameIsWhaaat/commenttree/internal/comment/handler/http"
	"github.com/MyNameIsWhaaat/commenttree/internal/comment/service"
	"github.com/MyNameIsWhaaat/commenttree/internal/comment/storage/postgres"
	"github.com/redis/go-redis/v9"
	pgxdriver "github.com/wb-go/wbf/dbpg/pgx-driver"
	wbflogger "github.com/wb-go/wbf/logger"
	"github.com/wb-go/wbf/zlog"
)

func main() {
	zlog.InitConsole()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		zlog.Logger.Fatal().Msg("DATABASE_URL is required")
	}

	appLogger, err := wbflogger.InitLogger(
		wbflogger.ZerologEngine,
		"commenttree",
		"dev",
	)
	if err != nil {
		zlog.Logger.Fatal().Err(err).Msg("failed to init wbf logger")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pg, err := pgxdriver.New(dsn, appLogger)
	if err != nil {
		zlog.Logger.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer pg.Close()

	if err := pg.Ping(ctx); err != nil {
		zlog.Logger.Fatal().Err(err).Msg("failed to ping database")
	}

	repo := postgres.New(pg.Pool)

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
			zlog.Logger.Warn().Err(err).Msg("redis not available")
			_ = rdb.Close()
			rdb = nil
		}
	}

	svc := service.New(repo, rdb)
	h := commenthttp.New(svc)

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           h.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)

	go func() {
		zlog.Logger.Info().Str("addr", srv.Addr).Msg("server starting")
		errCh <- srv.ListenAndServe()
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(stop)

	select {
	case sig := <-stop:
		zlog.Logger.Info().Str("signal", sig.String()).Msg("shutdown signal received")
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			zlog.Logger.Error().Err(err).Msg("server error")
		}
	}

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		zlog.Logger.Error().Err(err).Msg("http shutdown error")
	} else {
		zlog.Logger.Info().Msg("http server stopped")
	}

	if rdb != nil {
		_ = rdb.Close()
	}
}
