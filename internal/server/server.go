package server

import (
	"net/http"

	"reverse-watch/internal/config"
	"reverse-watch/internal/domain/models"
	"reverse-watch/internal/domain/service"
	adminkeys "reverse-watch/internal/handler/admin/keys"
	adminmarketplace "reverse-watch/internal/handler/admin/marketplace"
	"reverse-watch/internal/handler/marketplace/keys"
	rwmiddleware "reverse-watch/internal/middleware"
	"reverse-watch/internal/repository/private"
	"reverse-watch/internal/repository/public"
	"reverse-watch/internal/service/key"
	"reverse-watch/internal/service/marketplace"
	"reverse-watch/internal/service/reversal"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type Server struct {
	router chi.Router

	keyService         service.KeyService
	marketplaceService service.MarketplaceService
	reversalService    service.ReversalService
}

func New(cfg config.Config) *Server {
	privateRepo, err := private.NewPrivateRepository(cfg)
	if err != nil {
		panic(err)
	}
	publicRepo, err := public.NewPublicRepository(cfg)
	if err != nil {
		panic(err)
	}

	// Create Services
	keySvc := key.NewKeyService(privateRepo)
	marketplaceSvc := marketplace.NewMarketplaceService(privateRepo)
	reversalSvc := reversal.NewReversalService(privateRepo, publicRepo)

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(render.SetContentType(render.ContentTypeJSON))

	// Define routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/admin", func(r chi.Router) {
			r.Use(rwmiddleware.Middleware(keySvc))
			r.Use(rwmiddleware.RequirePermission(models.PermissionAdmin))
			r.Route("/keys", func(r chi.Router) {
				adminkeys.NewKeyHandler(keySvc).RegisterRoutes(r)
			})
			r.Route("/marketplace", func(r chi.Router) {
				adminmarketplace.NewMarketplaceHandler(keySvc, marketplaceSvc).RegisterRoutes(r)
			})
		})
		r.Route("/marketplace", func(r chi.Router) {
			r.Use(rwmiddleware.Middleware(keySvc))
			r.Use(rwmiddleware.RequirePermission(models.PermissionManage))
			r.Route("/keys", func(r chi.Router) {
				keys.NewKeyHandler(keySvc).RegisterRoutes(r)
			})
		})
	})

	return &Server{
		router:             r,
		keyService:         keySvc,
		marketplaceService: marketplaceSvc,
		reversalService:    reversalSvc,
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}
