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

package monitoring

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSnapshot(t *testing.T) {
	tests := []struct {
		name     string
		expected map[string]interface{}
		build    func(R *Registry)
	}{
		{
			"empty registry",
			nil,
			func(*Registry) {},
		},
		{
			"empty if metric is not exposed",
			nil,
			func(R *Registry) {
				NewInt(R, "test").Set(1)
			},
		},
		{
			"collect exposed metric",
			map[string]interface{}{"test": int64(1)},
			func(R *Registry) {
				NewInt(R, "test", Report).Set(1)
			},
		},
		{
			"do not report unexported namespace",
			map[string]interface{}{"test": int64(0)},
			func(R *Registry) {
				NewInt(R, "test", Report)
				NewInt(R, "unexported.test")
			},
		},
		{
			"do not report empty nested exported",
			map[string]interface{}{"test": int64(0)},
			func(R *Registry) {
				metrics := R.NewRegistry("exported", Report)
				NewInt(metrics, "unexported", DoNotReport)
				NewInt(R, "test", Report)
			},
		},
		{
			"export namespaced as nested-document from registry instance",
			map[string]interface{}{"exported": map[string]interface{}{"test": int64(0)}},
			func(R *Registry) {
				metrics := R.NewRegistry("exported", Report)
				NewInt(metrics, "test", Report)
				NewInt(R, "unexported.test")
			},
		},
		{
			"export unmarked namespaced as nested-document from registry instance",
			map[string]interface{}{"exported": map[string]interface{}{"test": int64(0)}},
			func(R *Registry) {
				metrics := R.NewRegistry("exported", Report)
				NewInt(metrics, "test")
				NewInt(R, "unexported.test")
			},
		},
		{
			"export namespaced as nested-document without intermediate registry instance",
			map[string]interface{}{"exported": map[string]interface{}{"test": int64(0)}},
			func(R *Registry) {
				NewInt(R, "exported.test", Report)
				NewInt(R, "unexported.test")
			},
		},
	}

	for i, test := range tests {
		t.Logf("run test (%v - %v): %v", i, test.name, test.expected)

		R := NewRegistry()
		test.build(R)
		snapshot := CollectStructSnapshot(R, Reported, false)

		t.Logf("  actual: %v", snapshot)
		assert.Equal(t, test.expected, snapshot)
	}
}
