// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package server

import (
	"fmt"
	"testing"

	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/stretchr/testify/assert"
)

func GetMetricProcessor() *metricProcessor {
	paths := []PathConfig{
		{
			Namespace: "foo",
			Path:      "/foo",
			Fields: mapstr.M{
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
