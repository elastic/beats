package config

import (
	"path/filepath"
	"testing"

	"github.com/elastic/libbeat/cfgfile"
	"github.com/stretchr/testify/assert"
)

func TestReadConfig(t *testing.T) {
	absPath, err := filepath.Abs("../tests/files/")

	assert.NotNil(t, absPath)
	assert.Nil(t, err)

	config := &Config{}

	err = cfgfile.Read(config, absPath+"/config.yml")

	assert.Nil(t, err)

	assert.Equal(t, uint64(1024), config.Filebeat.SpoolSize)
	assert.Equal(t, "/prospectorConfigs/", config.Filebeat.ConfigDir)

	prospectors := config.Filebeat.Prospectors

	// Check if multiple paths were read in
	assert.Equal(t, 3, len(prospectors))

	// Check if full array can be read. Assumed that are ordered same as in config file
	assert.Equal(t, 2, len(prospectors[0].Paths))
	assert.Equal(t, "/var/log/s*.log", prospectors[0].Paths[1])
	assert.Equal(t, "log", prospectors[0].Input)
	assert.Equal(t, 3, len(prospectors[0].Fields))
	assert.Equal(t, 1, len(prospectors[0].Fields["review"]))
	assert.Equal(t, "24h", prospectors[0].IgnoreOlder)
	assert.Equal(t, "10s", prospectors[0].ScanFrequency)

	assert.Equal(t, "stdin", prospectors[2].Input)
	assert.Equal(t, 0, len(prospectors[2].Paths))
	assert.Equal(t, "", prospectors[1].ScanFrequency)
}

func TestReadConfig2(t *testing.T) {
	// Tests with different params from config file
	absPath, err := filepath.Abs("../tests/files/")

	assert.NotNil(t, absPath)
	assert.Nil(t, err)

	config := &Config{}

	// Reads second config file
	err = cfgfile.Read(config, absPath+"/config2.yml")
	assert.Nil(t, err)

	assert.Equal(t, uint64(0), config.Filebeat.SpoolSize)
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

	assert.Equal(t, 4, len(config.Filebeat.Prospectors))
}
