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

package memory

import (
	"github.com/pkg/errors"

	"github.com/menderesk/beats/v7/libbeat/metric/system/resolve"
	"github.com/menderesk/beats/v7/libbeat/opt"
	"github.com/menderesk/go-windows"
)

// get is the windows implementation of get for memory metrics
func get(_ resolve.Resolver) (Memory, error) {

	memData := Memory{}

	memoryStatusEx, err := windows.GlobalMemoryStatusEx()
	if err != nil {
		return memData, errors.Wrap(err, "Error fetching global memory status")
	}
	memData.Total = opt.UintWith(memoryStatusEx.TotalPhys)
	memData.Free = opt.UintWith(memoryStatusEx.AvailPhys)

	memData.Used.Bytes = opt.UintWith(memoryStatusEx.TotalPhys - memoryStatusEx.AvailPhys)

	// We shouldn't really be doing this, but we also don't want to make breaking changes right now,
	// and memory.actual is used by quite a few visualizations
	memData.Actual.Free = memData.Free
	memData.Actual.Used.Bytes = memData.Used.Bytes

	memData.Swap.Free = opt.UintWith(memoryStatusEx.AvailPageFile)
	memData.Swap.Total = opt.UintWith(memoryStatusEx.TotalPageFile)
	memData.Swap.Used.Bytes = opt.UintWith(memoryStatusEx.TotalPageFile - memoryStatusEx.AvailPageFile)

	return memData, nil
}
