package probe

import (
	"context"
	"fmt"
	"net"

	"github.com/yourusername/bulwark/internal/logging"
	"github.com/yourusername/bulwark/internal/state"
)

// TCPProbe performs TCP connection checks
type TCPProbe struct {
	host   string
	port   int
	config Config
	logger *logging.Logger
}

// NewTCPProbe creates a new TCP probe
func NewTCPProbe(host string, port int, config Config, logger *logging.Logger) *TCPProbe {
	return &TCPProbe{
		host:   host,
		port:   port,
		config: config,
		logger: logger.WithComponent("tcp-probe"),
	}
}

// Type returns the probe type
func (p *TCPProbe) Type() state.ProbeType {
	return state.ProbeTypeTCP
}

// Execute runs the TCP probe
func (p *TCPProbe) Execute(ctx context.Context) *state.ProbeResult {
	p.logger.Debug().
		Str("host", p.host).
		Int("port", p.port).
		Msg("Starting TCP probe")

	success, duration, message := executeWithRetries(ctx, p.config, func(ctx context.Context) error {
		return p.checkTCP(ctx)
	})

	result := &state.ProbeResult{
		Type:     p.Type(),
		Success:  success,
		Duration: duration,
		Message:  message,
	}

	if success {
		p.logger.Info().
			Str("address", fmt.Sprintf("%s:%d", p.host, p.port)).
			Dur("duration", duration).
			Msg("TCP probe succeeded")
	} else {
		p.logger.Warn().
			Str("address", fmt.Sprintf("%s:%d", p.host, p.port)).
			Str("error", message).
			Msg("TCP probe failed")
	}

	return result
}

// checkTCP performs a single TCP connection check
func (p *TCPProbe) checkTCP(ctx context.Context) error {
	address := fmt.Sprintf("%s:%d", p.host, p.port)

	dialer := &net.Dialer{
		Timeout: p.config.Timeout,
	}

	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	// Successfully connected, close immediately
	conn.Close()
	return nil
}
