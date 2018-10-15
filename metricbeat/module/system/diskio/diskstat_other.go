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

// +build darwin,cgo freebsd windows

package diskio

import (
	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/disk"
)

func NewDiskIOStat() *DiskIOStat {
	d := &DiskIOStat{}
	d.lastDiskIOCounters = make(map[string]disk.IOCountersStat)
	return d
}

func (stat *DiskIOStat) OpenSampling() error {
	return nil
}

func (stat *DiskIOStat) CalIOStatistics(counter disk.IOCountersStat) (DiskIOMetric, error) {
	var result DiskIOMetric
	return result, errors.New("Not implemented out of linux")
}

func (stat *DiskIOStat) CloseSampling() {
	return
}
