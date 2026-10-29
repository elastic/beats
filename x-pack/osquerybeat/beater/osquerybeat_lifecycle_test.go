// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/osqd"
)

// A Stop that arrives before Run (for example a receiver Shutdown right after
// Start) must not be lost: Run has to return without starting osqueryd.
func TestStopBeforeRunReturnsImmediately(t *testing.T) {
	var factoryCalls atomic.Int32
	ob, b, _ := newStatusTestBeater(t, func(ob *osquerybeat) {
		ob.osquerydFactory = func(string, ...osqd.Option) (osqd.Runner, error) {
			factoryCalls.Add(1)
			return &mockOsqueryd{}, nil
		}
	})

	ob.Stop()

	errCh := make(chan error, 1)
	go func() {
		errCh <- ob.Run(b)
	}()

	select {
	case err := <-errCh:
		assert.NoError(t, err, "Run after Stop should return nil")
	case <-time.After(time.Second):
		t.Fatal("Run did not return after a Stop delivered before it")
	}
	require.Zero(t, factoryCalls.Load(), "osqueryd must not be created after Stop")
}
