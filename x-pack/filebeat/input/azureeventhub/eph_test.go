// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azureeventhub

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	invalidConfig = azureInputConfig{
		SAKey:            "invalid_key",
		SAName:           "storage",
		SAContainer:      ephContainerName,
		ConnectionString: "invalid_connection_string",
		ConsumerGroup:    "$Default",
	}
)

func TestRunWithEPH(t *testing.T) {
	input := azureInput{config: invalidConfig}
	// decoding error when key is invalid
	err := input.runWithEPH()
	assert.Error(t, err, '7')
}
