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

package configutil

import (
	"os"
	"strconv"
	"time"

	"github.com/pkg/errors"

	"go.elastic.co/apm/internal/wildcard"
)

// ParseDurationEnv gets the value of the environment variable envKey
// and, if set, parses it as a duration. If the environment variable
// is unset, defaultDuration is returned.
func ParseDurationEnv(envKey string, defaultDuration time.Duration) (time.Duration, error) {
	value := os.Getenv(envKey)
	if value == "" {
		return defaultDuration, nil
	}
	d, err := ParseDuration(value)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to parse %s", envKey)
	}
	return d, nil
}

// ParseSizeEnv gets the value of the environment variable envKey
// and, if set, parses it as a size. If the environment variable
// is unset, defaultSize is returned.
func ParseSizeEnv(envKey string, defaultSize Size) (Size, error) {
	value := os.Getenv(envKey)
	if value == "" {
		return defaultSize, nil
	}
	s, err := ParseSize(value)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to parse %s", envKey)
	}
	return s, nil
}

// ParseBoolEnv gets the value of the environment variable envKey
// and, if set, parses it as a boolean. If the environment variable
// is unset, defaultValue is returned.
func ParseBoolEnv(envKey string, defaultValue bool) (bool, error) {
	value := os.Getenv(envKey)
	if value == "" {
		return defaultValue, nil
	}
	b, err := strconv.ParseBool(value)
	if err != nil {
		return false, errors.Wrapf(err, "failed to parse %s", envKey)
	}
	return b, nil
}

// ParseListEnv gets the value of the environment variable envKey
// and, if set, parses it as a list separated by sep. If the environment
// variable is unset, defaultValue is returned.
func ParseListEnv(envKey, sep string, defaultValue []string) []string {
	value := os.Getenv(envKey)
	if value == "" {
		return defaultValue
	}
	return ParseList(value, sep)
}

// ParseWildcardPatternsEnv gets the value of the environment variable envKey
// and, if set, parses it as a list of wildcard patterns. If the environment
// variable is unset, defaultValue is returned.
func ParseWildcardPatternsEnv(envKey string, defaultValue wildcard.Matchers) wildcard.Matchers {
	value := os.Getenv(envKey)
	if value == "" {
		return defaultValue
	}
	return ParseWildcardPatterns(value)
}
