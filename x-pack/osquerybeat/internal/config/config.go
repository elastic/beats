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
	DefaultNamespace               = "default"
	DefaultDataset                 = "osquery_manager.result"
	DefaultType                    = "logs"
	DefaultActionResponsesDataset  = "osquery_manager.action.responses"
	DefaultQueryProfileDataset     = "osquery_manager.query_profile"
	DefaultQueryProfileMaxProfiles = 64
)

var datastreamPrefix = fmt.Sprintf("%s-%s-", DefaultType, DefaultDataset)
var queryProfileDatastreamPrefix = fmt.Sprintf("%s-%s-", DefaultType, DefaultQueryProfileDataset)

type StreamConfig struct {
	ID         string                 `config:"id"`
	Query      string                 `config:"query"`       // the SQL query to run
	Interval   int                    `config:"interval"`    // an interval in seconds to run the query (subject to splay/smoothing). It has a maximum value of 604,800 (1 week).
	Platform   string                 `config:"platform"`    // restrict this query to a given platform, default is 'all' platforms; you may use commas to set multiple platforms
	Version    string                 `config:"version"`     // only run on osquery versions greater than or equal-to this version string
	ECSMapping map[string]interface{} `config:"ecs_mapping"` // ECS mapping definition where the key is the source field in osquery result and the value is the destination fields in ECS
	// Profile enables per-query profiling for this stream. Published profiles for live-style
	// paths reflect the osqueryd process while extension queries run (serialized on the beat
	// client; native osqueryd schedules may still contribute load). Requires an input stream
	// with dataset osquery_manager.query_profile to publish events.
	Profile bool `config:"profile" json:"profile,omitempty"`
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
	Linux            *InstallPlatformConfig `config:"linux"`
	Darwin           *InstallPlatformConfig `config:"darwin"`
	Windows          *InstallPlatformConfig `config:"windows"`
	AllowInsecureURL bool                   `config:"allow_insecure_url"`
	SSL              *tlscommon.Config      `config:"ssl"`
}

type InstallPlatformConfig struct {
	SSL   *tlscommon.Config      `config:"ssl"`
	AMD64 *InstallArtifactConfig `config:"amd64"`
	ARM64 *InstallArtifactConfig `config:"arm64"`
}

type InstallArtifactConfig struct {
	ArtifactURL      string            `config:"artifact_url"`
	SHA256           string            `config:"sha256"`
	AllowInsecureURL *bool             `config:"allow_insecure_url"`
	SSL              *tlscommon.Config `config:"ssl"`
}

func (c InstallConfig) Enabled() bool {
	for _, platformCfg := range []*InstallPlatformConfig{c.Linux, c.Darwin, c.Windows} {
		if platformCfg == nil {
			continue
		}
		if hasArtifactConfig(platformCfg.AMD64) || hasArtifactConfig(platformCfg.ARM64) {
			return true
		}
	}
	return false
}

func (c InstallConfig) EnabledForPlatform(goos, goarch string) bool {
	_, enabled := c.SelectedForPlatform(goos, goarch)
	return enabled
}

func (c InstallConfig) PlatformConfig(goos string) *InstallPlatformConfig {
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

func (c InstallConfig) SelectedForPlatform(goos, goarch string) (InstallArtifactConfig, bool) {
	// PlatformConfig may return nil; ArchConfig is safe to call on nil receiver.
	if archCfg := c.PlatformConfig(goos).ArchConfig(goarch); hasArtifactConfig(archCfg) {
		return *archCfg, true
	}
	return InstallArtifactConfig{}, false
}

func hasArtifactConfig(cfg *InstallArtifactConfig) bool {
	return cfg != nil && strings.TrimSpace(cfg.ArtifactURL) != ""
}

func cloneInstallArtifactConfig(cfg *InstallArtifactConfig) *InstallArtifactConfig {
	if cfg == nil {
		return nil
	}
	clone := *cfg
	if cfg.AllowInsecureURL != nil {
		val := *cfg.AllowInsecureURL
		clone.AllowInsecureURL = &val
	}
	return &clone
}

func cloneInstallPlatformConfig(cfg *InstallPlatformConfig) *InstallPlatformConfig {
	if cfg == nil {
		return nil
	}
	clone := *cfg
	clone.AMD64 = cloneInstallArtifactConfig(cfg.AMD64)
	clone.ARM64 = cloneInstallArtifactConfig(cfg.ARM64)
	return &clone
}

func cloneInstallConfig(cfg InstallConfig) InstallConfig {
	clone := cfg
	clone.Linux = cloneInstallPlatformConfig(cfg.Linux)
	clone.Darwin = cloneInstallPlatformConfig(cfg.Darwin)
	clone.Windows = cloneInstallPlatformConfig(cfg.Windows)
	return clone
}

func (c InstallConfig) AllowInsecureURLForPlatform(goos, goarch string) bool {
	platformCfg := c.PlatformConfig(goos)
	if platformCfg != nil {
		if archCfg := platformCfg.ArchConfig(goarch); archCfg != nil && archCfg.AllowInsecureURL != nil {
			return *archCfg.AllowInsecureURL
		}
	}
	return c.AllowInsecureURL
}

func (c InstallConfig) SSLForPlatform(goos, goarch string) *tlscommon.Config {
	platformCfg := c.PlatformConfig(goos)
	if platformCfg != nil {
		if archCfg := platformCfg.ArchConfig(goarch); archCfg != nil && archCfg.SSL != nil {
			return archCfg.SSL
		}
		if platformCfg.SSL != nil {
			return platformCfg.SSL
		}
	}
	return c.SSL
}

func (c *InstallPlatformConfig) ArchConfig(goarch string) *InstallArtifactConfig {
	if c == nil {
		return nil
	}
	switch goarch {
	case "amd64":
		return c.AMD64
	case "arm64":
		return c.ARM64
	default:
		return nil
	}
}

func (c InstallConfig) NormalizeAndValidate() (InstallConfig, error) {
	normalized := cloneInstallConfig(c)
	platforms := []struct {
		name string
		cfg  *InstallPlatformConfig
	}{
		{name: "linux", cfg: normalized.Linux},
		{name: "darwin", cfg: normalized.Darwin},
		{name: "windows", cfg: normalized.Windows},
	}

	for _, platform := range platforms {
		if platform.cfg == nil {
			continue
		}
		arches := []struct {
			name string
			cfg  *InstallArtifactConfig
		}{
			{name: "amd64", cfg: platform.cfg.AMD64},
			{name: "arm64", cfg: platform.cfg.ARM64},
		}
		for _, arch := range arches {
			if arch.cfg == nil {
				continue
			}
			if err := normalizeAndValidateArtifactConfig(
				arch.cfg,
				fmt.Sprintf("osquery.elastic_options.install.%s.%s", platform.name, arch.name),
				normalized.AllowInsecureURLForPlatform(platform.name, arch.name),
			); err != nil {
				return InstallConfig{}, err
			}
		}
	}

	return normalized, nil
}

func normalizeAndValidateArtifactConfig(cfg *InstallArtifactConfig, configPath string, allowInsecure bool) error {
	cfg.ArtifactURL = strings.TrimSpace(cfg.ArtifactURL)
	cfg.SHA256 = strings.ToLower(strings.TrimSpace(cfg.SHA256))

	if cfg.ArtifactURL == "" && cfg.SHA256 == "" {
		return nil
	}
	if cfg.ArtifactURL == "" {
		return fmt.Errorf("%s.artifact_url is required when sha256 is set", configPath)
	}
	if cfg.SHA256 == "" {
		return fmt.Errorf("%s.sha256 is required when artifact_url is set", configPath)
	}

	hashBytes, err := hex.DecodeString(cfg.SHA256)
	if err != nil || len(hashBytes) != 32 {
		return fmt.Errorf("%s.sha256 must be a valid SHA256 hex string", configPath)
	}

	u, err := url.Parse(cfg.ArtifactURL)
	if err != nil {
		return fmt.Errorf("invalid %s.artifact_url: %w", configPath, err)
	}
	if u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("%s.artifact_url must be an absolute URL", configPath)
	}
	if !allowInsecure && strings.ToLower(u.Scheme) != "https" {
		return fmt.Errorf("%s.artifact_url must use https unless osquery.elastic_options.install.allow_insecure_url is true", configPath)
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

func QueryProfileDatastream(namespace string) string {
	if namespace == "" {
		namespace = DefaultNamespace
	}
	return queryProfileDatastreamPrefix + namespace
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

// GetQueryProfileStorageConfig returns live query profile storage settings from the first input if available.
func GetQueryProfileStorageConfig(inputs []InputConfig) QueryProfileStorageConfig {
	if len(inputs) == 0 {
		return QueryProfileStorageConfig{}
	}
	if inputs[0].Osquery == nil || inputs[0].Osquery.ElasticOptions == nil || inputs[0].Osquery.ElasticOptions.QueryProfileStorage == nil {
		return QueryProfileStorageConfig{}
	}
	return *inputs[0].Osquery.ElasticOptions.QueryProfileStorage
}
