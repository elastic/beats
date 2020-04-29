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
	"github.com/pkg/errors"

	"github.com/elastic/go-sysinfo/types"
)

func parseMemInfo(content []byte) (*types.HostMemoryInfo, error) {
	memInfo := &types.HostMemoryInfo{
		Metrics: map[string]uint64{},
	}

	hasAvailable := false
	err := parseKeyValue(content, ":", func(key, value []byte) error {
		num, err := parseBytesOrNumber(value)
		if err != nil {
			return errors.Wrapf(err, "failed to parse %v value of %v", string(key), string(value))
		}

		k := string(key)
		switch k {
		case "MemTotal":
			memInfo.Total = num
		case "MemAvailable":
			hasAvailable = true
			memInfo.Available = num
		case "MemFree":
			memInfo.Free = num
		case "SwapTotal":
			memInfo.VirtualTotal = num
		case "SwapFree":
			memInfo.VirtualFree = num
		default:
			memInfo.Metrics[k] = num
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	memInfo.Used = memInfo.Total - memInfo.Free
	memInfo.VirtualUsed = memInfo.VirtualTotal - memInfo.VirtualFree

	// MemAvailable was added in kernel 3.14.
	if !hasAvailable {
		// Linux uses this for the calculation (but we are using a simpler calculation).
		// https://git.kernel.org/pub/scm/linux/kernel/git/torvalds/linux.git/commit/?id=34e431b0ae398fc54ea69ff85ec700722c9da773
		memInfo.Available = memInfo.Free + memInfo.Metrics["Buffers"] + memInfo.Metrics["Cached"]
	}

	return memInfo, nil
}
