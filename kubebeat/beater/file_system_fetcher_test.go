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

func TestFileFetcherFetchDirectoryFromFileSystem(t *testing.T) {
	outerDirectoryName := "file-fetcher-test-1"
	dir, err := ioutil.TempDir("", outerDirectoryName)
	if err != nil {
		t.Fatal(err)
	}

	innerDirectoryName := "file-fetcher-test-2"
	innerDir, err := ioutil.TempDir(dir, innerDirectoryName)
	if err != nil {
		t.Fatal(err)
	}

	resourcesNames := []string{"file1.txt", "file2.txt", "file3.txt"}
	defer os.RemoveAll(dir)

	for _, fileName := range resourcesNames {
		file := filepath.Join(innerDir, fileName)
		if err = ioutil.WriteFile(file, []byte("test txt\n"), 0600); err != nil {
			assert.Fail(t, "Could not able to write a new file", err)
		}
	}

	path := []string{dir, dir + "/*", dir+ "/**/*"}
	fileFetcher := NewFileFetcher(path)
	results, err := fileFetcher.Fetch()

	if err != nil {
		assert.Fail(t, "Fetcher was not able to fetch files from FS", err)
	}

	assert.Equal(t, len(results), 5)

	directories := []string{filepath.Base(dir),filepath.Base(innerDir)}
	allFilesName := append(resourcesNames, directories...)

	//All inner files should exist in the final result
	for i := 0; i < len(results); i++ {

		fileSystemDataResources := results[i].(FileSystemResourceData)
		assert.Contains(t, allFilesName, fileSystemDataResources.FileName)
	}
}
