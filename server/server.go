package server

import (
	"fmt"
	"net/http"

	"reverse-watch/config"
	"reverse-watch/domain/repository"
	"reverse-watch/logging"
	rwmiddleware "reverse-watch/middleware"
	"reverse-watch/repository/factory"
	"reverse-watch/secret"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Server struct {
	r       chi.Router
	factory repository.Factory
}

func New(cfg config.Config) (*Server, error) {
	keygen := secret.NewKeyGenerator(cfg.Environment)
	f, err := factory.NewFactory(cfg, keygen)
	if err != nil {
		logging.Log.Errorf("failed to create factory: %v", err)
		return nil, fmt.Errorf("failed to create factory: %w", err)
	}

	r := chi.NewRouter()

	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)

	r.Use(rwmiddleware.FactoryMiddleware(f))

	// TODO(zach): Define routes

	return &Server{
		r:       r,
		factory: f,
	}, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.r.ServeHTTP(w, r)
}

func (s *Server) Close() error {
	return s.factory.Close()
}
