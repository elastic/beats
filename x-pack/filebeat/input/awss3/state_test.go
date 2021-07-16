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

// +build !windows

package awss3

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type stateTestCase struct {
	states [2]State
	isSame bool
}

func TestStateIsEqual(t *testing.T) {
	lastModifed := time.Now()
	tests := map[string]stateTestCase{
		"two states pointing to the same key with same size and same last modified": {
			[2]State{
				State{
					Bucket:       "bucket a",
					Key:          "/key/to/this/file/1",
					Size:         100,
					LastModified: lastModifed,
				},
				State{
					Bucket:       "bucket a",
					Key:          "/key/to/this/file/1",
					Size:         100,
					LastModified: lastModifed,
				},
			},
			true,
		},
		"two states pointing to the same key with different size and same last modified": {
			[2]State{
				State{
					Bucket:       "bucket a",
					Key:          "/key/to/this/file/1",
					Size:         100,
					LastModified: lastModifed,
				},
				State{
					Bucket:       "bucket a",
					Key:          "/key/to/this/file/1",
					Size:         101,
					LastModified: lastModifed,
				},
			},
			false,
		},
		"two states pointing to the same key with same size and different last modified": {
			[2]State{
				State{
					Bucket:       "bucket a",
					Key:          "/key/to/this/file/1",
					Size:         100,
					LastModified: time.Now(),
				},
				State{
					Bucket:       "bucket a",
					Key:          "/key/to/this/file/1",
					Size:         100,
					LastModified: time.Now().Add(10 * time.Second),
				},
			},
			false,
		},
		"two states pointing to different key": {
			[2]State{
				State{
					Bucket:       "bucket a",
					Key:          "/key/to/this/file/1",
					Size:         100,
					LastModified: lastModifed,
				},
				State{
					Bucket:       "bucket a",
					Key:          "/key/to/this/file/2",
					Size:         100,
					LastModified: lastModifed,
				},
			},
			false,
		},
		"two states pointing to different bcuket": {
			[2]State{
				State{
					Bucket:       "bucket b",
					Key:          "/key/to/this/file/1",
					Size:         100,
					LastModified: lastModifed,
				},
				State{
					Bucket:       "bucket a",
					Key:          "/key/to/this/file/1",
					Size:         100,
					LastModified: lastModifed,
				},
			},
			false,
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			isSame := test.states[0].IsEqual(&test.states[1])
			assert.Equal(t, isSame, test.isSame)
		})
	}
}
