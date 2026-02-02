package server

import (
	"fmt"
	"net/http"

	"reverse-watch/config"
	"reverse-watch/logging"
	rwmiddleware "reverse-watch/middleware"
	"reverse-watch/repository/factory"
	"reverse-watch/repository/private"
	"reverse-watch/repository/public"
	"reverse-watch/secret"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Server struct {
	r chi.Router
}

func New(cfg config.Config) (*Server, error) {
	keygen := secret.NewKeyGenerator(cfg.Environment)
	privateDB, err := private.NewPrivateRepository(cfg, keygen)
	if err != nil {
		logging.Log.Errorf("failed to create private repository: %v", err)
		return nil, fmt.Errorf("failed to create private repository: %v", err)
	}
	publicDB, err := public.NewPublicRepository(cfg)
	if err != nil {
		logging.Log.Errorf("failed to create public repository: %v", err)
		return nil, fmt.Errorf("failed to create public repository: %v", err)
	}

	r := chi.NewRouter()

	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)

	f := factory.NewFactory(privateDB, publicDB, keygen)
	r.Use(rwmiddleware.FactoryMiddleware(f))

	// TODO(zach): Define routes

	return &Server{
		r: r,
	}, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.r.ServeHTTP(w, r)
}
