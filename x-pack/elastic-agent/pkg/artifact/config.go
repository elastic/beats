// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package artifact

import (
	"runtime"
	"strings"
	"time"

	"github.com/elastic/beats/v7/libbeat/common/transport/httpcommon"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
)

// Config is a configuration used for verifier and downloader
type Config struct {
	// OperatingSystem: operating system [linux, windows, darwin]
	OperatingSystem string `json:"-" config:",ignore"`

	// Architecture: target architecture [32, 64]
	Architecture string `json:"-" config:",ignore"`

	// SourceURI: source of the artifacts, e.g https://artifacts.elastic.co/downloads/
	SourceURI string `json:"sourceURI" config:"sourceURI"`

	// TargetDirectory: path to the directory containing downloaded packages
	TargetDirectory string `json:"targetDirectory" config:"target_directory"`

	// InstallPath: path to the directory containing installed packages
	InstallPath string `yaml:"installPath" config:"install_path"`

	// DropPath: path where elastic-agent can find installation files for download.
	// Difference between this and TargetDirectory is that when fetching packages (from web or fs) they are stored in TargetDirectory
	// DropPath specifies where Filesystem downloader can find packages which will then be placed in TargetDirectory. This can be
	// local or network disk.
	// If not provided FileSystem Downloader will fallback to /beats subfolder of elastic-agent directory.
	DropPath string `yaml:"dropPath" config:"drop_path"`

	httpcommon.HTTPTransportSettings `config:",inline" yaml:",inline"` // Note: use anonymous struct for json inline
}

// DefaultConfig creates a config with pre-set default values.
func DefaultConfig() *Config {
	transport := httpcommon.DefaultHTTPTransportSettings()

	// binaries are a getting bit larger it might take >30s to download them
	transport.Timeout = 120 * time.Second

	return &Config{
		SourceURI:             "https://artifacts.elastic.co/downloads/",
		TargetDirectory:       paths.Downloads(),
		InstallPath:           paths.Install(),
		HTTPTransportSettings: transport,
	}
}

// OS returns the configured operating system or falls back to runtime.GOOS
func (c *Config) OS() string {
	if c.OperatingSystem != "" {
		return c.OperatingSystem
	}

	switch runtime.GOOS {
	case "windows":
		c.OperatingSystem = "windows"
	case "darwin":
		c.OperatingSystem = "darwin"
	default:
		c.OperatingSystem = "linux"
	}

	return c.OperatingSystem
}

// Arch returns the configured architecture or falls back to 32bit
func (c *Config) Arch() string {
	if c.Architecture != "" {
		return c.Architecture
	}

	arch := "32"
	if strings.Contains(runtime.GOARCH, "arm64") {
		arch = "arm64"
	} else if strings.Contains(runtime.GOARCH, "64") {
		arch = "64"
	}

	c.Architecture = arch
	return c.Architecture
}
