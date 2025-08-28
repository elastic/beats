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

package cmd

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/heartbeat/monitors/plugin"
)

// Test all required plugins are exported by this module, since it's the
// one imported by agentbeat: https://github.com/elastic/beats/pull/39818
func TestRootCmdPlugins(t *testing.T) {
	t.Parallel()
	plugins := []string{"http", "tcp", "icmp"}
	for _, p := range plugins {
		t.Run(fmt.Sprintf("%s plugin", p), func(t *testing.T) {
			_, found := plugin.GlobalPluginsReg.Get(p)
			assert.True(t, found)
		})
	}
}
