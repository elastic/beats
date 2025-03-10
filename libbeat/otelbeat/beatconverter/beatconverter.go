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

package beatconverter

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/confmap"

	"github.com/elastic/beats/v7/libbeat/cloudid"
	elasticsearchtranslate "github.com/elastic/beats/v7/libbeat/otelbeat/oteltranslate/outputs/elasticsearch"
	"github.com/elastic/elastic-agent-libs/config"
)

// list of supported beatreceivers
var supportedReceivers = []string{"filebeatreceiver"} // Add more beat receivers to this list when we add support

type converter struct{}

// NewFactory returns a factory for a  confmap.Converter,
func NewFactory() confmap.ConverterFactory {
	return confmap.NewConverterFactory(newConverter)
}

func newConverter(set confmap.ConverterSettings) confmap.Converter {
	return converter{}
}

// Convert converts [beatreceiver].output to OTel config here
func (c converter) Convert(_ context.Context, conf *confmap.Conf) error {

	for _, beatreceiver := range supportedReceivers {
		var out map[string]any

		var beatReceiverConfigKey = "receivers::" + beatreceiver
		// check if supported beat receiver is configured. Skip translation logic if not
		if v := conf.Get(beatReceiverConfigKey); v == nil {
			continue
		}

		// handle cloud id if set
		if conf.IsSet(beatReceiverConfigKey + "::cloud") {
			if err := handleCloudId(beatreceiver, conf); err != nil {
				return fmt.Errorf("error handling cloud id %w", err)
			}
		}

		receiverCfg, _ := conf.Sub(beatReceiverConfigKey)
		output, _ := receiverCfg.Sub("output")

		if len(output.ToStringMap()) > 1 {
			return fmt.Errorf("multiple outputs are not supported")
		}

		for key, output := range output.ToStringMap() {
			switch key {
			case "elasticsearch":
				esConfig := config.MustNewConfigFrom(output)
				esOTelConfig, err := elasticsearchtranslate.ToOTelConfig(esConfig)
				if err != nil {
					return fmt.Errorf("cannot convert elasticsearch config: %w", err)
				}

				// when output.queue is set by user or it comes from "preset" config, promote it to global level
				if ok := esConfig.HasField("queue"); ok {
					if err := promoteOutputQueueSettings(beatreceiver, esConfig, conf); err != nil {
						return err
					}
				}

				out = map[string]any{
					"service::pipelines::logs::exporters": []string{"elasticsearch"},
					"exporters": map[string]any{
						"elasticsearch": esOTelConfig,
					},
				}
				err = conf.Merge(confmap.NewFromStringMap(out))
				if err != nil {
					return err
				}
			default:
				return fmt.Errorf("output type %q is unsupported in OTel mode", key)
			}
		}

		// Replace output.[configured-output] with output.otelconsumer
		out = map[string]any{
			beatReceiverConfigKey + "::output": nil,
		}
		err := conf.Merge(confmap.NewFromStringMap(out))
		if err != nil {
			return err
		}
		out = map[string]any{
			beatReceiverConfigKey + "::output::otelconsumer": nil,
		}

		// inject log level
		receiverConfig, err := config.NewConfigFrom(receiverCfg.ToStringMap())
		if err != nil {
			return fmt.Errorf("Error getting receiver config: %w", err)
		}

		if level, _ := receiverConfig.String("logging.level", -1); level != "" {
			out["service::telemetry::logs::level"], err = getOTelLogLevel(level)
			if err != nil {
				return fmt.Errorf("error injecting log level: %w", err)

			}
		}

		err = conf.Merge(confmap.NewFromStringMap(out))
		if err != nil {
			return err
		}
	}

	return nil
}

func handleCloudId(beatReceiverConfigKey string, conf *confmap.Conf) error {

	receiverCfg, _ := conf.Sub("receivers::" + beatReceiverConfigKey)
	beatCfg := config.MustNewConfigFrom(receiverCfg.ToStringMap())

	// Handle cloud.id the same way Beats does, this will also handle
	// extracting the Kibana URL
	if err := cloudid.OverwriteSettings(beatCfg); err != nil {
		return fmt.Errorf("cannot read cloudid: %w", err)
	}

	var beatOutput map[string]any
	err := beatCfg.Unpack(&beatOutput)
	if err != nil {
		return err
	}

	out := map[string]any{
		"receivers::" + beatReceiverConfigKey: beatOutput,
	}
	err = conf.Merge(confmap.NewFromStringMap(out))
	if err != nil {
		return err
	}

	// we set this to nil to ensure cloudid check does not throw error when output is next set to otelconsumer
	out = map[string]any{
		"receivers::" + beatReceiverConfigKey + "::cloud": nil,
	}
	err = conf.Merge(confmap.NewFromStringMap(out))
	if err != nil {
		return err
	}

	return nil
}

// promoteOutputQueueSettings promotes output.queue settings to global level
func promoteOutputQueueSettings(beatReceiverConfigKey string, outputConfig *config.C, conf *confmap.Conf) error {

	var queueOutput map[string]any
	err := outputConfig.Unpack(&queueOutput)
	if err != nil {
		return err
	}
	out := map[string]any{
		"receivers::" + beatReceiverConfigKey + "::queue": queueOutput["queue"],
	}
	err = conf.Merge(confmap.NewFromStringMap(out))
	if err != nil {
		return err
	}

	return nil
}
