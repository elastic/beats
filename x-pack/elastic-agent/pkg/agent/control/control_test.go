// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package control_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/logp"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/control/client"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/control/server"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/release"
)

func TestServerClient_Version(t *testing.T) {
	srv := server.New(newErrorLogger(t), nil)
	err := srv.Start()
	require.NoError(t, err)
	defer srv.Stop()

	c := client.New()
	err = c.Connect(context.Background())
	require.NoError(t, err)
	defer c.Disconnect()

	ver, err := c.Version(context.Background())
	require.NoError(t, err)

	assert.Equal(t, client.Version{
		Version:   release.Version(),
		Commit:    release.Commit(),
		BuildTime: release.BuildTime(),
		Snapshot:  release.Snapshot(),
	}, ver)
}

func newErrorLogger(t *testing.T) *logger.Logger {
	t.Helper()

	loggerCfg := logger.DefaultLoggingConfig()
	loggerCfg.Level = logp.ErrorLevel

	log, err := logger.NewFromConfig("", loggerCfg)
	require.NoError(t, err)
	return log
}
