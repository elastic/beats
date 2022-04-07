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

package haproxy

import (
	"testing"

	mbtest "github.com/elastic/beats/v8/metricbeat/mb/testing"

	"github.com/stretchr/testify/assert"
)

func TestHostParser(t *testing.T) {
	tests := []struct {
		host, expected string
	}{
		{"localhost", "tcp://localhost"},
		{"localhost:123", "tcp://localhost:123"},
		{"tcp://localhost:123", "tcp://localhost:123"},
		{"unix:///var/lib/haproxy/stats", "unix:///var/lib/haproxy/stats"},
	}

	m := mbtest.NewTestModule(t, map[string]interface{}{})

	for _, test := range tests {
		hi, err := HostParser(m, test.host)
		if err != nil {
			t.Error("failed on", test.host, err)
			continue
		}
		assert.Equal(t, test.expected, hi.URI)
	}
}
