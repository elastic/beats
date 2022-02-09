// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFromConfig(t *testing.T) {
	l := newErrorLogger(t)
	cfg := &Config{
		Address: "0.0.0.0",
		Port:    9876,
	}
	srv, err := NewFromConfig(l, cfg, &StubHandler{})
	require.NoError(t, err)
	assert.Equal(t, "0.0.0.0:9876", srv.getListenAddr())
}
