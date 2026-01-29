package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"reverse-watch/domain/models"
)

func TestRequirePermissions(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		permissions    []models.Permissions
		setup          func() *http.Request
		wantStatusCode int
	}{
		{
			name:        "exactPermissions",
			permissions: []models.Permissions{models.PermissionRead},
			setup: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "http://testing", nil)
				key := &models.Key{
					Permissions: models.PermissionRead,
				}
				ctx := context.WithValue(r.Context(), KeyContextKey, key)
				return r.WithContext(ctx)
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name:        "multipleExactPermissions",
			permissions: []models.Permissions{models.PermissionManage, models.PermissionExport},
			setup: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "http://testing", nil)
				key := &models.Key{
					Permissions: models.PermissionManage | models.PermissionExport,
				}
				ctx := context.WithValue(r.Context(), KeyContextKey, key)
				return r.WithContext(ctx)
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name:        "noKeyInContext",
			permissions: []models.Permissions{models.PermissionRead},
			setup: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "http://testing", nil)
			},
			wantStatusCode: http.StatusForbidden,
		},
		{
			name:        "insufficientPermissions",
			permissions: []models.Permissions{models.PermissionWrite},
			setup: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "http://testing", nil)
				key := &models.Key{
					Permissions: models.PermissionRead,
				}
				ctx := context.WithValue(r.Context(), KeyContextKey, key)
				return r.WithContext(ctx)
			},
			wantStatusCode: http.StatusForbidden,
		},
		{
			name:        "multiplePermissionsOneMissing",
			permissions: []models.Permissions{models.PermissionRead, models.PermissionWrite},
			setup: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "http://testing", nil)
				key := &models.Key{
					Permissions: models.PermissionRead,
				}
				ctx := context.WithValue(r.Context(), KeyContextKey, key)
				return r.WithContext(ctx)
			},
			wantStatusCode: http.StatusForbidden,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fn := func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}
			dummyHandler := http.HandlerFunc(fn)

			middlewareFunc := RequirePermissions(tc.permissions...)
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
