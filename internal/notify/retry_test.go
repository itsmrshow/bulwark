package notify

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestSendWithRetry_Success(t *testing.T) {
	err := sendWithRetry(context.Background(), "test", func(ctx context.Context) (int, error) {
		return 200, nil
	})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestSendWithRetry_RetryThenSuccess(t *testing.T) {
	attempt := 0
	err := sendWithRetry(context.Background(), "test", func(ctx context.Context) (int, error) {
		attempt++
		if attempt < 3 {
			return 500, fmt.Errorf("server error")
		}
		return 200, nil
	})
	if err != nil {
		t.Errorf("expected success after retries, got %v", err)
	}
	if attempt != 3 {
		t.Errorf("expected 3 attempts, got %d", attempt)
	}
}

func TestSendWithRetry_AllFail(t *testing.T) {
	err := sendWithRetry(context.Background(), "test", func(ctx context.Context) (int, error) {
		return 500, fmt.Errorf("server error")
	})
	if err == nil {
		t.Error("expected error after all retries")
	}
}

func TestSendWithRetry_ClientError_NoRetry(t *testing.T) {
	attempt := 0
	err := sendWithRetry(context.Background(), "test", func(ctx context.Context) (int, error) {
		attempt++
		return 400, fmt.Errorf("bad request")
	})
	if err == nil {
		t.Error("expected error for client error")
	}
	if attempt != 1 {
		t.Errorf("expected 1 attempt for 4xx error, got %d", attempt)
	}
}

func TestSendWithRetry_RateLimit_Retries(t *testing.T) {
	attempt := 0
	err := sendWithRetry(context.Background(), "test", func(ctx context.Context) (int, error) {
		attempt++
		if attempt < 3 {
			return 429, fmt.Errorf("rate limited")
		}
		return 200, nil
	})
	if err != nil {
		t.Errorf("expected success after rate limit retries, got %v", err)
	}
	if attempt != 3 {
		t.Errorf("expected 3 attempts for rate limit, got %d", attempt)
	}
}

func TestSendWithRetry_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := sendWithRetry(ctx, "test", func(ctx context.Context) (int, error) {
		return 500, fmt.Errorf("server error")
	})
	if err == nil {
		t.Error("expected error when context cancelled")
	}
}
