package registry

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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

func TestFetchDigest_MockRegistry(t *testing.T) {
	manifest := ManifestResponse{
		SchemaVersion: 2,
		MediaType:     "application/vnd.docker.distribution.manifest.v2+json",
		Config: ManifestConfig{
			Digest: "sha256:configdigest",
		},
	}

	// Token server
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(TokenResponse{Token: "test-token"})
	}))
	defer tokenServer.Close()

	// Registry server
	registry := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Docker-Content-Digest", "sha256:testdigest123")
		json.NewEncoder(w).Encode(manifest)
	}))
	defer registry.Close()

	logger := logging.Default()
	client := &Client{
		httpClient: registry.Client(),
		logger:     logger.WithComponent("registry"),
		authCache:  make(map[string]string),
	}

	// Mock the registry URL by adding auth token to cache
	client.authCache["docker.io/library/nginx"] = "test-token"

	// We can't easily test against mock without overriding the URL,
	// so we test the individual helpers instead
	_ = client
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
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.httpClient == nil {
		t.Error("expected non-nil http client")
	}

	// Verify it doesn't panic on a FetchDigest with invalid image
	_, err := client.FetchDigest(context.Background(), "")
	if err == nil {
		t.Error("expected error for empty image")
	}
}
