// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/libbeat/common/otelbeat"
)

func TestOtel(t *testing.T) {
	rootDir, _ := filepath.Abs("../filebeat-otel.yml")

	// Create the command
	cmd := otelbeat.OTelCmd("filebeat")
	cmd.SetArgs([]string{"otel", "--config", rootDir})

	// Set up a context with a timeout to avoid indefinite blocking
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Run the command in a goroutine
	errCh := make(chan error, 1)
	go func() {
		err := cmd.Execute()
		errCh <- err
	}()

	// Wait for 15s to check there were no errors in starting the otel collector
	select {
	case err := <-errCh:
		// Assert no error occurred
		require.NoError(t, err, "cmd.RunE should not return an error")
	case <-ctx.Done():
		return
	}
}
