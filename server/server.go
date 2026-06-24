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

			if cfg.HTTP.AllowFirefoxExtensions {
				// Firefox extension IDs are randomly generated for each user.
				// Therefore, we're scoping requests made from Firefox extensions to specific endpoints only.
				if firefoxExtensionOrigin.MatchString(origin) {
					if strings.HasPrefix(r.RequestURI, "/api/v1/users/") {
						return true
					}
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

	// Serve the Astro-built dashboard from web/dist. `npm run build`
	// emits the hashed bundles under web/dist/_astro and copies
	// public/static verbatim to web/dist/static, so we hand both
	// prefixes to a single FileServer and fall back to index.html for
	// the root document.
	fs := http.FileServer(http.Dir("web/dist"))
	r.Handle("/static/*", fs)
	r.Handle("/_astro/*", fs)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/dist/index.html")
	})

	r.Mount("/api", api.Router())

	return &Server{
		r: r,
	}, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.r.ServeHTTP(w, r)
}
