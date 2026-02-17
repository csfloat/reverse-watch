package marketplace

import (
	"time"

	"reverse-watch/ratelimit"

	"github.com/go-chi/chi/v5"
)

func Router() chi.Router {
	r := chi.NewRouter()
	r.Use(ratelimit.ThrottleByAPIKey(time.Hour, 2_000))

	r.Post("/", onboardMarketplace)
	r.Patch("/{slug}", updateMarketplace)
	return r
}
