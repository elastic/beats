package config

import (
	"github.com/elastic/libbeat/cfgfile"
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
)

func TestReadConfig(t *testing.T) {
	absPath, err := filepath.Abs("../tests/files/")

	assert.NotNil(t, absPath)
	assert.Nil(t, err)

	config := &Config{}

	err = cfgfile.Read(config, absPath+"/config.yml")

	assert.Nil(t, err)

	files := config.Filebeat.Files

	// Check if multiple paths were read in
	assert.Equal(t, 3, len(files))

	// Check if full array can be read. Assumed that are ordered same as in config file
	assert.Equal(t, 2, len(files[0].Paths))
	assert.Equal(t, "/var/log/s*.log", files[0].Paths[1])
	assert.Equal(t, "log", files[0].Input)
	assert.Equal(t, 3, len(files[0].Fields))
	assert.Equal(t, 1, len(files[0].Fields["review"]))
	assert.Equal(t, "24h", files[0].IgnoreOlder)

	assert.Equal(t, "stdin", files[2].Input)
	assert.Equal(t, 0, len(files[2].Paths))

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

	assert.Equal(t, absPath+"/config.yml", files[0])
	assert.Equal(t, absPath+"/config2.yml", files[1])
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

	assert.Equal(t, 4, len(config.Filebeat.Files))
}
