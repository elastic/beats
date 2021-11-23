package beater

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFileFetcherFetchFilesFromFileSystem(t *testing.T) {

	dir, err := ioutil.TempDir("", "file-fetcher-test")
	if err != nil {
		t.Fatal(err)
	}

	defer os.RemoveAll(dir)
	file := filepath.Join(dir, "file.txt")
	if err = ioutil.WriteFile(file, []byte("test txt\n"), 0600); err != nil {
		assert.Fail(t, "Could not able to write a new file", err)
	}

	filePaths := []string{file}
	fileFetcher := NewFileFetcher(filePaths)
	results, err := fileFetcher.Fetch()

	if err != nil {
		assert.Fail(t, "Fetcher was not able to fetch files from FS", err)
	}
	result := results[0].(FileSystemResourceData)

	assert.Equal(t, file, result.Path)
	assert.Equal(t, "600", result.FileMode)
}
