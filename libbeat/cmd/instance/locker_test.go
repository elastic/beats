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

//go:build !integration
// +build !integration

package instance

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/paths"
)

// TestLocker tests that two beats pointing to the same data path cannot
// acquire the same lock.
func TestLocker(t *testing.T) {
	// Setup temporary data folder for test + clean it up at end of test
	tmpDataDir, err := ioutil.TempDir("", "data")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDataDir)

	origDataPath := paths.Paths.Data
	defer func() {
		paths.Paths.Data = origDataPath
	}()
	paths.Paths.Data = tmpDataDir

	// Setup two beats with same name and data path
	const beatName = "testbeat"

	b1 := &Beat{}
	b1.Info.Beat = beatName

	b2 := &Beat{}
	b2.Info.Beat = beatName

	// Try to get a lock for the first beat. Expect it to succeed.
	bl1 := newLocker(b1)
	err = bl1.lock()
	assert.NoError(t, err)

	// Try to get a lock for the second beat. Expect it to fail because the
	// first beat already has the lock.
	bl2 := newLocker(b2)
	err = bl2.lock()
	assert.EqualError(t, err, ErrAlreadyLocked.Error())
}
