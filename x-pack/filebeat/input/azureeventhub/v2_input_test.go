// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix

package azureeventhub

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	inputv2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

func TestRunUpdatesStatusToStartingAndFailed(t *testing.T) {
	input, err := newEventHubInputV2(azureInputConfig{}, logp.NewLogger(inputName))
	require.NoError(t, err)

	eventHubInputV2, ok := input.(*eventHubInputV2)
	require.True(t, ok)

	// Mock the setup function to return an error, so the Run function exits early.
	// This allows us to test the initial status update without running the full input.
	eventHubInputV2.setupFn = func(ctx context.Context) error {
		return errors.New("setup failure")
	}

	ctx := t.Context()

	statusReporter := newMockStatusReporter()
	inputTestCtx := inputv2.Context{
		Logger:          logp.NewLogger(inputName),
		Cancelation:     ctx,
		StatusReporter:  statusReporter,
		MetricsRegistry: monitoring.NewRegistry(),
	}

	// The Run function is expected to return the error from the mock setup function.
	err = eventHubInputV2.Run(inputTestCtx, nil)
	require.Error(t, err, "setup failure")

	// Verify that the status was updated to Starting and then to Failed and Stopped.
	assert.Len(t, statusReporter.statuses, 4)
	assert.Equal(t, status.Starting, statusReporter.statuses[0])
	assert.Equal(t, status.Configuring, statusReporter.statuses[1])
	assert.Equal(t, status.Failed, statusReporter.statuses[2])
	assert.Equal(t, status.Stopped, statusReporter.statuses[3])
}

// mockStatusReporter is a mock implementation of the status.Reporter interface.
// It is used to verify that the input updates its status correctly.
type mockStatusReporter struct {
	mutex    sync.Mutex
	statuses []status.Status
}

func newMockStatusReporter() *mockStatusReporter {
	return &mockStatusReporter{}
}

func (r *mockStatusReporter) UpdateStatus(status status.Status, msg string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.statuses = append(r.statuses, status)
}
