// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
)

func TestNewFromConfig(t *testing.T) {
	cfg := config.MustNewConfigFrom(map[string]interface{}{
		"grpc": map[string]interface{}{
			"address": "0.0.0.0",
			"port":    9876,
		},
	})
	l := newErrorLogger(t)
	srv, err := NewFromConfig(l, cfg, &StubHandler{})
	require.NoError(t, err)
	assert.Equal(t, "0.0.0.0:9876", srv.getListenAddr())
}
