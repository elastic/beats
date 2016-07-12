// +build !integration

package config

import (
	"path/filepath"
	"testing"

	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/stretchr/testify/assert"
)

func TestReadConfig2(t *testing.T) {
	// Tests with different params from config file
	absPath, err := filepath.Abs("../tests/files/")

	assert.NotNil(t, absPath)
	assert.Nil(t, err)

	config := &Config{}

	// Reads second config file
	err = cfgfile.Read(config, absPath+"/config2.yml")
	assert.Nil(t, err)

	assert.Equal(t, uint64(0), config.SpoolSize)
}

func TestGetConfigFiles_File(t *testing.T) {
	absPath, err := filepath.Abs("../tests/files/")

	assert.NotNil(t, absPath)
	assert.Nil(t, err)

	files, err := getConfigFiles(absPath + "/config.yml")

	assert.Nil(t, err)
	assert.Equal(t, 1, len(files))

	assert.Equal(t, absPath+"/config.yml", files[0])
}

func TestGetConfigFiles_Dir(t *testing.T) {
	absPath, err := filepath.Abs("../tests/files/")

	assert.NotNil(t, absPath)
	assert.Nil(t, err)

	files, err := getConfigFiles(absPath)

	assert.Nil(t, err)
	assert.Equal(t, 2, len(files))

	assert.Equal(t, filepath.Join(absPath, "/config.yml"), files[0])
	assert.Equal(t, filepath.Join(absPath, "/config2.yml"), files[1])
}

func TestGetConfigFiles_EmptyDir(t *testing.T) {
	absPath, err := filepath.Abs("../tests/files/")

	assert.NotNil(t, absPath)
	assert.Nil(t, err)

	files, err := getConfigFiles(absPath + "/logs")

	assert.Nil(t, err)
	assert.Equal(t, 0, len(files))
}

func TestGetConfigFiles_Invalid(t *testing.T) {
	absPath, err := filepath.Abs("../tests/files/")

	assert.NotNil(t, absPath)
	assert.Nil(t, err)

	// Invalid directory
	files, err := getConfigFiles(absPath + "/qwerwer")

	assert.NotNil(t, err)
	assert.Nil(t, files)
}

func TestMergeConfigFiles(t *testing.T) {
	absPath, err := filepath.Abs("../tests/files/")

	assert.NotNil(t, absPath)
	assert.Nil(t, err)

	files, err := getConfigFiles(absPath)

	assert.Nil(t, err)
	assert.Equal(t, 2, len(files))

	config := &Config{}
	mergeConfigFiles(files, config)

	assert.Equal(t, 4, len(config.Prospectors))
}
