package executor

import "testing"

func TestIsSameComposeService(t *testing.T) {
	tests := []struct {
		name          string
		targetProject string
		targetService string
		selfProject   string
		selfService   string
		expect        bool
	}{
		{
			name:          "matches same project and service",
			targetProject: "bulwark",
			targetService: "bulwark",
			selfProject:   "bulwark",
			selfService:   "bulwark",
			expect:        true,
		},
		{
			name:          "different service",
			targetProject: "bulwark",
			targetService: "web",
			selfProject:   "bulwark",
			selfService:   "bulwark",
			expect:        false,
		},
		{
			name:          "missing values",
			targetProject: "",
			targetService: "bulwark",
			selfProject:   "bulwark",
			selfService:   "bulwark",
			expect:        false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isSameComposeService(tc.targetProject, tc.targetService, tc.selfProject, tc.selfService)
			if got != tc.expect {
				t.Fatalf("expected %v, got %v", tc.expect, got)
			}
		})
	}
}

func TestAllowSelfUpdate(t *testing.T) {
	if !allowSelfUpdate("true") {
		t.Fatal("expected true to enable self-update")
	}
	if !allowSelfUpdate("YES") {
		t.Fatal("expected YES to enable self-update")
	}
	if allowSelfUpdate("false") {
		t.Fatal("expected false to disable self-update")
	}
}
