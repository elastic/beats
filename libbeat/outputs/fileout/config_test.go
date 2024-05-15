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

package fileout

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestConfig(t *testing.T) {
	for name, test := range map[string]struct {
		config         *config.C
		useWindowsPath bool
		assertion      func(t *testing.T, config *fileOutConfig, err error)
	}{
		"default config": {
			config: config.MustNewConfigFrom([]byte(`{ }`)),
			assertion: func(t *testing.T, actual *fileOutConfig, err error) {
				expectedConfig := &fileOutConfig{
					NumberOfFiles:   7,
					RotateEveryKb:   10 * 1024,
					Permissions:     0600,
					RotateOnStartup: true,
				}

				assert.Equal(t, expectedConfig, actual)
				assert.Nil(t, err)
			},
		},
		"config given with posix path": {
			config: config.MustNewConfigFrom(mapstr.M{
				"number_of_files": 10,
				"rotate_every_kb": 5 * 1024,
				"path":            "/tmp/packetbeat/%{+yyyy-MM-dd-mm-ss-SSSSSS}",
				"filename":        "pb",
			}),
			assertion: func(t *testing.T, actual *fileOutConfig, err error) {
				assert.Equal(t, uint(10), actual.NumberOfFiles)
				assert.Equal(t, uint(5*1024), actual.RotateEveryKb)
				assert.Equal(t, true, actual.RotateOnStartup)
				assert.Equal(t, uint32(0600), actual.Permissions)
				assert.Equal(t, "pb", actual.Filename)

				path, runErr := actual.Path.Run(time.Date(2024, 1, 2, 3, 4, 5, 67890, time.UTC))
				assert.Nil(t, runErr)

				assert.Equal(t, "/tmp/packetbeat/2024-01-02-04-05-000067", path)
				assert.Nil(t, err)
			},
		},
		"config given with windows path": {
			useWindowsPath: true,
			config: config.MustNewConfigFrom(mapstr.M{
				"number_of_files": 10,
				"rotate_every_kb": 5 * 1024,
				"path":            "c:\\tmp\\packetbeat\\%{+yyyy-MM-dd-mm-ss-SSSSSS}",
				"filename":        "pb",
			}),
			assertion: func(t *testing.T, actual *fileOutConfig, err error) {
				assert.Equal(t, uint(10), actual.NumberOfFiles)
				assert.Equal(t, uint(5*1024), actual.RotateEveryKb)
				assert.Equal(t, true, actual.RotateOnStartup)
				assert.Equal(t, uint32(0600), actual.Permissions)
				assert.Equal(t, "pb", actual.Filename)

				path, runErr := actual.Path.Run(time.Date(2024, 1, 2, 3, 4, 5, 67890, time.UTC))
				assert.Nil(t, runErr)

				assert.Equal(t, "c:\\tmp\\packetbeat\\2024-01-02-04-05-000067", path)
				assert.Nil(t, err)
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			isWindowsPath = test.useWindowsPath
			cfg, err := readConfig(test.config)
			test.assertion(t, cfg, err)
		})
	}
}
