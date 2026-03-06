// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Config is put into a different package to prevent cyclic imports in case
// it is needed in several locations

package config

import (
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"

	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
)

// Default index name for ad-hoc queries, since the dataset is defined at the stream level, for example:
// streams:
// - id: '123456'
//   data_stream:
// 	dataset: osquery_manager.result
// 	type: logs
//   query: select * from usb_devices

const (
	DefaultNamespace              = "default"
	DefaultDataset                = "osquery_manager.result"
	DefaultType                   = "logs"
	DefaultActionResponsesDataset = "osquery_manager.action.responses"
)

var datastreamPrefix = fmt.Sprintf("%s-%s-", DefaultType, DefaultDataset)

type StreamConfig struct {
	ID         string                 `config:"id"`
	Query      string                 `config:"query"`       // the SQL query to run
	Interval   int                    `config:"interval"`    // an interval in seconds to run the query (subject to splay/smoothing). It has a maximum value of 604,800 (1 week).
	Platform   string                 `config:"platform"`    // restrict this query to a given platform, default is 'all' platforms; you may use commas to set multiple platforms
	Version    string                 `config:"version"`     // only run on osquery versions greater than or equal-to this version string
	ECSMapping map[string]interface{} `config:"ecs_mapping"` // ECS mapping definition where the key is the source field in osquery result and the value is the destination fields in ECS
}

type DatastreamConfig struct {
	Namespace string `config:"namespace"`
	Dataset   string `config:"dataset"`
	Type      string `config:"type"`
}

type InputConfig struct {
	Name       string                  `config:"name"`
	Type       string                  `config:"type"`
	Datastream DatastreamConfig        `config:"data_stream"` // Datastream configuration
	Processors processors.PluginConfig `config:"processors"`

	// Full Osquery configuration
	Osquery *OsqueryConfig `config:"osquery"`

	// Deprecated
	Streams   []StreamConfig `config:"streams"`
	Platform  string         `config:"iplatform"` // restrict all queries to a given platform, default is 'all' platforms; you may use commas to set multiple platforms
	Version   string         `config:"iversion"`  // only run the queries with osquery versions greater than or equal-to this version string
	Discovery []string       `config:"discovery"` // a list of discovery queries https://osquery.readthedocs.io/en/stable/deployment/configuration/#discovery-queries
}

type Config struct {
	Inputs []InputConfig `config:"inputs"`
}

type InstallConfig struct {
	Linux            *InstallArtifactConfig `config:"linux"`
	Darwin           *InstallArtifactConfig `config:"darwin"`
	Windows          *InstallArtifactConfig `config:"windows"`
	AllowInsecureURL bool                   `config:"allow_insecure_url"`
	SSL              *tlscommon.Config      `config:"ssl"`
}

type InstallArtifactConfig struct {
	ArtifactURL      string            `config:"artifact_url"`
	SHA256           string            `config:"sha256"`
	AllowInsecureURL *bool             `config:"allow_insecure_url"`
	SSL              *tlscommon.Config `config:"ssl"`
}

func (c InstallConfig) Enabled() bool {
	return c.EnabledForPlatform("linux") || c.EnabledForPlatform("darwin") || c.EnabledForPlatform("windows")
}

func (c InstallConfig) EnabledForPlatform(goos string) bool {
	platformCfg := c.PlatformConfig(goos)
	if platformCfg == nil {
		return false
	}
	return strings.TrimSpace(platformCfg.ArtifactURL) != ""
}

func (c InstallConfig) PlatformConfig(goos string) *InstallArtifactConfig {
	switch goos {
	case "linux":
		return c.Linux
	case "darwin":
		return c.Darwin
	case "windows":
		return c.Windows
	default:
		return nil
	}
}

func (c InstallConfig) SelectedForPlatform(goos string) (InstallArtifactConfig, bool) {
	platformCfg := c.PlatformConfig(goos)
	if platformCfg == nil || strings.TrimSpace(platformCfg.ArtifactURL) == "" {
		return InstallArtifactConfig{}, false
	}
	return *platformCfg, true
}

func (c InstallConfig) AllowInsecureURLForPlatform(goos string) bool {
	platformCfg := c.PlatformConfig(goos)
	if platformCfg != nil && platformCfg.AllowInsecureURL != nil {
		return *platformCfg.AllowInsecureURL
	}
	return c.AllowInsecureURL
}

func (c InstallConfig) SSLForPlatform(goos string) *tlscommon.Config {
	platformCfg := c.PlatformConfig(goos)
	if platformCfg != nil && platformCfg.SSL != nil {
		return platformCfg.SSL
	}
	return c.SSL
}

func (c *InstallConfig) NormalizeAndValidate() error {
	platforms := []struct {
		name string
		cfg  *InstallArtifactConfig
	}{
		{name: "linux", cfg: c.Linux},
		{name: "darwin", cfg: c.Darwin},
		{name: "windows", cfg: c.Windows},
	}

	for _, platform := range platforms {
		if platform.cfg == nil {
			continue
		}
		platform.cfg.ArtifactURL = strings.TrimSpace(platform.cfg.ArtifactURL)
		platform.cfg.SHA256 = strings.ToLower(strings.TrimSpace(platform.cfg.SHA256))

		if platform.cfg.ArtifactURL == "" && platform.cfg.SHA256 == "" {
			continue
		}
		if platform.cfg.ArtifactURL == "" {
			return fmt.Errorf("osquery.elastic_options.install.%s.artifact_url is required when sha256 is set", platform.name)
		}
		if platform.cfg.SHA256 == "" {
			return fmt.Errorf("osquery.elastic_options.install.%s.sha256 is required when artifact_url is set", platform.name)
		}

		hashBytes, err := hex.DecodeString(platform.cfg.SHA256)
		if err != nil || len(hashBytes) != 32 {
			return fmt.Errorf("osquery.elastic_options.install.%s.sha256 must be a valid SHA256 hex string", platform.name)
		}

		u, err := url.Parse(platform.cfg.ArtifactURL)
		if err != nil {
			return fmt.Errorf("invalid osquery.elastic_options.install.%s.artifact_url: %w", platform.name, err)
		}
		if u.Scheme == "" || u.Host == "" {
			return fmt.Errorf("osquery.elastic_options.install.%s.artifact_url must be an absolute URL", platform.name)
		}
		if !c.AllowInsecureURLForPlatform(platform.name) && strings.ToLower(u.Scheme) != "https" {
			return fmt.Errorf("osquery.elastic_options.install.%s.artifact_url must use https unless osquery.elastic_options.install.allow_insecure_url is true", platform.name)
		}
	}

	return nil
}

var DefaultConfig = Config{}

func Datastream(namespace string) string {
	if namespace == "" {
		namespace = DefaultNamespace
	}
	return datastreamPrefix + namespace
}

// GetOsqueryOptions Returns options from the first input if available
func GetOsqueryOptions(inputs []InputConfig) map[string]interface{} {
	if len(inputs) == 0 {
		return nil
	}
	if inputs[0].Osquery == nil {
		return nil
	}
	return inputs[0].Osquery.Options
}

// GetOsqueryInstallConfig returns custom osquery install settings from the first input if available.
func GetOsqueryInstallConfig(inputs []InputConfig) InstallConfig {
	if len(inputs) == 0 {
		return InstallConfig{}
	}
	if inputs[0].Osquery == nil || inputs[0].Osquery.ElasticOptions == nil || inputs[0].Osquery.ElasticOptions.Install == nil {
		return InstallConfig{}
	}
	return *inputs[0].Osquery.ElasticOptions.Install
}
