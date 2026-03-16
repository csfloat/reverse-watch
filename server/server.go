package server

import (
	"net/http"

	"reverse-watch/api"
	"reverse-watch/config"
	"reverse-watch/domain/repository"
	rwmiddleware "reverse-watch/middleware"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Server struct {
	r chi.Router
}

func New(cfg config.Config, factory repository.Factory) (*Server, error) {
	r := chi.NewRouter()

	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)

	if cfg.TrustProxy {
		r.Use(rwmiddleware.CloudflareIP)
	}

	r.Use(rwmiddleware.FactoryMiddleware(factory))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/index.html")
	})

	r.Mount("/api", api.Router())

	return &Server{
		r: r,
	}, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.r.ServeHTTP(w, r)
}
