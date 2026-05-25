package stats

import (
	"time"

	"reverse-watch/ratelimit"

	"github.com/go-chi/chi/v5"
)

func Router() chi.Router {
	r := chi.NewRouter()
	throttle := ratelimit.ThrottleByIP(time.Minute, 60)
	r.With(throttle).Get("/summary", summaryHandler)
	r.With(throttle).Get("/reversals/daily", dailyHandler)
	return r
}
