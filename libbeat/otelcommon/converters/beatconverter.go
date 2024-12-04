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

// list of supported beatreceivers
var supportedReceivers = []string{"filebeatreceiver"} // Add more beat receivers to this list when we add support

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

	for _, receiverbeat := range supportedReceivers {
		var out map[string]any

		// check if supported beat receiver is configured. Skip translation logic if not
		if v := conf.Get("receivers::" + receiverbeat); v == nil {
			continue
		}

		receiverCfg, _ := conf.Sub("receivers::" + receiverbeat)
		outputs, _ := receiverCfg.Sub("output")

		for key := range outputs.ToStringMap() {
			switch key {
			case "elasticsearch":
				escfg := config.MustNewConfigFrom(receiverCfg.ToStringMap())
				esCfg, err := elasticsearch.ToOTelConfig(escfg)
				if err != nil {
					return fmt.Errorf("cannot convert elasticsearch config: %w", err)
				}
				out = map[string]any{
					"service::pipelines::logs::exporters": []string{"elasticsearch"},
					"exporters": map[string]any{
						"elasticsearch": esCfg,
					},
				}
				err = conf.Merge(confmap.NewFromStringMap(out))
				if err != nil {
					return err
				}
			case "kafka", "logstash":
				return fmt.Errorf("%s is currently unsupported in otel mode", key)
			default:
				return fmt.Errorf("%s is unsupported output", key)
			}
		}

		// Replace output.elasticsearch with output.otelconsumer
		out = map[string]any{
			"receivers::" + receiverbeat + "::output": nil,
		}
		err := conf.Merge(confmap.NewFromStringMap(out))
		if err != nil {
			return err
		}
		out = map[string]any{
			"receivers::" + receiverbeat + "::output::otelconsumer": nil,
		}
		err = conf.Merge(confmap.NewFromStringMap(out))
		if err != nil {
			return err
		}
	}

	return nil
}
