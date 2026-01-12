package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"strings"
	"time"
)


type RetryConfig struct {
	MaxAttempts   int
	InitialDelay  time.Duration
	MaxDelay      time.Duration
	BackoffFactor float64
	Jitter        float64
	OnRetry       func(attempt int, err error, nextDelay time.Duration)
}


func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:   3,
		InitialDelay:  1 * time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
		Jitter:        0.1,
	}
}


var retryableErrors = []string{
	"connection reset",
	"connection refused",
	"timeout",
	"temporary failure",
	"network is unreachable",
	"no such host",
	"TLS handshake timeout",
	"i/o timeout",
	"EOF",
	"broken pipe",
}


var nonRetryableErrors = []string{
	"access denied",
	"AccessDenied",
	"InvalidAccessKeyId",
	"SignatureDoesNotMatch",
	"NoSuchBucket",
	"InvalidBucketName",
	"forbidden",
	"unauthorized",
	"invalid key",
}


func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	errLower := strings.ToLower(errStr)


	for _, pattern := range nonRetryableErrors {
		if strings.Contains(errLower, strings.ToLower(pattern)) {
			return false
		}
	}


	for _, pattern := range retryableErrors {
		if strings.Contains(errLower, strings.ToLower(pattern)) {
			return true
		}
	}


	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	return false
}


func WithRetry(ctx context.Context, cfg RetryConfig, fn func() error) error {
	var lastErr error

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err


		if !IsRetryableError(err) {
			return fmt.Errorf("non-retryable error: %w", err)
		}


		if attempt == cfg.MaxAttempts {
			break
		}


		delay := float64(cfg.InitialDelay)
		for i := 1; i < attempt; i++ {
			delay *= cfg.BackoffFactor
		}
		if delay > float64(cfg.MaxDelay) {
			delay = float64(cfg.MaxDelay)
		}


		if cfg.Jitter > 0 {
			jitter := delay * cfg.Jitter * (rand.Float64()*2 - 1)
			delay += jitter
		}

		nextDelay := time.Duration(delay)


		if cfg.OnRetry != nil {
			cfg.OnRetry(attempt, err, nextDelay)
		}


		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(nextDelay):
		}
	}

	return fmt.Errorf("failed after %d attempts: %w", cfg.MaxAttempts, lastErr)
}


type RetryingProvider struct {
	Provider Provider
	Config   RetryConfig
}


func NewRetryingProvider(p Provider, cfg RetryConfig) *RetryingProvider {
	return &RetryingProvider{
		Provider: p,
		Config:   cfg,
	}
}


func (r *RetryingProvider) Upload(ctx context.Context, key string, data io.Reader) error {




	return WithRetry(ctx, r.Config, func() error {
		return r.Provider.Upload(ctx, key, data)
	})
}


func (r *RetryingProvider) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	var result io.ReadCloser
	var resultErr error

	err := WithRetry(ctx, r.Config, func() error {
		var err error
		result, err = r.Provider.Download(ctx, key)
		resultErr = err
		return err
	})

	if err != nil {
		return nil, err
	}
	return result, resultErr
}


func (r *RetryingProvider) List(ctx context.Context, prefix string) ([]BackupItem, error) {
	var result []BackupItem

	err := WithRetry(ctx, r.Config, func() error {
		var err error
		result, err = r.Provider.List(ctx, prefix)
		return err
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}
