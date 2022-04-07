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

//go:build !windows
// +build !windows

package file

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/common/file"
)

type stateTestCase struct {
	states [2]State
	isSame bool
}

func TestINodeDeviceIdentifier(t *testing.T) {
	tests := map[string]stateTestCase{
		"two states poiting to the same file": {
			[2]State{
				{
					Source:      "/path/to/this/file/1",
					FileStateOS: file.StateOS{Inode: 1, Device: 1},
				},
				{
					Source:      "/path/to/this/file/2",
					FileStateOS: file.StateOS{Inode: 1, Device: 1},
				},
			},
			true,
		},
		"two states poiting to different files": {
			[2]State{
				{
					Source:      "/path/to/this/file/1",
					FileStateOS: file.StateOS{Inode: 1, Device: 1},
				},
				{
					Source:      "/path/to/this/file/2",
					FileStateOS: file.StateOS{Inode: 2, Device: 1},
				},
			},
			false,
		},
	}

	identifier, _ := newINodeDeviceIdentifier(nil)
	for name, test := range tests {
		test := test
		for i := 0; i < len(test.states); i++ {
			test.states[i].Id, test.states[i].IdentifierName = identifier.GenerateID(test.states[i])
		}

		t.Run(name, func(t *testing.T) {
			isSame := test.states[0].IsEqual(&test.states[1])
			assert.Equal(t, isSame, test.isSame)
		})
	}
}

func TestPathIdentifier(t *testing.T) {
	tests := map[string]stateTestCase{
		"two states poiting to the same file": {
			[2]State{
				{
					Source:      "/path/to/this/file/1",
					FileStateOS: file.StateOS{Inode: 1, Device: 1},
				},
				{
					Source:      "/path/to/this/file/1",
					FileStateOS: file.StateOS{Inode: 1, Device: 1},
				},
			},
			true,
		},
		"two states poiting to different files": {
			[2]State{
				{
					Source:      "/path/to/this/file/1",
					FileStateOS: file.StateOS{Inode: 1, Device: 1},
				},
				{
					Source:      "/path/to/this/file/2",
					FileStateOS: file.StateOS{Inode: 2, Device: 1},
				},
			},
			false,
		},
	}

	identifier, _ := newPathIdentifier(nil)
	for name, test := range tests {
		test := test
		for i := 0; i < len(test.states); i++ {
			test.states[i].Id, test.states[i].IdentifierName = identifier.GenerateID(test.states[i])
		}
		t.Run(name, func(t *testing.T) {
			isSame := test.states[0].IsEqual(&test.states[1])
			assert.Equal(t, isSame, test.isSame)
		})
	}
}

func TestInodeMarkerIdentifier(t *testing.T) {
	tests := map[string]stateTestCase{
		"two states poiting to the same file i.": {
			[2]State{
				{
					Source:      "/path/to/this/file/1",
					FileStateOS: file.StateOS{Inode: 1, Device: 1},
				},
				{
					Source:      "/path/to/this/file/1",
					FileStateOS: file.StateOS{Inode: 1, Device: 1},
				},
			},
			true,
		},
		"two states poiting to the same file ii.": {
			[2]State{
				{
					Source:      "/path/to/this/file/1",
					FileStateOS: file.StateOS{Inode: 1, Device: 1},
				},
				{
					Source:      "/path/to/this/file/1",
					FileStateOS: file.StateOS{Inode: 1, Device: 2},
				},
			},
			true,
		},
		"two states poiting to different files i.": {
			[2]State{
				{
					Source:      "/path/to/this/file/1",
					FileStateOS: file.StateOS{Inode: 1, Device: 1},
				},
				{
					Source:      "/path/to/this/file/2",
					FileStateOS: file.StateOS{Inode: 2, Device: 1},
				},
			},
			false,
		},
		"two states poiting to different files ii.": {
			[2]State{
				{
					Source:      "/path/to/this/file/1",
					FileStateOS: file.StateOS{Inode: 1, Device: 1},
				},
				{
					Source:      "/path/to/this/file/1",
					FileStateOS: file.StateOS{Inode: 2, Device: 3},
				},
			},
			false,
		},
	}

	identifier := newMockInodeMarkerIdentifier()
	for name, test := range tests {
		test := test
		for i := 0; i < len(test.states); i++ {
			test.states[i].Id, test.states[i].IdentifierName = identifier.GenerateID(test.states[i])
		}
		t.Run(name, func(t *testing.T) {
			isSame := test.states[0].IsEqual(&test.states[1])
			assert.Equal(t, isSame, test.isSame)
		})
	}
}

func newMockInodeMarkerIdentifier() StateIdentifier {
	cfg := common.MustNewConfigFrom(map[string]string{"path": filepath.Join("testdata", "identifier_marker")})
	i, err := newINodeMarkerIdentifier(cfg)
	fmt.Println(err)
	return i
}
