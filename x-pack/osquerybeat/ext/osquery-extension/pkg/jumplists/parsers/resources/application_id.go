// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

// generate the application_id_generated.go file
//go:generate go run ../../generate -output=application_id_generated.go
package resources

import (
	"path/filepath"
	"strings"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
)

func LookupApplicationId(appId string) string {
	if _, ok := jumpListAppIds[appId]; ok {
		return jumpListAppIds[appId]
	}
	return ""
}

type ApplicationId struct {
	Id   string
	Name string
}

func NewApplicationId(id string) ApplicationId {
	return ApplicationId{Id: id, Name: LookupApplicationId(id)}
}

func GetAppIdFromFileName(filePath string, log *logger.Logger) ApplicationId {
	fileName := filepath.Base(filePath)
	dotIndex := strings.Index(fileName, ".")
	if dotIndex != -1 {
		return NewApplicationId(fileName[:dotIndex])
	}
	return NewApplicationId("")
}
