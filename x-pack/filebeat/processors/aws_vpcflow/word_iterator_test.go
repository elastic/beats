// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws_vpcflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWordIterator(t *testing.T) {
	testCases := []struct {
		sentence string
		words    []string
	}{
		{"", nil},
		{" ", nil},
		{"  ", nil},
		{"word", []string{"word"}},
		{" word", []string{"word"}},
		{"word ", []string{"word"}},
		{" word ", []string{"word"}},
		{"foo  bar baz", []string{"foo", "bar", "baz"}},
	}

	for _, tc := range testCases {
		itr := wordIterator{source: tc.sentence}

		var collectedWords []string
		for itr.Next() {
			collectedWords = append(collectedWords, itr.Word())
		}

		assert.Equal(t, tc.words, collectedWords)
		assert.Equal(t, len(tc.words), itr.Count())
	}
}
