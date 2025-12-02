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

package oteltest

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

// AssertMapsEqual asserts that documents m1 and m2 are equal under name
// flattening. Fields in ignoredFields are ignored unless they are missing
// from both documents, in which case the assertion fails.
func AssertMapsEqual(t *testing.T, m1, m2 mapstr.M, ignoredFields []string, msg string) {
	t.Helper()

	flatM1 := m1.Flatten()
	flatM2 := m2.Flatten()
	for _, f := range ignoredFields {
		hasKeyM1, _ := flatM1.HasKey(f)
		hasKeyM2, _ := flatM2.HasKey(f)

		if !hasKeyM1 && !hasKeyM2 {
			assert.Failf(t, msg, "ignored field %q does not exist in either map, please remove it from the ignored fields", f)
		}

		flatM1.Delete(f)
		flatM2.Delete(f)
	}
	if !cmp.Equal(flatM1, flatM2) {
		t.Error(cmp.Diff(flatM1, flatM2))
	}
}
