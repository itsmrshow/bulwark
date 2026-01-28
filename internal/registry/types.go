package registry

import (
	"fmt"
	"strings"
)

// ImageReference represents a parsed Docker image reference
type ImageReference struct {
	Registry   string // e.g., "docker.io", "ghcr.io"
	Repository string // e.g., "library/nginx", "user/image"
	Tag        string // e.g., "latest", "1.0.0"
	Digest     string // e.g., "sha256:abc123..."
}

// ParseImageReference parses an image reference into components
// Supports formats:
//   - nginx:latest
//   - docker.io/library/nginx:latest
//   - ghcr.io/user/image:v1.0.0
//   - nginx@sha256:abc123...
func ParseImageReference(image string) (*ImageReference, error) {
	if image == "" {
		return nil, fmt.Errorf("empty image reference")
	}

	ref := &ImageReference{
		Registry: "docker.io", // Default to Docker Hub
		Tag:      "latest",    // Default tag
	}

	// Handle digest
	if strings.Contains(image, "@") {
		parts := strings.Split(image, "@")
		image = parts[0]
		ref.Digest = parts[1]
		ref.Tag = "" // Digest overrides tag
	}

	// Handle tag
	if strings.Contains(image, ":") {
		parts := strings.Split(image, ":")
		// Check if this is a port (registry:port/repo) or a tag (repo:tag)
		if strings.Contains(parts[len(parts)-1], "/") {
			// This is a port, not a tag
			image = strings.Join(parts, ":")
		} else {
			// This is a tag
			ref.Tag = parts[len(parts)-1]
			image = strings.Join(parts[:len(parts)-1], ":")
		}
	}

	// Parse registry and repository
	parts := strings.Split(image, "/")

	switch len(parts) {
	case 1:
		// Just image name: nginx
		ref.Repository = "library/" + parts[0]
	case 2:
		// Check if first part is a registry
		if strings.Contains(parts[0], ".") || strings.Contains(parts[0], ":") {
			// registry/repo: ghcr.io/image
			ref.Registry = parts[0]
			ref.Repository = parts[1]
		} else {
			// user/repo: user/nginx
			ref.Repository = parts[0] + "/" + parts[1]
		}
	case 3:
		// registry/user/repo: ghcr.io/user/nginx
		ref.Registry = parts[0]
		ref.Repository = parts[1] + "/" + parts[2]
	default:
		// registry/org/team/repo: registry.example.com/org/team/image
		ref.Registry = parts[0]
		ref.Repository = strings.Join(parts[1:], "/")
	}

	return ref, nil
}

// String returns the full image reference
func (r *ImageReference) String() string {
	var sb strings.Builder

	if r.Registry != "" && r.Registry != "docker.io" {
		sb.WriteString(r.Registry)
		sb.WriteString("/")
	}

	sb.WriteString(r.Repository)

	if r.Digest != "" {
		sb.WriteString("@")
		sb.WriteString(r.Digest)
	} else if r.Tag != "" {
		sb.WriteString(":")
		sb.WriteString(r.Tag)
	}

	return sb.String()
}

// IsDockerHub returns true if this is a Docker Hub image
func (r *ImageReference) IsDockerHub() bool {
	return r.Registry == "docker.io" || r.Registry == "registry-1.docker.io"
}

// ManifestURL returns the URL to fetch the manifest
func (r *ImageReference) ManifestURL() string {
	registry := r.Registry
	if r.IsDockerHub() {
		registry = "registry-1.docker.io"
	}

	reference := r.Tag
	if r.Digest != "" {
		reference = r.Digest
	}

	return fmt.Sprintf("https://%s/v2/%s/manifests/%s", registry, r.Repository, reference)
}

// AuthURL returns the auth URL for this registry
func (r *ImageReference) AuthURL() string {
	if r.IsDockerHub() {
		return "https://auth.docker.io/token"
	}
	return fmt.Sprintf("https://%s/v2/", r.Registry)
}
