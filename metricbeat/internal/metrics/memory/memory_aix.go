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

/*
#cgo LDFLAGS: -L/usr/lib -lperfstat

#include <libperfstat.h>
#include <procinfo.h>
#include <unistd.h>
#include <utmp.h>
#include <sys/mntctl.h>
#include <sys/proc.h>
#include <sys/types.h>
#include <sys/vmount.h>

*/
import "C"

import (
	"fmt"
	"os"

	"github.com/elastic/beats/v7/libbeat/metric/system/resolve"
	"github.com/elastic/beats/v7/libbeat/opt"
)

var system struct {
	ticks    uint64
	btime    uint64
	pagesize uint64
}

func init() {
	// sysconf(_SC_CLK_TCK) returns the number of ticks by second.
	system.ticks = uint64(C.sysconf(C._SC_CLK_TCK))
	system.pagesize = uint64(os.Getpagesize())
}

func get(_ resolve.Resolver) (Memory, error) {
	memData := Memory{}
	meminfo := C.perfstat_memory_total_t{}
	_, err := C.perfstat_memory_total(nil, &meminfo, C.sizeof_perfstat_memory_total_t, 1)
	if err != nil {
		return memData, fmt.Errorf("perfstat_memory_total: %s", err)
	}

	totalMem := uint64(meminfo.real_total) * system.pagesize
	freeMem := uint64(meminfo.real_free) * system.pagesize

	memData.Total = opt.UintWith(totalMem)
	memData.Free = opt.UintWith(freeMem)

	kern := uint64(meminfo.numperm) * system.pagesize // number of pages in file cache

	memData.Used.Bytes = opt.UintWith(totalMem - freeMem)
	memData.Actual.Free = opt.UintWith(freeMem + kern)
	memData.Actual.Used.Bytes = opt.UintWith(memData.Used.Bytes.ValueOr(0) - kern)

	return memData, nil
}
