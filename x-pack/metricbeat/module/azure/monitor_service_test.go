// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azure

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetResourceTypeFromID(t *testing.T) {
	path := "subscriptions/qw3e45r6t-23ws-1234-6587-1234ed4532/resourceGroups/obs-infrastructure/providers/Microsoft.Compute/virtualMachines/obstestmemleak"
	rType := getResourceTypeFromId(path)
	assert.Equal(t, rType, "Microsoft.Compute/virtualMachines")
}

func TestGetResourceNameFromID(t *testing.T) {
	path := "subscriptions/qw3e45r6t-23ws-1234-6587-1234ed4532/resourceGroups/obs-infrastructure/providers/Microsoft.Compute/virtualMachines/obstestmemleak"
	name := getResourceNameFromId(path)
	assert.Equal(t, name, "obstestmemleak")
}
