// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package googlecloud

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetRegionName(t *testing.T) {
	availabilityZone := "us-central1-a"
	region := getRegionName(availabilityZone)
	assert.Equal(t, "us-central1", region)
}
