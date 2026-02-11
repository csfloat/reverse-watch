package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func TestThrottleByIP(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		limit          uint64
		duration       time.Duration
		numRequests    int
		remoteAddr     string
		wantStatusCode int
		checkHeaders   func(t *testing.T, w *http.Response)
	}{
		{
			name:           "belowLimit",
			limit:          10,
			duration:       time.Minute,
			numRequests:    5,
			remoteAddr:     "192.168.0.1:1234",
			wantStatusCode: http.StatusOK,
			checkHeaders: func(t *testing.T, w *http.Response) {
				limit, err := strconv.ParseUint(w.Header.Get("X-RateLimit-Limit"), 10, 64)
				if err != nil {
					t.Fatalf("failed to parse limit: %v", err)
				}

				if limit != 10 {
					t.Errorf("wanted limit %d, got %d", 10, limit)
				}

				remaining, err := strconv.ParseUint(w.Header.Get("X-RateLimit-Remaining"), 10, 64)
				if err != nil {
					t.Fatalf("failed to parse remaining: %v", err)
				}

				if remaining != 5 {
					t.Errorf("wanted remaining %d, got %d", 5, remaining)
				}

				resetTimeStr, err := strconv.ParseInt(w.Header.Get("X-RateLimit-Reset"), 10, 64)
				if err != nil {
					t.Fatalf("failed to parse reset time: %v", err)
				}

				resetTime := time.Unix(resetTimeStr, 0).UTC()
				maxReset := time.Now().Add(time.Minute + 5*time.Second)
				if resetTime.After(maxReset) {
					t.Errorf("X-RateLimit-Reset time is too far in the future")
				}
			},
		},
		{
			name:           "atLimit",
			limit:          10,
			duration:       time.Minute,
			numRequests:    10,
			remoteAddr:     "192.168.0.2:1234",
			wantStatusCode: http.StatusOK,
			checkHeaders: func(t *testing.T, w *http.Response) {
				limit, err := strconv.ParseUint(w.Header.Get("X-RateLimit-Limit"), 10, 64)
				if err != nil {
					t.Fatalf("failed to parse limit: %v", err)
				}

				if limit != 10 {
					t.Errorf("wanted limit %d, got %d", 10, limit)
				}

				remaining, err := strconv.ParseUint(w.Header.Get("X-RateLimit-Remaining"), 10, 64)
				if err != nil {
					t.Fatalf("failed to parse remaining: %v", err)
				}

				if remaining != 0 {
					t.Errorf("wanted remaining %d, got %d", 0, remaining)
				}

				resetTimeStr, err := strconv.ParseInt(w.Header.Get("X-RateLimit-Reset"), 10, 64)
				if err != nil {
					t.Fatalf("failed to parse reset time: %v", err)
				}

				resetTime := time.Unix(resetTimeStr, 0).UTC()
				maxReset := time.Now().Add(time.Minute + 5*time.Second)
				if resetTime.After(maxReset) {
					t.Errorf("X-RateLimit-Reset time is too far in the future")
				}
			},
		},
		{
			name:           "overLimit",
			limit:          10,
			duration:       time.Minute,
			numRequests:    11,
			remoteAddr:     "192.168.0.3:1234",
			wantStatusCode: http.StatusTooManyRequests,
			checkHeaders: func(t *testing.T, w *http.Response) {
				limit, err := strconv.ParseUint(w.Header.Get("X-RateLimit-Limit"), 10, 64)
				if err != nil {
					t.Fatalf("failed to parse limit: %v", err)
				}

				if limit != 10 {
					t.Errorf("wanted limit %d, got %d", 10, limit)
				}

				remaining, err := strconv.ParseUint(w.Header.Get("X-RateLimit-Remaining"), 10, 64)
				if err != nil {
					t.Fatalf("failed to parse remaining: %v", err)
				}

				if remaining != 0 {
					t.Errorf("wanted remaining %d, got %d", 0, remaining)
				}

				resetTime, err := strconv.ParseInt(w.Header.Get("X-RateLimit-Reset"), 10, 64)
				if err != nil {
					t.Fatalf("failed to parse reset time: %v", err)
				}

				reset := time.Unix(resetTime, 0).UTC()
				maxReset := time.Now().Add(time.Minute + 5*time.Second)
				if reset.After(maxReset) {
					t.Errorf("X-RateLimit-Reset time is too far in the future")
				}

				retryAfter, err := strconv.ParseInt(w.Header.Get("Retry-After"), 10, 64)
				if err != nil {
					t.Fatalf("failed to parse Retry-After header: %v", err)
				}

				if resetTime != retryAfter {
					t.Errorf("wanted Retry-After %v, got %v", resetTime, retryAfter)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fn := func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}
			next := http.HandlerFunc(fn)

			middleware := ThrottleByIP(tc.duration, tc.limit)
			handler := middleware(next)

			var w *httptest.ResponseRecorder
			for i := 0; i < tc.numRequests; i++ {
				w = httptest.NewRecorder()
				r := httptest.NewRequest(http.MethodGet, "http://testing", nil)
				r.RemoteAddr = tc.remoteAddr
				handler.ServeHTTP(w, r)
			}

			if w.Code != tc.wantStatusCode {
				t.Errorf("wanted status code %d, got %d", tc.wantStatusCode, w.Code)
			}

			tc.checkHeaders(t, w.Result())
		})
	}
}
