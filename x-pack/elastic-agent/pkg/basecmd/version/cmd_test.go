// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package version

import (
	"io/ioutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"github.com/elastic/beats/v7/libbeat/logp"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/control/server"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/cli"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/release"
)

func TestCmdBinaryOnly(t *testing.T) {
	streams, _, out, _ := cli.NewTestingIOStreams()
	cmd := NewCommandWithArgs(streams)
	cmd.Flags().Set("binary-only", "true")
	err := cmd.Execute()
	require.NoError(t, err)
	version, err := ioutil.ReadAll(out)

	require.NoError(t, err)
	assert.True(t, strings.Contains(string(version), "Binary: "))
	assert.False(t, strings.Contains(string(version), "Daemon: "))
}

func TestCmdBinaryOnlyYAML(t *testing.T) {
	streams, _, out, _ := cli.NewTestingIOStreams()
	cmd := NewCommandWithArgs(streams)
	cmd.Flags().Set("binary-only", "true")
	cmd.Flags().Set("yaml", "true")
	err := cmd.Execute()
	require.NoError(t, err)
	version, err := ioutil.ReadAll(out)

	require.NoError(t, err)

	var output Output
	err = yaml.Unmarshal(version, &output)
	require.NoError(t, err)

	assert.Nil(t, output.Daemon)
	assert.Equal(t, release.Info(), *output.Binary)
}

func TestCmdDaemon(t *testing.T) {
	srv := server.New(newErrorLogger(t), nil, nil, nil)
	require.NoError(t, srv.Start())
	defer srv.Stop()

	streams, _, out, _ := cli.NewTestingIOStreams()
	cmd := NewCommandWithArgs(streams)
	err := cmd.Execute()
	require.NoError(t, err)
	version, err := ioutil.ReadAll(out)

	require.NoError(t, err)
	assert.True(t, strings.Contains(string(version), "Binary: "))
	assert.True(t, strings.Contains(string(version), "Daemon: "))
}

func TestCmdDaemonYAML(t *testing.T) {
	srv := server.New(newErrorLogger(t), nil, nil, nil)
	require.NoError(t, srv.Start())
	defer srv.Stop()

	streams, _, out, _ := cli.NewTestingIOStreams()
	cmd := NewCommandWithArgs(streams)
	cmd.Flags().Set("yaml", "true")
	err := cmd.Execute()
	require.NoError(t, err)
	version, err := ioutil.ReadAll(out)

	require.NoError(t, err)

	var output Output
	err = yaml.Unmarshal(version, &output)
	require.NoError(t, err)

	assert.Equal(t, release.Info(), *output.Daemon)
	assert.Equal(t, release.Info(), *output.Binary)
}

func TestCmdDaemonErr(t *testing.T) {
	// srv not started
	streams, _, out, _ := cli.NewTestingIOStreams()
	cmd := NewCommandWithArgs(streams)
	err := cmd.Execute()
	require.Error(t, err)
	version, err := ioutil.ReadAll(out)
	require.NoError(t, err)

	assert.True(t, strings.Contains(string(version), "Binary: "))
	assert.True(t, strings.Contains(string(version), "Daemon: "))
}

func TestCmdDaemonErrYAML(t *testing.T) {
	// srv not started
	streams, _, out, _ := cli.NewTestingIOStreams()
	cmd := NewCommandWithArgs(streams)
	cmd.Flags().Set("yaml", "true")
	err := cmd.Execute()
	require.Error(t, err)
	version, err := ioutil.ReadAll(out)

	require.NoError(t, err)
	var output Output
	err = yaml.Unmarshal(version, &output)
	require.NoError(t, err)

	assert.Nil(t, output.Daemon)
	assert.Equal(t, release.Info(), *output.Binary)
}

func newErrorLogger(t *testing.T) *logger.Logger {
	t.Helper()

	loggerCfg := logger.DefaultLoggingConfig()
	loggerCfg.Level = logp.ErrorLevel

	log, err := logger.NewFromConfig("", loggerCfg, false)
	require.NoError(t, err)
	return log
}
