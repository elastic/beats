// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mage

import (
	"os"
	"strings"
)

type functionbeatProvider struct {
	Name   string
	GOOS   string
	GOARCH string
}

var (
	// SelectedProviders is the list of selected providers
	SelectedProviders = getConfiguredProviders()

	availableProviders = []functionbeatProvider{
		{Name: "aws", GOOS: "linux", GOARCH: "amd64"},
	}
)

func getConfiguredProviders() []functionbeatProvider {
	providersList := os.Getenv("PROVIDERS")
	if len(providersList) == 0 {
		return availableProviders
	}

	providers := make([]functionbeatProvider, 0)
	for _, name := range strings.Split(providersList, ",") {
		for _, provider := range availableProviders {
			if provider.Name == name {
				providers = append(providers, provider)
			}
		}
	}

	return providers
}
