package steam

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"reverse-watch/domain/models"
	"reverse-watch/errors"
	"reverse-watch/util"
)

const maxVanityLen = 64
const maxSteamAPIBodyBytes = 64 << 10

type Service struct {
	client *http.Client
	keys   *util.Ring[string]
}

type Options struct {
	HTTPClient *http.Client
	WebAPIKeys *util.Ring[string]
}

func New(opts Options) *Service {
	client := opts.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	keys := opts.WebAPIKeys
	if keys == nil {
		keys = util.NewRing[string](nil)
	}

	return &Service{
		client: client,
		keys:   keys,
	}
}

func (s *Service) ResolveSteamID(ctx context.Context, raw string) (*models.SteamID, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, errors.New(errors.BadRequest, "missing vanityUrl")
	}

	// This endpoint should accept a vanity URL, not a raw numeric SteamID64.
	if _, err := models.ToSteamID(raw); err == nil {
		return nil, errors.New(errors.BadRequest, "expected vanityUrl, got steam id")
	}

	u, err := url.Parse(raw)
	if err != nil {
		return nil, errors.New(errors.BadRequest, "invalid vanityUrl", err)
	}

	// If the scheme is missing, default to https so the host/path parsing works.
	if u.Scheme == "" {
		u, err = url.Parse("https://" + raw)
		if err != nil {
			return nil, errors.New(errors.BadRequest, "invalid vanityUrl", err)
		}
	}

	if !strings.EqualFold(u.Scheme, "http") && !strings.EqualFold(u.Scheme, "https") {
		return nil, errors.New(errors.BadRequest, "url scheme must be http or https")
	}

	host := strings.ToLower(strings.TrimPrefix(u.Hostname(), "www."))
	if host != "steamcommunity.com" {
		return nil, errors.New(errors.BadRequest, fmt.Sprintf("expected steamcommunity.com, got %q", host))
	}

	path := strings.Trim(u.Path, "/")
	segments := strings.Split(path, "/")
	if len(segments) < 2 || !strings.EqualFold(segments[0], "id") {
		return nil, errors.New(errors.BadRequest, "expected steam vanity URL path /id/{vanity}")
	}

	vanity := segments[1]
	if vanity == "" {
		return nil, errors.New(errors.BadRequest, "missing vanity url")
	}
	if len(vanity) > maxVanityLen {
		return nil, errors.New(errors.BadRequest, "vanity too long")
	}
	return s.resolveVanity(ctx, vanity)
}

func (s *Service) resolveVanity(ctx context.Context, vanity string) (*models.SteamID, error) {
	key, ok := s.keys.Next()
	if !ok {
		return nil, errors.New(errors.BadRequest, "steam web api keys not configured")
	}
	return resolveVanityWebAPI(ctx, s.client, key, vanity)
}

func resolveVanityWebAPI(ctx context.Context, client *http.Client, apiKey, vanity string) (*models.SteamID, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, errors.New(errors.BadRequest, "empty steam web api key")
	}
	vanity = strings.TrimSpace(vanity)
	if vanity == "" {
		return nil, errors.New(errors.BadRequest, "empty vanity")
	}

	q := url.Values{}
	q.Set("key", apiKey)
	q.Set("vanityurl", vanity)
	q.Set("url_type", "1")
	reqURL := "https://api.steampowered.com/ISteamUser/ResolveVanityURL/v1/?" + q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, errors.New(errors.InternalServerError, "failed to create request", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.New(errors.InternalServerError, "steam web api request failed", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(errors.BadRequest, fmt.Sprintf("steam api returned status %d", resp.StatusCode))
	}

	// Decode directly from the response body (still bounded to avoid large reads).
	var envelope struct {
		Response struct {
			Success int    `json:"success"`
			SteamID string `json:"steamid"`
			Message string `json:"message"`
		} `json:"response"`
	}

	if err := json.NewDecoder(io.LimitReader(resp.Body, maxSteamAPIBodyBytes)).Decode(&envelope); err != nil {
		return nil, errors.New(errors.JSONDecode, "failed to decode steam api json", err)
	}

	if envelope.Response.Success != 1 {
		msg := strings.TrimSpace(envelope.Response.Message)
		if msg == "" {
			return nil, errors.New(errors.BadRequest, fmt.Sprintf("steam api could not resolve vanity (success=%d)", envelope.Response.Success))
		}
		return nil, errors.New(errors.BadRequest, "steam api: "+msg)
	}
	if envelope.Response.SteamID == "" {
		return nil, errors.New(errors.BadRequest, "steam api returned empty steamid")
	}

	id, err := models.ToSteamID(envelope.Response.SteamID)
	if err != nil {
		return nil, errors.New(errors.BadRequest, "steam api returned invalid steamid", err)
	}
	return id, nil
}
