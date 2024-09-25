package elasticsearch

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/cloudid"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/elasticsearchexporter"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/exporter/exporterbatcher"
)

// ToOtelConfig converts a Beat config into an OTel elasticsearch exporter config
func ToOtelConfig(beatCfg *config.C) (*elasticsearchexporter.Config, error) {
	// Handle cloud.id
	if err := cloudid.OverwriteSettings(beatCfg); err != nil {
		return nil, fmt.Errorf("cannot read cloudid: %w", err)
	}

	esRawCfg, err := beatCfg.Child("output.elasticsearch", -1)
	if err != nil {
		panic(err)
	}
	esCfg := defaultConfig
	if err := esRawCfg.Unpack(&esCfg); err != nil {
		return nil, err
	}

	esToOTelOptions := struct {
		Index    string `config:"index"`
		Pipeline string `config:"pipeline"`
		ProxyURL string `config:"proxy_url"`
	}{}

	if err := esRawCfg.Unpack(&esToOTelOptions); err != nil {
		return nil, fmt.Errorf("cannot parse Elasticsearch config: %w", err)

	}

	// The ES output by default builds a list containing one entry of the
	// `hosts` list for each worker. Here we only read the original
	// `hosts` list and pass it to the OTel configuration
	hostsConfig := struct {
		Hosts []string `config:"hosts"  validate:"required"`
	}{}
	if err := esRawCfg.Unpack(&hostsConfig); err != nil {
		return nil, fmt.Errorf("cannot read 'hosts' from Elasticsearch config: %w", err)
	}

	workersCfg := outputs.HostWorkerCfg{}
	if err := esRawCfg.Unpack(&workersCfg); err != nil {
		return nil, fmt.Errorf("cannot read worker/workers from Elasticsearch config: %w", err)
	}

	headers := make(map[string]configopaque.String, len(esCfg.Headers))
	for k, v := range esCfg.Headers {
		headers[k] = configopaque.String(v)
	}

	otelcfg := elasticsearchexporter.Config{
		Authentication: elasticsearchexporter.AuthenticationSettings{
			User:     esCfg.Username,
			Password: configopaque.String(esCfg.Password),
			APIKey:   configopaque.String(esCfg.APIKey),
		},
		Index:      esToOTelOptions.Index,
		Pipeline:   esToOTelOptions.Pipeline,
		Endpoints:  hostsConfig.Hosts,
		NumWorkers: workersCfg.NumWorkers(),

		// HTTP Client configuration
		ClientConfig: confighttp.ClientConfig{
			ProxyURL: esToOTelOptions.ProxyURL,
			Headers:  headers,
		},

		// Backoff settings
		Retry: elasticsearchexporter.RetrySettings{
			Enabled:         true,
			InitialInterval: esCfg.Backoff.Init,
			MaxInterval:     esCfg.Backoff.Max,
		},

		// Batching configuration
		Batcher: elasticsearchexporter.BatcherConfig{
			MaxSizeConfig: exporterbatcher.MaxSizeConfig{
				MaxSizeItems: esCfg.BulkMaxSize,
			},
		},
	}

	otelcfg.Endpoints = hostsConfig.Hosts

	return &otelcfg, nil
}
