package proc

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"testing/fstest"
)

var cmdData = `/usr/bin/processName --config=/etc/conf/config.conf`

func TestReadCmdLineFS(t *testing.T) {
	pid := "1"
	testCases := []struct {
		fstest fstest.MapFS
		assert func(string, error)
	}{
		{
			fstest.MapFS{
				"hello.txt": {
					Data: []byte("hello, world"),
				},
				"proc/1/cmdline": {
					Data: []byte(cmdData),
				},
				"proc/1/stat": {
					Data: []byte("some data"),
				},
			},
			func(result string, err error) {
				assert.Nil(t, err)
				assert.Equal(t, result, cmdData)
			}},
		{
			fstest.MapFS{
				"hello.txt": {
					Data: []byte("hello, world"),
				},
			},
			func(result string, err error) {
				assert.Error(t, err)
				assert.Equal(t, result, "")
			}},
	}

	for _, testCase := range testCases {
		result, err := ReadCmdLineFS(testCase.fstest, pid)
		testCase.assert(result, err)
	}
}
