package server

import (
	"errors"
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
	"gorm.io/gorm"
)

type Server struct {
	r chi.Router

	private *gorm.DB
	public  *gorm.DB
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
		r:       r,
		private: privateDB,
		public:  publicDB,
	}, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.r.ServeHTTP(w, r)
}

func (s *Server) Close() error {
	var errs []error
	if err := closeConn(s.private); err != nil {
		errs = append(errs, err)
	}
	if err := closeConn(s.public); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func closeConn(conn *gorm.DB) error {
	db, err := conn.DB()
	if err != nil {
		return err
	}
	return db.Close()
}
