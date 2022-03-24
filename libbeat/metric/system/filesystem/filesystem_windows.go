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

package filesystem

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/opt"
	"github.com/elastic/gosigar/sys/windows"
)

func parseMounts(_ string, filter func(FSStat) bool) ([]FSStat, error) {
	drives, err := windows.GetAccessPaths()
	if err != nil {
		return nil, fmt.Errorf("GetAccessPaths failed: %w", err)
	}

	driveList := []FSStat{}
	for _, drive := range drives {
		fsType, err := windows.GetFilesystemType(drive)
		if err != nil {
			return nil, fmt.Errorf("GetFilesystemType failed: %w", err)
		}
		if fsType != "" {
			driveList = append(driveList, FSStat{
				Directory: drive,
				Device:    drive,
				Type:      fsType,
			})
		}
	}

	return driveList, nil
}

func (fs *FSStat) GetUsage() error {
	freeBytesAvailable, totalNumberOfBytes, totalNumberOfFreeBytes, err := windows.GetDiskFreeSpaceEx(fs.Directory)
	if err != nil {
		return errors.Wrap(err, "GetDiskFreeSpaceEx failed")
	}

	fs.Total = opt.UintWith(totalNumberOfBytes)
	fs.Free = opt.UintWith(totalNumberOfFreeBytes)
	fs.Avail = opt.UintWith(freeBytesAvailable)

	fs.fillMetrics()

	return nil
}
