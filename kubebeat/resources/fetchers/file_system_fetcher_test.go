package fetchers

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFileFetcherFetchASingleFile(t *testing.T) {
	directoryName := "test-outer-dir"
	files := []string{"file.txt"}
	dir := createDirectoriesWithFiles(t, "", directoryName, files)
	defer os.RemoveAll(dir)

	filePaths := []string{filepath.Join(dir, files[0])}
	fileFetcher := NewFileFetcher(filePaths)
	results, err := fileFetcher.Fetch()

	assert.Nil(t, err, "resources.Fetcher was not able to fetch files from FS")
	assert.Equal(t, 1, len(results))

	result := results[0].Resource.(FileSystemResource)
	assert.Equal(t, files[0], result.FileName)
	assert.Equal(t, "600", result.FileMode)
}

func TestFileFetcherFetchTwoPatterns(t *testing.T) {
	outerDirectoryName := "test-outer-dir"
	outerFiles := []string{"output.txt", "output1.txt"}
	outerDir := createDirectoriesWithFiles(t, "", outerDirectoryName, outerFiles)
	defer os.RemoveAll(outerDir)

	path := []string{filepath.Join(outerDir, outerFiles[0]), filepath.Join(outerDir, outerFiles[1])}
	fileFetcher := NewFileFetcher(path)
	results, err := fileFetcher.Fetch()

	assert.Nil(t, err, "resources.Fetcher was not able to fetch files from FS")
	assert.Equal(t, 2, len(results))

	firstResult := results[0].Resource.(FileSystemResource)
	assert.Equal(t, outerFiles[0], firstResult.FileName)
	assert.Equal(t, "600", firstResult.FileMode)

	secResult := results[1].Resource.(FileSystemResource)
	assert.Equal(t, outerFiles[1], secResult.FileName)
	assert.Equal(t, "600", secResult.FileMode)
}

func TestFileFetcherFetchDirectoryOnly(t *testing.T) {
	directoryName := "test-outer-dir"
	files := []string{"file.txt"}
	dir := createDirectoriesWithFiles(t, "", directoryName, files)
	defer os.RemoveAll(dir)

	filePaths := []string{filepath.Join(dir)}
	fileFetcher := NewFileFetcher(filePaths)
	results, err := fileFetcher.Fetch()

	assert.Nil(t, err, "resources.Fetcher was not able to fetch files from FS")
	assert.Equal(t, 1, len(results))
	result := results[0].Resource.(FileSystemResource)

	expectedResult := filepath.Base(dir)
	assert.Equal(t, expectedResult, result.FileName)
}

func TestFileFetcherFetchOuterDirectoryOnly(t *testing.T) {
	outerDirectoryName := "test-outer-dir"
	outerFiles := []string{"output.txt"}
	outerDir := createDirectoriesWithFiles(t, "", outerDirectoryName, outerFiles)
	defer os.RemoveAll(outerDir)

	innerDirectoryName := "test-inner-dir"
	innerFiles := []string{"innerFolderFile.txt"}
	innerDir := createDirectoriesWithFiles(t, outerDir, innerDirectoryName, innerFiles)

	path := []string{outerDir + "/*"}
	fileFetcher := NewFileFetcher(path)
	results, err := fileFetcher.Fetch()

	assert.Nil(t, err, "resources.Fetcher was not able to fetch files from FS")
	assert.Equal(t, 2, len(results))

	//All inner files should exist in the final result
	expectedResult := []string{"output.txt", filepath.Base(innerDir)}
	for i := 0; i < len(results); i++ {
		fileSystemDataResources := results[i].Resource.(FileSystemResource)
		assert.Contains(t, expectedResult, fileSystemDataResources.FileName)
	}
}

func TestFileFetcherFetchDirectoryRecursively(t *testing.T) {
	outerDirectoryName := "test-outer-dir"
	outerFiles := []string{"output.txt"}
	outerDir := createDirectoriesWithFiles(t, "", outerDirectoryName, outerFiles)
	defer os.RemoveAll(outerDir)

	innerDirectoryName := "test-inner-dir"
	innerFiles := []string{"innerFolderFile.txt"}
	innerDir := createDirectoriesWithFiles(t, outerDir, innerDirectoryName, innerFiles)

	innerInnerDirectoryName := "test-inner-inner-dir"
	innerInnerFiles := []string{"innerInnerFolderFile.txt"}
	innerInnerDir := createDirectoriesWithFiles(t, innerDir, innerInnerDirectoryName, innerInnerFiles)

	path := []string{outerDir + "/**"}
	fileFetcher := NewFileFetcher(path)
	results, err := fileFetcher.Fetch()

	assert.Nil(t, err, "resources.Fetcher was not able to fetch files from FS")
	assert.Equal(t, 6, len(results))

	directories := []string{filepath.Base(outerDir), filepath.Base(innerDir), filepath.Base(innerInnerDir)}
	allFilesName := append(append(append(innerFiles, directories...), outerFiles...), innerInnerFiles...)

	//All inner files should exist in the final result
	for i := 0; i < len(results); i++ {
		fileSystemDataResources := results[i].Resource.(FileSystemResource)
		assert.Contains(t, allFilesName, fileSystemDataResources.FileName)
	}
}

// This function creates a new directory with files inside and returns the path of the new directory
func createDirectoriesWithFiles(t *testing.T, dirPath string, dirName string, filesToWriteInDirectory []string) string {
	dirPath, err := ioutil.TempDir(dirPath, dirName)
	if err != nil {
		t.Fatal(err)
	}
	for _, fileName := range filesToWriteInDirectory {
		file := filepath.Join(dirPath, fileName)
		assert.Nil(t, ioutil.WriteFile(file, []byte("test txt\n"), 0600), "Could not able to write a new file")
	}
	return dirPath
}
