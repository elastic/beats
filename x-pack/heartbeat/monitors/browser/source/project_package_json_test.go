// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.
//go:build linux || darwin || synthetics

package source

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProjectPackageJSONIncludesAllowScripts(t *testing.T) {
	symlinkPath := "file:/usr/share/elastic-agent/.node/node/lib/node_modules/@elastic/synthetics"
	pkg := newProjectPackageJSON(symlinkPath)

	raw, err := json.Marshal(pkg)
	require.NoError(t, err)

	var decoded map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &decoded))

	assert.Equal(t, "project-journey", decoded["name"])
	assert.Equal(t, true, decoded["private"])

	deps, ok := decoded["dependencies"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, symlinkPath, deps["@elastic/synthetics"])

	allow, ok := decoded["allowScripts"].(map[string]interface{})
	require.True(t, ok, "package.json must include allowScripts for npm >= 11.16")
	assert.Equal(t, true, allow["@elastic/synthetics"])
	assert.Equal(t, true, allow[symlinkPath])
	assert.Equal(t, true, allow["esbuild"])
	assert.Equal(t, true, allow["playwright-chromium"])
	assert.Equal(t, true, allow["sharp"])
}
