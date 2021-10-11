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

//go:build !integration
// +build !integration

package eslegclient

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUrlEncode(t *testing.T) {
	params := map[string]string{
		"q": "agent:appserver1",
	}
	url := urlEncode("", params)

	if url != "q=agent%3Aappserver1" {
		t.Errorf("Fail to encode params: %s", url)
	}

	params = map[string]string{
		"wife":    "sarah",
		"husband": "joe",
	}

	url = urlEncode("", params)

	if url != "husband=joe&wife=sarah" {
		t.Errorf("Fail to encode params: %s", url)
	}
}

func TestMakePath(t *testing.T) {
	path, err := makePath("twitter", "tweet", "1")
	if err != nil {
		t.Errorf("Fail to create path: %s", err)
	}
	if path != "/twitter/tweet/1" {
		t.Errorf("Wrong path created: %s", path)
	}

	path, err = makePath("twitter", "", "_refresh")
	if err != nil {
		t.Errorf("Fail to create path: %s", err)
	}
	if path != "/twitter/_refresh" {
		t.Errorf("Wrong path created: %s", path)
	}

	path, err = makePath("", "", "_bulk")
	if err != nil {
		t.Errorf("Fail to create path: %s", err)
	}
	if path != "/_bulk" {
		t.Errorf("Wrong path created: %s", path)
	}
	path, err = makePath("twitter", "", "")
	if err != nil {
		t.Errorf("Fail to create path: %s", err)
	}
	if path != "/twitter" {
		t.Errorf("Wrong path created: %s", path)
	}
}

func TestAddToURL(t *testing.T) {
	type Test struct {
		url      string
		path     string
		pipeline string
		params   map[string]string
		expected string
	}
	tests := []Test{
		{
			url:      "localhost:9200",
			path:     "/path",
			pipeline: "",
			params:   make(map[string]string),
			expected: "localhost:9200/path",
		},
		{
			url:      "localhost:9200/",
			path:     "/path",
			pipeline: "",
			params:   make(map[string]string),
			expected: "localhost:9200/path",
		},
		{
			url:      "localhost:9200",
			path:     "/path",
			pipeline: "pipeline_1",
			params:   make(map[string]string),
			expected: "localhost:9200/path?pipeline=pipeline_1",
		},
		{
			url:      "localhost:9200/",
			path:     "/path",
			pipeline: "",
			params: map[string]string{
				"param": "value",
			},
			expected: "localhost:9200/path?param=value",
		},
	}
	for _, test := range tests {
		url := addToURL(test.url, test.path, test.pipeline, test.params)
		assert.Equal(t, url, test.expected)
	}
}
