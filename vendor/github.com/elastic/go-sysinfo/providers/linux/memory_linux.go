// Copyright 2018 Elasticsearch Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package linux

import (
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

func MemTotal() (uint64, error) {
	v, err := findValue("/proc/meminfo", ":", "MemTotal")
	if err != nil {
		return 0, errors.Wrap(err, "failed to get mem total")
	}

	parts := strings.Fields(v)
	if len(parts) != 2 && parts[1] == "kB" {
		return 0, errors.Errorf("failed to parse mem total '%v'", v)
	}

	kB, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to parse mem total '%v'", parts[0])
	}

	return kB * 1024, nil
}
