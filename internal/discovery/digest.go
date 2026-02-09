package discovery

import (
	"context"
	"strings"

	"github.com/itsmrshow/bulwark/internal/docker"
	"github.com/itsmrshow/bulwark/internal/registry"
)

// resolveRepoDigest returns a repo digest that matches the image reference.
// Falls back to the image ID if no repo digest is available.
func resolveRepoDigest(ctx context.Context, dockerClient *docker.Client, imageName, imageID string) string {
	if imageID == "" {
		return ""
	}

	inspect, err := dockerClient.ImageInspect(ctx, imageID)
	if err != nil {
		return imageID
	}

	ref, err := registry.ParseImageReference(imageName)
	if err != nil {
		return imageID
	}

	candidates := []string{ref.Repository}
	if ref.Registry != "" && ref.Registry != "docker.io" {
		candidates = append(candidates, ref.Registry+"/"+ref.Repository)
	} else {
		candidates = append(candidates,
			"docker.io/"+ref.Repository,
			"registry-1.docker.io/"+ref.Repository,
		)
	}

	for _, repoDigest := range inspect.RepoDigests {
		parts := strings.SplitN(repoDigest, "@", 2)
		if len(parts) != 2 {
			continue
		}
		repo := parts[0]
		digest := parts[1]
		for _, candidate := range candidates {
			if repo == candidate {
				return digest
			}
		}
	}

	if len(inspect.RepoDigests) == 1 {
		parts := strings.SplitN(inspect.RepoDigests[0], "@", 2)
		if len(parts) == 2 {
			return parts[1]
		}
	}

	return imageID
}
