// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package streaming

import (
	"context"
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

func TestConnectWebSocketRetriesHonorContextCancellation(t *testing.T) {
	tests := []struct {
		name string
		cfg  retry
	}{
		{
			name: "infinite retries",
			cfg: retry{
				InfiniteRetries: true,
				WaitMin:         time.Second,
				WaitMax:         10 * time.Second,
			},
		},
		{
			name: "finite retries",
			cfg: retry{
				MaxAttempts: 100,
				WaitMin:     time.Second,
				WaitMax:     10 * time.Second,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			log := logptest.NewTestingLogger(t, "")

			// Cancel after a short delay so the first dial attempt
			// fails and the backoff wait is interrupted.
			time.AfterFunc(50*time.Millisecond, cancel)

			cfg := config{
				URL:   &urlConfig{URL: &url.URL{Scheme: "ws", Host: "localhost:0", Path: "/unreachable"}},
				Retry: &tt.cfg,
			}

			start := time.Now()
			_, _, err := connectWebSocket(ctx, cfg, cfg.URL.String(), noopReporter{}, log) //nolint:bodyclose // resp is always nil in this test
			elapsed := time.Since(start)

			if !errors.Is(err, context.Canceled) {
				t.Errorf("connectWebSocket() error = %v; want context.Canceled", err)
			}
			if elapsed >= tt.cfg.WaitMin {
				t.Errorf("connectWebSocket() took %v; want less than WaitMin (%v)", elapsed, tt.cfg.WaitMin)
			}
		})
	}
}
