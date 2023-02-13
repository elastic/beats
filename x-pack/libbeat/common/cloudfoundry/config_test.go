// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package cloudfoundry

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/go-ucfg"
)

func TestValidation(t *testing.T) {
	var noIdOrSecret Config
	assert.Error(t, ucfg.New().Unpack(&noIdOrSecret))

	var noId Config
	assert.Error(t, ucfg.MustNewFrom(mapstr.M{
		"api_address":   "https://api.dev.cfdev.sh",
		"client_secret": "client_secret",
		"shard_id":      "beats-test-1",
	}).Unpack(&noId))

	var noSecret Config
	assert.Error(t, ucfg.MustNewFrom(mapstr.M{
		"api_address": "https://api.dev.cfdev.sh",
		"client_id":   "client_id",
		"shard_id":    "beats-test-1",
	}).Unpack(&noSecret))

	var noAPI Config
	assert.Error(t, ucfg.MustNewFrom(mapstr.M{
		"client_id":     "client_id",
		"client_secret": "client_secret",
		"shard_id":      "beats-test-1",
	}).Unpack(&noAPI))

	var noShardID Config
	assert.Error(t, ucfg.MustNewFrom(mapstr.M{
		"api_address":   "https://api.dev.cfdev.sh",
		"client_id":     "client_id",
		"client_secret": "client_secret",
	}).Unpack(&noShardID))

	var valid Config
	assert.NoError(t, ucfg.MustNewFrom(mapstr.M{
		"api_address":   "https://api.dev.cfdev.sh",
		"client_id":     "client_id",
		"client_secret": "client_secret",
		"shard_id":      "beats-test-1",
	}).Unpack(&valid))
}

func TestInitDefaults(t *testing.T) {
	var cfCfg Config
	assert.NoError(t, ucfg.MustNewFrom(mapstr.M{
		"api_address":   "https://api.dev.cfdev.sh",
		"client_id":     "client_id",
		"client_secret": "client_secret",
		"shard_id":      "beats-test-1",
	}).Unpack(&cfCfg))
	assert.Equal(t, ConsumerVersionV1, cfCfg.Version)
}
