// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one

// or more contributor license agreements. Licensed under the Elastic License;

// you may not use this file except in compliance with the Elastic License.

//go:build windows

package resources

import (
	"testing"
)

func TestApplicationID(t *testing.T) {
	for appId, name := range jumpListAppIds {
		applicationId := NewApplicationId(appId)
		if applicationId.Name != name {
			t.Errorf("NewApplicationId(%s) = %s, want %s", appId, applicationId.Name, name)
		}
	}
}
