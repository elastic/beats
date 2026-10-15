// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/osqd"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOsqueryExitHelper(t *testing.T) {
	if code := os.Getenv("OSQUERY_TEST_EXIT"); code != "" {
		value, err := strconv.Atoi(code)
		if err != nil {
			os.Exit(2)
		}
		os.Exit(value)
	}
}

func osqueryExitError(t *testing.T, code int) error {
	t.Helper()
	executable, err := os.Executable()
	require.NoError(t, err, "locate the test executable")
	command := exec.CommandContext(t.Context(), executable, "-test.run=^TestOsqueryExitHelper$")
	command.Env = append(os.Environ(), "OSQUERY_TEST_EXIT="+strconv.Itoa(code))
	err = command.Run()
	require.Error(t, err, "helper must exit unsuccessfully")
	var exitError *exec.ExitError
	require.ErrorAs(t, err, &exitError, "helper must produce a real process exit error")
	return err
}

func TestOsqueryExitClassification(t *testing.T) {
	exit78 := osqueryExitError(t, 78)
	otherExit := osqueryExitError(t, 1)
	for _, tc := range []struct {
		name        string
		ready       bool
		err         error
		recoverable bool
	}{
		{"startup exit 78", false, exit78, false},
		{"runtime exit 78", true, exit78, true},
		{"wrapped runtime exit 78", true, fmt.Errorf("process: %w", exit78), true},
		{"other runtime exit", true, otherExit, false},
		{"cancelled runtime exit", true, errors.Join(context.Canceled, exit78), false},
		{"deadline runtime exit", true, errors.Join(context.DeadlineExceeded, exit78), false},
		{"clean exit", true, nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var state osqueryRunState
			if tc.ready {
				state.ready()
			}
			err := state.exit(tc.err)
			assert.ErrorIs(t, err, tc.err, "preserve the original process error")
			assert.Equal(t, tc.recoverable, isRecoverableOsqueryError(err), "only runtime exit 78 adds recovery")
			state.ready()
			assert.Equal(t, osqueryExited, state.state.Load(), "late config callback must not revive an exited process")
		})
	}
}

func TestOsqueryRunnerRuntimeExit78ThenStartupFailure(t *testing.T) {
	exit78 := osqueryExitError(t, 78)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	var runs atomic.Int32
	runner := newOsqueryRunner(logptest.NewTestingLogger(t, "runner"))
	done := make(chan error, 1)
	go func() {
		done <- runner.Run(ctx, func(ctx context.Context, _ osqd.Flags, _ config.ExtensionsConfig, inputs <-chan []config.InputConfig) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-inputs:
			}
			var state osqueryRunState
			// More than five failures verifies that the existing unlimited retry
			// semantics remain intact. The replacement's startup failure is terminal.
			if runs.Add(1) <= 6 {
				state.ready()
			}
			return state.exit(exit78)
		})
	}()
	require.NoError(t, runner.Update(ctx, []config.InputConfig{{Osquery: &config.OsqueryConfig{}}}), "start runner")
	select {
	case err := <-done:
		assert.ErrorIs(t, err, exit78, "replacement startup failure must remain terminal")
		assert.Equal(t, int32(7), runs.Load(), "runtime errors must restart without a retry cap")
	case <-time.After(10 * time.Second):
		t.Fatal("runner did not restart and return its startup failure")
	}
}
