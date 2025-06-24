// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package cat_template

import (
	"os"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

var defaultMappingObject []byte

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

	if defaultMappingObject == nil {
		defaultMappingObject = getDefaultMappingObject(t)
	}

	return string(defaultMappingObject)
}

func getTemplateResponse(t *testing.T, templateNames []string, ignoredNames []string) []byte {
	mappings := "{"
	added := 0

	for _, name := range templateNames {
		if slices.Contains(ignoredNames, name) {
			continue
		}

		if added != 0 {
			mappings += ","
		}

		mappings += `"` + name + `":` + getMappingObject(t, name)

		added++
	}

	return []byte(mappings + "}")
}
