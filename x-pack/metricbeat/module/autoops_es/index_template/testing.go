// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package index_template

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func getDefaultMappingObject(t *testing.T) []byte {
	unnamedDefaultMapping, err := os.ReadFile("./_meta/test/mapping.json")
	require.NoError(t, err)

	return unnamedDefaultMapping
}

func getMappingObject(t *testing.T, templateName string) string {
	if _, err := os.Stat("./_meta/test/" + templateName + ".mapping.json"); err == nil {
		mapping, err := os.ReadFile("./_meta/test/" + templateName + ".mapping.json")
		require.NoError(t, err)

		return string(mapping)
	}

	return strings.Replace(string(getDefaultMappingObject(t)), "$name", templateName, -1)
}

func getTemplateResponse(t *testing.T, templateNames []string, ignoredNames []string) []byte {
	mappings := `{"index_templates":[`
	added := 0

	for _, name := range templateNames {
		if slices.Contains(ignoredNames, name) {
			continue
		}

		if added != 0 {
			mappings += ","
		}

		mappings += getMappingObject(t, name)

		added++
	}

	return []byte(mappings + "]}")
}
