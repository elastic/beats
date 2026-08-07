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
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-system-metrics/dev-tools/systemtests"
)

func TestFileSystemList(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	skipTypes := []string{"cdrom", "tracefs", "overlay", "fuse.lxcfs", "fuse.gvfsd-fuse", "nsfs", "squashfs", "vmhgfs"}
	hostfs := systemtests.DockerTestResolver(logger)
	//Exclude FS types that will give us a permission error
	fss, err := GetFilesystems(hostfs, BuildFilterWithList(skipTypes))
	if err != nil {
		t.Fatal("GetFileSystemList", err)
	}
	assert.True(t, (len(fss) > 0))

	for _, fs := range fss {
		err := fs.GetUsage()
		assert.NoError(t, err, "filesystem=%#v: %v", fs, err)

	}
}

func TestFileSystemListFiltering(t *testing.T) {
	if runtime.GOOS == "windows" {
		// Windows doesn't like these unix paths, the OS-specific code in stdlib will return different results.
		t.Skip("These cases don't need to work on Windows")
	}
	logger := logptest.NewTestingLogger(t, "")
	fakeDevDir := t.TempDir()

	cases := []struct {
		description   string
		fss, expected []FSStat
	}{
		{
			description: "basic filter test to remove duplicates",
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

		filtered := filterFileSystemList(logger, c.fss)
		ok := assert.ElementsMatch(t, c.expected, filtered, c.description)
		if !ok {
			t.FailNow()
		}
	}
}

// Emulate the filtering process that would normally happen inside the callbacks from platform-specific code
func filterFileSystemList(logger *logp.Logger, stats []FSStat) []FSStat {
	hostfs := systemtests.DockerTestResolver(logger)
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
