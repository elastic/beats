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

package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTryToInt(t *testing.T) {
	tests := []struct {
		input   interface{}
		result  int
		resultB bool
	}{
		{
			int(4),
			int(4),
			true,
		},
		{
			int64(3),
			int(3),
			true,
		},
		{
			"5",
			int(5),
			true,
		},
		{
			uint32(12),
			int(12),
			true,
		},
		{
			"abc",
			0,
			false,
		},
		{
			[]string{"123"},
			0,
			false,
		},
		{
			uint64(55),
			int(55),
			true,
		},
	}

	for _, test := range tests {
		a, b := TryToInt(test.input)
		assert.Equal(t, a, test.result)
		assert.Equal(t, b, test.resultB)
	}
}
