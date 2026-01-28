package executor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Definition describes a safe update pointer for loose containers.
type Definition struct {
	ComposePath string
	Service     string
}

// ParseDefinition parses bulwark.definition in the format:
// compose:/abs/path/to/compose.yml#service=serviceName
func ParseDefinition(definition string) (Definition, error) {
	if strings.TrimSpace(definition) == "" {
		return Definition{}, fmt.Errorf("definition is empty")
	}

	if !strings.HasPrefix(definition, "compose:") {
		return Definition{}, fmt.Errorf("definition must start with compose:")
	}

	raw := strings.TrimPrefix(definition, "compose:")
	parts := strings.SplitN(raw, "#", 2)
	if len(parts) != 2 {
		return Definition{}, fmt.Errorf("definition must include #service=")
	}

	path := parts[0]
	fragment := parts[1]
	if path == "" {
		return Definition{}, fmt.Errorf("compose path is empty")
	}
	if !filepath.IsAbs(path) {
		return Definition{}, fmt.Errorf("compose path must be absolute")
	}

	info, err := os.Stat(path)
	if err != nil {
		return Definition{}, fmt.Errorf("compose path not found: %w", err)
	}
	if info.IsDir() {
		return Definition{}, fmt.Errorf("compose path must be a file")
	}

	if !strings.HasPrefix(fragment, "service=") {
		return Definition{}, fmt.Errorf("definition fragment must start with service=")
	}
	service := strings.TrimPrefix(fragment, "service=")
	if service == "" {
		return Definition{}, fmt.Errorf("service name is empty")
	}

	return Definition{
		ComposePath: path,
		Service:     service,
	}, nil
}
