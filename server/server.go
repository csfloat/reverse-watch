package server

import (
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
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

	// Static files are read into memory and written in a single response
	// body. We avoid http.ServeFile / http.FileServer because the sendfile
	// fast path on some local setups truncates large responses at the
	// first TCP segment. Files served from here are tiny (HTML + a handful
	// of logos/icons), so the read-once cost is negligible.
	r.Get("/", serveStaticFile("static/index.html", "text/html; charset=utf-8"))
	r.Get("/static/*", staticDirHandler("static"))

	r.Mount("/api", api.Router())

	return &Server{
		r: r,
	}, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.r.ServeHTTP(w, r)
}

// serveStaticFile returns a handler that serves exactly the file at path
// with the given content-type. The file is read once per request.
func serveStaticFile(path, contentType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b, err := os.ReadFile(path)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Content-Length", strconv.Itoa(len(b)))
		_, _ = w.Write(b)
	}
}

// staticDirHandler serves files from baseDir for any request matching
// the chi wildcard /<prefix>/*. Path traversal is rejected. Content-Type
// is inferred from the file extension, falling back to content sniffing.
func staticDirHandler(baseDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rest := strings.TrimPrefix(r.URL.Path, "/static/")
		if rest == "" || strings.Contains(rest, "..") {
			http.NotFound(w, r)
			return
		}
		full := filepath.Join(baseDir, filepath.Clean("/"+rest))
		b, err := os.ReadFile(full)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		ctype := mime.TypeByExtension(filepath.Ext(full))
		if ctype == "" {
			ctype = http.DetectContentType(b)
		}
		w.Header().Set("Content-Type", ctype)
		w.Header().Set("Content-Length", strconv.Itoa(len(b)))
		_, _ = w.Write(b)
	}
}
