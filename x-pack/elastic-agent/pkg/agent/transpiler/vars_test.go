// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package transpiler

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVars_Replace(t *testing.T) {
	vars := mustMakeVars(map[string]interface{}{
		"un-der_score": map[string]interface{}{
			"key1": "data1",
			"key2": "data2",
			"list": []string{
				"array1",
				"array2",
			},
			"dict": map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			},
		},
		"other": map[string]interface{}{
			"data": "info",
		},
	})
	tests := []struct {
		Input   string
		Result  Node
		Error   bool
		NoMatch bool
	}{
		{
			"${un-der_score.key1}",
			NewStrVal("data1"),
			false,
			false,
		},
		{
			"${un-der_score.missing}",
			NewStrVal(""),
			false,
			true,
		},
		{
			"${un-der_score.missing|un-der_score.key2}",
			NewStrVal("data2"),
			false,
			false,
		},
		{
			"${un-der_score.missing|un-der_score.missing2|other.data}",
			NewStrVal("info"),
			false,
			false,
		},
		{
			"${un-der_score.missing|'fallback'}",
			NewStrVal("fallback"),
			false,
			false,
		},
		{
			`${un-der_score.missing|||||||||"fallback"}`,
			NewStrVal("fallback"),
			false,
			false,
		},
		{
			`${"direct"}`,
			NewStrVal("direct"),
			false,
			false,
		},
		{
			`${un-der_score.}`,
			NewStrVal(""),
			true,
			false,
		},
		{
			`${un-der_score.missing|"oth}`,
			NewStrVal(""),
			true,
			false,
		},
		{
			`${un-der_score.missing`,
			NewStrVal(""),
			true,
			false,
		},
		{
			`${un-der_score.missing  ${other}`,
			NewStrVal(""),
			true,
			false,
		},
		{
			`${}`,
			NewStrVal(""),
			true,
			false,
		},
		{
			"around ${un-der_score.key1} the var",
			NewStrVal("around data1 the var"),
			false,
			false,
		},
		{
			"multi ${un-der_score.key1} var ${ un-der_score.missing |     un-der_score.key2      } around",
			NewStrVal("multi data1 var data2 around"),
			false,
			false,
		},
		{
			`multi ${un-der_score.key1} var ${  un-der_score.missing|  'other"s with space'  } around`,
			NewStrVal(`multi data1 var other"s with space around`),
			false,
			false,
		},
		{
			`start ${  un-der_score.missing|  'others | with space'  } end`,
			NewStrVal(`start others | with space end`),
			false,
			false,
		},
		{
			`start ${  un-der_score.missing|  'other\'s with space'  } end`,
			NewStrVal(`start other's with space end`),
			false,
			false,
		},
		{
			`${un-der_score.list}`,
			NewList([]Node{
				NewStrVal("array1"),
				NewStrVal("array2"),
			}),
			false,
			false,
		},
		{
			`list inside string ${un-der_score.list} causes no match`,
			NewList([]Node{
				NewStrVal("array1"),
				NewStrVal("array2"),
			}),
			false,
			true,
		},
		{
			`${un-der_score.dict}`,
			NewDict([]Node{
				NewKey("key1", NewStrVal("value1")),
				NewKey("key2", NewStrVal("value2")),
			}),
			false,
			false,
		},
		{
			`dict inside string ${un-der_score.dict} causes no match`,
			NewDict([]Node{
				NewKey("key1", NewStrVal("value1")),
				NewKey("key2", NewStrVal("value2")),
			}),
			false,
			true,
		},
	}
	for _, test := range tests {
		t.Run(test.Input, func(t *testing.T) {
			res, err := vars.Replace(test.Input)
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
	processers := Processors{
		{
			"add_fields": map[string]interface{}{
				"dynamic": "added",
			},
		},
	}
	vars, err := NewVarsWithProcessors(
		map[string]interface{}{
			"testing": map[string]interface{}{
				"key1": "data1",
			},
			"dynamic": map[string]interface{}{
				"key1": "dynamic1",
				"list": []string{
					"array1",
					"array2",
				},
				"dict": map[string]string{
					"key1": "value1",
					"key2": "value2",
				},
			},
		},
		"dynamic",
		processers)
	require.NoError(t, err)

	res, err := vars.Replace("${testing.key1}")
	require.NoError(t, err)
	assert.Equal(t, NewStrVal("data1"), res)

	res, err = vars.Replace("${dynamic.key1}")
	require.NoError(t, err)
	assert.Equal(t, NewStrValWithProcessors("dynamic1", processers), res)

	res, err = vars.Replace("${other.key1|dynamic.key1}")
	require.NoError(t, err)
	assert.Equal(t, NewStrValWithProcessors("dynamic1", processers), res)

	res, err = vars.Replace("${dynamic.list}")
	require.NoError(t, err)
	assert.Equal(t, processers, res.Processors())
	assert.Equal(t, NewListWithProcessors([]Node{
		NewStrVal("array1"),
		NewStrVal("array2"),
	}, processers), res)

	res, err = vars.Replace("${dynamic.dict}")
	require.NoError(t, err)
	assert.Equal(t, processers, res.Processors())
	assert.Equal(t, NewDictWithProcessors([]Node{
		NewKey("key1", NewStrVal("value1")),
		NewKey("key2", NewStrVal("value2")),
	}, processers), res)
}
