// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package templates

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetTemplateIndexPatternsToFilterOut(t *testing.T) {
	t.Run("should return default values when no environment variable is set", func(t *testing.T) {
		actual := GetTemplateIndexPatternsToFilterOut()
		assert.ElementsMatch(t, defaultExcludedTemplatePatterns, actual)
	})

	t.Run("should return patterns parsed from environment variable value if set", func(t *testing.T) {
		// Set the environment variable
		t.Setenv(IGNORE_TEMPLATES_BY_INDEX_PATTERN_NAME_NAME, ".custom-pattern1,.custom-pattern2")

		expected := []string{".custom-pattern1", ".custom-pattern2"}

		actual := GetTemplateIndexPatternsToFilterOut()
		assert.ElementsMatch(t, expected, actual)
	})

	t.Run("should return empty list when environment variable is set to empty string", func(t *testing.T) {
		// Set the environment variable to an empty string
		t.Setenv(IGNORE_TEMPLATES_BY_INDEX_PATTERN_NAME_NAME, "")

		// Expected result is an empty list
		expected := []string{}
		actual := GetTemplateIndexPatternsToFilterOut()
		assert.Equal(t, expected, actual)
	})

	t.Run("should return empty list when environment variable is set to blank spaces", func(t *testing.T) {
		// Set the environment variable to a string with only spaces
		t.Setenv(IGNORE_TEMPLATES_BY_INDEX_PATTERN_NAME_NAME, "    ")

		// Expected result is an empty list
		expected := []string{}
		actual := GetTemplateIndexPatternsToFilterOut()
		assert.Equal(t, expected, actual)
	})

	t.Run("should remove spaces around comma-separated values in environment variable", func(t *testing.T) {
		// Set the environment variable to a value with spaces around the commas
		t.Setenv(IGNORE_TEMPLATES_BY_INDEX_PATTERN_NAME_NAME, "  .custom-pattern1 , .custom-pattern2  ,   .custom-pattern3  ")

		// Expected result is the values split by comma, with no leading/trailing spaces
		expected := []string{".custom-pattern1", ".custom-pattern2", ".custom-pattern3"}
		actual := GetTemplateIndexPatternsToFilterOut()
		assert.ElementsMatch(t, expected, actual)
	})

	t.Run("should use default values if the env variable is malformed", func(t *testing.T) {
		t.Setenv(IGNORE_TEMPLATES_BY_INDEX_PATTERN_NAME_NAME, "  ,  ,   ")

		actual := GetTemplateIndexPatternsToFilterOut()
		assert.ElementsMatch(t, defaultExcludedTemplatePatterns, actual)
	})
}

func TestGetTemplateNamesToFilterOut(t *testing.T) {
	const envVar = IGNORE_TEMPLATES_BY_NAME_NAME

	t.Run("should return empty slice if var is empty", func(t *testing.T) {
		t.Setenv(envVar, "")

		actual := GetTemplateNamesToFilterOut()
		if actual != nil {
			t.Errorf("expected nil, actual %v", actual)
		}
	})

	t.Run("should return slice with single value", func(t *testing.T) {
		t.Setenv(envVar, "template1")

		actual := GetTemplateNamesToFilterOut()
		expected := []string{"template1"}
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("expected %v, actual %v", expected, actual)
		}
	})

	t.Run("should return multiple values", func(t *testing.T) {
		t.Setenv(envVar, "template1,template2")

		actual := GetTemplateNamesToFilterOut()
		expected := []string{"template1", "template2"}
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("expected %v, actual %v", expected, actual)
		}
	})
}
