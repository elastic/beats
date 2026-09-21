// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

//go:build linux && !integration

package service

import (
	"context"
	"os"
	"runtime"
	"syscall"
	"testing"

	"github.com/coreos/go-systemd/v22/dbus"
	dbusRaw "github.com/godbus/dbus/v5"
	"github.com/stretchr/testify/require"
)

func TestDbusEnvConnection(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Test is linux-only")
	}

	// Set specific env var
	// This format is for the newer versions of the godbus/dbus library
	// Older versions use a format with out the `path` prefix.
	err := os.Setenv("DBUS_SYSTEM_BUS_ADDRESS", "unix:path=/var/run/dbus/system_bus_socket")
	require.NoError(t, err)

	// call internal dbus functions
	// This calls a lower-level bus library
	fetcher, err := instrospectForUnitMethods()
	require.NoError(t, err)
	require.NotNil(t, fetcher)

	// test the higher-level systemd library
	sysConn, err := dbus.NewWithContext(context.Background())
	require.NoError(t, err)
	t.Cleanup(sysConn.Close)
}

func TestClosePropertyFDs(t *testing.T) {
	r, w, err := os.Pipe()
	require.NoError(t, err, "failed to create pipe")

	dup, err := syscall.Dup(int(w.Fd()))
	require.NoError(t, err, "failed to dup pipe fd")
	require.NoError(t, w.Close(), "failed to close original write end")
	require.NoError(t, r.Close(), "failed to close read end")

	closePropertyFDs(map[string]any{
		"SomeFD": dbusRaw.UnixFD(dup),
	})

	err = syscall.Close(dup)
	require.Error(t, err, "expected close of already-closed fd to fail")
}
