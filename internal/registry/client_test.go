package registry

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/itsmrshow/bulwark/internal/logging"
)

func TestCompareDigests(t *testing.T) {
	tests := []struct {
		current   string
		remote    string
		different bool
	}{
		{"sha256:abc123", "sha256:abc123", false},
		{"abc123", "sha256:abc123", false},
		{"sha256:abc123", "abc123", false},
		{"abc123", "def456", true},
		{"sha256:abc123", "sha256:def456", true},
		{"", "", false},
	}

	for _, tt := range tests {
		got := CompareDigests(tt.current, tt.remote)
		if got != tt.different {
			t.Errorf("CompareDigests(%q, %q) = %v, want %v", tt.current, tt.remote, got, tt.different)
		}
	}
}

func TestParseImageReference(t *testing.T) {
	tests := []struct {
		image      string
		registry   string
		repository string
		tag        string
		wantErr    bool
	}{
		{"nginx", "docker.io", "library/nginx", "latest", false},
		{"nginx:1.25", "docker.io", "library/nginx", "1.25", false},
		{"user/app:v1", "docker.io", "user/app", "v1", false},
		{"ghcr.io/user/app:v2", "ghcr.io", "user/app", "v2", false},
		{"registry.example.com/org/app:latest", "registry.example.com", "org/app", "latest", false},
		{"", "", "", "", true},
		// A container running from a dangling image reports a bare digest with
		// no repository. Resolving it would query docker.io/library/sha256 and
		// get a guaranteed 401, so it must be rejected before any request.
		{"sha256:979202fb35994753a2adea609876fc24157edcf9a54c0acb0d180d6872c42960", "", "", "", true},
	}

	for _, tt := range tests {
		ref, err := ParseImageReference(tt.image)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseImageReference(%q) error = %v, wantErr %v", tt.image, err, tt.wantErr)
			continue
		}
		if tt.wantErr {
			continue
		}
		if ref.Registry != tt.registry {
			t.Errorf("ParseImageReference(%q).Registry = %q, want %q", tt.image, ref.Registry, tt.registry)
		}
		if ref.Repository != tt.repository {
			t.Errorf("ParseImageReference(%q).Repository = %q, want %q", tt.image, ref.Repository, tt.repository)
		}
		if ref.Tag != tt.tag {
			t.Errorf("ParseImageReference(%q).Tag = %q, want %q", tt.image, ref.Tag, tt.tag)
		}
	}
}

func TestParseImageReference_Digest(t *testing.T) {
	ref, err := ParseImageReference("nginx@sha256:abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ref.Digest != "sha256:abc123" {
		t.Errorf("expected digest=sha256:abc123, got %s", ref.Digest)
	}
	if ref.Tag != "" {
		t.Errorf("expected empty tag with digest, got %s", ref.Tag)
	}
}

func TestParseImageReference_TagAndDigest(t *testing.T) {
	// Docker Compose v2 pins running containers in repo:tag@sha256:... form.
	// Both the tag and digest must be preserved so FetchDigest can drop the
	// pin and resolve the tag's current digest.
	ref, err := ParseImageReference("lscr.io/linuxserver/radarr:latest@sha256:f08dda38e7d12e5a722d9a5cb6e54acaf63c8598fefeefec88effe0c0d0038dd")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ref.Registry != "lscr.io" {
		t.Errorf("Registry = %q, want lscr.io", ref.Registry)
	}
	if ref.Repository != "linuxserver/radarr" {
		t.Errorf("Repository = %q, want linuxserver/radarr", ref.Repository)
	}
	if ref.Tag != "latest" {
		t.Errorf("Tag = %q, want latest", ref.Tag)
	}
	if ref.Digest != "sha256:f08dda38e7d12e5a722d9a5cb6e54acaf63c8598fefeefec88effe0c0d0038dd" {
		t.Errorf("Digest = %q, want sha256:f08dda...", ref.Digest)
	}
}

func TestParseAuthChallenge(t *testing.T) {
	tests := []struct {
		header string
		realm  string
		svc    string
	}{
		{
			`Bearer realm="https://auth.docker.io/token",service="registry.docker.io",scope="repository:library/nginx:pull"`,
			"https://auth.docker.io/token",
			"registry.docker.io",
		},
		{"", "", ""},
		{"Basic realm=test", "", ""},
	}

	for _, tt := range tests {
		params := parseAuthChallenge(tt.header)
		if tt.realm == "" {
			if params != nil && params["realm"] != "" {
				t.Errorf("expected nil params for %q", tt.header)
			}
			continue
		}
		if params["realm"] != tt.realm {
			t.Errorf("realm = %q, want %q", params["realm"], tt.realm)
		}
		if params["service"] != tt.svc {
			t.Errorf("service = %q, want %q", params["service"], tt.svc)
		}
	}
}

// newTestClient wires a Client to a local TLS test server. Because
// ManifestURL() always builds https://<registry>/..., pointing ref.Registry at
// the httptest host is enough to exercise the real request path.
func newTestClient(srv *httptest.Server) *Client {
	c := NewClient(logging.Default())
	c.httpClient = srv.Client()
	return c
}

func testImage(srv *httptest.Server, repo string) string {
	return strings.TrimPrefix(srv.URL, "https://") + "/" + repo
}

func TestFetchDigest_MockRegistry(t *testing.T) {
	manifest := ManifestResponse{
		SchemaVersion: 2,
		MediaType:     "application/vnd.docker.distribution.manifest.v2+json",
		Config:        ManifestConfig{Digest: "sha256:configdigest"},
	}

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Docker-Content-Digest", "sha256:testdigest123")
		_ = json.NewEncoder(w).Encode(manifest)
	}))
	defer srv.Close()

	client := newTestClient(srv)
	digest, err := client.FetchDigest(context.Background(), testImage(srv, "library/nginx:latest"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if digest != "sha256:testdigest123" {
		t.Errorf("digest = %q, want sha256:testdigest123", digest)
	}
}

func TestFetchDigest_CachesAcrossCalls(t *testing.T) {
	var manifestHits int32

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&manifestHits, 1)
		w.Header().Set("Docker-Content-Digest", "sha256:cached")
		_ = json.NewEncoder(w).Encode(ManifestResponse{SchemaVersion: 2})
	}))
	defer srv.Close()

	client := newTestClient(srv)
	image := testImage(srv, "library/nginx:latest")

	for i := 0; i < 5; i++ {
		if _, err := client.FetchDigest(context.Background(), image); err != nil {
			t.Fatalf("call %d: unexpected error: %v", i, err)
		}
	}

	if got := atomic.LoadInt32(&manifestHits); got != 1 {
		t.Errorf("manifest requests = %d, want 1 (digest cache should absorb repeats)", got)
	}

	client.InvalidateDigests()
	if _, err := client.FetchDigest(context.Background(), image); err != nil {
		t.Fatalf("unexpected error after invalidate: %v", err)
	}
	if got := atomic.LoadInt32(&manifestHits); got != 2 {
		t.Errorf("manifest requests after invalidate = %d, want 2", got)
	}
}

func TestFetchDigest_CachesFailures(t *testing.T) {
	var hits int32

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"errors":[{"code":"MANIFEST_UNKNOWN"}]}`))
	}))
	defer srv.Close()

	client := newTestClient(srv)
	image := testImage(srv, "user/missing:latest")

	for i := 0; i < 3; i++ {
		if _, err := client.FetchDigest(context.Background(), image); err == nil {
			t.Fatalf("call %d: expected error", i)
		}
	}

	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Errorf("manifest requests = %d, want 1 (failures must be negatively cached)", got)
	}
}

func TestFetchDigest_RefreshesExpiredToken(t *testing.T) {
	const goodToken = "fresh-token"
	var manifestHits int32

	var srv *httptest.Server
	srv = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/token") {
			_ = json.NewEncoder(w).Encode(TokenResponse{Token: goodToken, ExpiresIn: 300})
			return
		}
		atomic.AddInt32(&manifestHits, 1)
		if r.Header.Get("Authorization") != "Bearer "+goodToken {
			w.Header().Set("WWW-Authenticate",
				`Bearer realm="`+srv.URL+`/token",service="test",scope="repository:user/app:pull"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Docker-Content-Digest", "sha256:afterrefresh")
		_ = json.NewEncoder(w).Encode(ManifestResponse{SchemaVersion: 2})
	}))
	defer srv.Close()

	client := newTestClient(srv)
	host := strings.TrimPrefix(srv.URL, "https://")

	// Seed an already-expired token: the client must discard it and re-auth
	// rather than returning a hard 401.
	client.setCachedToken(host+"/user/app", "stale-token", -time.Minute)

	digest, err := client.FetchDigest(context.Background(), testImage(srv, "user/app:latest"))
	if err != nil {
		t.Fatalf("expected token refresh to recover, got error: %v", err)
	}
	if digest != "sha256:afterrefresh" {
		t.Errorf("digest = %q, want sha256:afterrefresh", digest)
	}
}

func TestFetchDigest_DoesNotCacheCancelledRequests(t *testing.T) {
	var hits int32

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.Header().Set("Docker-Content-Digest", "sha256:ok")
		_ = json.NewEncoder(w).Encode(ManifestResponse{SchemaVersion: 2})
	}))
	defer srv.Close()

	client := newTestClient(srv)
	image := testImage(srv, "user/app:latest")

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.FetchDigest(cancelled, image); err == nil {
		t.Fatal("expected error from cancelled context")
	}

	digest, err := client.FetchDigest(context.Background(), image)
	if err != nil {
		t.Fatalf("cancellation poisoned the cache: %v", err)
	}
	if digest != "sha256:ok" {
		t.Errorf("digest = %q, want sha256:ok", digest)
	}
}

func TestCachedToken_RespectsExpiry(t *testing.T) {
	client := NewClient(logging.Default())

	client.setCachedToken("docker.io/library/nginx", "live", time.Minute)
	if tok, ok := client.cachedToken("docker.io/library/nginx"); !ok || tok != "live" {
		t.Errorf("cachedToken = (%q, %v), want (live, true)", tok, ok)
	}

	// A non-positive TTL means "registry did not say", so it falls back to the
	// default lifetime rather than expiring immediately.
	client.setCachedToken("docker.io/library/nginx", "no-ttl", 0)
	if tok, ok := client.cachedToken("docker.io/library/nginx"); !ok || tok != "no-ttl" {
		t.Errorf("cachedToken = (%q, %v), want (no-ttl, true)", tok, ok)
	}

	client.authCache["docker.io/library/nginx"] = cachedAuth{
		token:   "dead",
		expires: time.Now().Add(-time.Second),
	}
	if tok, ok := client.cachedToken("docker.io/library/nginx"); ok {
		t.Errorf("cachedToken returned expired token %q", tok)
	}
}

func TestImageReference_ManifestURL(t *testing.T) {
	ref := &ImageReference{
		Registry:   "docker.io",
		Repository: "library/nginx",
		Tag:        "latest",
	}
	url := ref.ManifestURL()
	expected := "https://registry-1.docker.io/v2/library/nginx/manifests/latest"
	if url != expected {
		t.Errorf("ManifestURL() = %q, want %q", url, expected)
	}
}

func TestImageReference_IsDockerHub(t *testing.T) {
	tests := []struct {
		registry string
		expected bool
	}{
		{"docker.io", true},
		{"registry-1.docker.io", true},
		{"ghcr.io", false},
	}
	for _, tt := range tests {
		ref := &ImageReference{Registry: tt.registry}
		if got := ref.IsDockerHub(); got != tt.expected {
			t.Errorf("IsDockerHub(%q) = %v, want %v", tt.registry, got, tt.expected)
		}
	}
}

func TestNewClient(t *testing.T) {
	logger := logging.Default()
	client := NewClient(logger)
	if client.httpClient == nil {
		t.Error("expected non-nil http client")
	}
	if client.digestTTL != DefaultDigestTTL {
		t.Errorf("digestTTL = %v, want %v", client.digestTTL, DefaultDigestTTL)
	}

	// Verify it doesn't panic on a FetchDigest with invalid image
	_, err := client.FetchDigest(context.Background(), "")
	if err == nil {
		t.Error("expected error for empty image")
	}
}
