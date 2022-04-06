// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix
// +build !aix

package azureeventhub

import (
	"testing"

	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/stretchr/testify/assert"
)

var invalidConfig = azureInputConfig{
	SAKey:            "invalid_key",
	SAName:           "storage",
	SAContainer:      ephContainerName,
	ConnectionString: "invalid_connection_string",
	ConsumerGroup:    "$Default",
}

func TestRunWithEPH(t *testing.T) {
	input := azureInput{config: invalidConfig}
	// decoding error when key is invalid
	err := input.runWithEPH()
	assert.Error(t, err, '7')
}

func TestGetAzureEnvironment(t *testing.T) {
	resMan := ""
	env, err := getAzureEnvironment(resMan)
	assert.NoError(t, err)
	assert.Equal(t, env, azure.PublicCloud)
	resMan = "https://management.microsoftazure.de/"
	env, err = getAzureEnvironment(resMan)
	assert.NoError(t, err)
	assert.Equal(t, env, azure.GermanCloud)
	resMan = "http://management.invalidhybrid.com/"
	env, err = getAzureEnvironment(resMan)
	assert.Errorf(t, err, "invalid character 'F' looking for beginning of value")
	resMan = "<no value>"
	env, err = getAzureEnvironment(resMan)
	assert.NoError(t, err)
	assert.Equal(t, env, azure.PublicCloud)
}
