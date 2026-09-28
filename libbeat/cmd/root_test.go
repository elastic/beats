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
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/cmd/instance"
	"github.com/elastic/elastic-agent-libs/config"
)

func TestGenRootCmdWithSettingsExposesHostnameFlag(t *testing.T) {
	// GenRootCmdWithSettings changes the default of -c, so restore it for other tests.
	t.Cleanup(func() {
		if c := flag.Lookup("c"); c != nil {
			if sf, ok := c.Value.(*config.StringsFlag); ok {
				sf.SetDefault("beat.yml")
			}
		}
		instance.HostnameFlag = ""
	})

	settings := instance.Settings{Name: "testbeat", Version: "0.0.1"}
	rootCmd := GenRootCmdWithSettings(nil, settings)
	f := rootCmd.PersistentFlags().Lookup("hostname")
	require.NotNil(t, f, "--hostname must be registered on the root command")
	assert.False(t, f.Hidden, "--hostname must be visible to users")
	// Parsing through Cobra must populate the value consumed by instance.Beat.
	require.NoError(t, rootCmd.ParseFlags([]string{"--hostname", "test-node"}))
	assert.Equal(t, "test-node", instance.HostnameFlag)
}
