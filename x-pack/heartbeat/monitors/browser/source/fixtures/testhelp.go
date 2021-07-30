// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fixtures

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTodosFiles(t *testing.T, dir string) {
	expected := []string{
		"node_modules",
		"package.json",
		"helpers.ts",
		"add-remove.journey.ts",
		"basics.journey.ts",
	}
	for _, file := range expected {
		_, err := os.Stat(path.Join(dir, file))
		// assert, not require, because we want to proceed to the close bit
		assert.NoError(t, err)
	}
}
