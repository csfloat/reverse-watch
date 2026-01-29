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
		setup          func() (*http.Request, error)
		wantStatusCode int
	}{
		{
			name: "noAuthHeader",
			setup: func() (*http.Request, error) {
				return httptest.NewRequest(http.MethodGet, "http://testing", nil), nil
			},
			wantStatusCode: http.StatusUnauthorized,
		},
		{
			name: "invalidAuthHeader",
			setup: func() (*http.Request, error) {
				req := httptest.NewRequest(http.MethodGet, "http://testing", nil)
				req.Header.Set("Authorization", "test-token")
				return req, nil
			},
			wantStatusCode: http.StatusUnauthorized,
		},
		{
			name: "invalidToken",
			setup: func() (*http.Request, error) {
				req := httptest.NewRequest(http.MethodGet, "http://testing", nil)
				req.Header.Set("Authorization", "Bearer test-token")
				return req, nil
			},
			wantStatusCode: http.StatusUnauthorized,
		},
		{
			name: "validToken",
			setup: func() (*http.Request, error) {
				testMarketplace := &models.Marketplace{
					Slug:     "test-marketplace",
					Name:     "Test Marketplace",
					IsActive: true,
				}
				testutil.Insert(t, db, testMarketplace)

				testKey, err := keygen.GenerateSecretKey()
				if err != nil {
					return nil, err
				}

				id, err := testKey.ID()
				if err != nil {
					return nil, err
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
					return nil, err
				}

				value := fmt.Sprintf("Bearer %s", formattedKey)
				req := httptest.NewRequest(http.MethodGet, "http://testing", nil)
				req.Header.Set("Authorization", value)
				return req, nil
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
			r, err := tc.setup()
			if err != nil {
				t.Fatalf("setup(): %v", err)
			}

			handler.ServeHTTP(w, r)

			if w.Code != tc.wantStatusCode {
				t.Errorf("got status code %d, wanted %d", w.Code, tc.wantStatusCode)
			}
		})
	}
}
