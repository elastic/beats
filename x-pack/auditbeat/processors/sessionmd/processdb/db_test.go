// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux

package processdb

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/logp"
)

var logger = logp.NewLogger("processdb")

func TestGetTTYType(t *testing.T) {
	assert.Equal(t, TTYConsole, getTTYType(4, 0))
	assert.Equal(t, Pts, getTTYType(136, 0))
	assert.Equal(t, TTY, getTTYType(4, 64))
	assert.Equal(t, TTYUnknown, getTTYType(1000, 1000))
}
