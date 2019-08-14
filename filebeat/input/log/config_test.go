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

package log

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/filebeat/harvester"
)

func TestCleanOlderError(t *testing.T) {
	config := config{
		CleanInactive: 10 * time.Hour,
	}

	err := config.Validate()
	assert.Error(t, err)
}

func TestCleanOlderIgnoreOlderError(t *testing.T) {
	config := config{
		CleanInactive: 10 * time.Hour,
		IgnoreOlder:   15 * time.Hour,
	}

	err := config.Validate()
	assert.Error(t, err)
}

func TestCleanOlderIgnoreOlderErrorEqual(t *testing.T) {
	config := config{
		CleanInactive: 10 * time.Hour,
		IgnoreOlder:   10 * time.Hour,
	}

	err := config.Validate()
	assert.Error(t, err)
}

func TestCleanOlderIgnoreOlder(t *testing.T) {
	config := config{
		CleanInactive: 10*time.Hour + defaultConfig.ScanFrequency + 1*time.Second,
		IgnoreOlder:   10 * time.Hour,
		Paths:         []string{"hello"},
		ForwarderConfig: harvester.ForwarderConfig{
			Type: "log",
		},
	}

	err := config.Validate()
	assert.NoError(t, err)
}
