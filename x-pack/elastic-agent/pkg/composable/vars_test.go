// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package composable

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVars_Replace(t *testing.T) {
	vars := &Vars{
		Mapping: map[string]interface{}{
			"un-der_score": map[string]interface{}{
				"key1": "data1",
				"key2": "data2",
			},
			"other": map[string]interface{}{
				"data": "info",
			},
		},
	}
	tests := []struct {
		Input   string
		Result  string
		Error   bool
		NoMatch bool
	}{
		{
			"{{un-der_score.key1}}",
			"data1",
			false,
			false,
		},
		{
			"{{un-der_score.missing}}",
			"",
			false,
			true,
		},
		{
			"{{un-der_score.missing|un-der_score.key2}}",
			"data2",
			false,
			false,
		},
		{
			"{{un-der_score.missing|un-der_score.missing2|other.data}}",
			"info",
			false,
			false,
		},
		{
			"{{un-der_score.missing|'fallback'}}",
			"fallback",
			false,
			false,
		},
		{
			`{{un-der_score.missing|||||||||"fallback"}}`,
			"fallback",
			false,
			false,
		},
		{
			`{{"direct"}}`,
			"direct",
			false,
			false,
		},
		{
			`{{un-der_score.}}`,
			"",
			true,
			false,
		},
		{
			`{{un-der_score.missing|"oth}}`,
			"",
			true,
			false,
		},
		{
			`{{un-der_score.missing`,
			"",
			true,
			false,
		},
		{
			`{{un-der_score.missing  {{other}}`,
			"",
			true,
			false,
		},
		{
			`{{}}`,
			"",
			true,
			false,
		},
		{
			"around {{un-der_score.key1}} the var",
			"around data1 the var",
			false,
			false,
		},
		{
			"multi {{un-der_score.key1}} var {{ un-der_score.missing |     un-der_score.key2      }} around",
			"multi data1 var data2 around",
			false,
			false,
		},
		{
			`multi {{un-der_score.key1}} var {{  un-der_score.missing|  'other"s with space'  }} around`,
			`multi data1 var other"s with space around`,
			false,
			false,
		},
		{
			`start {{  un-der_score.missing|  'others | with space'  }} end`,
			`start others | with space end`,
			false,
			false,
		},
		{
			`start {{  un-der_score.missing|  'other\'s with space'  }} end`,
			`start other's with space end`,
			false,
			false,
		},
	}
	for _, test := range tests {
		t.Run(test.Input, func(t *testing.T) {
			res, _, err := vars.Replace(test.Input)
			if test.Error {
				assert.Error(t, err)
			} else if test.NoMatch {
				assert.Error(t, ErrNoMatch, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.Result, res)
			}
		})
	}
}

func TestVars_ReplaceWithProcessors(t *testing.T) {
	processers := []map[string]interface{}{
		{
			"add_fields": map[string]interface{}{
				"dynamic": "added",
			},
		},
	}
	vars := &Vars{
		Mapping: map[string]interface{}{
			"testing": map[string]interface{}{
				"key1": "data1",
			},
			"dynamic": map[string]interface{}{
				"key1": "dynamic1",
			},
		},
		ProcessorsKey: "dynamic",
		Processors:    processers,
	}

	res, resProcessors, err := vars.Replace("{{testing.key1}}")
	require.NoError(t, err)
	assert.Nil(t, resProcessors)
	assert.Equal(t, "data1", res)

	res, resProcessors, err = vars.Replace("{{dynamic.key1}}")
	require.NoError(t, err)
	assert.Equal(t, processers, resProcessors)
	assert.Equal(t, "dynamic1", res)

	res, resProcessors, err = vars.Replace("{{other.key1|dynamic.key1}}")
	require.NoError(t, err)
	assert.Equal(t, processers, resProcessors)
	assert.Equal(t, "dynamic1", res)
}
