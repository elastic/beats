// +build !integration
// +build darwin freebsd linux openbsd windows

package filesystem

import (
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
				assert.NotEqual(t, "", stat.SysTypeName)
			}
		}
	}
}

func TestFileSystemListFiltering(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("These cases don't need to work on Windows")
	}

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
			description: "Don't repeat devices, sortest of dir names should be used",
			fss: []sigar.FileSystem{
				{DirName: "/", DevName: "/dev/sda1"},
				{DirName: "/bind", DevName: "/dev/sda1"},
			},
			expected: []sigar.FileSystem{
				{DirName: "/", DevName: "/dev/sda1"},
			},
		},
		{
			description: "Don't repeat devices, sortest of dir names should be used",
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
			description: "Don't repeat devices, sortest of dir names should be used, keep tmpfs",
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
	}

	for _, c := range cases {
		filtered := filterFileSystemList(c.fss)
		assert.ElementsMatch(t, c.expected, filtered, c.description)
	}
}

func TestFileSystemListFilteringWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("These cases only need to work on windows")
	}

	cases := []struct {
		description   string
		fss, expected []sigar.FileSystem
	}{
		{
			description: "Keep all filesystems in Windows",
			fss: []sigar.FileSystem{
				{DirName: "C:\\", DevName: "C:\\"},
				{DirName: "D:\\", DevName: "D:\\"},
			},
			expected: []sigar.FileSystem{
				{DirName: "C:\\", DevName: "C:\\"},
				{DirName: "D:\\", DevName: "D:\\"},
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
