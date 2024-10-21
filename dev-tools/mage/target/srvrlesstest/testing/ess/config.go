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

package ess

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	BaseUrl string `json:"base_url" yaml:"base_url"`
	ApiKey  string `json:"api_key" yaml:"api_key"`
}

func defaultConfig() *Config {
	baseURL := os.Getenv("TEST_INTEG_AUTH_ESS_URL")
	if baseURL == "" {
		baseURL = "https://cloud.elastic.co"
	}
	url := strings.TrimRight(baseURL, "/") + "/api/v1"
	return &Config{
		BaseUrl: url,
	}
}

// Merge overlays the provided configuration on top of
// this configuration.
func (c *Config) Merge(anotherConfig Config) {
	if anotherConfig.BaseUrl != "" {
		c.BaseUrl = anotherConfig.BaseUrl
	}

	if anotherConfig.ApiKey != "" {
		c.ApiKey = anotherConfig.ApiKey
	}
}

// GetESSAPIKey returns the ESS API key, if it exists
func GetESSAPIKey() (string, bool, error) {
	essAPIKeyFile, err := GetESSAPIKeyFilePath()
	if err != nil {
		return "", false, err
	}
	_, err = os.Stat(essAPIKeyFile)
	if os.IsNotExist(err) {
		return "", false, nil
	} else if err != nil {
		return "", false, fmt.Errorf("unable to check if ESS config directory exists: %w", err)
	}
	data, err := os.ReadFile(essAPIKeyFile)
	if err != nil {
		return "", true, fmt.Errorf("unable to read ESS API key: %w", err)
	}
	essAPIKey := strings.TrimSpace(string(data))
	return essAPIKey, true, nil
}

// GetESSAPIKeyFilePath returns the path to the ESS API key file
func GetESSAPIKeyFilePath() (string, error) {
	essAPIKeyFile := os.Getenv("TEST_INTEG_AUTH_ESS_APIKEY_FILE")
	if essAPIKeyFile == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("unable to determine user's home directory: %w", err)
		}
		essAPIKeyFile = filepath.Join(homeDir, ".config", "ess", "api_key.txt")
	}
	return essAPIKeyFile, nil
}
