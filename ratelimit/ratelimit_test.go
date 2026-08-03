package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"reverse-watch/domain/models"
	"reverse-watch/domain/models/constants"
	isecret "reverse-watch/domain/secret"
	"reverse-watch/internal/testutil"
	"reverse-watch/middleware"
	"reverse-watch/repository/factory"
	"reverse-watch/secret"

	"gorm.io/gorm"
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

				resetTimeStr := w.Header.Get("X-RateLimit-Reset")
				resetTime, err := time.Parse(time.RFC1123, resetTimeStr)
				if err != nil {
					t.Fatalf("failed to parse reset time: %v", err)
				}

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

				resetTimeStr := w.Header.Get("X-RateLimit-Reset")
				resetTime, err := time.Parse(time.RFC1123, resetTimeStr)
				if err != nil {
					t.Fatalf("failed to parse reset time: %v", err)
				}

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

				resetTimeStr := w.Header.Get("X-RateLimit-Reset")
				resetTime, err := time.Parse(time.RFC1123, resetTimeStr)
				if err != nil {
					t.Fatalf("failed to parse reset time: %v", err)
				}

				maxReset := time.Now().Add(time.Minute + 5*time.Second)
				if resetTime.After(maxReset) {
					t.Errorf("X-RateLimit-Reset time is too far in the future")
				}

				retryAfterStr := w.Header.Get("Retry-After")
				if retryAfterStr == "" {
					t.Errorf("wanted Retry-After header, got empty string")
				}

				retryAfter, err := strconv.ParseInt(retryAfterStr, 10, 64)
				if err != nil {
					t.Fatalf("failed to parse Retry-After header: %v", err)
				}

				if retryAfter > 60 {
					t.Errorf("wanted Retry-After <= %d, got %d", 60, retryAfter)
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

			throttleMiddleware := ThrottleByIP(tc.duration, tc.limit)
			handler := throttleMiddleware(next)

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

func TestThrottleByAPIKey(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		limit          uint64
		duration       time.Duration
		numRequests    int
		setup          func(db *gorm.DB, keygen isecret.KeyGenerator) *http.Request
		wantStatusCode int
		checkHeaders   func(t *testing.T, w *http.Response)
	}{
		{
			name:        "belowLimit",
			limit:       10,
			duration:    time.Minute,
			numRequests: 5,
			setup: func(db *gorm.DB, keygen isecret.KeyGenerator) *http.Request {
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "test-marketplace", keygen, models.PermissionWrite)

				r := httptest.NewRequest(http.MethodGet, "http://testing", nil)
				r.Header.Set("Authorization", "Bearer "+formattedKey)
				return r
			},
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

				resetTimeStr := w.Header.Get("X-RateLimit-Reset")
				resetTime, err := time.Parse(time.RFC1123, resetTimeStr)
				if err != nil {
					t.Fatalf("failed to parse reset time: %v", err)
				}

				maxReset := time.Now().Add(time.Minute + 5*time.Second)
				if resetTime.After(maxReset) {
					t.Errorf("X-RateLimit-Reset time is too far in the future")
				}
			},
		},
		{
			name:        "atLimit",
			limit:       10,
			duration:    time.Minute,
			numRequests: 10,
			setup: func(db *gorm.DB, keygen isecret.KeyGenerator) *http.Request {
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "test-marketplace", keygen, models.PermissionWrite)

				r := httptest.NewRequest(http.MethodGet, "http://testing", nil)
				r.Header.Set("Authorization", "Bearer "+formattedKey)
				return r
			},
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

				resetTimeStr := w.Header.Get("X-RateLimit-Reset")
				resetTime, err := time.Parse(time.RFC1123, resetTimeStr)
				if err != nil {
					t.Fatalf("failed to parse reset time: %v", err)
				}

				maxReset := time.Now().Add(time.Minute + 5*time.Second)
				if resetTime.After(maxReset) {
					t.Errorf("X-RateLimit-Reset time is too far in the future")
				}
			},
		},
		{
			name:        "overLimit",
			limit:       10,
			duration:    time.Minute,
			numRequests: 11,
			setup: func(db *gorm.DB, keygen isecret.KeyGenerator) *http.Request {
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "test-marketplace", keygen, models.PermissionWrite)

				r := httptest.NewRequest(http.MethodGet, "http://testing", nil)
				r.Header.Set("Authorization", "Bearer "+formattedKey)
				return r
			},
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

				resetTimeStr := w.Header.Get("X-RateLimit-Reset")
				resetTime, err := time.Parse(time.RFC1123, resetTimeStr)
				if err != nil {
					t.Fatalf("failed to parse reset time: %v", err)
				}

				maxReset := time.Now().Add(time.Minute + 5*time.Second)
				if resetTime.After(maxReset) {
					t.Errorf("X-RateLimit-Reset time is too far in the future")
				}

				retryAfterStr := w.Header.Get("Retry-After")
				if retryAfterStr == "" {
					t.Errorf("wanted Retry-After header, got empty string")
				}

				retryAfter, err := strconv.ParseInt(retryAfterStr, 10, 64)
				if err != nil {
					t.Fatalf("failed to parse Retry-After header: %v", err)
				}

				if retryAfter > 60 {
					t.Errorf("wanted Retry-After <= %d, got %d", 60, retryAfter)
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

			db := testutil.NewTestDB(t)
			keygen := secret.NewKeyGenerator(constants.EnvironmentDevelopment)
			f, err := factory.NewFactoryWithConfig(&factory.Config{
				PrivateDB: db,
				PublicDB:  db,
				KeyGen:    keygen,
			})
			if err != nil {
				t.Fatalf("NewFactoryWithConfig(): %v", err)
			}

			factoryMiddleware := middleware.FactoryMiddleware(f)
			throttleMiddleware := ThrottleByAPIKey(tc.duration, tc.limit)
			finalHandler := factoryMiddleware(
				middleware.AuthMiddleware(
					throttleMiddleware(next),
				),
			)

			var w *httptest.ResponseRecorder
			r := tc.setup(db, keygen)
			for i := 0; i < tc.numRequests; i++ {
				w = httptest.NewRecorder()
				finalHandler.ServeHTTP(w, r)
			}

			if w.Code != tc.wantStatusCode {
				t.Errorf("got status code %d, wanted %d", w.Code, tc.wantStatusCode)
			}

			tc.checkHeaders(t, w.Result())
		})
	}
}

func TestThrottleByMarketplace(t *testing.T) {
	t.Parallel()

	db := testutil.NewTestDB(t)
	keygen := secret.NewKeyGenerator(constants.EnvironmentDevelopment)
	f, err := factory.NewFactoryWithConfig(&factory.Config{
		PrivateDB: db,
		PublicDB:  db,
		KeyGen:    keygen,
	})
	if err != nil {
		t.Fatalf("NewFactoryWithConfig(): %v", err)
	}

	testMarketplace, _, firstFormattedKey := testutil.SetupMarketplaceWithKey(t, db, "test-marketplace", keygen, models.PermissionWrite)
	secondSecretKey, err := keygen.GenerateSecretKey()
	if err != nil {
		t.Fatalf("GenerateSecretKey(): %v", err)
	}
	secondKeyID, err := secondSecretKey.ID()
	if err != nil {
		t.Fatalf("ID(): %v", err)
	}
	secondKey := &models.Key{
		ID:              secondKeyID,
		Environment:     keygen.Environment(),
		MarketplaceSlug: testMarketplace.Slug,
		Permissions:     models.PermissionWrite,
	}
	testutil.Insert(t, db, secondKey)
	secondFormattedKey, err := secondSecretKey.Format()
	if err != nil {
		t.Fatalf("Format(): %v", err)
	}

	_, _, otherMarketplaceFormattedKey := testutil.SetupMarketplaceWithKey(t, db, "other-marketplace", keygen, models.PermissionWrite)

	fn := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
	next := http.HandlerFunc(fn)

	factoryMiddleware := middleware.FactoryMiddleware(f)
	throttleMiddleware := ThrottleByMarketplace(time.Minute, 1)
	finalHandler := factoryMiddleware(
		middleware.AuthMiddleware(
			throttleMiddleware(next),
		),
	)

	requestWithKey := func(formattedKey string) *http.Request {
		r := httptest.NewRequest(http.MethodGet, "http://testing", nil)
		r.Header.Set("Authorization", "Bearer "+formattedKey)
		return r
	}

	w := httptest.NewRecorder()
	finalHandler.ServeHTTP(w, requestWithKey(firstFormattedKey))
	if w.Code != http.StatusOK {
		t.Fatalf("first marketplace request got status code %d, wanted %d", w.Code, http.StatusOK)
	}

	w = httptest.NewRecorder()
	finalHandler.ServeHTTP(w, requestWithKey(secondFormattedKey))
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("second same-marketplace key got status code %d, wanted %d", w.Code, http.StatusTooManyRequests)
	}

	w = httptest.NewRecorder()
	finalHandler.ServeHTTP(w, requestWithKey(otherMarketplaceFormattedKey))
	if w.Code != http.StatusOK {
		t.Fatalf("different marketplace got status code %d, wanted %d", w.Code, http.StatusOK)
	}
}
