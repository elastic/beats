// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration

package integration

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/osqd"
	"github.com/elastic/beats/v7/x-pack/otel/oteltest"
)

func receiverTestHome(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		return t.TempDir()
	}
	// macOS's default TMPDIR can exceed the Unix socket limit once osquery
	// appends its extension route. Keep the fallback socket base private and short.
	home, err := os.MkdirTemp("/tmp", "osq-")
	require.NoError(t, err, "create a short private receiver home")
	t.Cleanup(func() {
		assert.NoError(t, os.RemoveAll(home), "remove receiver home after shutdown")
	})
	t.Setenv("TMPDIR", home)
	return home
}

// ensureOsquerydAvailable skips the test if osqueryd is not available on the host.
func ensureOsquerydAvailable(t *testing.T) {
	t.Helper()
	// Honor a supplied binary directory, including macOS app-bundle paths.
	if dir := filepath.Clean(os.Getenv("OSQUERYBEAT_BINARY_DIR")); dir != "" && dir != "." {
		candidate := osqd.OsquerydPathForPlatform(runtime.GOOS, dir)
		if _, err := os.Stat(candidate); err == nil {
			return
		}
	}
	// Fall back to common install locations.
	for _, candidate := range []string{
		"osqueryd",
		"/usr/bin/osqueryd",
		"/usr/local/bin/osqueryd",
		"/opt/osquery/bin/osqueryd",
	} {
		if _, err := exec.LookPath(candidate); err == nil {
			return
		}
	}
	t.Skip("osqueryd not found; skipping osquerybeat OTel integration test")
}

// nopActionExtension supplies the action registration required by BeatReceiver.Start.
type nopActionExtension struct{}

func (nopActionExtension) Start(context.Context, component.Host) error { return nil }
func (nopActionExtension) Shutdown(context.Context) error              { return nil }
func (nopActionExtension) RegisterActionHandler(_ string, _ func(context.Context, map[string]any) (map[string]any, error)) error {
	return nil
}
func (nopActionExtension) UnregisterActionHandler(_ string) {}

// mockActionHost tracks receiver status and supplies its action registration.
type mockActionHost struct {
	*oteltest.MockHost
}

func (h *mockActionHost) GetExtensions() map[component.ID]component.Component {
	return map[component.ID]component.Component{
		component.MustNewID("elastic_diagnostics"): nopActionExtension{},
	}
}

// makeReceiverSettings builds receiver.Settings with an observed logger teed into
// sharedCore.  The otelcol.component.* context fields are added so that
// zapLogs.FilterField can separate logs from different receivers.
func makeReceiverSettings(factory receiver.Factory, name string, sharedCore zapcore.Core) receiver.Settings {
	id := component.NewIDWithName(factory.Type(), name)
	core := sharedCore.With([]zapcore.Field{
		zap.String("otelcol.component.id", id.String()),
		zap.String("otelcol.component.kind", "receiver"),
		zap.String("otelcol.signal", "logs"),
	})
	return receiver.Settings{
		ID: id,
		TelemetrySettings: component.TelemetrySettings{
			Logger: zap.New(core),
		},
	}
}
