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

package cgv2

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/metric/system/cgroup/cgcommon"
)

//IOSubsystem is the replacement for the bulkio controller in cgroupsV1
type IOSubsystem struct {
	cgcommon.Metadata
	Stats    map[string]IOStat            `json:"stats" struct:"stats"`
	Pressure map[string]cgcommon.Pressure `json:"pressure" struct:"pressure"`
}

// IOStat carries io.Stat data for the controllers
// This data is broken down per-device, based on the maj-minor device ID
type IOStat struct {
	Read      IOMetric `json:"read" struct:"read"`
	Write     IOMetric `json:"write" struct:"write"`
	Discarded IOMetric `json:"discarded" struct:"discarded"`
}

// IOMetric groups together the common IO sub-metrics by bytes and IOOps count
type IOMetric struct {
	Bytes uint64 `json:"bytes" struct:"bytes"`
	IOs   uint64 `json:"ios" struct:"ios"`
}

// Get fetches metrics for the IO subsystem
// resolveDevIDs determines if Get will try to resolve the major-minor ID pairs reported by io.stat
// are resolved to a device name
func (io *IOSubsystem) Get(path string, resolveDevIDs bool) error {
	var err error
	io.Stats, err = getIOStats(path, resolveDevIDs)
	if err != nil {
		return errors.Wrapf(err, "error getting io.stats for path %s", path)
	}

	io.Pressure, err = cgcommon.GetPressure(filepath.Join(path, "io.pressure"))
	if err != nil {
		return errors.Wrapf(err, "error fetching io.pressure for path %s:", path)
	}

	return nil
}

// getIOStats fetches and formats the io.stats object
func getIOStats(path string, resolveDevIDs bool) (map[string]IOStat, error) {
	stats := make(map[string]IOStat)
	file := filepath.Join(path, "io.stat")
	f, err := os.Open(file)
	if err != nil {
		return stats, errors.Wrap(err, "error reading cpu.stat")
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		devMetric := IOStat{}
		var major, minor uint64
		_, err := fmt.Sscanf(sc.Text(), "%d:%d rbytes=%d wbytes=%d rios=%d wios=%d dbytes=%d dios=%d", &major, &minor, &devMetric.Read.Bytes, &devMetric.Write.Bytes, &devMetric.Read.IOs, &devMetric.Write.IOs, &devMetric.Discarded.Bytes, &devMetric.Discarded.IOs)
		if err != nil {
			return stats, errors.Wrapf(err, "error scanning file: %s", file)
		}

		// try to find the device name associated with the major/minor pair
		// This isn't guarenteed to work, for a number of reasons, so we'll need to fall back
		var found bool
		var devName string
		if resolveDevIDs {
			found, devName, err = fetchDeviceName(major, minor)
			if err != nil {
				return nil, errors.Wrapf(err, "error looking up device ID %d:%d", major, minor)
			}
		}

		if found {
			stats[devName] = devMetric
		} else {
			idKey := fmt.Sprintf("%d:%d", major, minor)
			stats[idKey] = devMetric
		}
	}

	return stats, nil
}

// fetchDeviceName will attempt to find a device name associated with a major/minor pair
// the bool indicates if a device was found.
func fetchDeviceName(major, minor uint64) (bool, string, error) {
	// iterate over /dev/ and pull major and minor values
	found := false
	var devName string
	walkFunc := func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() && path != "/dev/" {
			return fs.SkipDir
		}
		if d.Type() != fs.ModeDevice {
			return nil
		}
		fInfo, err := d.Info()
		if err != nil {
			return nil
		}
		infoT, ok := fInfo.Sys().(*syscall.Stat_t)
		if !ok {
			return nil
		}
		devID := infoT.Rdev
		// do some bitmapping to extract the major and minor device values
		// The odd duplicated logic here is to deal with 32 and 64 bit values.
		// see bits/sysmacros.h
		curMajor := ((devID & 0xfffff00000000000) >> 32) | ((devID & 0x00000000000fff00) >> 8)
		curMinor := ((devID & 0x00000000000000ff) >> 0) | ((devID & 0x00000ffffff00000) >> 12)
		if curMajor == major && curMinor == minor {
			found = true
			devName = d.Name()
		}
		return nil
	}

	err := filepath.WalkDir("/dev/", walkFunc)
	if err != nil {
		return false, "", errors.Wrap(err, "error walking /dev/")
	}

	return found, devName, nil
}
