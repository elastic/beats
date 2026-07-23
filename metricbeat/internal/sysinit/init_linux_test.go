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

package sysinit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/management"
	"github.com/elastic/beats/v7/metricbeat/mb"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/paths"
	"github.com/elastic/gosigar"
)

type fakeMetricSet struct {
	mb.BaseMetricSet
}

func (m *fakeMetricSet) Fetch(r mb.ReporterV2) {}

func newFakeMetricSet(base mb.BaseMetricSet) (mb.MetricSet, error) {
	var ms mb.ReportingMetricSetV2 = &fakeMetricSet{BaseMetricSet: base}
	return ms, nil
}

func TestHostFSReloadResetsProcessGlobals(t *testing.T) {
	origProc := os.Getenv("HOST_PROC")
	origSys := os.Getenv("HOST_SYS")
	origEtc := os.Getenv("HOST_ETC")
	origProcd := gosigar.Procd
	t.Cleanup(func() {
		restoreEnv("HOST_PROC", origProc)
		restoreEnv("HOST_SYS", origSys)
		restoreEnv("HOST_ETC", origEtc)
		gosigar.Procd = origProcd
	})

	prevUnderAgent := management.UnderAgent()
	management.SetUnderAgent(true)
	t.Cleanup(func() { management.SetUnderAgent(prevUnderAgent) })

	info := beat.Info{Logger: logp.NewNopLogger(), Paths: paths.New()}

	reg := mb.NewRegister()
	require.NoError(t, reg.AddModule("system", InitSystemModule), "register system module")
	require.NoError(t, reg.AddMetricSet("system", "fake", newFakeMetricSet), "register fake metricset")

	// Configure hostfs to a custom root (as Agent would).
	_, _, err := mb.NewModule(mustConfig(t, map[string]any{
		"module":     "system",
		"metricsets": []string{"fake"},
		"hostfs":     "/hostfs",
		"period":     "10s",
	}), reg, info)
	require.NoError(t, err, "create module with hostfs set")
	assertHostFS(t, "/hostfs")

	// Change to a different hostfs path.
	_, _, err = mb.NewModule(mustConfig(t, map[string]any{
		"module":     "system",
		"metricsets": []string{"fake"},
		"hostfs":     "/other",
		"period":     "10s",
	}), reg, info)
	require.NoError(t, err, "create module with different hostfs")
	assertHostFS(t, "/other")

	// Unset hostfs on reload — globals must return to the default rooted at "/".
	_, _, err = mb.NewModule(mustConfig(t, map[string]any{
		"module":     "system",
		"metricsets": []string{"fake"},
		"period":     "10s",
	}), reg, info)
	require.NoError(t, err, "create module with hostfs unset")
	assertHostFS(t, "/")
}

func TestHostFSReloadWithoutAgent(t *testing.T) {
	origProc := os.Getenv("HOST_PROC")
	origSys := os.Getenv("HOST_SYS")
	origEtc := os.Getenv("HOST_ETC")
	origProcd := gosigar.Procd
	t.Cleanup(func() {
		restoreEnv("HOST_PROC", origProc)
		restoreEnv("HOST_SYS", origSys)
		restoreEnv("HOST_ETC", origEtc)
		gosigar.Procd = origProcd
	})

	prevUnderAgent := management.UnderAgent()
	management.SetUnderAgent(false)
	t.Cleanup(func() { management.SetUnderAgent(prevUnderAgent) })

	info := beat.Info{Logger: logp.NewNopLogger(), Paths: paths.New()}

	reg := mb.NewRegister()
	require.NoError(t, reg.AddModule("system", InitSystemModule), "register system module")
	require.NoError(t, reg.AddMetricSet("system", "fake", newFakeMetricSet), "register fake metricset")

	_, _, err := mb.NewModule(mustConfig(t, map[string]any{
		"module":     "system",
		"metricsets": []string{"fake"},
		"hostfs":     "/hostfs",
		"period":     "10s",
	}), reg, info)
	require.NoError(t, err, "create module with hostfs set")
	assertHostFS(t, "/hostfs")

	_, _, err = mb.NewModule(mustConfig(t, map[string]any{
		"module":     "system",
		"metricsets": []string{"fake"},
		"period":     "10s",
	}), reg, info)
	require.NoError(t, err, "create module with hostfs unset")
	assertHostFS(t, "/")
}

func assertHostFS(t *testing.T, root string) {
	t.Helper()
	assert.Equal(t, filepath.Join(root, "proc"), os.Getenv("HOST_PROC"), "HOST_PROC")
	assert.Equal(t, filepath.Join(root, "sys"), os.Getenv("HOST_SYS"), "HOST_SYS")
	assert.Equal(t, filepath.Join(root, "etc"), os.Getenv("HOST_ETC"), "HOST_ETC")
	assert.Equal(t, filepath.Join(root, "proc"), gosigar.Procd, "gosigar.Procd")
}

func mustConfig(t *testing.T, v map[string]any) *conf.C {
	t.Helper()
	c, err := conf.NewConfigFrom(v)
	require.NoError(t, err, "build config")
	return c
}

func restoreEnv(key, value string) {
	if value == "" {
		_ = os.Unsetenv(key)
		return
	}
	_ = os.Setenv(key, value)
}
