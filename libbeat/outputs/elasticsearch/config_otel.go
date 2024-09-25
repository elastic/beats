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

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/elasticsearchexporter"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/exporter/exporterbatcher"

	"github.com/elastic/beats/v7/libbeat/cloudid"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/elastic-agent-libs/config"
)

// ToOtelConfig converts a Beat config into an OTel elasticsearch exporter config
func ToOtelConfig(beatCfg *config.C) (*elasticsearchexporter.Config, error) {
	// Handle cloud.id the same way Beats does, this will also handle
	// extracting the Kibana URL (which is required to handle ILM on
	// Beats side (currently not supported by ES OTel exporter).
	if err := cloudid.OverwriteSettings(beatCfg); err != nil {
		return nil, fmt.Errorf("cannot read cloudid: %w", err)
	}

	esRawCfg, err := beatCfg.Child("output.elasticsearch", -1)
	if err != nil {
		panic(err)
	}
	escfg := defaultConfig
	if err := esRawCfg.Unpack(&escfg); err != nil {
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

	// The workers config is can be configured using two keys, so we leverage
	// the already existing code to handle it by using `output.HostWorkerCfg`.
	workersCfg := outputs.HostWorkerCfg{}
	if err := esRawCfg.Unpack(&workersCfg); err != nil {
		return nil, fmt.Errorf("cannot read worker/workers from Elasticsearch config: %w", err)
	}

	headers := make(map[string]configopaque.String, len(escfg.Headers))
	for k, v := range escfg.Headers {
		headers[k] = configopaque.String(v)
	}

	otelTLSConfg, err := outputs.TLSCommonToOtel(escfg.Transport.TLS)
	if err != nil {
		return nil, fmt.Errorf("cannot convert SSL config into OTel: %w", err)
	}

	otelcfg := elasticsearchexporter.Config{
		Index:      esToOTelOptions.Index,    // index
		Pipeline:   esToOTelOptions.Pipeline, // pipeline
		Endpoints:  esToOTelOptions.Hosts,    // hosts, protocol, path, port
		NumWorkers: workersCfg.NumWorkers(),  // worker/workers

		Authentication: elasticsearchexporter.AuthenticationSettings{
			User:     escfg.Username,                      // username
			Password: configopaque.String(escfg.Password), // password
			APIKey:   configopaque.String(escfg.APIKey),   //api_key
		},

		// HTTP Client configuration
		ClientConfig: confighttp.ClientConfig{
			ProxyURL:        esToOTelOptions.ProxyURL,         // proxy_url
			Headers:         headers,                          // headers
			Timeout:         escfg.Transport.Timeout,          // timeout
			IdleConnTimeout: &escfg.Transport.IdleConnTimeout, // idle_connection_connection_timeout
			TLSSetting:      otelTLSConfg,
		},

		// Backoff settings
		Retry: elasticsearchexporter.RetrySettings{
			Enabled:         true,
			InitialInterval: escfg.Backoff.Init, // backoff.init
			MaxInterval:     escfg.Backoff.Max,  // backoff.max
		},

		// Batching configuration
		Batcher: elasticsearchexporter.BatcherConfig{
			MaxSizeConfig: exporterbatcher.MaxSizeConfig{
				MaxSizeItems: escfg.BulkMaxSize, // bulk_max_size
			},
		},
	}

	return &otelcfg, nil
}
