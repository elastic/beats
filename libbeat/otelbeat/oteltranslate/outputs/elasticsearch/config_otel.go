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

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/transport/kerberos"
	oteltranslate "github.com/elastic/beats/v7/libbeat/otelbeat/oteltranslate"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/elasticsearch"
	"github.com/elastic/beats/v7/libbeat/publisher/queue/memqueue"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

// TODO: add  following unuspported params to below struct
// indices
// pipelines
// parameters
// setup.ilm.* -> supported but the logic is not in place yet
// proxy_disable -> supported but the logic is not in place yet
// proxy_headers
type unsupportedConfig struct {
	LoadBalance        bool              `config:"loadbalance"`
	NonIndexablePolicy *config.Namespace `config:"non_indexable_policy"`
	AllowOlderVersion  bool              `config:"allow_older_versions"`
	EscapeHTML         bool              `config:"escape_html"`
	Kerberos           *kerberos.Config  `config:"kerberos"`
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

	Index:    "filebeat-9.0.0", // TODO. Default value should be filebeat-%{[agent.version]}
	Pipeline: "",
	ProxyURL: "",
	Preset:   "custom", // default is custom if not set
	HostWorkerCfg: outputs.HostWorkerCfg{
		Workers: 1,
	},
}

// ToOTelConfig converts a Beat config into OTel elasticsearch exporter config
// Ensure cloudid is handled before calling this method
// Note: This method may override output queue settings defined by user.
func ToOTelConfig(output *config.C) (map[string]any, error) {
	escfg := defaultOptions
	// check if unsupported configuration is provided
	temp := unsupportedConfig{}
	if err := output.Unpack(&temp); err != nil {
		return nil, err
	}
	if !isStructEmpty(temp) {
		return nil, fmt.Errorf("these configuration parameters are not supported %+v", temp)
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

	// get number of workers
	workers := 1 // Default value is 1
	if escfg.NumWorkers() > 1 {
		workers = escfg.NumWorkers()
	}
	otelYAMLCfg := map[string]any{
		"logs_index":  escfg.Index, // index
		"endpoints":   hosts,       // hosts, protocol, path, port
		"num_workers": workers,     // worker/workers

		// ClientConfig
		"timeout":           escfg.Transport.Timeout,         // timeout
		"idle_conn_timeout": escfg.Transport.IdleConnTimeout, // idle_connection_timeout

		// max_conns_per_host is a "hard" limit on number of open connections.
		// Ideally, escfg.NumWorkers() should map to num_consumer, but we had a bug in upstream
		// where it could spin as many goroutines as it liked.
		// Given that batcher implementation can change and it has a history of such changes,
		// let's keep max_conns_per_host setting for now and remove it once exporterhelper is stable.
		"max_conns_per_host": escfg.NumWorkers(),

		// Retry
		"retry": map[string]any{
			"enabled":          true,
			"initial_interval": escfg.Backoff.Init, // backoff.init
			"max_interval":     escfg.Backoff.Max,  // backoff.max
			"max_retries":      escfg.MaxRetries,   // max_retries
		},

		"sending_queue": map[string]any{
			"batch": map[string]any{
				"max_size": escfg.BulkMaxSize, // bulk_max_size
				"min_size": 0,                 // 0 means immediately trigger a flush
				"sizer":    "items",
			},
			"enabled":           true,
			"queue_size":        getQueueSize(logger, output),
			"block_on_overflow": true,
			"wait_for_result":   true,
			"num_consumers":     escfg.NumWorkers(),
		},

		"mapping": map[string]any{
			"mode": "bodymap",
		},
	}

	// Compression
	otelYAMLCfg["compression"] = "none"
	if escfg.CompressionLevel > 0 {
		otelYAMLCfg["compression"] = "gzip"
		otelYAMLCfg["compression_params"] = map[string]any{
			"level": escfg.CompressionLevel,
		}
	}

	// Authentication
	setIfNotNil(otelYAMLCfg, "user", escfg.Username)                                             // username
	setIfNotNil(otelYAMLCfg, "password", escfg.Password)                                         // password
	setIfNotNil(otelYAMLCfg, "api_key", base64.StdEncoding.EncodeToString([]byte(escfg.APIKey))) // api_key

	setIfNotNil(otelYAMLCfg, "headers", escfg.Headers)    // headers
	setIfNotNil(otelYAMLCfg, "tls", otelTLSConfg)         // tls config
	setIfNotNil(otelYAMLCfg, "proxy_url", escfg.ProxyURL) // proxy_url
	setIfNotNil(otelYAMLCfg, "pipeline", escfg.Pipeline)  // pipeline

	return otelYAMLCfg, nil
}

<<<<<<< HEAD
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
=======
// log warning for unsupported config
func checkUnsupportedConfig(cfg *config.C, logger *logp.Logger) error {
	// check if unsupported configuration is provided
	temp := unsupportedConfig{}
	if err := cfg.Unpack(&temp); err != nil {
		return err
	}

	if !isStructEmpty(temp) {
		return fmt.Errorf("these configuration parameters are not supported %+v: %w", temp, errors.ErrUnsupported)
	}

	// check for dictionary like parameters that we do not support yet
	if cfg.HasField("indices") {
		return fmt.Errorf("indices is currently not supported: %w", errors.ErrUnsupported)
	} else if cfg.HasField("pipelines") {
		return fmt.Errorf("pipelines is currently not supported: %w", errors.ErrUnsupported)
	} else if cfg.HasField("parameters") {
		return fmt.Errorf("parameters is currently not supported: %w", errors.ErrUnsupported)
	} else if cfg.HasField("proxy_headers") {
		return fmt.Errorf("proxy_headers is currently not supported: %w", errors.ErrUnsupported)
	} else if value, err := cfg.Bool("allow_older_versions", -1); err == nil && !value {
		return fmt.Errorf("allow_older_versions:false is currently not supported: %w", errors.ErrUnsupported)
	}
	return nil
>>>>>>> 02b596f4a ([beatreceiver] - Disable `force_attempt_http2` and map ES config to elasticsearchexporter (#46111))
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

func getQueueSize(logger *logp.Logger, output *config.C) int {
	size, err := output.Int("queue.mem.events", -1)
	if err != nil {
		logger.Debugf("Failed to get queue size: %v", err)
		return memqueue.DefaultEvents // return default queue.mem.events for sending_queue in case of an errr
	}
	return int(size)
}
