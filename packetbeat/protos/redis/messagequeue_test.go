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

package redis

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type testMessage int

func (t testMessage) Size() int {
	return int(t)
}

func TestMessageList_Append(t *testing.T) {
	for _, test := range []struct {
		title    string
		maxBytes int64
		maxCount int32
		input    []int
		expected []int
	}{
		{
			title:    "unbounded queue",
			maxBytes: 0,
			maxCount: 0,
			input:    []int{1, 2, 3, 4, 5},
			expected: []int{1, 2, 3, 4, 5},
		},
		{
			title:    "count limited",
			maxBytes: 0,
			maxCount: 3,
			input:    []int{1, 2, 3, 4, 5},
			expected: []int{3, 4, 5},
		},
		{
			title:    "count limit boundary",
			maxBytes: 0,
			maxCount: 3,
			input:    []int{1, 2, 3},
			expected: []int{1, 2, 3},
		},
		{
			title:    "size limited",
			maxBytes: 10,
			maxCount: 0,
			input:    []int{1, 2, 3, 4, 5},
			expected: []int{4, 5},
		},
		{
			title:    "size limited boundary",
			maxBytes: 10,
			maxCount: 0,
			input:    []int{1, 2, 3, 4},
			expected: []int{1, 2, 3, 4},
		},
		{
			title:    "excess size",
			maxBytes: 10,
			maxCount: 0,
			input:    []int{1, 2, 3, 100},
			expected: []int{100},
		},
		{
			title:    "excess size 2",
			maxBytes: 10,
			maxCount: 0,
			input:    []int{100, 1},
			expected: []int{1},
		},
		{
			title:    "excess size 3",
			maxBytes: 10,
			maxCount: 0,
			input:    []int{1, 2, 3, 4, 5, 5},
			expected: []int{5, 5},
		},
		{
			title:    "both",
			maxBytes: 10,
			maxCount: 3,
			input:    []int{3, 4, 2, 1},
			expected: []int{4, 2, 1},
		},
	} {
		t.Run(test.title, func(t *testing.T) {
			conf := MessageQueueConfig{
				MaxBytes:    test.maxBytes,
				MaxMessages: test.maxCount,
			}
			q := NewMessageQueue(conf)
			for _, elem := range test.input {
				q.Append(testMessage(elem))
			}
			var result []int
			for !q.IsEmpty() {
				msg := q.Pop()
				if !assert.NotNil(t, msg) {
					t.FailNow()
				}
				value, ok := msg.(testMessage)
				if !assert.True(t, ok) {
					t.FailNow()
				}
				result = append(result, value.Size())
			}
			assert.Equal(t, test.expected, result)
		})
	}
}
