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

package otelcommon

import (
	"github.com/elastic/beats/v7/x-pack/filebeat/fbreceiver"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/elasticsearchexporter"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/debugexporter"
	"go.opentelemetry.io/collector/otelcol"
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

	if err != nil {
		return otelcol.Factories{}, nil
	}

	return otelcol.Factories{
		Receivers: receivers,
		Exporters: exporters,
	}, nil

}