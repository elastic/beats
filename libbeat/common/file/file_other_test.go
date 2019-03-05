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

// +build !windows,!integration

package file

import (
	"io/ioutil"
	"math"
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetOSFileState(t *testing.T) {
	file, err := ioutil.TempFile("", "")
	assert.Nil(t, err)

	fileinfo, err := file.Stat()
	assert.Nil(t, err)

	state := GetOSState(fileinfo)

	assert.True(t, state.Inode > 0)

	if runtime.GOOS == "openbsd" {
		// The first device on OpenBSD has an ID of 0 so allow this.
		assert.True(t, state.Device >= 0, "Device %d", state.Device)
	} else {
		assert.True(t, state.Device > 0, "Device %d", state.Device)
	}
}

func TestGetOSFileStateStat(t *testing.T) {
	file, err := ioutil.TempFile("", "")
	assert.Nil(t, err)

	fileinfo, err := os.Stat(file.Name())
	assert.Nil(t, err)

	state := GetOSState(fileinfo)

	assert.True(t, state.Inode > 0)

	if runtime.GOOS == "openbsd" {
		// The first device on OpenBSD has an ID of 0 so allow this.
		assert.True(t, state.Device >= 0, "Device %d", state.Device)
	} else {
		assert.True(t, state.Device > 0, "Device %d", state.Device)
	}
}

func BenchmarkStateString(b *testing.B) {
	var samples [50]uint64
	for i, v := 0, uint64(0); i < len(samples); i, v = i+1, v+math.MaxUint64/uint64(len(samples)) {
		samples[i] = v
	}

	for i := 0; i < b.N; i++ {
		for _, inode := range samples {
			for _, device := range samples {
				st := StateOS{Inode: inode, Device: device}
				if st.String() == "" {
					b.Fatal("empty state string")
				}
			}
		}
	}
}
