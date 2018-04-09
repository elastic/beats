// +build !windows,!integration

package file

import (
	"io/ioutil"
	"math"
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetOSFileState(t *testing.T) {
	file, err := ioutil.TempFile("", "")
	assert.Nil(t, err)

	fileinfo, err := file.Stat()
	assert.Nil(t, err)

	state := GetOSState(fileinfo)

	assert.True(t, state.Inode > 0)

	if runtime.GOOS == "openbsd" {
		// The first device on OpenBSD has an ID of 0 so allow this.
		assert.True(t, state.Device >= 0, "Device %d", state.Device)
	} else {
		assert.True(t, state.Device > 0, "Device %d", state.Device)
	}
}

func TestGetOSFileStateStat(t *testing.T) {
	file, err := ioutil.TempFile("", "")
	assert.Nil(t, err)

	fileinfo, err := os.Stat(file.Name())
	assert.Nil(t, err)

	state := GetOSState(fileinfo)

	assert.True(t, state.Inode > 0)

	if runtime.GOOS == "openbsd" {
		// The first device on OpenBSD has an ID of 0 so allow this.
		assert.True(t, state.Device >= 0, "Device %d", state.Device)
	} else {
		assert.True(t, state.Device > 0, "Device %d", state.Device)
	}
}

func BenchmarkStateString(b *testing.B) {
	var samples [50]uint64
	for i, v := 0, uint64(0); i < len(samples); i, v = i+1, v+math.MaxUint64/uint64(len(samples)) {
		samples[i] = v
	}

	for i := 0; i < b.N; i++ {
		for _, inode := range samples {
			for _, device := range samples {
				st := StateOS{Inode: inode, Device: device}
				if st.String() == "" {
					b.Fatal("empty state string")
				}
			}
		}
	}
}
