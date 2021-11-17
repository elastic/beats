package beater

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestFileFetcherFetchFilesFromFileSystem(t *testing.T) {

	dir, err := ioutil.TempDir("", "file-fetcher-test")
	if err != nil {
		t.Fatal(err)
	}

	defer os.RemoveAll(dir)
	file := filepath.Join(dir, "file.txt")
	if err = ioutil.WriteFile(file, []byte("test txt\n"), 0600); err != nil {
		t.Fatal(err)
	}

	filePaths := []string{file}
	fileFetcher := NewFileFetcher(filePaths)
	results, err := fileFetcher.Fetch()

	if err != nil {
		assert.Fail(t, "Fetcher did not work")
	}
	result := results.([]FileSystemResourceData)[0]

	assert.Equal(t, file, result.Path)
	assert.Equal(t, "600", result.FileMode)
}
