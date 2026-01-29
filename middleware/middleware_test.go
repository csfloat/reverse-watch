package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"reverse-watch/domain/models"
	"reverse-watch/domain/models/constants"
	"reverse-watch/internal/testutil"
	"reverse-watch/repository/private"
	"reverse-watch/secret"
)

func TestAuthMiddleware(t *testing.T) {
	t.Parallel()

	db := testutil.NewTestDB(t)
	keygen := secret.NewKeyGenerator(constants.EnvironmentDevelopment)
	keyRepo := private.NewKeyRepository(db, keygen)

	testCases := []struct {
		name           string
		setup          func() *http.Request
		wantStatusCode int
	}{
		{
			name: "noAuthHeader",
			setup: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "http://testing", nil)
			},
			wantStatusCode: http.StatusUnauthorized,
		},
		{
			name: "invalidAuthHeader",
			setup: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "http://testing", nil)
				req.Header.Set("Authorization", "test-token")
				return req
			},
			wantStatusCode: http.StatusUnauthorized,
		},
		{
			name: "invalidToken",
			setup: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "http://testing", nil)
				req.Header.Set("Authorization", "Bearer test-token")
				return req
			},
			wantStatusCode: http.StatusUnauthorized,
		},
		{
			name: "validToken",
			setup: func() *http.Request {
				testMarketplace := &models.Marketplace{
					Slug:     "test-marketplace",
					Name:     "Test Marketplace",
					IsActive: true,
				}
				testutil.Insert(t, db, testMarketplace)

				testKey, err := keygen.GenerateSecretKey()
				if err != nil {
					t.Fatalf("GenerateSecretKey(): %v", err)
				}

				id, err := testKey.ID()
				if err != nil {
					t.Fatalf("ID(): %v", err)
				}

				key := &models.Key{
					ID:              id,
					Environment:     keygen.Environment(),
					MarketplaceSlug: testMarketplace.Slug,
					Permissions:     models.PermissionWrite,
				}
				testutil.Insert(t, db, key)

				formattedKey, err := testKey.Format()
				if err != nil {
					t.Fatalf("Format(): %v", err)
				}

				value := fmt.Sprintf("Bearer %s", formattedKey)
				req := httptest.NewRequest(http.MethodGet, "http://testing", nil)
				req.Header.Set("Authorization", value)
				return req
			},
			wantStatusCode: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fn := func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}
			dummyHandler := http.HandlerFunc(fn)

			middlewareFunc := AuthMiddleware(keyRepo)
			handler := middlewareFunc(dummyHandler)

			w := httptest.NewRecorder()
			r := tc.setup()

			handler.ServeHTTP(w, r)

			if w.Code != tc.wantStatusCode {
				t.Errorf("got status code %d, wanted %d", w.Code, tc.wantStatusCode)
			}
		})
	}
}
