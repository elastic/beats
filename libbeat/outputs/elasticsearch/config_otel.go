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

	"go.opentelemetry.io/collector/config/configopaque"

	"github.com/elastic/beats/v7/libbeat/cloudid"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/elastic-agent-libs/config"
)

// toOTelConfig converts a Beat config into an OTel elasticsearch exporter config
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

	hosts := []string{}
	for _, h := range esToOTelOptions.Hosts {
		esURL, err := common.MakeURL(escfg.Protocol, escfg.Path, h, 9200)
		if err != nil {
			return nil, fmt.Errorf("cannot generate ES URL from host %q", err)
		}
		hosts = append(hosts, esURL)
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

	otelTLSConfg, err := outputs.TLSCommonToOTel(escfg.Transport.TLS)
	if err != nil {
		return nil, fmt.Errorf("cannot convert SSL config into OTel: %w", err)
	}

	otelYAMLCfg := map[string]any{
		"logs_index":  esToOTelOptions.Index, // index
		"index":       esToOTelOptions.Index,
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
		"tls":               otelTLSConfg,                     //TODO: convert it to map[string]any

		// Retry
		"retry": map[string]any{
			"enabled":          true,
			"initial_interval": escfg.Backoff.Init, // backoff.init
			"max_interval":     escfg.Backoff.Max,  // backoff.max
		},

		// Batcher
		"batcher": map[string]any{
			"enabled":        true,
			"max_size_items": escfg.BulkMaxSize, // bulk_max_size
		},

		// TODO (Tiago): Trying to make things work, remove later
		"mapping": map[string]any{
			"mode": "ecs",
		},
	}

	return otelYAMLCfg, nil
}
