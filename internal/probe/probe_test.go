package probe

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestExecuteWithRetries_Success(t *testing.T) {
	config := Config{
		Timeout:      1 * time.Second,
		Retries:      3,
		RetryBackoff: 10 * time.Millisecond,
	}

	success, _, msg := executeWithRetries(context.Background(), config, func(ctx context.Context) error {
		return nil
	})

	if !success {
		t.Error("expected success")
	}
	if msg != "probe succeeded" {
		t.Errorf("expected 'probe succeeded', got %q", msg)
	}
}

func TestExecuteWithRetries_RetryThenPass(t *testing.T) {
	config := Config{
		Timeout:      1 * time.Second,
		Retries:      3,
		RetryBackoff: 10 * time.Millisecond,
	}

	attempt := 0
	success, _, _ := executeWithRetries(context.Background(), config, func(ctx context.Context) error {
		attempt++
		if attempt < 3 {
			return fmt.Errorf("not ready")
		}
		return nil
	})

	if !success {
		t.Error("expected success after retries")
	}
	if attempt != 3 {
		t.Errorf("expected 3 attempts, got %d", attempt)
	}
}

func TestExecuteWithRetries_AllFail(t *testing.T) {
	config := Config{
		Timeout:      1 * time.Second,
		Retries:      2,
		RetryBackoff: 10 * time.Millisecond,
	}

	success, _, msg := executeWithRetries(context.Background(), config, func(ctx context.Context) error {
		return fmt.Errorf("always fails")
	})

	if success {
		t.Error("expected failure")
	}
	if msg != "always fails" {
		t.Errorf("expected 'always fails', got %q", msg)
	}
}

func TestExecuteWithRetries_ContextCancel(t *testing.T) {
	config := Config{
		Timeout:      1 * time.Second,
		Retries:      5,
		RetryBackoff: 1 * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel after first attempt
	attempt := 0
	success, _, _ := executeWithRetries(ctx, config, func(ctx context.Context) error {
		attempt++
		if attempt == 1 {
			cancel()
		}
		return fmt.Errorf("failed")
	})

	if success {
		t.Error("expected failure after context cancel")
	}
}

func TestExecuteWithRetries_Duration(t *testing.T) {
	config := Config{
		Timeout:      1 * time.Second,
		Retries:      1,
		RetryBackoff: 10 * time.Millisecond,
	}

	_, duration, _ := executeWithRetries(context.Background(), config, func(ctx context.Context) error {
		return nil
	})

	if duration <= 0 {
		t.Error("expected positive duration")
	}
}
