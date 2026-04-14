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

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
)

func Test_readFileViaNTFS(t *testing.T) {
	log := logger.New(os.Stdout, true)

	// Read the actual amcache hive using the readFileViaNTFS function to ensure it can read the file and return valid registry data.
	amcachePath := "C:\\Windows\\AppCompat\\Programs\\Amcache.hve"

	// Read the file using
	data, err := readFileViaNTFS(amcachePath)
	assert.NoError(t, err, "readFileViaNTFS() failed: %v", err)
	assert.NotEmpty(t, data, "readFileViaNTFS() returned empty data")

	magic := []byte{0x72, 0x65, 0x67, 0x66} // "regf"
	log.Infof("readFileViaNTFS() returned data with magic: %x", data[:5])
	assert.True(t, bytes.HasPrefix(data, magic), "readFileViaNTFS() returned data with incorrect magic")

	registry, err := regparser.NewRegistry(bytes.NewReader(data))
	assert.NoError(t, err, "failed to create registry from NTFS data")
	assert.NotNil(t, registry, "registry is nil")

	keyNode := registry.OpenKey("Root\\InventoryApplication")
	assert.NotNil(t, keyNode, "failed to open key Root\\InventoryApplication")
}
