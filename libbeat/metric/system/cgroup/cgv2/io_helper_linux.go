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

//go:build linux
// +build linux

package cgv2

import (
	"io/fs"
	"path/filepath"
	"syscall"

	"github.com/pkg/errors"
)

// fetchDeviceName will attempt to find a device name associated with a major/minor pair
// the bool indicates if a device was found.
func fetchDeviceName(major uint64, minor uint64) (bool, string, error) {
	// iterate over /dev/ and pull major and minor values
	found := false
	var devName string
	var curMajor uint64
	var curMinor uint64
	var devID uint64
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
		devID = uint64(infoT.Rdev)

		// do some bitmapping to extract the major and minor device values
		// The odd duplicated logic here is to deal with 32 and 64 bit values.
		// see bits/sysmacros.h
		curMajor = ((devID & 0xfffff00000000000) >> 32) | ((devID & 0x00000000000fff00) >> 8)
		curMinor = ((devID & 0x00000000000000ff) >> 0) | ((devID & 0x00000ffffff00000) >> 12)
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
