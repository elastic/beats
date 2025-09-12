// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package utils

import (
	"os"
)

var resourceID string = ""

// GetResourceID returns the global resource ID.
func GetResourceID() string {
	return resourceID
}

// GetAndSetResourceID returns the global resource ID, setting it if not already set.
func GetAndSetResourceID() string {
	if resourceID != "" {
		return resourceID
	}

	if deploymentID := os.Getenv("DEPLOYMENT_ID"); deploymentID != "" {
		SetResourceID(deploymentID)
		return deploymentID
	} else if projectID := os.Getenv("PROJECT_ID"); projectID != "" {
		SetResourceID(projectID)
		return projectID
	} else if resourceID := os.Getenv("RESOURCE_ID"); resourceID != "" {
		SetResourceID(resourceID)
		return resourceID
	}

	return ""
}

// SetResourceID sets the global resource ID.
func SetResourceID(id string) {
	resourceID = id
}

// ClearResourceID is intended to be used by tests to clear state.
func ClearResourceID() {
	resourceID = ""
}
