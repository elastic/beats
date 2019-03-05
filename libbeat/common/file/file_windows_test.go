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

// +build !integration

package file

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetOSState(t *testing.T) {
	file, err := ioutil.TempFile("", "")
	assert.Nil(t, err)

	fileinfo, err := file.Stat()
	assert.Nil(t, err)

	state := GetOSState(fileinfo)

	assert.True(t, state.IdxHi > 0)
	assert.True(t, state.IdxLo > 0)
	assert.True(t, state.Vol > 0)
}

func TestGetOSStateStat(t *testing.T) {
	file, err := ioutil.TempFile("", "")
	assert.Nil(t, err)

	fileinfo, err := os.Stat(file.Name())
	assert.Nil(t, err)

	state := GetOSState(fileinfo)

	assert.True(t, state.IdxHi > 0)
	assert.True(t, state.IdxLo > 0)
	assert.True(t, state.Vol > 0)
}
