// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package jumplists

import (
	"path/filepath"
	"strings"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
)

// LookupApplicationId looks up the application name for a given application id.
// knownAppIds is a map of application ids to application names, and is generated using go generate
// the generate directive is in the jumplists.go file
func LookupApplicationId(appId string) string {
	if _, ok := knownAppIds[appId]; ok {
		return knownAppIds[appId]
	}
	return ""
}

// ApplicationId is a struct that contains the application id and name.
// It is used to store the application id and name for a given jumplist.
type ApplicationId struct {
	Id   string `osquery:"application_id"`
	Name string `osquery:"application_name"`
}

// NewApplicationId creates a new ApplicationId object.
func NewApplicationId(id string) *ApplicationId {
	return &ApplicationId{Id: id, Name: LookupApplicationId(id)}
}

// GetAppIdFromFileName extracts the application id from the file name.
// It is used to create a new ApplicationId object from the file name.
func GetAppIdFromFileName(filePath string, log *logger.Logger) *ApplicationId {
	fileName := filepath.Base(filePath)
	dotIndex := strings.Index(fileName, ".")
	if dotIndex != -1 {
		return NewApplicationId(fileName[:dotIndex])
	}
	return NewApplicationId("")
}
