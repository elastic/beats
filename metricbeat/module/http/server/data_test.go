package server

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
)

func GetMetricProcessor() *metricProcessor {
	paths := []PathConfig{
		{
			Namespace: "foo",
			Path:      "/foo",
			Fields: common.MapStr{
				"a": "b",
			},
		},
		{
			Namespace: "bar",
			Path:      "/bar",
		},
	}

	defaultPath := defaultHttpServerConfig().DefaultPath
	return NewMetricProcessor(paths, defaultPath)
}

func TestMetricProcessorAddPath(t *testing.T) {
	processor := GetMetricProcessor()
	temp := PathConfig{
		Namespace: "xyz",
		Path:      "/abc",
	}
	processor.AddPath(temp)
	out, _ := processor.paths[temp.Path]
	assert.NotNil(t, out)
	assert.Equal(t, out.Namespace, temp.Namespace)
}

func TestMetricProcessorDeletePath(t *testing.T) {
	processor := GetMetricProcessor()
	processor.RemovePath(processor.paths["bar"])
	_, ok := processor.paths["bar"]
	assert.Equal(t, ok, false)
}

func TestFindPath(t *testing.T) {
	processor := GetMetricProcessor()
	tests := []struct {
		a        string
		expected PathConfig
	}{
		{
			a:        "/foo/bar",
			expected: processor.paths["/foo"],
		},
		{
			a:        "/",
			expected: processor.defaultPath,
		},
		{
			a:        "/abc",
			expected: processor.defaultPath,
		},
	}

	for i, test := range tests {
		a, expected := test.a, test.expected
		name := fmt.Sprintf("%v: %v = %v", i, a, expected)

		t.Run(name, func(t *testing.T) {
			b := processor.findPath(a)
			assert.Equal(t, expected, *b)
		})
	}
}
