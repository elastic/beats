// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package templates

import "testing"

func GivenSomeIndexPatternsToExclude(t *testing.T, patterns ...string) {
	originalExclusionPatterns := TemplateIndexPatternsToIgnore

	TemplateIndexPatternsToIgnore = patterns

	t.Cleanup(func() {
		TemplateIndexPatternsToIgnore = originalExclusionPatterns
	})
}

func GivenSomeIndexNamesToExclude(t *testing.T, patterns ...string) {
	originalExclusionPatterns := TemplateIndexNamesToIgnore

	TemplateIndexNamesToIgnore = patterns

	t.Cleanup(func() {
		TemplateIndexNamesToIgnore = originalExclusionPatterns
	})
}

func GivenNoIndexPatternToExclude(t *testing.T) {
	GivenSomeIndexPatternsToExclude(t)
}
