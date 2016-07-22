// +build !integration
// +build darwin freebsd linux openbsd windows

package filesystem

import (
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFileSystemList(t *testing.T) {

	if runtime.GOOS == "darwin" && os.Getenv("TRAVIS") == "true" {
		t.Skip("FileSystem test fails on Travis/OSX with i/o error")
	}

	fss, err := GetFileSystemList()

	assert.Nil(t, err)
	assert.True(t, (len(fss) > 0))

	for _, fs := range fss {

		stat, err := GetFileSystemStat(fs)
		assert.NoError(t, err)

		assert.True(t, (stat.Total >= 0))
		assert.True(t, (stat.Free >= 0))
		assert.True(t, (stat.Avail >= 0))
		assert.True(t, (stat.Used >= 0))
	}
}
