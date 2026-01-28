package state

import (
	"encoding/json"
	"time"
)

// TargetType represents the type of target (compose or container)
type TargetType string

const (
	TargetTypeCompose   TargetType = "compose"
	TargetTypeContainer TargetType = "container"
)

// Target represents a managed Docker resource
type Target struct {
	ID        string      `json:"id"`
	Type      TargetType  `json:"type"`
	Name      string      `json:"name"`
	Path      string      `json:"path"` // For compose: path to docker-compose.yml
	Services  []Service   `json:"services"`
	Labels    Labels      `json:"labels"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

// Service represents a single service/container
type Service struct {
	ID            string       `json:"id"`
	TargetID      string       `json:"target_id"`
	Name          string       `json:"name"`
	Image         string       `json:"image"`
	CurrentDigest string       `json:"current_digest"`
	Labels        Labels       `json:"labels"`
	HealthCheck   *HealthCheck `json:"health_check,omitempty"`
	CreatedAt     time.Time    `json:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at"`
}

// HealthCheck represents Docker HEALTHCHECK configuration
type HealthCheck struct {
	Test        []string      `json:"test"`
	Interval    time.Duration `json:"interval"`
	Timeout     time.Duration `json:"timeout"`
	StartPeriod time.Duration `json:"start_period"`
	Retries     int           `json:"retries"`
}

// Policy represents the update policy
type Policy string

const (
	PolicyNotify     Policy = "notify"     // Check only, no updates
	PolicySafe       Policy = "safe"       // Update with all health checks
	PolicyAggressive Policy = "aggressive" // Update with minimal checks
)

// Tier represents the service tier
type Tier string

const (
	TierStateless Tier = "stateless"
	TierStateful  Tier = "stateful"
)

// Labels holds parsed Bulwark configuration from container labels
type Labels struct {
	Enabled    bool        `json:"enabled"`
	Policy     Policy      `json:"policy"`
	Tier       Tier        `json:"tier"`
	Probe      ProbeConfig `json:"probe"`
	Definition string      `json:"definition"` // For loose containers: "compose:/path/to/compose.yml:service-name"
}

// ProbeType represents the type of health probe
type ProbeType string

const (
	ProbeTypeNone      ProbeType = "none"
	ProbeTypeDocker    ProbeType = "docker"    // Use Docker HEALTHCHECK
	ProbeTypeHTTP      ProbeType = "http"
	ProbeTypeTCP       ProbeType = "tcp"
	ProbeTypeLog       ProbeType = "log"
	ProbeTypeStability ProbeType = "stability"
)

// ProbeConfig defines health check configuration
type ProbeConfig struct {
	Type         ProbeType     `json:"type"`
	HTTPUrl      string        `json:"http_url,omitempty"`
	HTTPStatus   int           `json:"http_status,omitempty"`   // Expected status code (default 200)
	TCPHost      string        `json:"tcp_host,omitempty"`
	TCPPort      int           `json:"tcp_port,omitempty"`
	LogPattern   string        `json:"log_pattern,omitempty"`   // Regex pattern
	WindowSec    int           `json:"window_sec,omitempty"`    // For log probe: time window
	StabilitySec int           `json:"stability_sec,omitempty"` // Seconds to wait before declaring success
}

// UpdateCheck represents an available update
type UpdateCheck struct {
	Target       *Target `json:"target"`
	Service      *Service `json:"service"`
	RemoteDigest string  `json:"remote_digest"`
	UpdateNeeded bool    `json:"update_needed"`
	PolicyAllows bool    `json:"policy_allows"`
	Reason       string  `json:"reason"` // Why update is/isn't allowed
}

// UpdateResult represents the outcome of an update
type UpdateResult struct {
	TargetID          string        `json:"target_id"`
	ServiceID         string        `json:"service_id"`
	ServiceName       string        `json:"service_name"`
	Success           bool          `json:"success"`
	OldDigest         string        `json:"old_digest"`
	NewDigest         string        `json:"new_digest"`
	ProbeResults      []ProbeResult `json:"probe_results"`
	RollbackPerformed bool          `json:"rollback_performed"`
	RollbackDigest    string        `json:"rollback_digest,omitempty"`
	Error             error         `json:"error,omitempty"`
	StartedAt         time.Time     `json:"started_at"`
	CompletedAt       time.Time     `json:"completed_at"`
}

// ProbeResult represents the result of a single probe
type ProbeResult struct {
	Type     ProbeType     `json:"type"`
	Success  bool          `json:"success"`
	Duration time.Duration `json:"duration"`
	Message  string        `json:"message"`
}

// StateRecord persists update history
type StateRecord struct {
	ID           int64     `json:"id"`
	TargetID     string    `json:"target_id"`
	ServiceName  string    `json:"service_name"`
	Image        string    `json:"image"`
	OldDigest    string    `json:"old_digest"`
	NewDigest    string    `json:"new_digest"`
	Success      bool      `json:"success"`
	RolledBack   bool      `json:"rolled_back"`
	ProbesPassed int       `json:"probes_passed"`
	ProbesFailed int       `json:"probes_failed"`
	ErrorMessage string    `json:"error_message,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// ToJSON converts a struct to JSON string (for SQLite storage)
func ToJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// FromJSON parses JSON string into a struct
func FromJSON(data string, v interface{}) error {
	return json.Unmarshal([]byte(data), v)
}

// DefaultLabels returns default label values
func DefaultLabels() Labels {
	return Labels{
		Enabled: false,
		Policy:  PolicySafe,
		Tier:    TierStateless,
		Probe: ProbeConfig{
			Type:         ProbeTypeNone,
			HTTPStatus:   200,
			StabilitySec: 10,
		},
	}
}
