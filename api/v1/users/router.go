package users

import (
	"time"

	"reverse-watch/ratelimit"

	"github.com/go-chi/chi/v5"
)

func Router() chi.Router {
	r := chi.NewRouter()
	r.With(ratelimit.ThrottleByIP(time.Minute, 15)).Get("/{steamId}", fetchUserStatus)
	return r
}
