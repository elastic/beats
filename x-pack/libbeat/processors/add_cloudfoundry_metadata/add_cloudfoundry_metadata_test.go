// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux darwin windows

package add_cloudfoundry_metadata

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

func TestNoClient(t *testing.T) {
	var testConfig = common.NewConfig()
	testConfig.SetString("client_id", -1, "client_id")
	testConfig.SetString("client_secret", -1, "client_secret")

	_, err := buildCloudFoundryMetadataProcessor(logp.L(), testConfig)
	assert.NoError(t, err, "initializing add_cloudfoundry_metadata processor")
}
