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

package memlog

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func init() {
	logp.DevelopmentSetup()
}

func TestRecoverFromCorruption(t *testing.T) {
	path := t.TempDir()
	defer os.RemoveAll(path)

	if err := copyPath(path, "testdata/1/logfile_incomplete/"); err != nil {
		t.Fatalf("Failed to copy test file to the temporary directory: %v", err)
	}

	store, err := openStore(logp.NewLogger("test"), path, 0660, 4096, false, func(_ uint64) bool {
		return false
	})
	assert.NoError(t, err)

	assert.Equal(t, true, store.disk.logInvalid)

	err = store.logOperation(&opSet{K: "key", V: mapstr.M{
		"field": 42,
	}})
	assert.NoError(t, err)
	assert.Equal(t, false, store.disk.logInvalid)
}
