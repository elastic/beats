// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsRecoverableOsqueryError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil",
			err:  nil,
			want: false,
		},
		{
			name: "extension ping timeout",
			err:  errors.New("extension ping failed: timeout after 200ms"),
			want: true,
		},
		{
			name: "extension ping socket write timeout",
			err:  errors.New("extension ping failed: write unix @->/var/run/1980071180/osquery.sock: i/o timeout"),
			want: true,
		},
		{
			name: "wrapped extension ping timeout",
			err:  fmt.Errorf("run osquery: %w", errors.New("extension ping failed: timeout after 200ms")),
			want: true,
		},
		{
			name: "eof",
			err:  errors.New("osquery failed: EOF"),
			want: true,
		},
		{
			name: "broken pipe",
			err:  errors.New("write: broken pipe"),
			want: true,
		},
		{
			name: "epipe op error",
			err:  &net.OpError{Op: "write", Err: syscall.EPIPE},
			want: true,
		},
		{
			name: "connection reset op error",
			err:  &net.OpError{Op: "read", Err: syscall.ECONNRESET},
			want: true,
		},
		{
			name: "net timeout",
			err:  &net.DNSError{Err: "i/o timeout", IsTimeout: true},
			want: true,
		},
		{
			name: "context canceled",
			err:  context.Canceled,
			want: false,
		},
		{
			name: "context deadline exceeded",
			err:  context.DeadlineExceeded,
			want: false,
		},
		{
			name: "unrelated error is not recoverable",
			err:  errors.New("invalid configuration"),
			want: false,
		},
		{
			name: "non-timeout net error is not recoverable",
			err:  &net.DNSError{Err: "no such host"},
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, isRecoverableOsqueryError(tc.err), "unexpected recoverability for %v", tc.err)
		})
	}
}

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

func TestRecoverableOsqueryProcessExit(t *testing.T) {
	exit78 := osqueryExitError(t, 78)
	assert.True(t, isRecoverableOsqueryError(fmt.Errorf("run: %w", exit78)), "wrapped EX_CONFIG logger shutdown must recover")
	assert.False(t, isRecoverableOsqueryError(osqueryExitError(t, 1)), "unclassified process exits must remain terminal")
	assert.False(t, isRecoverableOsqueryError(errors.Join(context.Canceled, exit78)), "cancellation must take precedence over process exit")
	assert.True(t, isRecoverableOsqueryError(ErrOsquerydExited), "unexpected successful process exit must recover")
}
