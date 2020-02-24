// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build !integration

package cloudfoundry

import (
	"testing"

	"github.com/elastic/beats/libbeat/common"

	"github.com/elastic/go-ucfg"

	"github.com/stretchr/testify/assert"
)

func TestValidation(t *testing.T) {
	var noIdOrSecret Config
	assert.Error(t, ucfg.New().Unpack(&noIdOrSecret))

	var noId Config
	assert.Error(t, ucfg.MustNewFrom(common.MapStr{
		"client_secret": "client_secret",
	}).Unpack(&noId))

	var noSecret Config
	assert.Error(t, ucfg.MustNewFrom(common.MapStr{
		"client_id": "client_id",
	}).Unpack(&noSecret))

	var valid Config
	assert.NoError(t, ucfg.MustNewFrom(common.MapStr{
		"client_id":     "client_id",
		"client_secret": "client_secret",
	}).Unpack(&valid))
}

func TestInitDefaults(t *testing.T) {
	var cfCfg Config
	assert.NoError(t, ucfg.MustNewFrom(common.MapStr{
		"client_id":     "client_id",
		"client_secret": "client_secret",
	}).Unpack(&cfCfg))
	assert.Len(t, cfCfg.ShardID, 36)
}
