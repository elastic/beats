// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package jumplists

import (
	"path/filepath"
	"strings"

	jumpliststypes "github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/tables/generated/jumplists"
)

// lookupApplicationID looks up the application name for a given application id.
// knownAppIds is a map of application ids to application names, and is generated using go generate
// the generate directive is in the jumplists.go file
func lookupApplicationID(appID string) string {
	if _, ok := knownAppIDs[appID]; ok {
		return knownAppIDs[appID]
	}
	return ""
}

// newApplicationID creates a new ApplicationId object.
func newApplicationID(id string) *jumpliststypes.ApplicationID {
	return &jumpliststypes.ApplicationID{ApplicationId: id, ApplicationName: lookupApplicationID(id)}
}

// getAppIdFromFileName extracts the application id from the file name.
// It is used to create a new ApplicationId object from the file name.
func getAppIdFromFileName(filePath string) *jumpliststypes.ApplicationID {
	fileName := filepath.Base(filePath)
	dotIndex := strings.Index(fileName, ".")
	if dotIndex != -1 {
		return newApplicationID(fileName[:dotIndex])
	}
	return newApplicationID("")
}
