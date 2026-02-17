package probe

import (
	"context"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/itsmrshow/bulwark/internal/logging"
	"github.com/itsmrshow/bulwark/internal/state"
)

func TestTCPProbe_Success(t *testing.T) {
	// Start a TCP listener
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)

	logger := logging.Default()
	config := Config{
		Timeout:      5 * time.Second,
		Retries:      1,
		RetryBackoff: 10 * time.Millisecond,
	}

	probe := NewTCPProbe("127.0.0.1", addr.Port, config, logger)
	result := probe.Execute(context.Background())

	if !result.Success {
		t.Errorf("expected success, got failure: %s", result.Message)
	}
	if result.Type != state.ProbeTypeTCP {
		t.Errorf("expected type=tcp, got %s", result.Type)
	}
}

func TestTCPProbe_Failure(t *testing.T) {
	// Use a port that's not listening
	// Find an available port, then close it
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}
	port, _ := strconv.Atoi(listener.Addr().(*net.TCPAddr).String()[len("127.0.0.1:"):])
	listener.Close() // Close immediately so it's not listening

	logger := logging.Default()
	config := Config{
		Timeout:      500 * time.Millisecond,
		Retries:      1,
		RetryBackoff: 10 * time.Millisecond,
	}

	probe := NewTCPProbe("127.0.0.1", port, config, logger)
	result := probe.Execute(context.Background())

	if result.Success {
		t.Error("expected failure for closed port")
	}
}

func TestTCPProbe_Type(t *testing.T) {
	probe := NewTCPProbe("localhost", 80, Config{}, logging.Default())
	if probe.Type() != state.ProbeTypeTCP {
		t.Errorf("expected type=tcp, got %s", probe.Type())
	}
}
