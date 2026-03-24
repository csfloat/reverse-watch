package models

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestParseSteamUserInput_plainID(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	id, err := ParseSteamUserInput(ctx, http.DefaultClient, " 76561197960287930 ")
	if err != nil {
		t.Fatalf("ParseSteamUserInput: %v", err)
	}
	if *id != 76561197960287930 {
		t.Fatalf("got %v", *id)
	}
}

func TestParseSteamUserInput_profileURL(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	cases := []string{
		"https://steamcommunity.com/profiles/76561197960287930",
		"https://steamcommunity.com/profiles/76561197960287930/",
		"steamcommunity.com/profiles/76561197960287930",
		"HTTP://STEAMCOMMUNITY.COM/profiles/76561197960287930",
	}
	for _, raw := range cases {
		t.Run(raw, func(t *testing.T) {
			id, err := ParseSteamUserInput(ctx, http.DefaultClient, raw)
			if err != nil {
				t.Fatalf("ParseSteamUserInput: %v", err)
			}
			if *id != 76561197960287930 {
				t.Fatalf("got %v", *id)
			}
		})
	}
}

func TestParseSteamUserInput_vanityURL_viaClient(t *testing.T) {
	t.Parallel()
	const want = 76561199802279371
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/id/testuser" || r.URL.Query().Get("xml") != "1" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprintf(w, `<?xml version="1.0"?><profile><steamID64>%d</steamID64></profile>`, want)
	}))
	t.Cleanup(srv.Close)

	client := &http.Client{
		Transport: &rewriteHostsTransport{hosts: map[string]string{"steamcommunity.com": srv.URL}, inner: http.DefaultTransport},
	}

	ctx := context.Background()
	raw := "https://steamcommunity.com/id/testuser"
	id, err := ParseSteamUserInput(ctx, client, raw)
	if err != nil {
		t.Fatalf("ParseSteamUserInput: %v", err)
	}
	if *id != want {
		t.Fatalf("got %v want %v", *id, want)
	}
}

func TestParseSteamUserInput_gabeLoganNewellVanityURL(t *testing.T) {
	t.Parallel()
	// Expected steamID64 from https://steamcommunity.com/id/GabeLoganNewell?xml=1
	const want = 76561197960287930
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/id/GabeLoganNewell" || r.URL.Query().Get("xml") != "1" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprintf(w, `<?xml version="1.0"?><profile><steamID64>%d</steamID64></profile>`, want)
	}))
	t.Cleanup(srv.Close)

	client := &http.Client{
		Transport: &rewriteHostsTransport{hosts: map[string]string{"steamcommunity.com": srv.URL}, inner: http.DefaultTransport},
	}

	ctx := context.Background()
	raw := "https://steamcommunity.com/id/GabeLoganNewell/ "
	id, err := ParseSteamUserInput(ctx, client, raw)
	if err != nil {
		t.Fatalf("ParseSteamUserInput: %v", err)
	}
	if *id != want {
		t.Fatalf("got %v want %v", *id, want)
	}
}

// rewriteHostsTransport sends listed hostnames to test server base URLs (e.g. httptest).
type rewriteHostsTransport struct {
	hosts map[string]string
	inner http.RoundTripper
}

func (t *rewriteHostsTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	dup := req.Clone(req.Context())
	host := strings.ToLower(strings.TrimPrefix(dup.URL.Hostname(), "www."))
	if realBase, ok := t.hosts[host]; ok {
		base, err := url.Parse(realBase)
		if err != nil {
			return nil, err
		}
		dup.URL.Scheme = base.Scheme
		dup.URL.Host = base.Host
	}
	return t.inner.RoundTrip(dup)
}

func TestResolveVanitySteamWebAPI(t *testing.T) {
	t.Parallel()
	const want = 76561197960287930
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ISteamUser/ResolveVanityURL/v1/" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("vanityurl") != "testuser" || r.URL.Query().Get("key") != "fake" {
			http.Error(w, "bad query", http.StatusBadRequest)
			return
		}
		fmt.Fprintf(w, `{"response":{"success":1,"steamid":"%d"}}`, want)
	}))
	t.Cleanup(srv.Close)

	client := &http.Client{
		Transport: &rewriteHostsTransport{
			hosts: map[string]string{"api.steampowered.com": srv.URL},
			inner: http.DefaultTransport,
		},
	}

	ctx := context.Background()
	id, err := ResolveVanitySteamWebAPI(ctx, client, "fake", "testuser")
	if err != nil {
		t.Fatalf("ResolveVanitySteamWebAPI: %v", err)
	}
	if *id != want {
		t.Fatalf("got %v want %v", *id, want)
	}
}

func TestParseSteamUserInputWithOpts_webAPI(t *testing.T) {
	t.Parallel()
	const want = 76561199802279371
	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ISteamUser/ResolveVanityURL/v1/" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprintf(w, `{"response":{"success":1,"steamid":"%d"}}`, want)
	}))
	t.Cleanup(apiSrv.Close)

	client := &http.Client{
		Transport: &rewriteHostsTransport{
			hosts: map[string]string{"api.steampowered.com": apiSrv.URL},
			inner: http.DefaultTransport,
		},
	}

	ctx := context.Background()
	id, err := ParseSteamUserInputWithOpts(ctx, client, "https://steamcommunity.com/id/foo", &SteamUserInputOpts{
		UseWebAPIForVanity: true,
		SteamWebAPIKey:     "k",
	})
	if err != nil {
		t.Fatalf("ParseSteamUserInputWithOpts: %v", err)
	}
	if *id != want {
		t.Fatalf("got %v want %v", *id, want)
	}
}

func TestParseSteamUserInputWithOpts_webAPIRequiresKey(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	_, err := ParseSteamUserInputWithOpts(ctx, http.DefaultClient, "https://steamcommunity.com/id/foo", &SteamUserInputOpts{
		UseWebAPIForVanity: true,
		SteamWebAPIKey:     "",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseSteamUserInput_invalid(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	client := http.DefaultClient
	for _, raw := range []string{
		"",
		"not-a-url",
		"https://example.com/profiles/76561197960287930",
		"https://steamcommunity.com/game/123",
		"76561197000000000", // below minimum for IsValid
	} {
		t.Run(raw, func(t *testing.T) {
			_, err := ParseSteamUserInput(ctx, client, raw)
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
