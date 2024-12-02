package otelcommon

import (
	"github.com/elastic/beats/v7/x-pack/filebeat/fbreceiver"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/elasticsearchexporter"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/debugexporter"
	"go.opentelemetry.io/collector/otelcol"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/batchprocessor"
	"go.opentelemetry.io/collector/processor/memorylimiterprocessor"
	"go.opentelemetry.io/collector/receiver"
)

// Component initializes collector components
func Component() (otelcol.Factories, error) {
	receivers, err := receiver.MakeFactoryMap(
		fbreceiver.NewFactory(),
	)
	if err != nil {
		return otelcol.Factories{}, nil
	}

	exporters, err := exporter.MakeFactoryMap(
		debugexporter.NewFactory(),
		elasticsearchexporter.NewFactory(),
	)
	if err != nil {
		return otelcol.Factories{}, nil
	}

	processors, err := processor.MakeFactoryMap(
		batchprocessor.NewFactory(),
		memorylimiterprocessor.NewFactory(),
	)
	if err != nil {
		return otelcol.Factories{}, nil
	}

	return otelcol.Factories{
		Receivers:  receivers,
		Exporters:  exporters,
		Processors: processors,
	}, nil

}
