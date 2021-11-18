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
		// TODO(ofir): Used Go native *testing.T for this error, but the testify
		// assert before. Should use the same in both places.
		t.Fatal(err)
	}

	filePaths := []string{file}
	fileFetcher := NewFileFetcher(filePaths)
	results, err := fileFetcher.Fetch()

	if err != nil {
		// TODO(ofir): The error err should be included in the failure output.
		assert.Fail(t, "Fetcher did not work")
	}
	result := results[0].(FileSystemResourceData)

	assert.Equal(t, file, result.Path)
	assert.Equal(t, "600", result.FileMode)
}
