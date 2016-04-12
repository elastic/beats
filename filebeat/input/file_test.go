// +build !integration

package input

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/elastic/beats/filebeat/config"
	"github.com/elastic/beats/libbeat/common"
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

	// create an existing registry file
	err = ioutil.WriteFile(filepath.Join(tempdir, "registry"),
		[]byte("existing filebeat"), 0x777)
	assert.NoError(t, err)

	// create a new registry.new file
	err = ioutil.WriteFile(filepath.Join(tempdir, "registry.new"),
		[]byte("new filebeat"), 0x777)
	assert.NoError(t, err)

	// rotate registry.new into registry
	err = SafeFileRotate(filepath.Join(tempdir, "registry"),
		filepath.Join(tempdir, "registry.new"))
	assert.NoError(t, err)

	contents, err := ioutil.ReadFile(filepath.Join(tempdir, "registry"))
	assert.NoError(t, err)
	assert.Equal(t, []byte("new filebeat"), contents)

	// do it again to make sure we deal with deleting the old file

	err = ioutil.WriteFile(filepath.Join(tempdir, "registry.new"),
		[]byte("new filebeat 1"), 0x777)
	assert.NoError(t, err)

	err = SafeFileRotate(filepath.Join(tempdir, "registry"),
		filepath.Join(tempdir, "registry.new"))
	assert.NoError(t, err)

	contents, err = ioutil.ReadFile(filepath.Join(tempdir, "registry"))
	assert.NoError(t, err)
	assert.Equal(t, []byte("new filebeat 1"), contents)

	// and again for good measure

	err = ioutil.WriteFile(filepath.Join(tempdir, "registry.new"),
		[]byte("new filebeat 2"), 0x777)
	assert.NoError(t, err)

	err = SafeFileRotate(filepath.Join(tempdir, "registry"),
		filepath.Join(tempdir, "registry.new"))
	assert.NoError(t, err)

	contents, err = ioutil.ReadFile(filepath.Join(tempdir, "registry"))
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

func TestFileEventToMapStrJSON(t *testing.T) {
	type io struct {
		Event         FileEvent
		ExpectedItems common.MapStr
	}

	text := "hello"

	tests := []io{
		{
			// by default, don't overwrite keys
			Event: FileEvent{
				DocumentType: "test_type",
				Text:         &text,
				JSONFields:   common.MapStr{"type": "test", "text": "hello"},
				JSONConfig:   &config.JSONConfig{KeysUnderRoot: true},
			},
			ExpectedItems: common.MapStr{
				"type": "test_type",
				"text": "hello",
			},
		},
		{
			// overwrite keys if asked
			Event: FileEvent{
				DocumentType: "test_type",
				Text:         &text,
				JSONFields:   common.MapStr{"type": "test", "text": "hello"},
				JSONConfig:   &config.JSONConfig{KeysUnderRoot: true, OverwriteKeys: true},
			},
			ExpectedItems: common.MapStr{
				"type": "test",
				"text": "hello",
			},
		},
		{
			// without keys_under_root, put everything in a json key
			Event: FileEvent{
				DocumentType: "test_type",
				Text:         &text,
				JSONFields:   common.MapStr{"type": "test", "text": "hello"},
				JSONConfig:   &config.JSONConfig{},
			},
			ExpectedItems: common.MapStr{
				"json": common.MapStr{"type": "test", "text": "hello"},
				"type": "test_type",
			},
		},
		{
			// when MessageKey is defined, the Text overwrites the value of that key
			Event: FileEvent{
				DocumentType: "test_type",
				Text:         &text,
				JSONFields:   common.MapStr{"type": "test", "text": "hi"},
				JSONConfig:   &config.JSONConfig{MessageKey: "text"},
			},
			ExpectedItems: common.MapStr{
				"json": common.MapStr{"type": "test", "text": "hello"},
				"type": "test_type",
			},
		},
	}

	for _, test := range tests {
		result := test.Event.ToMapStr()
		for k, v := range test.ExpectedItems {
			assert.Equal(t, v, result[k])
		}
	}
}
