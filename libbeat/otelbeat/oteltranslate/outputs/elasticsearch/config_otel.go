// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package elasticsearchtranslate

import (
	"encoding/base64"
	"fmt"
	"reflect"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/elasticsearchexporter"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/transport/kerberos"
	oteltranslate "github.com/elastic/beats/v7/libbeat/otelbeat/oteltranslate"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/elasticsearch"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

// setup.ilm.* -> supported but the logic is not in place yet
type unsupportedConfig struct {
	CompressionLevel   int               `config:"compression_level" `
	LoadBalance        bool              `config:"loadbalance"`
	NonIndexablePolicy *config.Namespace `config:"non_indexable_policy"`
	EscapeHTML         bool              `config:"escape_html"`
	Kerberos           *kerberos.Config  `config:"kerberos"`
	ProxyDisable       bool              `config:"proxy_disable"`
}

type esToOTelOptions struct {
	elasticsearch.ElasticsearchConfig `config:",inline"`
	outputs.HostWorkerCfg             `config:",inline"`

	Index    string `config:"index"`
	Pipeline string `config:"pipeline"`
	ProxyURL string `config:"proxy_url"`
	Preset   string `config:"preset"`
}

var defaultOptions = esToOTelOptions{
	ElasticsearchConfig: elasticsearch.DefaultConfig(),

	Index:    "", // Dynamic routing is disabled if index is set
	Pipeline: "",
	ProxyURL: "",
	Preset:   "custom", // default is custom if not set
}

// ToOTelConfig converts a Beat config into OTel elasticsearch exporter config
// Ensure cloudid is handled before calling this method
// Note: This method may override output queue settings defined by user.
func ToOTelConfig(output *config.C) (map[string]any, error) {
	escfg := defaultOptions

	// check for unsupported config
	err := unSupportedConfig(output)
	if err != nil {
		return nil, err
	}

	// apply preset here
	// It is important to apply preset before unpacking the config, as preset can override output fields
	preset, err := output.String("preset", -1)
	if err == nil {
		// Performance preset is present, apply it and log any fields that
		// were overridden
		overriddenFields, presetConfig, err := elasticsearch.ApplyPreset(preset, output)
		if err != nil {
			return nil, err
		}
		logp.Info("Applying performance preset '%v': %v",
			preset, config.DebugString(presetConfig, false))
		logp.Warn("Performance preset '%v' overrides user setting for field(s): %s",
			preset, strings.Join(overriddenFields, ","))
	}

	// unpack and validate ES config
	if err := output.Unpack(&escfg); err != nil {
		return nil, fmt.Errorf("failed unpacking config. %w", err)
	}

	if err := escfg.Validate(); err != nil {
		return nil, err
	}

	// Create url using host name, protocol and path
	hosts := []string{}
	for _, h := range escfg.Hosts {
		esURL, err := common.MakeURL(escfg.Protocol, escfg.Path, h, 9200)
		if err != nil {
			return nil, fmt.Errorf("cannot generate ES URL from host %w", err)
		}
		hosts = append(hosts, esURL)
	}

	// convert ssl configuration
	otelTLSConfg, err := oteltranslate.TLSCommonToOTel(escfg.Transport.TLS)
	if err != nil {
		return nil, fmt.Errorf("cannot convert SSL config into OTel: %w", err)
	}

	otelYAMLCfg := map[string]any{
		"endpoints": hosts, // hosts, protocol, path, port

		// ClientConfig
		"timeout":           escfg.Transport.Timeout,         // timeout
		"idle_conn_timeout": escfg.Transport.IdleConnTimeout, // idle_connection_timeout

		// Retry
		"retry": map[string]any{
			"enabled":          true,
			"initial_interval": escfg.Backoff.Init, // backoff.init
			"max_interval":     escfg.Backoff.Max,  // backoff.max
			"max_retries":      escfg.MaxRetries,   // max_retries

		},

		// Batcher is experimental
		"batcher": map[string]any{
			"enabled":  true,
			"max_size": escfg.BulkMaxSize, // bulk_max_size
			"min_size": 0,                 // 0 means immediately trigger a flush
		},

		"mapping": map[string]any{
			"mode": "bodymap",
		},
	}

	// Authentication
	setIfNotNil(otelYAMLCfg, "user", escfg.Username)                                             // username
	setIfNotNil(otelYAMLCfg, "password", escfg.Password)                                         // password
	setIfNotNil(otelYAMLCfg, "api_key", base64.StdEncoding.EncodeToString([]byte(escfg.APIKey))) // api_key

	setIfNotNil(otelYAMLCfg, "headers", escfg.Headers)    // headers
	setIfNotNil(otelYAMLCfg, "tls", otelTLSConfg)         // tls config
	setIfNotNil(otelYAMLCfg, "proxy_url", escfg.ProxyURL) // proxy_url
	setIfNotNil(otelYAMLCfg, "pipeline", escfg.Pipeline)  // pipeline
	// Dynamic routing is disabled if output.elasticsearch.index is set
	setIfNotNil(otelYAMLCfg, "logs_index", escfg.Index) // index

	if err := typeSafetyCheck(otelYAMLCfg); err != nil {
		return nil, err
	}

	return otelYAMLCfg, nil
}

// log warning for unsupported config
func unSupportedConfig(cfg *config.C) error {
	// check if unsupported configuration is provided
	temp := unsupportedConfig{}
	if err := cfg.Unpack(&temp); err != nil {
		return err
	}

	if !isStructEmpty(temp) {
		logp.Warn("these configuration parameters are not supported %+v", temp)
		return nil
	}

	// check for dictionary like parameters that we do not support yet
	if cfg.HasField("indices") {
		logp.Warn("indices is currently not supported")
	} else if cfg.HasField("pipelines") {
		logp.Warn("pipelines is currently not supported")
	} else if cfg.HasField("parameters") {
		logp.Warn("parameters is currently not supported")
	} else if cfg.HasField("proxy_headers") {
		logp.Warn("proxy_headers is currently not supported")
	} else if value, _ := cfg.Bool("allow_older_versions", -1); !value {
		logp.Warn("allow_older_versions:false is currently not supported")
	}

	return nil
}

// For type safety check
func typeSafetyCheck(value map[string]any) error {
	// the  value should match `elasticsearchexporter.Config` type.
	// it throws an error if non existing key names  are set
	var result elasticsearchexporter.Config
	d, _ := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Squash:      true,
		Result:      &result,
		ErrorUnused: true,
	})

	err := d.Decode(value)
	if err != nil {
		return err
	}
	return err
}

// Helper function to check if a struct is empty
func isStructEmpty(s any) bool {
	return reflect.DeepEqual(s, reflect.Zero(reflect.TypeOf(s)).Interface())
}

// Helper function to conditionally add fields to the map
func setIfNotNil(m map[string]any, key string, value any) {
	if value == nil {
		return
	}

	v := reflect.ValueOf(value)

	switch v.Kind() {
	case reflect.String:
		if v.String() != "" {
			m[key] = value
		}
	case reflect.Map, reflect.Slice:
		if v.Len() > 0 {
			m[key] = value
		}
	case reflect.Struct:
		if !isStructEmpty(value) {
			m[key] = value
		}
	default:
		m[key] = value
	}
}
