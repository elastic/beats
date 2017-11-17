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
