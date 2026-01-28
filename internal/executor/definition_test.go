package executor

import (
	"fmt"
	"os"
	"testing"
)

func TestParseDefinitionValid(t *testing.T) {
	tmp, err := os.CreateTemp("", "compose-*.yml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmp.Name())

	def := fmt.Sprintf("compose:%s#service=web", tmp.Name())
	parsed, err := ParseDefinition(def)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if parsed.ComposePath != tmp.Name() {
		t.Fatalf("expected compose path %q, got %q", tmp.Name(), parsed.ComposePath)
	}
	if parsed.Service != "web" {
		t.Fatalf("expected service %q, got %q", "web", parsed.Service)
	}
}

func TestParseDefinitionInvalid(t *testing.T) {
	tmp, err := os.CreateTemp("", "compose-*.yml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmp.Name())

	abs := tmp.Name()
	tests := []struct {
		name string
		def  string
	}{
		{"empty", ""},
		{"missing_prefix", abs + "#service=web"},
		{"missing_fragment", "compose:" + abs},
		{"missing_service", "compose:" + abs + "#service="},
		{"relative_path", "compose:relative/compose.yml#service=web"},
		{"missing_path", "compose:#service=web"},
		{"path_not_found", "compose:/no/such/file.yml#service=web"},
		{"bad_fragment", "compose:" + abs + "#name=web"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := ParseDefinition(test.def); err == nil {
				t.Fatalf("expected error for %q", test.def)
			}
		})
	}
}
