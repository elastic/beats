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

package linux

import (
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// ReadIntFromFile reads a single int value from a path and returns an int64.
// /sysfs contains a number of metrics broken out by values in individual files, so this is a useful helper function to have
func ReadIntFromFile(path string, base int) (int64, error) {

	raw, err := ioutil.ReadFile(path)
	if err != nil {
		return 0, errors.Wrapf(err, "error reading file %s", path)
	}

	clean := strings.TrimSpace(string(raw))

	intval, err := strconv.ParseInt(clean, 10, 64)
	if err != nil {
		return 0, errors.Wrapf(err, "error converting string: %s", clean)
	}

	return intval, nil
}
