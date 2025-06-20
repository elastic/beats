// Licensed to Elasticsearch B.V. under one or more contributor license agreements.
// See the NOTICE file distributed with this work for additional information regarding copyright ownership.
// Elasticsearch B.V. licenses this file to you under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software distributed
// under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
// either express or implied.  See the License for the specific language governing permissions and limitations under the License.

package ntp

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()
	assert.Equal(t, 5*time.Second, cfg.Timeout)
	assert.Equal(t, 4, cfg.Version)
}

func TestValidateConfig_Valid(t *testing.T) {
	cfg := config{
		Host:    "localhost:123",
		Timeout: 5 * time.Second,
		Version: 4,
	}
	assert.NoError(t, validateConfig(&cfg))
}

func TestValidateConfig_MissingHost(t *testing.T) {
	cfg := config{
		Timeout: 5 * time.Second,
		Version: 4,
	}
	err := validateConfig(&cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "NTP host must be set in config")
}

func TestValidateConfig_InvalidVersion(t *testing.T) {
	cfg := config{
		Host:    "localhost:123",
		Timeout: 5 * time.Second,
		Version: 2,
	}
	err := validateConfig(&cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "NTP version must be 3 or 4")
}

func TestValidateConfig_InvalidTimeout(t *testing.T) {
	cfg := config{
		Host:    "localhost:123",
		Timeout: 0,
		Version: 2,
	}
	err := validateConfig(&cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid NTP timeout: 0s")
}
