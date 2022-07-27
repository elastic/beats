// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package transpiler

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/common"
	corecomp "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/composable"
)

func TestVars_Replace(t *testing.T) {
	vars := mustMakeVars(map[string]interface{}{
		"un-der_score": map[string]interface{}{
			"key1":      "data1",
			"key2":      "data2",
			"with-dash": "dash-value",
			"list": []string{
				"array1",
				"array2",
			},
			"with/slash": "some/path",
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
			"${un-der_score.with-dash}",
			NewStrVal("dash-value"),
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
			`${"with:colon"}`,
			NewStrVal("with:colon"),
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
			`${un-der_score.missing|'with:colon'}`,
			NewStrVal("with:colon"),
			false,
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
			`${un-der_score.with/slash}`,
			NewStrVal(`some/path`),
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
		processers,
		nil)
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

func TestVars_ReplaceWithFetchContextProvider(t *testing.T) {
	processers := Processors{
		{
			"add_fields": map[string]interface{}{
				"dynamic": "added",
			},
		},
	}

	mockFetchProvider, err := MockContextProviderBuilder()
	require.NoError(t, err)

	fetchContextProviders := common.MapStr{
		"kubernetes_secrets": mockFetchProvider,
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
		processers,
		fetchContextProviders)
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

	res, err = vars.Replace("${kubernetes_secrets.test_namespace.testing_secret.secret_value}")
	require.NoError(t, err)
	assert.Equal(t, NewStrVal("mockedFetchContent"), res)
}

type contextProviderMock struct {
}

// MockContextProviderBuilder builds the mock context provider.
func MockContextProviderBuilder() (corecomp.ContextProvider, error) {
	return &contextProviderMock{}, nil
}

func (p *contextProviderMock) Fetch(key string) (string, bool) {
	return "mockedFetchContent", true
}

func (p *contextProviderMock) Run(comm corecomp.ContextProviderComm) error {
	return nil
}
