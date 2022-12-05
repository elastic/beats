// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azureeventhub

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStorageContainerValidate(t *testing.T) {
	var tests = []struct {
		input    string
		errIsNil bool
	}{
		{"a-valid-name", true},
		{"a", false},
		{"a-name-that-is-really-too-long-to-be-valid-and-should-never-be-used-no-matter-what", false},
		{"-not-valid", false},
		{"not-valid-", false},
		{"not--valid", false},
		{"capital-A-not-valid", false},
		{"no_underscores_either", false},
	}
	for _, test := range tests {
		err := storageContainerValidate(test.input)
		if (err == nil) != test.errIsNil {
			t.Errorf("storageContainerValidate(%s) = %v", test.input, err)
		}
	}
}

func TestValidate(t *testing.T) {
	t.Run("Sanitize storage account containers with underscores", func(t *testing.T) {
		config := azureInputConfig{
			ConnectionString: "sb://test-ns.servicebus.windows.net/;SharedAccessKeyName=RootManageSharedAccessKey;SharedAccessKey=SECRET",
			EventHubName:     "event_hub_00",
			SAName:           "teststorageaccount",
			SAKey:            "secret",
			SAContainer:      "filebeat-activitylogs-event_hub_00",
		}

		if err := config.Validate(); err != nil {
			t.Fatalf("unexpected validation error: %v", err)
		}

		assert.Equal(
			t,
			"filebeat-activitylogs-event-hub-00",
			config.SAContainer,
			"underscores (_) not replaced with hyphens (-)",
		)
	})
}
