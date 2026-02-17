package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/itsmrshow/bulwark/internal/logging"
	"github.com/itsmrshow/bulwark/internal/metrics"
)

// Client handles registry operations
type Client struct {
	httpClient *http.Client
	logger     *logging.Logger
	authCache  map[string]string // registry -> token
}

// NewClient creates a new registry client
func NewClient(logger *logging.Logger) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger:    logger.WithComponent("registry"),
		authCache: make(map[string]string),
	}
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

// FetchDigest fetches the digest for an image
func (c *Client) FetchDigest(ctx context.Context, image string) (string, error) {
	start := time.Now()

	ref, err := ParseImageReference(image)
	if err != nil {
		return "", fmt.Errorf("failed to parse image reference: %w", err)
	}

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
	manifest, digest, err := c.fetchManifest(ctx, ref, token)
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

// fetchManifest fetches the manifest from the registry
func (c *Client) fetchManifest(ctx context.Context, ref *ImageReference, token string) (*ManifestResponse, string, error) {
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
	if resp.StatusCode == http.StatusUnauthorized && token == "" {
		challenge := resp.Header.Get("WWW-Authenticate")
		_ = resp.Body.Close()
		if challenge != "" {
			if newToken, err := c.getTokenFromChallenge(ctx, ref, challenge); err == nil && newToken != "" {
				cacheKey := fmt.Sprintf("%s/%s", ref.Registry, ref.Repository)
				c.authCache[cacheKey] = newToken
				return c.fetchManifest(ctx, ref, newToken)
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

func (c *Client) getTokenFromChallenge(ctx context.Context, ref *ImageReference, challenge string) (string, error) {
	params := parseAuthChallenge(challenge)
	realm := params["realm"]
	if realm == "" {
		return "", fmt.Errorf("auth challenge missing realm")
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

	req, err := http.NewRequestWithContext(ctx, "GET", tokenURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create token request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch token: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	if tokenResp.Token != "" {
		return tokenResp.Token, nil
	}
	if tokenResp.AccessToken != "" {
		return tokenResp.AccessToken, nil
	}

	return "", fmt.Errorf("no token in response")
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
	// Check cache
	cacheKey := fmt.Sprintf("%s/%s", ref.Registry, ref.Repository)
	if token, ok := c.authCache[cacheKey]; ok {
		return token, nil
	}

	// Docker Hub uses a different auth flow
	if ref.IsDockerHub() {
		token, err := c.getDockerHubToken(ctx, ref)
		if err != nil {
			return "", err
		}
		c.authCache[cacheKey] = token
		return token, nil
	}

	// For other registries, try anonymous access first
	return "", nil
}

// getDockerHubToken gets a token for Docker Hub
func (c *Client) getDockerHubToken(ctx context.Context, ref *ImageReference) (string, error) {
	// Docker Hub token URL
	tokenURL := "https://auth.docker.io/token"

	// Build request
	params := url.Values{}
	params.Set("service", "registry.docker.io")
	params.Set("scope", fmt.Sprintf("repository:%s:pull", ref.Repository))

	req, err := http.NewRequestWithContext(ctx, "GET", tokenURL+"?"+params.Encode(), nil)
	if err != nil {
		return "", fmt.Errorf("failed to create token request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch token: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	if tokenResp.Token != "" {
		return tokenResp.Token, nil
	}
	if tokenResp.AccessToken != "" {
		return tokenResp.AccessToken, nil
	}

	return "", fmt.Errorf("no token in response")
}

// CompareDigests compares two digests and returns true if they're different
func CompareDigests(current, remote string) bool {
	// Normalize digests (remove sha256: prefix if present)
	current = strings.TrimPrefix(current, "sha256:")
	remote = strings.TrimPrefix(remote, "sha256:")

	return current != remote
}
