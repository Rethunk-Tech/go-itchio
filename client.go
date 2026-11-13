package itchio

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/Rethunk-Tech/go-httpkit"
)

const (
	// BaseURL is the server-side API root (itch.io/docs/api/serverside). Only this host reads the key from the
	// Authorization header; itch.io/api/1 wants it in the address, where an error quoting the address would leak it.
	BaseURL = "https://api.itch.io"

	requestTimeout = 20 * time.Second
	// maxBody caps one API answer, so a hostile response cannot grow a caller's memory without bound.
	maxBody = 16 << 20
)

var (
	// ErrNeedsKey means the Client has no API key; nothing is sent.
	ErrNeedsKey = errors.New("needs an itch.io API key")
	// ErrBadKey means itch.io rejected the key: it is mistyped or revoked.
	ErrBadKey = errors.New("itch.io rejected the API key")
)

// Config is what a Client is built from.
type Config struct {
	// Key is the user's own itch.io API key, sent as a bearer token. Empty makes every call that needs one fail with
	// ErrNeedsKey.
	Key string
	// BaseURL replaces the API root.
	BaseURL string
	// HTTP sends every request; nil means a client with a default timeout. A redirect to another origin is refused
	// so the key stays with the API.
	HTTP *http.Client
	// UserAgent is sent with every request when set.
	UserAgent string
}

// Client calls the itch.io API. It holds no state beyond its configuration, so it is safe for concurrent use and
// cheap to build again when the key changes.
type Client struct {
	key, base, agent string
	hc               *http.Client
	// dl reads a download redirect instead of following it, from the caller's client with its own redirect policy
	// left out: a chain of same-origin hops would be followed here and the signed address lost.
	dl *http.Client
}

// New builds a Client.
func New(cfg Config) *Client {
	// The caller's CheckRedirect must not run here: an error from it would fail the very redirect this client reads.
	dl := httpkit.WithRedirect(cfg.HTTP, httpkit.NoFollow)
	dl.CheckRedirect = httpkit.NoFollow
	return &Client{
		key: strings.TrimSpace(cfg.Key), base: strings.TrimRight(cmp.Or(cfg.BaseURL, BaseURL), "/"),
		agent: cfg.UserAgent, hc: httpkit.WithRedirect(cfg.HTTP, httpkit.SameOrigin),
		dl: dl,
	}
}

// get sends one GET with the key and decodes the 200 answer into out. A 401 or 403 is ErrBadKey, and so is a 200
// whose errors list names an invalid key, which is how itch.io also reports one. The key is in a header only, so no
// error quotes it.
func (c *Client) get(ctx context.Context, path string, params url.Values, out any) error {
	if c.key == "" {
		return ErrNeedsKey
	}
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	u := c.base + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, http.NoBody)
	if err != nil {
		return fmt.Errorf("itchio: %w", err)
	}
	c.auth(req)
	raw, err := httpkit.DoCapped(c.hc, req, maxBody, nil)
	if err != nil {
		return mapStatus(err)
	}
	if err := errorsFrom(raw); err != nil {
		return err
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("itchio: %w", err)
	}
	return nil
}

// auth sets the headers every call carries.
func (c *Client) auth(req *http.Request) {
	if c.agent != "" {
		req.Header.Set("User-Agent", c.agent)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.key)
}

// mapStatus names the answer a caller treats specially: an authentication failure, whichever of 401 or 403 it
// arrives as.
func mapStatus(err error) error {
	if se, ok := errors.AsType[*httpkit.StatusError](err); ok &&
		(se.Code == http.StatusUnauthorized || se.Code == http.StatusForbidden) {
		return ErrBadKey
	}
	return err
}

// errorsFrom reads itch.io's 200-with-errors answer, which carries no other payload, and reports nil for any other
// body.
func errorsFrom(raw []byte) error {
	var env struct {
		Errors []string `json:"errors"`
	}
	if json.Unmarshal(raw, &env) == nil && len(env.Errors) > 0 {
		if slices.Contains(env.Errors, "invalid key") {
			return ErrBadKey
		}
		return fmt.Errorf("itch.io: %s", strings.Join(env.Errors, "; "))
	}
	return nil
}

// Me reads the key's account.
func (c *Client) Me(ctx context.Context) (User, error) {
	var out struct {
		User User `json:"user"`
	}
	err := c.get(ctx, "/profile", nil, &out)
	return out.User, err
}

// SearchGames lists the games matching query. itch.io answers every hit on the first page and ignores a page number,
// so a later page is empty and asks for nothing.
func (c *Client) SearchGames(ctx context.Context, query string, page int) ([]Game, error) {
	if c.key == "" {
		return nil, ErrNeedsKey
	}
	if page > 1 {
		return []Game{}, nil
	}
	var out struct {
		Games []Game `json:"games"`
	}
	params := url.Values{"query": {strings.TrimSpace(query)}}
	if err := c.get(ctx, "/search/games", params, &out); err != nil {
		return nil, err
	}
	return out.Games, nil
}

// Uploads lists the files attached to a game.
func (c *Client) Uploads(ctx context.Context, gameID string) ([]Upload, error) {
	var out struct {
		Uploads []Upload `json:"uploads"`
	}
	if err := c.get(ctx, "/games/"+url.PathEscape(gameID)+"/uploads", nil, &out); err != nil {
		return nil, err
	}
	return out.Uploads, nil
}

// DownloadURL asks for an upload's file and returns the short-lived signed address itch.io answers with. The API
// sends that address as a redirect rather than as JSON, so the redirect is read and not followed: the key never
// travels to the file host.
func (c *Client) DownloadURL(ctx context.Context, uploadID string) (string, error) {
	if c.key == "" {
		return "", ErrNeedsKey
	}
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	u := c.base + "/uploads/" + url.PathEscape(uploadID) + "/download"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, http.NoBody)
	if err != nil {
		return "", fmt.Errorf("itchio: %w", err)
	}
	c.auth(req)
	var link string
	var code int
	var status string
	raw, err := httpkit.DoCapped(c.dl, req, maxBody,
		func(resp *http.Response) error {
			code, status = resp.StatusCode, resp.Status
			if code >= http.StatusMultipleChoices && code < http.StatusBadRequest {
				link = resp.Header.Get("Location")
			}
			return nil
		})
	if link != "" {
		return c.resolveLink(req.URL, link)
	}
	if err != nil {
		return "", mapStatus(err)
	}
	if err := errorsFrom(raw); err != nil {
		return "", err
	}
	return "", &httpkit.StatusError{Code: code, Status: status}
}

// resolveLink makes a Location absolute against the request and refuses one that leaves https for another scheme, so
// the signed address is never fetched in the clear when the API was reached over TLS.
func (c *Client) resolveLink(from *url.URL, loc string) (string, error) {
	u, err := from.Parse(loc)
	if err != nil {
		return "", fmt.Errorf("itchio: the download address is not a URL: %w", err)
	}
	if from.Scheme == "https" && u.Scheme != "https" {
		return "", fmt.Errorf("itchio: refusing a download address on %s://", u.Scheme)
	}
	return u.String(), nil
}
