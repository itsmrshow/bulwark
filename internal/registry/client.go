package registry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/itsmrshow/bulwark/internal/logging"
	"github.com/itsmrshow/bulwark/internal/metrics"
	"golang.org/x/sync/singleflight"
)

const (
	// DefaultDigestTTL is how long a successfully resolved digest is reused.
	// Digest lookups are the dominant source of registry traffic, and tags move
	// far more slowly than the UI polls, so caching them is what keeps Bulwark
	// under Docker Hub's anonymous pull limit.
	DefaultDigestTTL = 10 * time.Minute
	// DefaultDigestErrorTTL is the (shorter) reuse window for failed lookups.
	// Unresolvable images — private repos, dangling references — otherwise
	// re-fail on every single plan build.
	DefaultDigestErrorTTL = 5 * time.Minute
	// defaultTokenTTL is used when a registry omits expires_in.
	defaultTokenTTL = 5 * time.Minute
	// tokenExpiryMargin renews tokens slightly early to avoid racing expiry.
	tokenExpiryMargin = 30 * time.Second
)

type cachedAuth struct {
	token   string
	expires time.Time
}

type cachedDigest struct {
	digest  string
	err     error
	expires time.Time
}

// Client handles registry operations
type Client struct {
	httpClient *http.Client
	logger     *logging.Logger

	authMu     sync.RWMutex
	authCache  map[string]cachedAuth // registry/repository -> token
	tokenGroup singleflight.Group

	digestMu     sync.RWMutex
	digestCache  map[string]cachedDigest // registry/repository:reference -> digest
	digestTTL    time.Duration
	digestErrTTL time.Duration
	digestGroup  singleflight.Group
}

// NewClient creates a new registry client
func NewClient(logger *logging.Logger) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger:       logger.WithComponent("registry"),
		authCache:    make(map[string]cachedAuth),
		digestCache:  make(map[string]cachedDigest),
		digestTTL:    DefaultDigestTTL,
		digestErrTTL: DefaultDigestErrorTTL,
	}
}

// WithDigestTTL overrides how long resolved digests are cached. A non-positive
// TTL disables digest caching.
func (c *Client) WithDigestTTL(ttl time.Duration) *Client {
	c.digestMu.Lock()
	defer c.digestMu.Unlock()
	c.digestTTL = ttl
	if ttl <= 0 {
		c.digestErrTTL = 0
	} else if c.digestErrTTL > ttl {
		c.digestErrTTL = ttl
	}
	return c
}

// InvalidateDigests drops every cached digest so the next lookup hits the
// registry. Backs the manual refresh action in the UI.
func (c *Client) InvalidateDigests() {
	c.digestMu.Lock()
	defer c.digestMu.Unlock()
	c.digestCache = make(map[string]cachedDigest)
}

// ManifestResponse represents a Docker registry manifest
type ManifestResponse struct {
	SchemaVersion int             `json:"schemaVersion"`
	MediaType     string          `json:"mediaType"`
	Config        ManifestConfig  `json:"config"`
	Layers        []ManifestLayer `json:"layers"`
	Manifests     []ManifestEntry `json:"manifests,omitempty"` // For manifest lists
}

// ManifestConfig represents the config in a manifest
type ManifestConfig struct {
	MediaType string `json:"mediaType"`
	Size      int64  `json:"size"`
	Digest    string `json:"digest"`
}

// ManifestLayer represents a layer in a manifest
type ManifestLayer struct {
	MediaType string `json:"mediaType"`
	Size      int64  `json:"size"`
	Digest    string `json:"digest"`
}

// ManifestEntry represents an entry in a manifest list
type ManifestEntry struct {
	MediaType string           `json:"mediaType"`
	Size      int64            `json:"size"`
	Digest    string           `json:"digest"`
	Platform  ManifestPlatform `json:"platform"`
}

// ManifestPlatform represents platform information
type ManifestPlatform struct {
	Architecture string `json:"architecture"`
	OS           string `json:"os"`
	Variant      string `json:"variant,omitempty"`
}

// TokenResponse represents a Docker auth token response
type TokenResponse struct {
	Token       string `json:"token"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

// FetchDigest fetches the digest for an image, reusing a cached result when
// one is still fresh. Concurrent lookups of the same image collapse into a
// single registry request.
func (c *Client) FetchDigest(ctx context.Context, image string) (string, error) {
	ref, err := ParseImageReference(image)
	if err != nil {
		return "", fmt.Errorf("failed to parse image reference: %w", err)
	}

	// When a reference carries both a tag and a digest (e.g. Compose v2
	// pins running containers as repo:tag@sha256:...), querying by the
	// digest just echoes the pin back. The whole point of this call is
	// "what does the tag currently point to?" — so drop the digest and
	// resolve by tag.
	if ref.Tag != "" && ref.Digest != "" {
		ref.Digest = ""
	}

	cacheKey := ref.CacheKey()
	if digest, err, ok := c.cachedDigest(cacheKey); ok {
		return digest, err
	}

	type digestOutcome struct {
		digest string
		err    error
	}

	result, _, _ := c.digestGroup.Do(cacheKey, func() (interface{}, error) {
		if digest, err, ok := c.cachedDigest(cacheKey); ok {
			return digestOutcome{digest: digest, err: err}, nil
		}

		digest, err := c.fetchDigestUncached(ctx, ref)
		c.setCachedDigest(cacheKey, digest, err)
		return digestOutcome{digest: digest, err: err}, nil
	})

	outcome := result.(digestOutcome)
	return outcome.digest, outcome.err
}

func (c *Client) fetchDigestUncached(ctx context.Context, ref *ImageReference) (string, error) {
	start := time.Now()

	defer func() {
		metrics.DigestFetchDuration.WithLabelValues(ref.Registry).Observe(time.Since(start).Seconds())
	}()

	c.logger.Debug().
		Str("registry", ref.Registry).
		Str("repository", ref.Repository).
		Str("tag", ref.Tag).
		Msg("Fetching digest")

	// Get auth token if needed
	token, err := c.getAuthToken(ctx, ref)
	if err != nil {
		c.logger.Warn().Err(err).Msg("Failed to get auth token, trying without auth")
		token = ""
	}

	// Fetch manifest
	manifest, digest, err := c.fetchManifest(ctx, ref, token, true)
	if err != nil {
		return "", fmt.Errorf("failed to fetch manifest: %w", err)
	}

	// If we got a manifest list, we need to fetch the specific platform manifest
	if manifest.MediaType == "application/vnd.docker.distribution.manifest.list.v2+json" ||
		manifest.MediaType == "application/vnd.oci.image.index.v1+json" {
		if digest != "" {
			return digest, nil
		}

		c.logger.Debug().Msg("Got manifest list without digest header, selecting linux/amd64 platform")

		// Find linux/amd64 manifest
		for _, entry := range manifest.Manifests {
			if entry.Platform.OS == "linux" && entry.Platform.Architecture == "amd64" {
				return entry.Digest, nil
			}
		}

		// Fallback to first manifest
		if len(manifest.Manifests) > 0 {
			return manifest.Manifests[0].Digest, nil
		}
	}

	// Return the digest from the Docker-Content-Digest header
	if digest != "" {
		return digest, nil
	}

	// Fallback to config digest
	if manifest.Config.Digest != "" {
		return manifest.Config.Digest, nil
	}

	return "", fmt.Errorf("no digest found in manifest")
}

// fetchManifest fetches the manifest from the registry. When allowRetry is set
// a 401 triggers one re-authentication attempt: cached tokens are short-lived
// (Docker Hub issues 5-minute tokens), so an expired token must be replaced
// rather than reported as an auth failure.
func (c *Client) fetchManifest(ctx context.Context, ref *ImageReference, token string, allowRetry bool) (*ManifestResponse, string, error) {
	manifestURL := ref.ManifestURL()

	req, err := http.NewRequestWithContext(ctx, "GET", manifestURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.list.v2+json")
	req.Header.Add("Accept", "application/vnd.oci.image.manifest.v1+json")
	req.Header.Add("Accept", "application/vnd.oci.image.index.v1+json")

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch manifest: %w", err)
	}
	if resp.StatusCode == http.StatusUnauthorized && allowRetry {
		challenge := resp.Header.Get("WWW-Authenticate")
		_ = resp.Body.Close()

		cacheKey := fmt.Sprintf("%s/%s", ref.Registry, ref.Repository)
		// Whatever we sent was rejected; never hand it out again.
		c.invalidateToken(cacheKey)

		if challenge != "" {
			if newToken, ttl, err := c.getTokenFromChallenge(ctx, ref, challenge); err == nil && newToken != "" {
				c.setCachedToken(cacheKey, newToken, ttl)
				return c.fetchManifest(ctx, ref, newToken, false)
			}
		}
		if ref.IsDockerHub() {
			if newToken, ttl, err := c.getDockerHubToken(ctx, ref); err == nil && newToken != "" && newToken != token {
				c.setCachedToken(cacheKey, newToken, ttl)
				return c.fetchManifest(ctx, ref, newToken, false)
			}
		}
		return nil, "", fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	// Get digest from header
	digest := resp.Header.Get("Docker-Content-Digest")

	// Parse manifest
	var manifest ManifestResponse
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return nil, "", fmt.Errorf("failed to decode manifest: %w", err)
	}

	return &manifest, digest, nil
}

func (c *Client) getTokenFromChallenge(ctx context.Context, ref *ImageReference, challenge string) (string, time.Duration, error) {
	params := parseAuthChallenge(challenge)
	realm := params["realm"]
	if realm == "" {
		return "", 0, fmt.Errorf("auth challenge missing realm")
	}

	query := url.Values{}
	if service := params["service"]; service != "" {
		query.Set("service", service)
	}
	scope := params["scope"]
	if scope == "" {
		scope = fmt.Sprintf("repository:%s:pull", ref.Repository)
	}
	if scope != "" {
		query.Set("scope", scope)
	}

	tokenURL := realm
	if encoded := query.Encode(); encoded != "" {
		separator := "?"
		if strings.Contains(tokenURL, "?") {
			separator = "&"
		}
		tokenURL = tokenURL + separator + encoded
	}

	return c.requestToken(ctx, tokenURL)
}

// requestToken performs a registry token request and reports the token along
// with how long it stays valid.
func (c *Client) requestToken(ctx context.Context, tokenURL string) (string, time.Duration, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", tokenURL, nil)
	if err != nil {
		return "", 0, fmt.Errorf("failed to create token request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("failed to fetch token: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", 0, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", 0, fmt.Errorf("failed to decode token response: %w", err)
	}

	ttl := defaultTokenTTL
	if tokenResp.ExpiresIn > 0 {
		ttl = time.Duration(tokenResp.ExpiresIn) * time.Second
	}

	if tokenResp.Token != "" {
		return tokenResp.Token, ttl, nil
	}
	if tokenResp.AccessToken != "" {
		return tokenResp.AccessToken, ttl, nil
	}

	return "", 0, fmt.Errorf("no token in response")
}

func parseAuthChallenge(header string) map[string]string {
	challenge := strings.TrimSpace(header)
	if challenge == "" {
		return nil
	}
	parts := strings.SplitN(challenge, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return nil
	}

	params := make(map[string]string)
	for _, part := range strings.Split(parts[1], ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		value := strings.Trim(strings.TrimSpace(kv[1]), "\"")
		if key != "" {
			params[key] = value
		}
	}

	return params
}

// getAuthToken gets an auth token for the registry
func (c *Client) getAuthToken(ctx context.Context, ref *ImageReference) (string, error) {
	cacheKey := fmt.Sprintf("%s/%s", ref.Registry, ref.Repository)
	if token, ok := c.cachedToken(cacheKey); ok {
		return token, nil
	}

	// Docker Hub uses a different auth flow
	if ref.IsDockerHub() {
		token, err, _ := c.tokenGroup.Do(cacheKey, func() (interface{}, error) {
			if token, ok := c.cachedToken(cacheKey); ok {
				return token, nil
			}

			token, ttl, err := c.getDockerHubToken(ctx, ref)
			if err != nil {
				return "", err
			}
			c.setCachedToken(cacheKey, token, ttl)
			return token, nil
		})
		if err != nil {
			return "", err
		}
		return token.(string), nil
	}

	// For other registries, try anonymous access first
	return "", nil
}

func (c *Client) cachedToken(cacheKey string) (string, bool) {
	c.authMu.RLock()
	defer c.authMu.RUnlock()

	entry, ok := c.authCache[cacheKey]
	if !ok || time.Now().After(entry.expires) {
		return "", false
	}
	return entry.token, true
}

func (c *Client) setCachedToken(cacheKey, token string, ttl time.Duration) {
	if ttl <= 0 {
		ttl = defaultTokenTTL
	}
	// Renew a little before the registry would reject the token.
	if ttl > tokenExpiryMargin {
		ttl -= tokenExpiryMargin
	}

	c.authMu.Lock()
	defer c.authMu.Unlock()

	c.authCache[cacheKey] = cachedAuth{token: token, expires: time.Now().Add(ttl)}
}

func (c *Client) invalidateToken(cacheKey string) {
	c.authMu.Lock()
	defer c.authMu.Unlock()

	delete(c.authCache, cacheKey)
}

func (c *Client) cachedDigest(cacheKey string) (string, error, bool) {
	c.digestMu.RLock()
	defer c.digestMu.RUnlock()

	entry, ok := c.digestCache[cacheKey]
	if !ok || time.Now().After(entry.expires) {
		return "", nil, false
	}
	return entry.digest, entry.err, true
}

func (c *Client) setCachedDigest(cacheKey, digest string, err error) {
	c.digestMu.Lock()
	defer c.digestMu.Unlock()

	ttl := c.digestTTL
	if err != nil {
		// A cancelled or timed-out request says nothing about the image; caching
		// it would let one aborted HTTP request suppress lookups for minutes.
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return
		}
		ttl = c.digestErrTTL
	}
	if ttl <= 0 {
		return
	}

	c.digestCache[cacheKey] = cachedDigest{
		digest:  digest,
		err:     err,
		expires: time.Now().Add(ttl),
	}
}

// getDockerHubToken gets a token for Docker Hub
func (c *Client) getDockerHubToken(ctx context.Context, ref *ImageReference) (string, time.Duration, error) {
	// Docker Hub token URL
	tokenURL := "https://auth.docker.io/token"

	// Build request
	params := url.Values{}
	params.Set("service", "registry.docker.io")
	params.Set("scope", fmt.Sprintf("repository:%s:pull", ref.Repository))

	return c.requestToken(ctx, tokenURL+"?"+params.Encode())
}

// CompareDigests compares two digests and returns true if they're different
func CompareDigests(current, remote string) bool {
	// Normalize digests (remove sha256: prefix if present)
	current = strings.TrimPrefix(current, "sha256:")
	remote = strings.TrimPrefix(remote, "sha256:")

	return current != remote
}
