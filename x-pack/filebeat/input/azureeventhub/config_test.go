// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix
// +build !aix

package azureeventhub

import (
	"testing"
)

func TestStorageContainerValidate(t *testing.T) {
	tests := []struct {
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
	t.Run("Check event hub names containing underscores", func(t *testing.T) {
		config := azureInputConfig{
			ConnectionString: "sb://test-ns.servicebus.windows.net/;SharedAccessKeyName=RootManageSharedAccessKey;SharedAccessKey=SECRET",
			EventHubName:     "eventhub_00",
			SAName:           "teststorageaccount",
			SAKey:            "secret",
			SAContainer:      "filebeat-activitylogs-eventhub_00",
		}

		if err := config.Validate(); err != nil {
			t.Fatal(err)
		}

		if config.SAContainer != "filebeat-activitylogs-eventhub-00" {
			t.Errorf("underscores (_) not replaced with hyphens (-): %s", config.SAContainer)
		}
	})
}
