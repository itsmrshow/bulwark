package probe

import (
	"context"
	"fmt"
	"net/http"

	"github.com/yourusername/bulwark/internal/logging"
	"github.com/yourusername/bulwark/internal/state"
)

// HTTPProbe performs HTTP health checks
type HTTPProbe struct {
	url          string
	expectStatus int
	config       Config
	logger       *logging.Logger
}

// NewHTTPProbe creates a new HTTP probe
func NewHTTPProbe(url string, expectStatus int, config Config, logger *logging.Logger) *HTTPProbe {
	if expectStatus == 0 {
		expectStatus = 200 // Default to 200 OK
	}

	return &HTTPProbe{
		url:          url,
		expectStatus: expectStatus,
		config:       config,
		logger:       logger.WithComponent("http-probe"),
	}
}

// Type returns the probe type
func (p *HTTPProbe) Type() state.ProbeType {
	return state.ProbeTypeHTTP
}

// Execute runs the HTTP probe
func (p *HTTPProbe) Execute(ctx context.Context) *state.ProbeResult {
	p.logger.Debug().
		Str("url", p.url).
		Int("expect_status", p.expectStatus).
		Msg("Starting HTTP probe")

	success, duration, message := executeWithRetries(ctx, p.config, func(ctx context.Context) error {
		return p.checkHTTP(ctx)
	})

	result := &state.ProbeResult{
		Type:     p.Type(),
		Success:  success,
		Duration: duration,
		Message:  message,
	}

	if success {
		p.logger.Info().
			Str("url", p.url).
			Dur("duration", duration).
			Msg("HTTP probe succeeded")
	} else {
		p.logger.Warn().
			Str("url", p.url).
			Str("error", message).
			Msg("HTTP probe failed")
	}

	return result
}

// checkHTTP performs a single HTTP check
func (p *HTTPProbe) checkHTTP(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", p.url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{
		Timeout: p.config.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Allow up to 10 redirects
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != p.expectStatus {
		return fmt.Errorf("unexpected status code: got %d, expected %d", resp.StatusCode, p.expectStatus)
	}

	return nil
}
