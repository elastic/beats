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
)

// get is the linux implementation for fetching Memory data
func get(rootfs resolve.Resolver) (Memory, error) {
	table, err := ParseMeminfo(rootfs)
	if err != nil {
		return Memory{}, errors.Wrap(err, "error fetching meminfo")
	}

	memData := Memory{}

	var free, cached uint64
	var ok bool
	if total, ok := table["MemTotal"]; ok {
		memData.Total = opt.UintWith(total)
	}
	if free, ok = table["MemFree"]; ok {
		memData.Free = opt.UintWith(free)
	}
	if cached, ok = table["Cached"]; ok {
		memData.Cached = opt.UintWith(cached)
	}

	// overlook parsing issues here
	// On the very small chance some of these don't exist,
	// It's not the end of the world
	buffers, _ := table["Buffers"]

	if memAvail, ok := table["MemAvailable"]; ok {
		// MemAvailable is in /proc/meminfo (kernel 3.14+)
		memData.Actual.Free = opt.UintWith(memAvail)
	} else {
		// in the future we may want to find another way to do this.
		// "MemAvailable" and other more derivied metrics
		// Are very relative, and can be unhelpful in cerntain workloads
		// We may want to find a way to more clearly express to users
		// where a certain value is coming from and what it represents

		// The use of `cached` here is particularly concerning,
		// as under certain intense DB server workloads, the cached memory can be quite large
		// and give the impression that we've passed memory usage watermark
		memData.Actual.Free = opt.UintWith(free + buffers + cached)
	}

	memData.Used.Bytes = opt.UintWith(memData.Total.ValueOr(0) - memData.Free.ValueOr(0))
	memData.Actual.Used.Bytes = opt.UintWith(memData.Total.ValueOr(0) - memData.Actual.Free.ValueOr(0))

	// Populate swap data
	swapTotal, okST := table["SwapTotal"]
	if okST {
		memData.Swap.Total = opt.UintWith(swapTotal)
	}
	swapFree, okSF := table["SwapFree"]
	if okSF {
		memData.Swap.Free = opt.UintWith(swapFree)
	}

	if okSF && okST {
		memData.Swap.Used.Bytes = opt.UintWith(swapTotal - swapFree)
	}

	return memData, nil

}
