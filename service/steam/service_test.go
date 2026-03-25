package steam

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	reverrors "reverse-watch/errors"
	"reverse-watch/util"
)

type rewriteTransport struct {
	base          http.RoundTripper
	communityBase *url.URL
	apiBase       *url.URL
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	switch clone.URL.Hostname() {
	case "steamcommunity.com", "www.steamcommunity.com":
		clone.URL.Scheme = t.communityBase.Scheme
		clone.URL.Host = t.communityBase.Host
	case "api.steampowered.com":
		clone.URL.Scheme = t.apiBase.Scheme
		clone.URL.Host = t.apiBase.Host
	}
	return t.base.RoundTrip(clone)
}

func mustParseURL(t *testing.T, s string) *url.URL {
	t.Helper()
	u, err := url.Parse(s)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	return u
}

func TestResolveSteamID_Number(t *testing.T) {
	svc := New(Options{})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err := svc.ResolveSteamID(ctx, "76561198000000000")
	if err == nil {
		t.Fatalf("expected error for raw steam id")
	}
	var e *reverrors.Error
	if !stderrors.As(err, &e) {
		t.Fatalf("expected errors.Error, got %T: %v", err, err)
	}
	if e.Code != reverrors.BadRequest.Code {
		t.Fatalf("expected BadRequest code=%d, got code=%d", reverrors.BadRequest.Code, e.Code)
	}
}

func TestResolveSteamID_ProfileURL(t *testing.T) {
	svc := New(Options{})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err := svc.ResolveSteamID(ctx, "steamcommunity.com/profiles/76561198000000000")
	if err == nil {
		t.Fatalf("expected error for /profiles URL")
	}
}

func TestResolveSteamID_Vanity_WebAPI(t *testing.T) {
	community := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "should not call community xml fallback", http.StatusBadRequest)
	}))
	defer community.Close()

	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ISteamUser/ResolveVanityURL/v1/" {
			http.Error(w, "unexpected request path", http.StatusBadRequest)
			return
		}
		if got := r.URL.Query().Get("vanityurl"); got != "testuser" {
			http.Error(w, "unexpected vanityurl", http.StatusBadRequest)
			return
		}
		if got := r.URL.Query().Get("key"); got != "k1" {
			http.Error(w, "unexpected key", http.StatusBadRequest)
			return
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"response": map[string]any{
				"success": 1,
				"steamid": "76561198000000000",
			},
		})
	}))
	defer api.Close()

	client := &http.Client{
		Transport: &rewriteTransport{
			base:          http.DefaultTransport,
			communityBase: mustParseURL(t, community.URL),
			apiBase:       mustParseURL(t, api.URL),
		},
	}
	svc := New(Options{HTTPClient: client, WebAPIKeys: util.NewRing([]string{"k1"})})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	id, err := svc.ResolveSteamID(ctx, "https://steamcommunity.com/id/testuser")
	if err != nil {
		t.Fatalf("ResolveSteamID: %v", err)
	}
	if got := id.String(); got != "76561198000000000" {
		t.Fatalf("unexpected steamid: %s", got)
	}
}

func TestResolveSteamID_Vanity_WebAPI_RotatesKeys(t *testing.T) {
	var (
		mu   sync.Mutex
		keys []string
	)

	community := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "should not call xml", http.StatusBadRequest)
	}))
	defer community.Close()

	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		keys = append(keys, r.URL.Query().Get("key"))
		mu.Unlock()

		_ = json.NewEncoder(w).Encode(map[string]any{
			"response": map[string]any{
				"success": 1,
				"steamid": "76561198000000000",
			},
		})
	}))
	defer api.Close()

	client := &http.Client{
		Transport: &rewriteTransport{
			base:          http.DefaultTransport,
			communityBase: mustParseURL(t, community.URL),
			apiBase:       mustParseURL(t, api.URL),
		},
	}
	svc := New(Options{HTTPClient: client, WebAPIKeys: util.NewRing([]string{"k1", "k2"})})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if _, err := svc.ResolveSteamID(ctx, "steamcommunity.com/id/testuser"); err != nil {
		t.Fatalf("ResolveSteamID #1: %v", err)
	}
	if _, err := svc.ResolveSteamID(ctx, "steamcommunity.com/id/testuser"); err != nil {
		t.Fatalf("ResolveSteamID #2: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(keys) != 2 {
		t.Fatalf("expected 2 api calls, got %d", len(keys))
	}
	if keys[0] != "k1" || keys[1] != "k2" {
		t.Fatalf("expected key rotation k1,k2 got %v", keys)
	}
}
