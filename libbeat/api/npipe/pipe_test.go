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

package npipe

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsNPipe(t *testing.T) {
	t.Run("return true on named pipe", func(t *testing.T) {
		assert.True(t, IsNPipe("npipe:///hello"))
		assert.True(t, IsNPipe(`\\.\pipe\hello`))
	})

	t.Run("return false if its not a named pipe", func(t *testing.T) {
		assert.False(t, IsNPipe("unix:///tmp/ok.sock"))
	})
}
