package server

import (
	"net/http"

	"reverse-watch/api"
	"reverse-watch/config"
	"reverse-watch/domain/repository"
	rwmiddleware "reverse-watch/middleware"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

type Server struct {
	r chi.Router
}

func New(cfg config.Config, factory repository.Factory) (*Server, error) {
	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{
			"chrome-extension://jjicbefpemnphinccgikpdaagjebbnhg",
		},
		AllowedMethods:   []string{"GET", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)

	if cfg.TrustProxy {
		r.Use(rwmiddleware.CloudflareIP)
	}

	r.Use(rwmiddleware.FactoryMiddleware(factory))

	r.Mount("/api", api.Router())

	return &Server{
		r: r,
	}, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.r.ServeHTTP(w, r)
}
