package fetchers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Based on https://github.com/yargevad/filepathx/blob/master/filepathx.go

func TestGlobMatchingNonExistingPattern(t *testing.T) {
	directoryName := "test-outer-dir"
	fileName := "file.txt"
	dir := createDirectoriesWithFiles(t, "", directoryName, []string{fileName})
	defer os.RemoveAll(dir)

	filePath := filepath.Join(dir, fileName)
	matchedFiles, err := Glob(filePath + "/***")

	assert.Nil(t, err)
	assert.Nil(t, matchedFiles)
}

func TestGlobMatchingPathDoesNotExist(t *testing.T) {
	directoryName := "test-outer-dir"
	fileName := "file.txt"
	dir := createDirectoriesWithFiles(t, "", directoryName, []string{fileName})
	defer os.RemoveAll(dir)

	filePath := filepath.Join(dir, fileName)
	matchedFiles, err := Glob(filePath + "/abc")

	assert.Nil(t, err)
	assert.Nil(t, matchedFiles)
}

func TestGlobMatchingSingleFile(t *testing.T) {
	directoryName := "test-outer-dir"
	fileName := "file.txt"
	dir := createDirectoriesWithFiles(t, "", directoryName, []string{fileName})
	defer os.RemoveAll(dir)

	filePath := filepath.Join(dir, fileName)
	matchedFiles, err := Glob(filePath)

	assert.Nil(t, err, "Glob could not fetch results")
	assert.Equal(t, 1, len(matchedFiles))
	assert.Equal(t, matchedFiles[0], filePath)
}

func TestGlobDirectoryOnly(t *testing.T) {
	directoryName := "test-outer-dir"
	fileName := "file.txt"
	dir := createDirectoriesWithFiles(t, "", directoryName, []string{fileName})
	defer os.RemoveAll(dir)

	matchedFiles, err := Glob(dir)

	assert.Nil(t, err, "Glob could not fetch results")
	assert.Equal(t, 1, len(matchedFiles))
	assert.Equal(t, matchedFiles[0], dir)
}

func TestGlobOuterDirectoryOnly(t *testing.T) {
	outerDirectoryName := "test-outer-dir"
	outerFiles := []string{"output.txt"}
	outerDir := createDirectoriesWithFiles(t, "", outerDirectoryName, outerFiles)
	defer os.RemoveAll(outerDir)

	innerDirectoryName := "test-inner-dir"
	innerFiles := []string{"innerFolderFile.txt"}
	innerDir := createDirectoriesWithFiles(t, outerDir, innerDirectoryName, innerFiles)

	matchedFiles, err := Glob(outerDir + "/*")

	assert.Nil(t, err, "Glob could not fetch results")
	assert.Equal(t, 2, len(matchedFiles))
	assert.Equal(t, matchedFiles[0], filepath.Join(outerDir, outerFiles[0]))
	assert.Equal(t, matchedFiles[1], innerDir)
}

func TestGlobDirectoryRecursively(t *testing.T) {
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

	matchedFiles, err := Glob(outerDir + "/**")

	assert.Nil(t, err, "Glob could not fetch results")
	assert.Equal(t, 6, len(matchedFiles))

	//When using glob matching recursively the first outer folder is being sent without a '/'
	assert.Equal(t, matchedFiles[0], outerDir+"/")
	assert.Equal(t, matchedFiles[1], filepath.Join(outerDir, outerFiles[0]))
	assert.Equal(t, matchedFiles[2], innerDir)
	assert.Equal(t, matchedFiles[3], filepath.Join(innerDir, innerFiles[0]))
	assert.Equal(t, matchedFiles[4], innerInnerDir)
	assert.Equal(t, matchedFiles[5], filepath.Join(innerInnerDir, innerInnerFiles[0]))
}
