// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mage

import (
	"fmt"
	"os"
	"strings"
)

var (
	availableProviders = []ProviderDetails{
		{Name: "aws", Buildable: true, GOOS: "linux", GOARCH: "amd64"},
	}
)

// ProviderDetails stores information about the available cloud providers.
type ProviderDetails struct {
	Name      string
	Buildable bool
	GOOS      string
	GOARCH    string
}

// SelectedProviders is the list of selected providers
// Can be configured by setting PROVIDERS enviroment variable.
func SelectedProviders() ([]ProviderDetails, error) {
	providers := os.Getenv("PROVIDERS")
	if len(providers) == 0 {
		return availableProviders, nil
	}

	names := strings.Split(providers, ",")
	providerDetails := make([]ProviderDetails, len(names))
	for i, name := range names {
		p, err := findProviderDetails(name)
		if err != nil {
			return nil, err
		}
		providerDetails[i] = p
	}
	return providerDetails, nil
}

func findProviderDetails(name string) (ProviderDetails, error) {
	for _, p := range availableProviders {
		if p.Name == name {
			return p, nil
		}
	}

	return ProviderDetails{}, fmt.Errorf("no such provider")
}
