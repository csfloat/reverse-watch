package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"reverse-watch/api"
	"reverse-watch/config"
	"reverse-watch/database"
	"reverse-watch/logging"
	"reverse-watch/services/private"
	"reverse-watch/services/public"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

func main() {
	logging.Initialize()
	cfg := config.Load()

	logging.Log.Info("Starting Unified Reversal Database")

	privateDB, err := database.InitializePrivateDB(cfg)
	if err != nil {
		logging.Log.Fatalf("failed to initialize private database: %v", err)
	}
	publicDB, err := database.InitializePublicDB(cfg)
	if err != nil {
		logging.Log.Fatalf("failed to initialize public database: %v", err)
	}

	privateService := private.NewService(privateDB)
	publicService := public.NewService(publicDB)

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(render.SetContentType(render.ContentTypeJSON))

	r.Mount("/api", api.Router(privateService, publicService))

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	srv := http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%s", cfg.HTTP.Port),
		Handler: r,
	}

	go func() {
		logging.Log.Infof("Starting HTTP Server on %v", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()

	<-done

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer func() {
		cancel()
	}()

	if err := srv.Shutdown(ctx); err != nil {
		panic(err)
	}
}
