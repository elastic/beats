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

package readfile

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsLine(t *testing.T) {
	notLine := []byte("This is not a line")
	assert.False(t, isLine(notLine))

	notLine = []byte("This is not a line\n\r")
	assert.False(t, isLine(notLine))

	notLine = []byte("This is \n not a line")
	assert.False(t, isLine(notLine))

	line := []byte("This is a line \n")
	assert.True(t, isLine(line))

	line = []byte("This is a line\r\n")
	assert.True(t, isLine(line))
}

func TestLineEndingChars(t *testing.T) {
	line := []byte("Not ending line")
	assert.Equal(t, 0, lineEndingChars(line))

	line = []byte("N ending \n")
	assert.Equal(t, 1, lineEndingChars(line))

	line = []byte("RN ending \r\n")
	assert.Equal(t, 2, lineEndingChars(line))

	// This is an invalid option
	line = []byte("NR ending \n\r")
	assert.Equal(t, 0, lineEndingChars(line))
}
