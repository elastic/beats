// +build !integration

package config

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/common"
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

	assert.Equal(t, 4, len(config.Inputs))
}

func TestEnabledInputs(t *testing.T) {
	stdinEnabled, err := common.NewConfigFrom(map[string]interface{}{
		"type":    "stdin",
		"enabled": true,
	})
	if !assert.NoError(t, err) {
		return
	}

	udpDisabled, err := common.NewConfigFrom(map[string]interface{}{
		"type":    "udp",
		"enabled": false,
	})
	if !assert.NoError(t, err) {
		return
	}

	logDisabled, err := common.NewConfigFrom(map[string]interface{}{
		"type":    "log",
		"enabled": false,
	})
	if !assert.NoError(t, err) {
		return
	}

	t.Run("ListEnabledInputs", func(t *testing.T) {
		tests := []struct {
			name     string
			config   *Config
			expected []string
		}{
			{
				name:     "all inputs disabled",
				config:   &Config{Inputs: []*common.Config{udpDisabled, logDisabled}},
				expected: []string{},
			},
			{
				name:     "all inputs enabled",
				config:   &Config{Inputs: []*common.Config{stdinEnabled}},
				expected: []string{"stdin"},
			},
			{
				name:     "disabled and enabled inputs",
				config:   &Config{Inputs: []*common.Config{stdinEnabled, udpDisabled, logDisabled}},
				expected: []string{"stdin"},
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				assert.ElementsMatch(t, test.expected, test.config.ListEnabledInputs())
			})
		}
	})

	t.Run("IsInputEnabled", func(t *testing.T) {
		config := &Config{Inputs: []*common.Config{stdinEnabled, udpDisabled, logDisabled}}

		tests := []struct {
			name     string
			input    string
			expected bool
			config   *Config
		}{
			{name: "input exists and enabled", input: "stdin", expected: true, config: config},
			{name: "input exists and disabled", input: "udp", expected: false, config: config},
			{name: "input doesn't exist", input: "redis", expected: false, config: config},
			{name: "no inputs are enabled", input: "redis", expected: false, config: &Config{}},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				assert.Equal(t, test.expected, config.IsInputEnabled(test.input))
			})
		}
	})
}
