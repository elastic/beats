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

package release

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestVersion(t *testing.T) {
	t.Run("set version without qualifier", func(t *testing.T) {
		old := version
		defer func() { version = old }()
		version = "8.x.x"
		assert.Equal(t, Version(), version)
	})

	t.Run("set version with qualifier", func(t *testing.T) {
		old := version
		defer func() { version = old }()
		version = "8.x.x"
		qualifier = "alpha1"
		assert.Equal(t, Version(), version+"-"+qualifier)
	})

	t.Run("get commit hash", func(t *testing.T) {
		commit = "abc1234"
		assert.Equal(t, Commit(), commit)
	})

	t.Run("get build time", func(t *testing.T) {
		ts := time.Now().Format(time.RFC3339)
		old := buildTime
		defer func() { buildTime = old }()
		buildTime = ts
		assert.Equal(t, ts, BuildTime().Format(time.RFC3339))
	})
}
