// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fbprovider

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"go.opentelemetry.io/collector/confmap"

	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/outputs/elasticsearch"
	"github.com/elastic/elastic-agent-libs/config"
)

const schemeName = "file"

type provider struct{}

func NewFactory() confmap.ProviderFactory {
	return confmap.NewProviderFactory(newProvider)
}

func newProvider(confmap.ProviderSettings) confmap.Provider {
	return &provider{}
}

func (fmp *provider) Retrieve(_ context.Context, uri string, _ confmap.WatcherFunc) (*confmap.Retrieved, error) {
	if !strings.HasPrefix(uri, schemeName+":") {
		return nil, fmt.Errorf("%q uri is not supported by %q provider", uri, schemeName)
	}

	cfg, err := cfgfile.Load(filepath.Clean(uri[len(schemeName)+1:]), nil)
	if err != nil {
		return nil, err
	}

	esCfg, err := elasticsearch.ToOTelConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("cannot convert Filebeat config: %w", err)
	}

	// We need to edit the output settings from Filebeat:
	// first we create a new config with a single key `otelconsumer`
	// which is an empty map, then we replace the `output` by this
	// new config.
	//
	// Effectively we're replacing `output.elasticsearch` by
	// `output.otelconsumer`.
	otelConsumerCfg := config.NewConfig()
	otelConsumerCfg.SetChild("otelconsumer", -1, config.MustNewConfigFrom(map[string]any{}))
	cfg.SetChild("output", -1, otelConsumerCfg)

	var receiverMap map[string]any
	cfg.Unpack(&receiverMap)

	cfgMap := map[string]any{
		"exporters": map[string]any{
			"elasticsearch": esCfg,
			"debug":         map[string]any{},
		},
		"receivers": map[string]any{
			"filebeatreceiver": receiverMap,
		},
		"service": map[string]any{
			"pipelines": map[string]any{
				"logs": map[string]any{
					"exporters": []string{
						"debug",
						"elasticsearch",
					},
					"receivers": []string{"filebeatreceiver"},
				},
			},
		},
	}

	// TODO: Remove this debug statement
	s, _ := json.MarshalIndent(cfgMap, "", " ")
	fmt.Println(string(s))

	return confmap.NewRetrieved(cfgMap)
}

func (*provider) Scheme() string {
	return schemeName
}

func (*provider) Shutdown(context.Context) error {
	return nil
}
