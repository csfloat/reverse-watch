package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"reverse-watch/internal/testutil"
)

func TestFactoryMiddleware(t *testing.T) {
	var capturedRequest *http.Request
	fn := func(w http.ResponseWriter, r *http.Request) {
		capturedRequest = r
		w.WriteHeader(http.StatusOK)
	}
	next := http.HandlerFunc(fn)

	factory := testutil.NewTestFactory(t)
	factoryMiddleware := FactoryMiddleware(factory)
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
