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

package cgcommon

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var (

	// ErrInvalidFormat indicates a malformed key/value pair on a line.
	ErrInvalidFormat = errors.New("error invalid key/value format")
)

// ParseUintFromFile reads a single uint value from a file.
func ParseUintFromFile(path ...string) (uint64, error) {
	value, err := ioutil.ReadFile(filepath.Join(path...))
	if err != nil {
		// Not all features are implemented/enabled by each OS.
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	return ParseUint(value)
}

// ParseUint reads a single uint value. It will trip any whitespace before
// attempting to parse string. If the value is negative it will return 0.
func ParseUint(value []byte) (uint64, error) {
	strValue := string(bytes.TrimSpace(value))
	uintValue, err := strconv.ParseUint(strValue, 10, 64)
	if err != nil {
		// Munge negative values to 0.
		intValue, intErr := strconv.ParseInt(strValue, 10, 64)
		if intErr == nil && intValue < 0 {
			return 0, nil
		} else if intErr != nil && intErr.(*strconv.NumError).Err == strconv.ErrRange && intValue < 0 {
			return 0, nil
		}

		return 0, err
	}

	return uintValue, nil
}

// ParseCgroupParamKeyValue parses a cgroup param and returns the key name and value.
func ParseCgroupParamKeyValue(t string) (string, uint64, error) {
	parts := strings.Fields(t)
	if len(parts) != 2 {
		return "", 0, ErrInvalidFormat
	}

	value, err := ParseUint([]byte(parts[1]))
	if err != nil {
		return "", 0, fmt.Errorf("unable to convert param value (%q) to uint64: %v", parts[1], err)
	}

	return parts[0], value, nil
}
