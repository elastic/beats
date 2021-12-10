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

//go:build !integration && (darwin || freebsd || linux || openbsd || windows)
// +build !integration
// +build darwin freebsd linux openbsd windows

package filesystem

import (
	"io/ioutil"
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"

	sigar "github.com/elastic/gosigar"
)

func TestFileSystemList(t *testing.T) {
	if runtime.GOOS == "darwin" && os.Getenv("TRAVIS") == "true" {
		t.Skip("FileSystem test fails on Travis/OSX with i/o error")
	}

	fss, err := GetFileSystemList()
	if err != nil {
		t.Fatal("GetFileSystemList", err)
	}
	assert.True(t, (len(fss) > 0))

	for _, fs := range fss {
		if fs.TypeName == "cdrom" {
			continue
		}

		stat, err := GetFileSystemStat(fs)
		if os.IsPermission(err) {
			continue
		}

		if assert.NoError(t, err, "filesystem=%v: %v", fs, err) {
			assert.True(t, (stat.Total >= 0))
			assert.True(t, (stat.Free >= 0))
			assert.True(t, (stat.Avail >= 0))
			assert.True(t, (stat.Used >= 0))

			if runtime.GOOS != "windows" {
				assert.NotEqual(t, "", fs.SysTypeName)
			}
		}
	}
}

func TestFileSystemListFiltering(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("These cases don't need to work on Windows")
	}

	fakeDevDir, err := ioutil.TempDir(os.TempDir(), "dir")
	assert.Empty(t, err)
	defer os.RemoveAll(fakeDevDir)

	cases := []struct {
		description   string
		fss, expected []sigar.FileSystem
	}{
		{
			fss: []sigar.FileSystem{
				{DirName: "/", DevName: "/dev/sda1"},
				{DirName: "/", DevName: "/dev/sda1"},
			},
			expected: []sigar.FileSystem{
				{DirName: "/", DevName: "/dev/sda1"},
			},
		},
		{
			description: "Don't repeat devices, shortest of dir names should be used",
			fss: []sigar.FileSystem{
				{DirName: "/", DevName: "/dev/sda1"},
				{DirName: "/bind", DevName: "/dev/sda1"},
			},
			expected: []sigar.FileSystem{
				{DirName: "/", DevName: "/dev/sda1"},
			},
		},
		{
			description: "Don't repeat devices, shortest of dir names should be used",
			fss: []sigar.FileSystem{
				{DirName: "/bind", DevName: "/dev/sda1"},
				{DirName: "/", DevName: "/dev/sda1"},
			},
			expected: []sigar.FileSystem{
				{DirName: "/", DevName: "/dev/sda1"},
			},
		},
		{
			description: "Keep tmpfs",
			fss: []sigar.FileSystem{
				{DirName: "/run", DevName: "tmpfs"},
				{DirName: "/tmp", DevName: "tmpfs"},
			},
			expected: []sigar.FileSystem{
				{DirName: "/run", DevName: "tmpfs"},
				{DirName: "/tmp", DevName: "tmpfs"},
			},
		},
		{
			description: "Don't repeat devices, shortest of dir names should be used, keep tmpfs",
			fss: []sigar.FileSystem{
				{DirName: "/", DevName: "/dev/sda1"},
				{DirName: "/bind", DevName: "/dev/sda1"},
				{DirName: "/run", DevName: "tmpfs"},
			},
			expected: []sigar.FileSystem{
				{DirName: "/", DevName: "/dev/sda1"},
				{DirName: "/run", DevName: "tmpfs"},
			},
		},
		{
			description: "Don't keep the fs if the device is a directory (it'd be a bind mount)",
			fss: []sigar.FileSystem{
				{DirName: "/", DevName: "/dev/sda1"},
				{DirName: "/bind", DevName: fakeDevDir},
			},
			expected: []sigar.FileSystem{
				{DirName: "/", DevName: "/dev/sda1"},
			},
		},
		{
			description: "Don't filter out NFS",
			fss: []sigar.FileSystem{
				{DirName: "/srv/data", DevName: "192.168.42.42:/exports/nfs1"},
			},
			expected: []sigar.FileSystem{
				{DirName: "/srv/data", DevName: "192.168.42.42:/exports/nfs1"},
			},
		},
	}

	for _, c := range cases {
		filtered := filterFileSystemList(c.fss)
		assert.ElementsMatch(t, c.expected, filtered, c.description)
	}
}

func TestFilter(t *testing.T) {
	in := []sigar.FileSystem{
		{SysTypeName: "nfs"},
		{SysTypeName: "ext4"},
		{SysTypeName: "proc"},
		{SysTypeName: "smb"},
	}

	out := Filter(in, BuildTypeFilter("nfs", "smb", "proc"))

	if assert.Len(t, out, 1) {
		assert.Equal(t, "ext4", out[0].SysTypeName)
	}
}
