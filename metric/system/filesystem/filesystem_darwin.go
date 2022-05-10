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

/*
#include <stdlib.h>
#include <sys/sysctl.h>
#include <sys/mount.h>
#include <mach/mach_init.h>
#include <mach/mach_host.h>
#include <mach/host_info.h>
#include <libproc.h>
#include <mach/processor_info.h>
#include <mach/vm_map.h>
*/
import "C"

import (
	"bytes"
	"syscall"
)

func parseMounts(path string, filter func(FSStat) bool) ([]FSStat, error) {
	num, err := syscall.Getfsstat(nil, C.MNT_NOWAIT)
	if err != nil {
		return nil, err
	}

	buf := make([]syscall.Statfs_t, num)

	_, err = syscall.Getfsstat(buf, C.MNT_NOWAIT)
	if err != nil {
		return nil, err
	}

	fslist := make([]FSStat, 0, num)

	for i := 0; i < num; i++ {
		fs := FSStat{}
		fs.Directory = byteListToString(buf[i].Mntonname[:])
		fs.Device = byteListToString(buf[i].Mntfromname[:])
		fs.Type = byteListToString(buf[i].Fstypename[:])

		fslist = append(fslist, fs)
	}
	return fslist, nil
}

func byteListToString(raw []int8) string {
	byteList := make([]byte, len(raw))

	for pos, singleByte := range raw {
		byteList[pos] = byte(singleByte)
		if singleByte == 0 {
			break
		}
	}

	return string(bytes.Trim(byteList, "\x00"))
}
