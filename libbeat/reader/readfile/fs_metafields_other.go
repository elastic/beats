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
	"strconv"

	"github.com/elastic/beats/v7/libbeat/common/file"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const (
	deviceIDKey = "log.file.device_id"
	inodeKey    = "log.file.inode"
)

func setFileSystemMetadata(fi file.ExtendedFileInfo, fields mapstr.M) error {
	osstate := fi.GetOSState()
	_, err := fields.Put(deviceIDKey, strconv.FormatUint(osstate.Device, 10))
	if err != nil {
		return fmt.Errorf("failed to set %q: %w", deviceIDKey, err)
	}
	_, err = fields.Put(inodeKey, osstate.InodeString())
	if err != nil {
		return fmt.Errorf("failed to set %q: %w", inodeKey, err)
	}

	return nil
}
