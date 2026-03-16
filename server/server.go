package server

import (
	"net/http"
	"regexp"
	"strings"

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

	firefoxExtensionOrigin := regexp.MustCompile("^moz-extension://[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")

	r.Use(cors.Handler(cors.Options{
		AllowOriginFunc: func(r *http.Request, origin string) bool {
			for _, allowedOrigin := range cfg.HTTP.AllowedOrigins {
				if allowedOrigin == origin {
					return true
				}
			}

			// Firefox extension IDs are randomly generated for each user.
			// Therefore, we're scoping requests made from Firefox extensions to specific endpoints only.
			if firefoxExtensionOrigin.MatchString(origin) {
				if strings.HasPrefix(r.RequestURI, "/api/v1/users/") {
					return true
				}
			}
			return false
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
