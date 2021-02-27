// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package management

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func EnsureBlacklistItems(t *testing.T) {
	// NOTE: We do not permit to configure the console or the file output with CM for security reason.
	c := defaultConfig()
	v, _ := c.Blacklist.Patterns["output"]
	assert.Equal(t, "console|file", v)
}
