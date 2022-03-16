package proc

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"testing/fstest"
)

var io = `rchar: 45479191
wchar: 42997831
syscr: 1327670
syscw: 582115
read_bytes: 3784704
write_bytes: 8192`

func TestReadIOFS(t *testing.T) {
	pid := "1"
	testCases := []struct {
		fstest fstest.MapFS
		assert func(ProcIO, error)
	}{
		{
			fstest.MapFS{
				"hello.txt": {
					Data: []byte("hello, world"),
				},
				"proc/1/io": {
					Data: []byte(io),
				},
				"proc/1/stat": {
					Data: []byte("some data"),
				},
			},
			func(result ProcIO, err error) {
				assert.Nil(t, err)
				assert.Equal(t, result.ReadBytes, "3784704")
				assert.Equal(t, result.WriteBytes, "8192")
			}},
		{
			fstest.MapFS{
				"hello.txt": {
					Data: []byte("hello, world"),
				},
			},
			func(result ProcIO, err error) {
				assert.Error(t, err)
			}},
	}

	for _, testCase := range testCases {
		result, err := ReadIOFS(testCase.fstest, pid)
		testCase.assert(result, err)
	}
}
