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

//go:build aix || darwin || freebsd || linux
// +build aix darwin freebsd linux

package filesystem

import (
	"fmt"
	"syscall"

	"github.com/elastic/beats/v7/libbeat/opt"
)

// GetUsage returns the filesystem usage
func (fs *FSStat) GetUsage() error {
	stat := syscall.Statfs_t{}
	err := syscall.Statfs(fs.Directory, &stat)
	if err != nil {
		return fmt.Errorf("error in Statfs syscall: %w", err)
	}

	fs.Total = opt.UintWith(stat.Blocks).MultUint64OrNone(uint64(stat.Bsize))
	fs.Free = opt.UintWith(stat.Bfree).MultUint64OrNone(uint64(stat.Bsize))
	fs.Avail = opt.UintWith(stat.Bavail).MultUint64OrNone(uint64(stat.Bsize))
	fs.Files = opt.UintWith(stat.Files)
	fs.FreeFiles = opt.UintWith(stat.Ffree)

	fs.fillMetrics()

	return nil
}
