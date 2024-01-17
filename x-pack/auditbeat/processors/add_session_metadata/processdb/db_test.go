// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package processdb

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetTtyType(t *testing.T) {
	assert.Equal(t, TtyConsole, getTtyType(4, 0))
	assert.Equal(t, Pts, getTtyType(136, 0))
	assert.Equal(t, Tty, getTtyType(4, 64))
	assert.Equal(t, TtyUnknown, getTtyType(1000, 1000))
}
