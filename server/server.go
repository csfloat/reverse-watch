package server

import (
	"errors"
	"fmt"
	"net/http"

	"reverse-watch/config"
	"reverse-watch/domain/repository"
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

	privateRepo repository.PrivateRepository
	publicRepo  repository.PublicRepository
}

func New(cfg config.Config) (*Server, error) {
	keygen := secret.NewKeyGenerator(cfg.Environment)
	privateRepo, err := private.NewPrivateRepository(cfg, keygen)
	if err != nil {
		logging.Log.Errorf("failed to create private repository: %v", err)
		return nil, fmt.Errorf("failed to create private repository: %v", err)
	}
	publicRepo, err := public.NewPublicRepository(cfg)
	if err != nil {
		privateRepo.Close()
		logging.Log.Errorf("failed to create public repository: %v", err)
		return nil, fmt.Errorf("failed to create public repository: %v", err)
	}

	r := chi.NewRouter()

	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)

	f := factory.NewFactory(privateRepo, publicRepo)
	r.Use(rwmiddleware.FactoryMiddleware(f))

	// TODO(zach): Define routes

	return &Server{
		r:           r,
		privateRepo: privateRepo,
		publicRepo:  publicRepo,
	}, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.r.ServeHTTP(w, r)
}

func (s *Server) Close() error {
	var errs []error
	if err := s.privateRepo.Close(); err != nil {
		errs = append(errs, err)
	}
	if err := s.publicRepo.Close(); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}
