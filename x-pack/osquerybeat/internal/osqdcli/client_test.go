// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package osqdcli

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
)

func TestResolveHitTypes(t *testing.T) {

	tests := []struct {
		name          string
		hit, colTypes map[string]string
		res           map[string]interface{}
	}{
		{
			name: "empty",
			res:  map[string]interface{}{},
		},
		{
			name: "resolvable",
			hit: map[string]string{
				"pid":      "5551",
				"pid_int":  "5552",
				"pid_uint": "5553",
				"pid_text": "5543",
				"foo":      "bar",
			},
			colTypes: map[string]string{
				"pid":      "BIGINT",
				"pid_int":  "INTEGER",
				"pid_uint": "UNSIGNED_BIGINT",
				"pid_text": "TEXT",
			},
			res: map[string]interface{}{
				"pid":      int64(5551),
				"pid_int":  int64(5552),
				"pid_uint": uint64(5553),
				"pid_text": "5543",
				"foo":      "bar",
			},
		},
		{
			// Should preserve the field if it can not be parsed into the type
			name: "wrong type",
			hit: map[string]string{
				"data": "0,22,137,138,29754,49154,49155",
				"foo":  "bar",
			},
			colTypes: map[string]string{"data": "BIGINT"},
			res: map[string]interface{}{
				"data": "0,22,137,138,29754,49154,49155",
				"foo":  "bar",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res := resolveHitTypes(tc.hit, tc.colTypes)
			diff := cmp.Diff(tc.res, res)
			if diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestClient_NilLogger(t *testing.T) {
	// Test that Client can be created and used without a logger (nil logger should not panic)
	client := New("/tmp/test.sock")

	// Verify logger is nil
	assert.Nil(t, client.log)

	// Verify defaults are set
	assert.Equal(t, defaultTimeout, client.timeout)
	assert.Equal(t, defaultMaxTimeout, client.maxTimeout)
	assert.Equal(t, defaultConnectRetries, client.connectRetries)
	assert.NotNil(t, client.cache)
	assert.NotNil(t, client.cliLimiter)
}

func TestClient_WithOptions(t *testing.T) {
	// Test that options are properly applied
	customTimeout := 5 * time.Minute
	customMaxTimeout := 10 * time.Hour
	customRetries := 5

	client := New("/tmp/test.sock",
		WithTimeout(customTimeout),
		WithMaxTimeout(customMaxTimeout),
		WithConnectRetries(customRetries),
		WithLogger(nil), // Explicitly set nil logger
	)

	assert.Nil(t, client.log)
	assert.Equal(t, customTimeout, client.timeout)
	assert.Equal(t, customMaxTimeout, client.maxTimeout)
	assert.Equal(t, customRetries, client.connectRetries)
}

func TestRetry_NilLogger(t *testing.T) {
	// Test that retry can work with nil logger without panicking
	r := retry{
		maxRetry:  2,
		retryWait: 10 * time.Millisecond,
		log:       nil, // nil logger
	}

	callCount := 0
	err := r.Run(context.Background(), func(ctx context.Context) error {
		callCount++
		if callCount < 2 {
			return assert.AnError
		}
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 2, callCount, "should have retried once")
}
