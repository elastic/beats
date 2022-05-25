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

package unix

import (
	"testing"

	"github.com/stretchr/testify/assert"

	conf "github.com/elastic/elastic-agent-libs/config"
)

func TestErrorMissingPath(t *testing.T) {
	c := conf.MustNewConfigFrom(map[string]interface{}{
		"timeout":          1,
		"max_message_size": 1,
	})
	var config Config
	err := c.Unpack(&config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "need to specify the path to the unix socket")
}

func TestErrorOnEmptyLineDelimiterWhenStreamSocket(t *testing.T) {
	c := conf.MustNewConfigFrom(map[string]interface{}{
		"timeout":          1,
		"max_message_size": 1,
		"path":             "my-path",
		"socket_type":      "stream",
	})
	var config Config
	err := c.Unpack(&config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "line_delimiter cannot be empty when using stream socket")
}

func TestInvalidSocketType(t *testing.T) {
	c := conf.MustNewConfigFrom(map[string]interface{}{
		"timeout":          1,
		"max_message_size": 1,
		"path":             "my-path",
		"socket_type":      "invalid_type",
	})
	var config Config
	err := c.Unpack(&config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown socket type")
}
