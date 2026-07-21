package discovery

import "testing"

func TestResolveImageRef(t *testing.T) {
	const configImage = "lscr.io/linuxserver/bazarr:latest"

	tests := []struct {
		name      string
		listImage string
		config    string
		want      string
	}{
		{
			name:      "normal container keeps the listed reference",
			listImage: "lscr.io/linuxserver/bazarr:latest",
			config:    configImage,
			want:      "lscr.io/linuxserver/bazarr:latest",
		},
		{
			// When the local image is untagged, Docker reports a bare image ID
			// with no repository. Resolving it against a registry is impossible,
			// so fall back to what the container was created from.
			name:      "dangling image falls back to config",
			listImage: "sha256:979202fb35994753a2adea609876fc24157edcf9a54c0acb0d180d6872c42960",
			config:    configImage,
			want:      configImage,
		},
		{
			name:      "short image id falls back to config",
			listImage: "979202fb3599",
			config:    configImage,
			want:      configImage,
		},
		{
			name:      "no config to fall back to keeps the id",
			listImage: "sha256:979202fb35994753a2adea609876fc24157edcf9a54c0acb0d180d6872c42960",
			config:    "",
			want:      "sha256:979202fb35994753a2adea609876fc24157edcf9a54c0acb0d180d6872c42960",
		},
		{
			name:      "tagged image without a registry host is not an id",
			listImage: "postgres:16",
			config:    "postgres:16",
			want:      "postgres:16",
		},
		{
			name:      "bare repository name is not an id",
			listImage: "nginx",
			config:    "nginx",
			want:      "nginx",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveImageRef(tt.listImage, tt.config); got != tt.want {
				t.Errorf("resolveImageRef(%q, %q) = %q, want %q", tt.listImage, tt.config, got, tt.want)
			}
		})
	}
}
