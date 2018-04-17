// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package linux

import (
	"bufio"
	"bytes"
	"io/ioutil"

	"github.com/pkg/errors"
)

const procOneCgroup = "/proc/1/cgroup"

// IsContainerized returns true if this process is containerized.
func IsContainerized() (bool, error) {
	data, err := ioutil.ReadFile(procOneCgroup)
	if err != nil {
		return false, errors.Wrap(err, "failed to read process cgroups")
	}

	return isContainerizedCgroup(data)
}

func isContainerizedCgroup(data []byte) (bool, error) {
	s := bufio.NewScanner(bytes.NewReader(data))
	for n := 0; s.Scan(); n++ {
		line := s.Bytes()
		if len(line) == 0 || line[len(line)-1] == '/' {
			continue
		}

		if bytes.HasSuffix(line, []byte("init.scope")) {
			return false, nil
		}
	}

	return true, s.Err()
}
