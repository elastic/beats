// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mage

type functionbeatProvider struct {
	Name   string
	GOOS   string
	GOARCH string
}

var (
	// SelectedProviders is the list of selected providers
	SelectedProviders = []functionbeatProvider{
		{Name: "aws", GOOS: "darwin", GOARCH: "amd64"},
	}
)
