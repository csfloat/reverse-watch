package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"reverse-watch/domain/models/constants"
	"reverse-watch/internal/testutil"
	"reverse-watch/repository/factory"
	"reverse-watch/secret"
)

func TestFactoryMiddleware(t *testing.T) {
	var capturedRequest *http.Request
	fn := func(w http.ResponseWriter, r *http.Request) {
		capturedRequest = r
		w.WriteHeader(http.StatusOK)
	}
	next := http.HandlerFunc(fn)

	db := testutil.NewTestDB(t)
	f, err := factory.NewFactoryWithConfig(&factory.Config{
		PrivateDB: db,
		PublicDB:  db,
		KeyGen:    secret.NewKeyGenerator(constants.EnvironmentDevelopment),
	})
	if err != nil {
		t.Fatalf("NewFactoryWithConfig(): %v", err)
	}
	factoryMiddleware := FactoryMiddleware(f)
	handler := factoryMiddleware(next)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("got status code %d, wanted %d", w.Code, http.StatusOK)
	}

	if capturedRequest == nil {
		t.Fatalf("captured request is nil")
	}

	capturedFactory := capturedRequest.Context().Value(FactoryContextKey)
	if capturedFactory == nil {
		t.Fatalf("captured factory is nil")
	}
}
