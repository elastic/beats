// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.
package mage

import (
	"log"
	"os"
	"strings"
)

var (
	availableProviders = []string{
		"aws",
	}

	// SelectedProviders is the list of selected providers
	// Can be configured by setting PROVIDERS enviroment variable.
	SelectedProviders []string
)

func init() {
	SelectedProviders := getConfiguredProviders()
	log.Println("Selected providers:", SelectedProviders)
}

func getConfiguredProviders() []string {
	providers := os.Getenv("PROVIDERS")
	if len(providers) == 0 {
		return availableProviders
	}

	return strings.Split(providers, ",")
}
