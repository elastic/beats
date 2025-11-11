// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

//go:generate go run ../tools/scrape_app_ids.go
package parsers

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
