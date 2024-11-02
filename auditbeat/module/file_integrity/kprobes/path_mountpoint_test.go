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

package kprobes

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_readMountInfo(t *testing.T) {
	procContents := `19 42 0:19 / /sys rw,nosuid,nodev,noexec,relatime shared:6 - sysfs sysfs rw
42 0 252:1 / /etc/test/test rw,noatime shared:1 - xfs /dev/vda1 rw,attr2,inode64,noquota
20 42 0:4 / /proc rw,nosuid,nodev,noexec,relatime shared:5 - proc proc rw
23 21 0:20 / /dev/shm rw,nosuid,nodev shared:3 - tmpfs tmpfs rw
25 42 0:22 / /run rw,nosuid,nodev shared:23 - tmpfs tmpfs rw,mode=755
26 19 0:23 / /sys/fs/cgroup ro,nosuid,nodev,noexec shared:8 - tmpfs tmpfs ro,mode=755
42 0 252:1 / / rw,noatime shared:1 - xfs /dev/vda1 rw,attr2,inode64,noquota
45 19 0:8 / /sys/kernel/debug rw,relatime shared:26 - debugfs debugfs rw
46 20 0:39 / /proc/sys/fs/binfmt_misc rw,relatime shared:27 - autofs systemd-1 rw,fd=34,pgrp=1,timeout=0,minproto=5,maxproto=5,direct,pipe_ino=13706
47 42 259:0 / /boot/efi rw,noatime shared:28 - vfat /dev/vda128 rw,fmask=0077,dmask=0077,codepage=437,iocharset=ascii,shortname=winnt,errors=remount-ro
42 0 252:1 / /etc/test rw,noatime shared:1 - xfs /dev/vda1 rw,attr2,inode64,noquota
`

	sortedPaths := []string{
		"/proc/sys/fs/binfmt_misc",
		"/sys/kernel/debug",
		"/sys/fs/cgroup",
		"/etc/test/test",
		"/etc/test",
		"/boot/efi",
		"/dev/shm",
		"/proc",
		"/sys",
		"/run",
		"/",
	}

	reader := strings.NewReader(procContents)

	mounts, err := readMountInfo(reader)
	require.NoError(t, err)
	require.Len(t, mounts, 11)

	for i, path := range sortedPaths {
		require.Equal(t, path, mounts[i].Path)
	}

	require.Equal(t, mounts[10], &mount{
		Path:           "/",
		FilesystemType: "xfs",
		DeviceMajor:    252,
		DeviceMinor:    1,
		Subtree:        "/",
		ReadOnly:       false,
	})

	require.Equal(t, mounts[2], &mount{
		Path:           "/sys/fs/cgroup",
		FilesystemType: "tmpfs",
		DeviceMajor:    0,
		DeviceMinor:    23,
		Subtree:        "/",
		ReadOnly:       true,
	})

	require.Equal(t, mounts[0], &mount{
		Path:           "/proc/sys/fs/binfmt_misc",
		FilesystemType: "autofs",
		DeviceMajor:    0,
		DeviceMinor:    39,
		Subtree:        "/",
		ReadOnly:       false,
	})

	pathMountPoint := mounts.getMountByPath("/etc/test/")

	require.Equal(t, pathMountPoint, &mount{
		Path:           "/etc/test",
		FilesystemType: "xfs",
		DeviceMajor:    252,
		DeviceMinor:    1,
		Subtree:        "/",
		ReadOnly:       false,
	})

	pathMountPoint = mounts.getMountByPath("unknown")

	require.Nil(t, pathMountPoint)
}
