// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package release

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestVersion(t *testing.T) {
	t.Run("set version without qualifier", func(t *testing.T) {
		old := version
		defer func() { version = old }()
		version = "8.x.x"
		assert.Equal(t, Version(), version)
	})

	t.Run("set version with qualifier", func(t *testing.T) {
		old := version
		defer func() { version = old }()
		version = "8.x.x"
		qualifier = "alpha1"
		assert.Equal(t, Version(), version+"-"+qualifier)
	})

	t.Run("get commit hash", func(t *testing.T) {
		commit = "abc1234"
		assert.Equal(t, Commit(), commit)
	})

	t.Run("get build time", func(t *testing.T) {
		ts := time.Now().Format(time.RFC3339)
		old := buildTime
		defer func() { buildTime = old }()
		buildTime = ts
		assert.Equal(t, ts, BuildTime().Format(time.RFC3339))
	})
}
