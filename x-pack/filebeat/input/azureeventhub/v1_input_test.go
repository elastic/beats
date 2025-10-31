// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix

package azureeventhub

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStripConnectionString(t *testing.T) {
	tests := []struct {
		connectionString, expected string
	}{
		{
			"Endpoint=sb://something",
			"(redacted)",
		},
		{
			"Endpoint=sb://dummynamespace.servicebus.windows.net/;SharedAccessKeyName=DummyAccessKeyName;SharedAccessKey=5dOntTRytoC24opYThisAsit3is2B+OGY1US/fuL3ly=",
			"Endpoint=sb://dummynamespace.servicebus.windows.net/",
		},
		{
			"Endpoint=sb://dummynamespace.servicebus.windows.net/;SharedAccessKey=5dOntTRytoC24opYThisAsit3is2B+OGY1US/fuL3ly=",
			"Endpoint=sb://dummynamespace.servicebus.windows.net/",
		},
	}

	for _, tt := range tests {
		res := stripConnectionString(tt.connectionString)
		assert.Equal(t, res, tt.expected)
	}
}
