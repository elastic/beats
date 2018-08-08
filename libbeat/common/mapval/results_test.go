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

package mapval

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEmpty(t *testing.T) {
	r := NewResults()
	assert.True(t, r.Valid)
	assert.Empty(t, r.DetailedErrors().Fields)
	assert.Empty(t, r.Errors())
}

func TestWithError(t *testing.T) {
	r := NewResults()
	r.record(MustParsePath("foo"), KeyMissingVR)
	r.record(MustParsePath("bar"), ValidVR)

	assert.False(t, r.Valid)

	assert.Equal(t, KeyMissingVR, r.Fields["foo"][0])
	assert.Equal(t, ValidVR, r.Fields["bar"][0])

	assert.Equal(t, KeyMissingVR, r.DetailedErrors().Fields["foo"][0])
	assert.NotContains(t, r.DetailedErrors().Fields, "bar")

	assert.False(t, r.DetailedErrors().Valid)
	assert.NotEmpty(t, r.Errors())
}
