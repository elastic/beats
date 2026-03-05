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
	ArtifactURL      string            `config:"artifact_url"`
	SHA256           string            `config:"sha256"`
	InstallDir       string            `config:"install_dir"`
	AllowInsecureURL bool              `config:"allow_insecure_url"`
	SSL              *tlscommon.Config `config:"ssl"`
}

func (c InstallConfig) Enabled() bool {
	return strings.TrimSpace(c.ArtifactURL) != ""
}

func (c *InstallConfig) Validate() error {
	c.ArtifactURL = strings.TrimSpace(c.ArtifactURL)
	c.SHA256 = strings.ToLower(strings.TrimSpace(c.SHA256))
	c.InstallDir = strings.TrimSpace(c.InstallDir)

	if !c.Enabled() {
		return nil
	}

	if c.InstallDir != "" {
		return fmt.Errorf("osquery.elastic_options.install.install_dir is not supported; custom osquery install path is fixed to bundled osquery directory")
	}

	if c.SHA256 == "" {
		return fmt.Errorf("osquery.elastic_options.install.sha256 is required when osquery.elastic_options.install.artifact_url is set")
	}

	hashBytes, err := hex.DecodeString(c.SHA256)
	if err != nil || len(hashBytes) != 32 {
		return fmt.Errorf("osquery.elastic_options.install.sha256 must be a valid SHA256 hex string")
	}

	u, err := url.Parse(c.ArtifactURL)
	if err != nil {
		return fmt.Errorf("invalid osquery.elastic_options.install.artifact_url: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("osquery.elastic_options.install.artifact_url must be an absolute URL")
	}
	if !c.AllowInsecureURL && strings.ToLower(u.Scheme) != "https" {
		return fmt.Errorf("osquery.elastic_options.install.artifact_url must use https unless osquery.elastic_options.install.allow_insecure_url is true")
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
