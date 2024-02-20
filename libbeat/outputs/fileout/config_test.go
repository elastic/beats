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
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestConfig(t *testing.T) {
	for name, test := range map[string]struct {
		config    *config.C
		assertion func(t *testing.T, config *fileOutConfig, err error)
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
		"config given": {
			config: config.MustNewConfigFrom(mapstr.M{
				"number_of_files": 10,
				"rotate_every_kb": 5 * 1024,
				"path":            "/tmp/packetbeat",
				"filename":        "pb",
			}),
			assertion: func(t *testing.T, actual *fileOutConfig, err error) {
				expectedConfig := &fileOutConfig{
					NumberOfFiles:   10,
					RotateEveryKb:   5 * 1024,
					Permissions:     0600,
					RotateOnStartup: true,
					Path:            "/tmp/packetbeat",
					Filename:        "pb",
				}

				assert.Equal(t, expectedConfig, actual)
				assert.Nil(t, err)
			},
		},
		"use TIME_NOW": {
			config: config.MustNewConfigFrom(mapstr.M{
				"path":     "/tmp/${TIME_NOW}",
				"filename": "pb-${TIME_NOW}",
			}),
			assertion: func(t *testing.T, actual *fileOutConfig, err error) {
				assert.Equal(t, uint(7), actual.NumberOfFiles)
				assert.Equal(t, uint(10*1024), actual.RotateEveryKb)
				assert.Equal(t, true, actual.RotateOnStartup)
				assert.Equal(t, uint32(0600), actual.Permissions)

				timeNow := time.Now().UnixMilli()

				pathRegex := `\/tmp\/(\d{13})$`
				re := regexp.MustCompile(pathRegex)
				matchesPath := re.FindStringSubmatch(actual.Path)

				assert.NotNil(t, matchesPath)
				assert.Equal(t, 2, len(matchesPath))

				timeNowPath, err := strconv.ParseInt(matchesPath[1], 10, 64)
				assert.Nil(t, err)
				assert.LessOrEqual(t, timeNow, timeNowPath)

				fileRegex := `pb\-(\d{13})$`
				re = regexp.MustCompile(fileRegex)
				matchesFile := re.FindStringSubmatch(actual.Filename)

				assert.NotNil(t, matchesFile)
				assert.Equal(t, 2, len(matchesFile))

				timeNowFile, err := strconv.ParseInt(matchesFile[1], 10, 64)
				assert.Nil(t, err)
				assert.Equal(t, timeNowFile, timeNowPath)
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			cfg, err := readConfig(test.config)
			test.assertion(t, cfg, err)
		})
	}
}
