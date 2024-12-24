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

package file

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSafeFileRotateExistingFile(t *testing.T) {
	tempdir := t.TempDir()

	// create an existing registry file
	err := os.WriteFile(filepath.Join(tempdir, "registry"),
		[]byte("existing filebeat"), 0x777)
	assert.NoError(t, err)

	// create a new registry.new file
	err = os.WriteFile(filepath.Join(tempdir, "registry.new"),
		[]byte("new filebeat"), 0x777)
	assert.NoError(t, err)

	// rotate registry.new into registry
	err = SafeFileRotate(filepath.Join(tempdir, "registry"),
		filepath.Join(tempdir, "registry.new"))
	assert.NoError(t, err)

	contents, err := os.ReadFile(filepath.Join(tempdir, "registry"))
	assert.NoError(t, err)
	assert.Equal(t, []byte("new filebeat"), contents)

	// do it twice to make sure we deal with deleting the old file
	for i := 0; i < 2; i++ {
		expectedContents := []byte(fmt.Sprintf("new filebeat %d", i))
		err = os.WriteFile(filepath.Join(tempdir, "registry.new"),
			expectedContents, 0x777)
		assert.NoError(t, err)

		err = SafeFileRotate(filepath.Join(tempdir, "registry"),
			filepath.Join(tempdir, "registry.new"))
		assert.NoError(t, err)

		contents, err = os.ReadFile(filepath.Join(tempdir, "registry"))
		assert.NoError(t, err)
		assert.Equal(t, expectedContents, contents)
	}
}
