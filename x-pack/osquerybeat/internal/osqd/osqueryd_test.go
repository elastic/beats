// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package osqd

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/elastic/elastic-agent-libs/logp"

	"github.com/gofrs/uuid/v5"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunCheckWithTimeout(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		err := runCheckWithTimeout(t.Context(), time.Second, func(context.Context) error {
			return nil
		})
		require.NoError(t, err, "successful check should not return an error")
	})

	t.Run("ordinary failure", func(t *testing.T) {
		wantErr := errors.New("check failed")
		err := runCheckWithTimeout(t.Context(), time.Second, func(context.Context) error {
			return wantErr
		})
		require.ErrorIs(t, err, wantErr, "ordinary check failure should be preserved")
	})

	t.Run("timeout", func(t *testing.T) {
		timeout := 20 * time.Millisecond
		startedAt := time.Now()
		err := runCheckWithTimeout(t.Context(), timeout, func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		})
		require.ErrorIs(t, err, context.DeadlineExceeded, "check timeout should wrap context deadline exceeded")
		assert.Contains(t, err.Error(), "osqueryd check timed out after 20ms", "timeout error should identify the check and duration")
		assert.Less(t, time.Since(startedAt), time.Second, "test check should be terminated promptly")
	})

	t.Run("parent cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		err := runCheckWithTimeout(ctx, time.Second, func(checkCtx context.Context) error {
			<-checkCtx.Done()
			return checkCtx.Err()
		})
		require.ErrorIs(t, err, context.Canceled, "parent cancellation should be preserved")
		assert.NotContains(t, err.Error(), "timed out", "parent cancellation should not be reported as a check timeout")
	})
}

func TestNew(t *testing.T) {

	socketPath := "/var/run/foobar"

	extensionsTimeout := 5
	configurationRefreshIntervalSecs := 12
	configPluginName := "config_plugin_test"
	loggerPluginName := "logger_plugin_test"

	osq, err := newOsqueryD(
		socketPath,
		WithExtensionsTimeout(extensionsTimeout),
		WithConfigRefresh(configurationRefreshIntervalSecs),
		WithConfigPlugin(configPluginName),
		WithLoggerPlugin(loggerPluginName),
		WithCheckTimeout(30*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}

	diff := cmp.Diff(extensionsTimeout, osq.extensionsTimeout)
	if diff != "" {
		t.Error(diff)
	}

	diff = cmp.Diff(configurationRefreshIntervalSecs, osq.configRefreshInterval)
	if diff != "" {
		t.Error(diff)
	}
	diff = cmp.Diff(configPluginName, osq.configPlugin)
	if diff != "" {
		t.Error(diff)
	}

	diff = cmp.Diff(loggerPluginName, osq.loggerPlugin)
	if diff != "" {
		t.Error(diff)
	}

	diff = cmp.Diff(30*time.Second, osq.checkTimeout)
	if diff != "" {
		t.Error(diff)
	}
}

func TestWithCheckTimeoutIgnoredWhenNonPositive(t *testing.T) {
	osq, err := newOsqueryD("/var/run/foobar", WithCheckTimeout(0))
	require.NoError(t, err, "newOsqueryD should succeed")
	assert.Equal(t, defaultCheckTimeout, osq.checkTimeout, "non-positive WithCheckTimeout should keep the default")
}

func TestVerifyAutoloadFileMissing(t *testing.T) {
	dir := uuid.Must(uuid.NewV4()).String()
	extensionAutoloadPath := filepath.Join(dir, osqueryAutoload)
	mandatoryExtensionPath := filepath.Join(dir, extensionName)
	err := verifyAutoloadFile(extensionAutoloadPath, mandatoryExtensionPath)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected error: %v, got: %v", os.ErrNotExist, err)
	}
}

// TestPrepareAutoloadFile tests possibly different states of the osquery.autoload file and that it is restored into the workable state
func TestPrepareAutoloadFile(t *testing.T) {
	validLogger := logp.NewLogger("osqueryd_test")

	// Prepare the directory with extension
	dir := t.TempDir()
	mandatoryExtensionPath := filepath.Join(dir, extensionName)

	// Write fake extension file for testing
	err := os.WriteFile(mandatoryExtensionPath, nil, 0600)
	if err != nil {
		t.Fatal(err)
	}

	// Run prepareAutoloadFile and verify the file is created and valid
	extensionAutoloadPath := filepath.Join(dir, osqueryAutoload)
	if err := prepareAutoloadFile(extensionAutoloadPath, mandatoryExtensionPath, validLogger); err != nil {
		t.Fatalf("prepareAutoloadFile failed: %v", err)
	}
	if err := verifyAutoloadFile(extensionAutoloadPath, mandatoryExtensionPath); err != nil {
		t.Fatalf("verifyAutoloadFile failed: %v", err)
	}

	// Idempotency: running it again should still validate
	if err := prepareAutoloadFile(extensionAutoloadPath, mandatoryExtensionPath, validLogger); err != nil {
		t.Fatalf("prepareAutoloadFile second run failed: %v", err)
	}
	if err := verifyAutoloadFile(extensionAutoloadPath, mandatoryExtensionPath); err != nil {
		t.Fatalf("verifyAutoloadFile second run failed: %v", err)
	}
}

func TestRunNotifiesExitBeforeLogging(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a POSIX shell helper")
	}
	dir := t.TempDir()
	binary := osquerydPath(dir)
	require.NoError(t, os.MkdirAll(filepath.Dir(binary), 0750), "create helper directory")
	require.NoError(t, os.WriteFile(binary, []byte("#!/bin/sh\nexit 78\n"), 0700), "write process helper")
	logging := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	defer unblock()
	core, _ := observer.New(zapcore.DebugLevel)
	logger, err := logp.ConfigureWithCoreLocal(logp.Config{}, zapcore.RegisterHooks(core, func(entry zapcore.Entry) error {
		if strings.Contains(entry.Message, "osqueryd process exited with error") {
			close(logging)
			<-release
		}
		return nil
	}))
	require.NoError(t, err, "create blocking exit logger")
	q, err := newOsqueryD(filepath.Join(dir, "rpc.sock"), WithBinaryPath(dir), WithExtensionPath(binary), WithDataPath(dir), WithLogger(logger))
	require.NoError(t, err, "create daemon runner")
	exited := make(chan error, 1)
	done := make(chan error, 1)
	go func() { done <- q.Run(t.Context(), nil, func(err error) { exited <- err }) }()
	select {
	case <-logging:
	case <-time.After(5 * time.Second):
		t.Fatal("process did not reach exit logging")
	}
	select {
	case err := <-exited:
		var exitError *exec.ExitError
		require.ErrorAs(t, err, &exitError, "notify with the process error")
		assert.Equal(t, 78, exitError.ExitCode(), "preserve process exit code")
	default:
		t.Fatal("exit callback did not run before blocked logging")
	}
	select {
	case <-done:
		t.Fatal("Run returned while exit logging was blocked")
	default:
	}
	unblock()
	select {
	case err := <-done:
		require.Error(t, err, "return the process failure after logging")
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not finish after logging was released")
	}
}
