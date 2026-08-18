package stats

import (
	"time"

	"reverse-watch/ratelimit"

	"github.com/go-chi/chi/v5"
)

func Router() chi.Router {
	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(ratelimit.ThrottleByIP(time.Minute, 60))
		r.Get("/summary", summaryHandler)
		r.Get("/reversals/daily", dailyHandler)
	})
	return r
}
