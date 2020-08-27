package composable

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVars_Replace(t *testing.T) {
	vars := &Vars{
		Mapping: map[string]interface{}{
			"testing": map[string]interface{}{
				"key1": "data1",
				"key2": "data2",
			},
			"other": map[string]interface{}{
				"data": "info",
			},
		},
	}
	tests := []struct{
		Input string
		Result string
		Error bool
		NoMatch bool
	}{
		{
			"{{testing.key1}}",
			"data1",
			false,
			false,
		},
		{
			"{{testing.missing}}",
			"",
			false,
			true,
		},
		{
			"{{testing.missing|testing.key2}}",
			"data2",
			false,
			false,
		},
		{
			"{{testing.missing|testing.missing2|other.data}}",
			"info",
			false,
			false,
		},
		{
			"{{testing.missing|'fallback'}}",
			"fallback",
			false,
			false,
		},
		{
			`{{testing.missing|"fallback"}}`,
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
			`{{testing.}}`,
			"",
			true,
			false,
		},
		{
			`{{testing. key1}}`,
			"",
			true,
			false,
		},
		{
			`{{testing.missing|"oth}}`,
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
			"around {{testing.key1}} the var",
			"around data1 the var",
			false,
			false,
		},
		{
			"multi {{testing.key1}} var {{testing.missing|testing.key2}} around",
			"multi data1 var data2 around",
			false,
			false,
		},
		{
			"multi {{testing.key1}} var {{testing.missing|'other with space'}} around",
			"multi data1 var other with space around",
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
				assert.Error(t, NoMatchErr, err)
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
		Processors: processers,
	}

	res, resProcessors, err := vars.Replace("{{testing.key1}}")
	require.NoError(t, err)
	assert.Nil(t, resProcessors)
	assert.Equal(t, "data1", res)

	res, resProcessors, err = vars.Replace("{{dynamic.key1}}")
	require.NoError(t, err)
	assert.Equal(t, processers, resProcessors)
	assert.Equal(t, "dynamic1", res)
}
