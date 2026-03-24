package models

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

var steamID64XMLRe = regexp.MustCompile(`(?i)<steamID64>\s*(\d+)\s*</steamID64>`)

const maxVanityLen = 64
const maxSteamXMLBytes = 1 << 20
const maxSteamAPIBodyBytes = 64 << 10

// SteamUserInputOpts tweaks how vanity URLs (/id/{name}) are turned into a SteamID.
// Zero value: same as ParseSteamUserInput — uses the community profile ?xml=1 fetch.
type SteamUserInputOpts struct {
	// UseWebAPIForVanity calls ISteamUser/ResolveVanityURL instead of scraping ?xml=1.
	UseWebAPIForVanity bool
	// SteamWebAPIKey from https://steamcommunity.com/dev/apikey — required when UseWebAPIForVanity is true.
	SteamWebAPIKey string
}

// ParseSteamUserInput resolves a SteamID from a decimal 64-bit ID string, a
// steamcommunity.com /profiles/{id} URL, or /id/{vanity} URL (via Valve's ?xml=1 profile feed).
func ParseSteamUserInput(ctx context.Context, httpClient *http.Client, raw string) (*SteamID, error) {
	return ParseSteamUserInputWithOpts(ctx, httpClient, raw, nil)
}

// ParseSteamUserInputWithOpts is like ParseSteamUserInput but can resolve custom URLs via the Steam Web API.
func ParseSteamUserInputWithOpts(ctx context.Context, httpClient *http.Client, raw string, opts *SteamUserInputOpts) (*SteamID, error) {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	s := strings.TrimSpace(raw)
	if s == "" {
		return nil, fmt.Errorf("empty steam identifier")
	}
	if id, err := ToSteamID(s); err == nil {
		return id, nil
	}
	normalized := normalizeSteamProfileURL(s)
	u, err := url.Parse(normalized)
	if err != nil {
		return nil, fmt.Errorf("parse url: %w", err)
	}
	host := strings.ToLower(strings.TrimPrefix(u.Hostname(), "www."))
	if host != "steamcommunity.com" {
		return nil, fmt.Errorf("not a steam community url")
	}
	path := strings.Trim(u.Path, "/")
	segments := strings.Split(path, "/")
	if len(segments) >= 2 && strings.EqualFold(segments[0], "profiles") {
		idStr := segments[1]
		if idStr == "" {
			return nil, fmt.Errorf("missing profile id")
		}
		return ToSteamID(idStr)
	}
	if len(segments) >= 2 && strings.EqualFold(segments[0], "id") {
		vanity := segments[1]
		if vanity == "" {
			return nil, fmt.Errorf("missing vanity url")
		}
		if len(vanity) > maxVanityLen {
			return nil, fmt.Errorf("vanity too long")
		}
		if opts != nil && opts.UseWebAPIForVanity {
			key := strings.TrimSpace(opts.SteamWebAPIKey)
			if key == "" {
				return nil, fmt.Errorf("steam web api key is required for web api vanity resolution")
			}
			return ResolveVanitySteamWebAPI(ctx, httpClient, key, vanity)
		}
		return resolveSteamVanityXML(ctx, httpClient, vanity)
	}
	return nil, fmt.Errorf("unrecognized steam profile path")
}

func normalizeSteamProfileURL(s string) string {
	s = strings.TrimSpace(s)
	lower := strings.ToLower(s)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return s
	}
	if strings.HasPrefix(lower, "steamcommunity.com") || strings.HasPrefix(lower, "www.steamcommunity.com") {
		return "https://" + s
	}
	return s
}

// resolveSteamVanityXML hits steamcommunity.com/id/{vanity}?xml=1 and pulls the 64-bit ID
// from the response. Fine for the odd manual lookup; don’t use this for bulk scraping—Steam
// will rate-limit or block you. For high volume, cache results, throttle requests, or use
// the Web API ResolveVanityURL instead.
func resolveSteamVanityXML(ctx context.Context, client *http.Client, vanity string) (*SteamID, error) {
	reqURL := "https://steamcommunity.com/id/" + url.PathEscape(vanity) + "?xml=1"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "reverse-watch/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("steam profile returned status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxSteamXMLBytes))
	if err != nil {
		return nil, err
	}
	m := steamID64XMLRe.FindSubmatch(body)
	if m == nil {
		return nil, fmt.Errorf("steam id not found in profile response")
	}
	return ToSteamID(string(m[1]))
}

// ResolveVanitySteamWebAPI turns a custom profile slug into SteamID64 using
// ISteamUser/ResolveVanityURL. You need an API key from https://steamcommunity.com/dev/apikey.
func ResolveVanitySteamWebAPI(ctx context.Context, client *http.Client, apiKey, vanity string) (*SteamID, error) {
	if client == nil {
		client = http.DefaultClient
	}
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, fmt.Errorf("empty steam web api key")
	}
	vanity = strings.TrimSpace(vanity)
	if vanity == "" {
		return nil, fmt.Errorf("empty vanity")
	}

	q := url.Values{}
	q.Set("key", apiKey)
	q.Set("vanityurl", vanity)
	q.Set("url_type", "1")
	reqURL := "https://api.steampowered.com/ISteamUser/ResolveVanityURL/v1/?" + q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "reverse-watch/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxSteamAPIBodyBytes))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("steam api returned status %d", resp.StatusCode)
	}

	var envelope struct {
		Response struct {
			Success int    `json:"success"`
			SteamID string `json:"steamid"`
			Message string `json:"message"`
		} `json:"response"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("decode steam api json: %w", err)
	}
	// success 1 = OK; 42 is the usual "no match" code.
	if envelope.Response.Success != 1 {
		msg := strings.TrimSpace(envelope.Response.Message)
		if msg == "" {
			return nil, fmt.Errorf("steam api could not resolve vanity (success=%d)", envelope.Response.Success)
		}
		return nil, fmt.Errorf("steam api: %s", msg)
	}
	if envelope.Response.SteamID == "" {
		return nil, fmt.Errorf("steam api returned empty steamid")
	}
	return ToSteamID(envelope.Response.SteamID)
}
