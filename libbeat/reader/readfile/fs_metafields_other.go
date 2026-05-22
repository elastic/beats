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

//go:build !windows

package readfile

import (
	"fmt"
	"os/user"
	"strconv"

	"github.com/elastic/beats/v7/libbeat/common/file"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// platformFileFields is the number of fields setFileSystemMetadata always writes
// into the log.file submap on this platform (device_id and inode).
// Optional fields (owner, group) are counted separately in metafields.go.
const platformFileFields = 2

// Keys written into the log.file submap by setFileSystemMetadata.
const (
	deviceIDKey = "device_id"
	inodeKey    = "inode"
	ownerKey    = "owner"
	groupKey    = "group"
)

func setFileSystemMetadata(fi file.ExtendedFileInfo, fileMap mapstr.M, includeOwner bool, includeGroup bool) error {
	osstate := fi.GetOSState()
	fileMap[deviceIDKey] = strconv.FormatUint(osstate.Device, 10)
	fileMap[inodeKey] = osstate.InodeString()

	if includeOwner {
		o, err := user.LookupId(strconv.FormatUint(osstate.UID, 10))
		if err != nil {
			return fmt.Errorf("failed to lookup uid %q: %w", osstate.UID, err)
		}
		fileMap[ownerKey] = o.Username
	}

	if includeGroup {
		g, err := user.LookupGroupId(strconv.FormatUint(osstate.GID, 10))
		if err != nil {
			return fmt.Errorf("failed to lookup gid %q: %w", osstate.GID, err)
		}
		fileMap[groupKey] = g.Name
	}
	return nil
}
