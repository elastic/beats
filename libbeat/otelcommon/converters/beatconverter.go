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

// [beatreceiver].output is unpacked to OTel config here
func (c converter) Convert(_ context.Context, conf *confmap.Conf) error {
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
			return fmt.Errorf("%s is currently unsupported in otel mode", key)
		case "default":
			return fmt.Errorf("%s is unsupported output", key)
		}
	}

	// Replace output.elasticsearch with output.otelconsumer
	out = map[string]any{
		"receivers::filebeatreceiver::output": nil,
	}
	conf.Merge(confmap.NewFromStringMap(out))
	out = map[string]any{
		"receivers::filebeatreceiver::output::otelconsumer": nil,
	}

	return conf.Merge(confmap.NewFromStringMap(out))
}
