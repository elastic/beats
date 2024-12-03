package converters

import (
	"context"
	"fmt"

	"github.com/elastic/beats/v7/libbeat/outputs/elasticsearch"
	"github.com/elastic/elastic-agent-libs/config"
	"go.opentelemetry.io/collector/confmap"
	"go.uber.org/zap"
)

type converter struct {
	logger *zap.Logger
}

// NewFactory returns a factory for a  confmap.Converter,
// which expands all environment variables for a given confmap.Conf.
func NewFactory() confmap.ConverterFactory {
	return confmap.NewConverterFactory(newConverter)
}

func newConverter(set confmap.ConverterSettings) confmap.Converter {
	return converter{
		logger: set.Logger,
	}
}

// converts beats output config to OTel config ß
func (c converter) Convert(_ context.Context, conf *confmap.Conf) error {
	// var err error
	fmt.Println(conf.ToStringMap())
	var out map[string]any
	receiverCfg, _ := conf.Sub("receivers::filebeatreceiver")
	outputs, _ := receiverCfg.Sub("output")

	for key := range outputs.ToStringMap() {
		switch key {
		case "elasticsearch":
			escfg := config.MustNewConfigFrom(receiverCfg.ToStringMap())
			esCfg, err := elasticsearch.ToOTelConfig(escfg)
			if err != nil {
				return fmt.Errorf("cannot convert Filebeat config: %w", err)
			}
			out = map[string]any{
				"service::pipelines::logs::exporters": []string{"elasticsearch"},
				"exporters": map[string]any{
					"elasticsearch": esCfg,
				},
			}
			conf.Merge(confmap.NewFromStringMap(out))
		case "kafka":
			return fmt.Errorf("unsupported output")
		case "default":
		}
	}

	out = map[string]any{
		"receivers::filebeatreceiver::output": nil,
	}
	conf.Merge(confmap.NewFromStringMap(out))
	out = map[string]any{
		"receivers::filebeatreceiver::output::otelconsumer": nil,
	}

	return conf.Merge(confmap.NewFromStringMap(out))
}
