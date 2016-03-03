package input

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TODO: Tests to be implemented
// * Check file renaming
// * Check file ids for moved files (windows)

func TestIsSameFile(t *testing.T) {
	absPath, err := filepath.Abs("../tests/files/")

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

func TestSafeFileRotateExistingFile(t *testing.T) {

	tempdir, err := ioutil.TempDir("", "")
	assert.NoError(t, err)
	defer func() {
		assert.NoError(t, os.RemoveAll(tempdir))
	}()

	// create an existing .filebeat file
	err = ioutil.WriteFile(filepath.Join(tempdir, ".filebeat"),
		[]byte("existing filebeat"), 0x777)
	assert.NoError(t, err)

	// create a new .filebeat.new file
	err = ioutil.WriteFile(filepath.Join(tempdir, ".filebeat.new"),
		[]byte("new filebeat"), 0x777)
	assert.NoError(t, err)

	// rotate .filebeat.new into .filebeat
	err = SafeFileRotate(filepath.Join(tempdir, ".filebeat"),
		filepath.Join(tempdir, ".filebeat.new"))
	assert.NoError(t, err)

	contents, err := ioutil.ReadFile(filepath.Join(tempdir, ".filebeat"))
	assert.NoError(t, err)
	assert.Equal(t, []byte("new filebeat"), contents)

	// do it again to make sure we deal with deleting the old file

	err = ioutil.WriteFile(filepath.Join(tempdir, ".filebeat.new"),
		[]byte("new filebeat 1"), 0x777)
	assert.NoError(t, err)

	err = SafeFileRotate(filepath.Join(tempdir, ".filebeat"),
		filepath.Join(tempdir, ".filebeat.new"))
	assert.NoError(t, err)

	contents, err = ioutil.ReadFile(filepath.Join(tempdir, ".filebeat"))
	assert.NoError(t, err)
	assert.Equal(t, []byte("new filebeat 1"), contents)

	// and again for good measure

	err = ioutil.WriteFile(filepath.Join(tempdir, ".filebeat.new"),
		[]byte("new filebeat 2"), 0x777)
	assert.NoError(t, err)

	err = SafeFileRotate(filepath.Join(tempdir, ".filebeat"),
		filepath.Join(tempdir, ".filebeat.new"))
	assert.NoError(t, err)

	contents, err = ioutil.ReadFile(filepath.Join(tempdir, ".filebeat"))
	assert.NoError(t, err)
	assert.Equal(t, []byte("new filebeat 2"), contents)
}

func TestFileEventToMapStr(t *testing.T) {
	// Test 'fields' is not present when it is nil.
	event := FileEvent{}
	mapStr := event.ToMapStr()
	_, found := mapStr["fields"]
	assert.False(t, found)
}
