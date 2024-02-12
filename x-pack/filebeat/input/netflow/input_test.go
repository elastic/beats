// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration

package netflow

import (
	"testing"

	"github.com/elastic/beats/v7/libbeat/tests/resources"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/stretchr/testify/require"
)

func TestNewInputDone(t *testing.T) {

	goroutines := resources.NewGoroutinesChecker()
	defer goroutines.Check(t)

	config, err := conf.NewConfigFrom(mapstr.M{})
	require.NoError(t, err)

	_, err = Plugin(logp.NewLogger("netflow_test")).Manager.Create(config)
	require.NoError(t, err)
}
