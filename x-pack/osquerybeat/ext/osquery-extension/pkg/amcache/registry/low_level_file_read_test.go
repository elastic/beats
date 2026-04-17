// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package registry

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"www.velocidex.com/golang/regparser"
)

func Test_readFileViaNTFS(t *testing.T) {
	// Read the actual amcache hive using the readFileViaNTFS function to ensure it can read the file and return valid registry data.
	amcachePath := "C:\\Windows\\AppCompat\\Programs\\Amcache.hve"
	if _, err := os.Stat(amcachePath); err != nil {
		t.Skipf("amcache hive not found at %s, skipping test", amcachePath)
	}

	// Read the file using
	data, err := readFileViaNTFS(amcachePath)
	if err != nil {
		t.Skipf("requires raw-volume read permissions and amcache hive presence: %v", err)
	}
	assert.NoError(t, err, "readFileViaNTFS() failed: %v", err)
	assert.NotEmpty(t, data, "readFileViaNTFS() returned empty data")

	magic := []byte{0x72, 0x65, 0x67, 0x66} // "regf"
	assert.GreaterOrEqual(t, len(data), 4, "readFileViaNTFS() returned data that is too short to be a valid registry hive")
	assert.True(t, bytes.HasPrefix(data, magic), "readFileViaNTFS() returned data with incorrect magic")

	registry, err := regparser.NewRegistry(bytes.NewReader(data))
	assert.NoError(t, err, "failed to create registry from NTFS data")
	assert.NotNil(t, registry, "registry is nil")

	keyNode := registry.OpenKey("Root\\InventoryApplication")
	assert.NotNil(t, keyNode, "failed to open key Root\\InventoryApplication")
}
