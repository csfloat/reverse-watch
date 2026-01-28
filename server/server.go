package server

import (
	"net/http"

	"reverse-watch/config"
	"reverse-watch/domain/repository"
	"reverse-watch/repository/private"
	"reverse-watch/repository/public"
	"reverse-watch/secret"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Server struct {
	router chi.Router

	privateRepo repository.PrivateRepository
	publicRepo  repository.PublicRepository
}

func New(cfg config.Config) *Server {
	keygen := secret.NewKeyGenerator(cfg.Environment)
	privateRepo, err := private.NewPrivateRepository(cfg, keygen)
	if err != nil {
		panic(err)
	}
	publicRepo, err := public.NewPublicRepository(cfg)
	if err != nil {
		panic(err)
	}

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// TODO(zach): Define routes

	return &Server{
		router:      r,
		privateRepo: privateRepo,
		publicRepo:  publicRepo,
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}
