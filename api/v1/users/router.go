package users

import (
	"time"

	"reverse-watch/ratelimit"

	"github.com/go-chi/chi/v5"
)

func Router(steamWebAPIKey string) chi.Router {
	r := chi.NewRouter()
	r.With(ratelimit.ThrottleByIP(time.Minute, 100)).Get("/{steamId}", fetchUserStatus(steamWebAPIKey))
	return r
}
