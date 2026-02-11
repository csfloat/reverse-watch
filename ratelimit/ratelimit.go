package ratelimit

import (
	"net/http"
	"strconv"
	"time"

	"reverse-watch/errors"
	"reverse-watch/logging"
	"reverse-watch/render"

	"github.com/sethvargo/go-limiter"
	"github.com/sethvargo/go-limiter/httplimit"
	"github.com/sethvargo/go-limiter/memorystore"
)

func keyByIP(r *http.Request) (string, error) {
	return r.RemoteAddr, nil
}

func ThrottleByIP(dur time.Duration, limit uint64) func(http.Handler) http.Handler {
	return newLimiter(dur, limit, keyByIP)
}

func newThrottlerWithLimiter(keyFunc httplimit.KeyFunc, store limiter.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			key, err := keyFunc(r)
			if err != nil {
				logging.Log.Errorf("failed to get key from request: %v", err)
				render.Error(w, r, &errors.Limiter)
				return
			}

			limit, remaining, reset, ok, err := store.Take(ctx, key)
			if err != nil {
				logging.Log.Errorf("failed to fetch key %v from store: %v", key, err)
				render.Error(w, r, &errors.Limiter)
				return
			}

			resetTime := time.Unix(0, int64(reset)).UTC().Format(time.RFC1123)

			w.Header().Set("X-RateLimit-Limit", strconv.FormatUint(limit, 10))
			w.Header().Set("X-RateLimit-Remaining", strconv.FormatUint(remaining, 10))
			w.Header().Set("X-RateLimit-Reset", resetTime)

			if !ok {
				w.Header().Set("Retry-After", resetTime)
				render.Error(w, r, &errors.RateLimited)
				return
			}
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}

func newLimiter(dur time.Duration, limit uint64, keyFunc httplimit.KeyFunc) func(http.Handler) http.Handler {
	store, err := memorystore.New(&memorystore.Config{
		Tokens:        limit,
		Interval:      dur,
		SweepMinTTL:   dur,
		SweepInterval: time.Hour,
	})
	if err != nil {
		panic(err)
	}
	return newThrottlerWithLimiter(keyFunc, store)
}
