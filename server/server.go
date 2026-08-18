package server

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
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

// webDistDir is the directory (relative to the server's working directory)
// containing the Astro-built dashboard. It is produced by `npm run build`
// in web/ and is intentionally gitignored, so it must exist at runtime.
const webDistDir = "web/dist"

type Server struct {
	r chi.Router
}

// verifyWebDist fails fast when the Astro build output is missing. web/dist is
// gitignored, so any run path that skips the frontend build (e.g. `go run`
// locally, or a deploy that doesn't run `npm run build`) would otherwise
// silently serve 404s for the dashboard. Requiring index.html turns that into
// an obvious startup error with remediation instructions.
func verifyWebDist(dir string) error {
	index := filepath.Join(dir, "index.html")
	if _, err := os.Stat(index); err != nil {
		return fmt.Errorf("dashboard assets not found at %q: the Astro build output (%s) is gitignored and must be built before starting the server. Run `npm ci && npm run build` in web/, or use the Docker image which builds it automatically: %w", index, dir, err)
	}
	return nil
}

func New(cfg config.Config, factory repository.Factory) (*Server, error) {
	if err := verifyWebDist(webDistDir); err != nil {
		return nil, err
	}

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
	fs := http.FileServer(http.Dir(webDistDir))
	r.Handle("/static/*", fs)
	r.Handle("/_astro/*", fs)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(webDistDir, "index.html"))
	})

	r.Mount("/api", api.Router())

	return &Server{
		r: r,
	}, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.r.ServeHTTP(w, r)
}
