// +build !integration

package file

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsSameFile(t *testing.T) {
	absPath, err := filepath.Abs("../../tests/files/")

	assert.NotNil(t, absPath)
	assert.Nil(t, err)

	fileInfo1, err := os.Stat(absPath + "/logs/test.log")
	fileInfo2, err := os.Stat(absPath + "/logs/system.log")

	assert.Nil(t, err)
	assert.NotNil(t, fileInfo1)
	assert.NotNil(t, fileInfo2)

	file1 := &File{
		FileInfo: fileInfo1,
	}

	file2 := &File{
		FileInfo: fileInfo2,
	}

	file3 := &File{
		FileInfo: fileInfo2,
	}

	assert.False(t, file1.IsSameFile(file2))
	assert.False(t, file2.IsSameFile(file1))

	assert.True(t, file1.IsSameFile(file1))
	assert.True(t, file2.IsSameFile(file2))

	assert.True(t, file3.IsSameFile(file2))
	assert.True(t, file2.IsSameFile(file3))
}
