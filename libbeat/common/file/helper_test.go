// +build !integration

package file

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
