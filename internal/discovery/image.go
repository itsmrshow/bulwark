package discovery

import (
	"strings"

	"github.com/itsmrshow/bulwark/internal/docker"
)

// configImage returns the image a container was created from, if inspection
// exposed it.
func configImage(inspect docker.ContainerJSON) string {
	if inspect.Config == nil {
		return ""
	}
	return inspect.Config.Image
}

// resolveImageRef returns the image reference to track updates against.
//
// Docker's container list reports a repository reference only while the local
// image is still tagged. Once the tag moves elsewhere the image goes dangling
// and the list reports a bare image ID instead — a reference with no repository,
// which cannot be resolved against any registry. The container's config still
// records what it was created from, so use that as the fallback.
func resolveImageRef(listImage, configImage string) string {
	if configImage != "" && isImageID(listImage) {
		return configImage
	}
	return listImage
}

// isImageID reports whether the reference is a raw image ID rather than a
// repository reference.
func isImageID(image string) bool {
	if image == "" {
		return true
	}
	if strings.Contains(image, "/") {
		return false
	}

	digits := image
	if after, ok := strings.CutPrefix(image, "sha256:"); ok {
		digits = after
	} else {
		// Without the algorithm prefix, only an untagged hex string can be an
		// ID; anything carrying a tag separator is a repository reference.
		if strings.Contains(image, ":") || len(digits) < 12 {
			return false
		}
	}

	if digits == "" {
		return false
	}
	for _, r := range digits {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}
