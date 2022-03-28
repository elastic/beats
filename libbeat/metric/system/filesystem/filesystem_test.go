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
	"io/ioutil"
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/transform/typeconv"
	"github.com/elastic/beats/v7/libbeat/metric/system/resolve"
)

func TestMountList(t *testing.T) {
	hostfs := resolve.NewTestResolver("/")

	result, err := GetFilesystems(hostfs, nil)
	assert.NoError(t, err, "GetFilesystems")

	t.Logf("Usage:")

	for _, res := range result {
		err := res.GetUsage()
		assert.NoError(t, err, "getUsage")
		out := common.MapStr{}
		err = typeconv.Convert(&out, res)
		assert.NoError(t, err, "typeconv")
		t.Logf("Usage: %s", out.StringToPrint())
	}
}

func TestFileSystemList(t *testing.T) {
	if runtime.GOOS == "darwin" && os.Getenv("TRAVIS") == "true" {
		t.Skip("FileSystem test fails on Travis/OSX with i/o error")
	}
	hostfs := resolve.NewTestResolver("/")
	//Exclude FS types that will give us a permission error
	fss, err := GetFilesystems(hostfs, BuildFilterWithList([]string{"cdrom", "tracefs", "overlay", "fuse.lxcfs", "fuse.gvfsd-fuse", "nsfs", "squashfs"}))
	if err != nil {
		t.Fatal("GetFileSystemList", err)
	}
	assert.True(t, (len(fss) > 0))

	for _, fs := range fss {

		err := fs.GetUsage()

		assert.NoError(t, err, "filesystem=%v: %v", fs, err)

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
		fss, expected []FSStat
	}{
		{
			fss: []FSStat{
				{Directory: "/", Device: "/dev/sda1"},
				{Directory: "/", Device: "/dev/sda1"},
			},
			expected: []FSStat{
				{Directory: "/", Device: "/dev/sda1"},
			},
		},
		{
			description: "Don't repeat devices, shortest of dir names should be used",
			fss: []FSStat{
				{Directory: "/", Device: "/dev/sda1"},
				{Directory: "/bind", Device: "/dev/sda1"},
			},
			expected: []FSStat{
				{Directory: "/", Device: "/dev/sda1"},
			},
		},
		{
			description: "Don't repeat devices, shortest of dir names should be used",
			fss: []FSStat{
				{Directory: "/bind", Device: "/dev/sda1"},
				{Directory: "/", Device: "/dev/sda1"},
			},
			expected: []FSStat{
				{Directory: "/", Device: "/dev/sda1"},
			},
		},
		{
			description: "Keep tmpfs",
			fss: []FSStat{
				{Directory: "/run", Device: "tmpfs"},
				{Directory: "/tmp", Device: "tmpfs"},
			},
			expected: []FSStat{
				{Directory: "/run", Device: "tmpfs"},
				{Directory: "/tmp", Device: "tmpfs"},
			},
		},
		{
			description: "Don't repeat devices, shortest of dir names should be used, keep tmpfs",
			fss: []FSStat{
				{Directory: "/", Device: "/dev/sda1"},
				{Directory: "/bind", Device: "/dev/sda1"},
				{Directory: "/run", Device: "tmpfs"},
			},
			expected: []FSStat{
				{Directory: "/", Device: "/dev/sda1"},
				{Directory: "/run", Device: "tmpfs"},
			},
		},
		{
			description: "Don't keep the fs if the device is a directory (it'd be a bind mount)",
			fss: []FSStat{
				{Directory: "/", Device: "/dev/sda1"},
				{Directory: "/bind", Device: fakeDevDir},
			},
			expected: []FSStat{
				{Directory: "/", Device: "/dev/sda1"},
			},
		},
		{
			description: "Don't filter out NFS",
			fss: []FSStat{
				{Directory: "/srv/data", Device: "192.168.42.42:/exports/nfs1"},
			},
			expected: []FSStat{
				{Directory: "/srv/data", Device: "192.168.42.42:/exports/nfs1"},
			},
		},
	}

	for _, c := range cases {

		filtered := filterFileSystemList(c.fss)
		ok := assert.ElementsMatch(t, c.expected, filtered, c.description)
		if !ok {
			t.FailNow()
		}
	}
}

// Emulate the filtering process that would normally happen inside the callbacks from platform-specific code
func filterFileSystemList(stats []FSStat) []FSStat {
	hostfs := resolve.NewTestResolver("/")
	filtered := []FSStat{}
	for _, stat := range stats {
		if avoidFileSystem(stat) && buildDefaultFilters(hostfs)(stat) {
			filtered = append(filtered, stat)
		}

	}

	return filterDuplicates(filtered)
}

func TestFilter(t *testing.T) {
	in := []FSStat{
		{Type: "nfs"},
		{Type: "ext4"},
		{Type: "proc"},
		{Type: "smb"},
	}
	filter := BuildFilterWithList([]string{"nfs", "smb", "proc"})
	out := []FSStat{}
	for _, fs := range in {
		if filter(fs) {
			out = append(out, fs)
		}
	}

	if assert.Len(t, out, 1) {
		assert.Equal(t, "ext4", out[0].Type)
	}
}
