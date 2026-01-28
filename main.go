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
	"reverse-watch/logging"
	"reverse-watch/server"
)

func main() {
	logging.Initialize()
	cfg := config.Load()
	models.InitSnowflakeGenerator(0, 0)

	logging.Log.Info("Starting Reverse Watch")

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	srv := server.New(cfg)
	httpSrv := http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%s", cfg.HTTP.Port),
		Handler: srv,
	}

	go func() {
		logging.Log.Infof("Starting HTTP Server on %v", httpSrv.Addr)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()

	<-done

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Close db connections
	if err := srv.Close(); err != nil {
		panic(err)
	}

	if err := httpSrv.Shutdown(ctx); err != nil {
		panic(err)
	}
}
