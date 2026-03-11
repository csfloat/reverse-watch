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

	"reverse-watch/config"
	"reverse-watch/domain/models"
	"reverse-watch/ingestors"
	"reverse-watch/logging"
	"reverse-watch/repository/factory"
	"reverse-watch/secret"
	"reverse-watch/server"
)

func main() {
	logging.Initialize()
	cfg := config.Load()
	models.InitSnowflakeGenerator(0 /* workerID */, 0 /* processID */)

	keygen := secret.NewKeyGenerator(cfg.Environment)
	f, err := factory.NewFactory(cfg, keygen)
	if err != nil {
		logging.Log.Errorf("failed to create factory: %v", err)
		panic(err)
	}

	logging.Log.Info("Starting Reverse Watch")

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	srv, err := server.New(cfg, f)
	if err != nil {
		panic(err)
	}

	ingestorManager := ingestors.New(f, &cfg, logging.Log)
	ingestorManager.StartIngestors()

	httpSrv := &http.Server{
		Addr:              fmt.Sprintf("0.0.0.0:%s", cfg.HTTP.Port),
		Handler:           srv,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		logging.Log.Infof("Starting HTTP Server on %v", httpSrv.Addr)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()

	<-done

	logging.Log.Info("Shutting down server connections gracefully")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpSrv.Shutdown(ctx); err != nil {
		panic(err)
	}

	ingestorManager.Stop()

	if err := f.Close(); err != nil {
		panic(err)
	}
}
