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

package elasticsearch

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/elasticsearchexporter"
	"go.opentelemetry.io/collector/config/configopaque"

	"github.com/elastic/beats/v7/libbeat/cloudid"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/transport/kerberos"
	oteltranslate "github.com/elastic/beats/v7/libbeat/otelbeat/oteltranslate"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/elastic-agent-libs/config"
)

// TODO: add  following unuspported params to below struct
// indices
// pipelines
// parameters
// preset
// setup.ilm.* -> supported but the logic is not in place yet
// proxy_disable -> supported but the logic is not in place yet
// proxy_headers
type unsupportedConfig struct {
	CompressionLevel   int               `config:"compression_level" `
	LoadBalance        bool              `config:"loadbalance"`
	NonIndexablePolicy *config.Namespace `config:"non_indexable_policy"`
	AllowOlderVersion  bool              `config:"allow_older_versions"`
	EscapeHTML         bool              `config:"escape_html"`
	Kerberos           *kerberos.Config  `config:"kerberos"`
	MaxRetries         int               `config:"max_retries"`
	BulkMaxSize        int               `config:"bulk_max_size"`
}

// ToOTelConfig converts a Beat config into an OTel elasticsearch exporter config
func ToOTelConfig(beatCfg *config.C) (map[string]any, error) {
	// Handle cloud.id the same way Beats does, this will also handle
	// extracting the Kibana URL (which is required to handle ILM on
	// Beats side (currently not supported by ES OTel exporter).
	if err := cloudid.OverwriteSettings(beatCfg); err != nil {
		return nil, fmt.Errorf("cannot read cloudid: %w", err)
	}

	esRawCfg, err := beatCfg.Child("output.elasticsearch", -1)
	if err != nil {
		return nil, fmt.Errorf("could not parse Elasticsearch output configuration: %w", err)
	}
	escfg := defaultConfig

	// check if unsupported configuration is provided
	temp := unsupportedConfig{}
	if err := esRawCfg.Unpack(&temp); err != nil {
		return nil, err
	}
	if temp != (unsupportedConfig{}) {
		return nil, fmt.Errorf("these configuration parameters are not supported %+v", temp)
	}

	// unpack and validate ES config
	if err := esRawCfg.Unpack(&escfg); err != nil {
		return nil, fmt.Errorf("failed unpacking config. %w", err)
	}

	if err := escfg.Validate(); err != nil {
		return nil, err
	}

	esToOTelOptions := struct {
		Index    string   `config:"index"`
		Pipeline string   `config:"pipeline"`
		ProxyURL string   `config:"proxy_url"`
		Hosts    []string `config:"hosts"  validate:"required"`
	}{}

	if err := esRawCfg.Unpack(&esToOTelOptions); err != nil {
		return nil, fmt.Errorf("cannot parse Elasticsearch config: %w", err)
	}

	hosts := []string{}
	for _, h := range esToOTelOptions.Hosts {
		esURL, err := common.MakeURL(escfg.Protocol, escfg.Path, h, 9200)
		if err != nil {
			return nil, fmt.Errorf("cannot generate ES URL from host %w", err)
		}
		hosts = append(hosts, esURL)
	}

	// The workers config  can be configured using two keys, so we leverage
	// the already existing code to handle it by using `output.HostWorkerCfg`.
	workersCfg := outputs.HostWorkerCfg{}
	if err := esRawCfg.Unpack(&workersCfg); err != nil {
		return nil, fmt.Errorf("cannot read worker/workers from Elasticsearch config: %w", err)
	}

	headers := make(map[string]configopaque.String, len(escfg.Headers))
	for k, v := range escfg.Headers {
		headers[k] = configopaque.String(v)
	}

	otelTLSConfg, err := oteltranslate.TLSCommonToOTel(escfg.Transport.TLS)
	if err != nil {
		return nil, fmt.Errorf("cannot convert SSL config into OTel: %w", err)
	}

	otelYAMLCfg := map[string]any{
		"logs_index":  esToOTelOptions.Index,    // index
		"pipeline":    esToOTelOptions.Pipeline, // pipeline
		"endpoints":   hosts,                    // hosts, protocol, path, port
		"num_workers": workersCfg.NumWorkers(),  // worker/workers

		// Authentication
		"user":     escfg.Username, // username
		"password": escfg.Password, // password
		"api_key":  escfg.APIKey,   // api_key

		// ClientConfig
		"proxy_url":         esToOTelOptions.ProxyURL,         // proxy_url
		"headers":           headers,                          // headers
		"timeout":           escfg.Transport.Timeout,          // timeout
		"idle_conn_timeout": &escfg.Transport.IdleConnTimeout, // idle_connection_connection_timeout
		"tls":               otelTLSConfg,                     // tls config

		// Retry
		"retry": map[string]any{
			"enabled":          true,
			"initial_interval": escfg.Backoff.Init, // backoff.init
			"max_interval":     escfg.Backoff.Max,  // backoff.max
		},

		// Batcher is experimental and by not setting it, we are using the exporter's default batching mechanism
		// "batcher": map[string]any{
		// 	"enabled":        true,
		// 	"max_size_items": escfg.BulkMaxSize, // bulk_max_size
		// },
	}

	// For type safety check only
	// the returned valued should match `elasticsearchexporter.Config` type.
	// it throws an error if non existing key names  are set
	var result elasticsearchexporter.Config
	d, _ := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Squash:      true,
		Result:      &result,
		ErrorUnused: true,
	})

	err = d.Decode(otelYAMLCfg)
	if err != nil {
		return nil, err
	}

	// TODO:
	// // validates all required fields are set
	// err = result.Validate()
	// if err != nil {
	// 	return nil, err
	// }

	return otelYAMLCfg, nil
}
