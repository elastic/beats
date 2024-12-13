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

package wmi

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	wmi "github.com/microsoft/wmi/pkg/wmiinstance"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host is defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("windows", "wmi", New)
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	config Config
}

const WMIDefaultNamespace = "root\\cimv2"

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The windows wmi metricset is beta.")

	config := NewDefaultConfig()
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	err := config.ValidateConnectionParameters()
	if err != nil {
		return nil, err
	}

	err = config.CompileQueries()
	if err != nil {
		return nil, err
	}

	m := &MetricSet{
		BaseMetricSet: base,
		config:        config,
	}

	return m, nil
}

// Fetch method implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {

	var err error

	sm := wmi.NewWmiSessionManager()
	defer sm.Dispose()

	session, err := sm.GetSession(m.config.Namespace, m.config.Host, "", m.config.User, m.config.Password)

	if err != nil {
		return fmt.Errorf("could not initialize session %w", err)
	}
	_, err = session.Connect()
	if err != nil {
		return fmt.Errorf("could not connect session %w", err)
	}
	defer session.Dispose()

	for _, queryConfig := range m.config.Queries {

		query := queryConfig.QueryStr

		rows, err := session.QueryInstances(query)
		if err != nil {
			logp.Warn("Could not execute query %v", err)
			continue
		}

		for _, instance := range rows {
			event := mb.Event{
				MetricSetFields: mapstr.M{
					"class":     queryConfig.Class,
					"namespace": m.config.Namespace,
					"host":      m.config.Host,
				},
			}

			if m.config.IncludeQueries {
				event.MetricSetFields.Put("query", query)
			}

			// Get only the required properites
			properties := queryConfig.Fields

			// If the Fields array is empty we retrieve all fields
			if len(queryConfig.Fields) == 0 {
				properties = instance.GetClass().GetPropertiesNames()
			}

			for _, fieldName := range properties {
				fieldValue, err := instance.GetProperty(fieldName)
				if err != nil {
					logp.Err("Unable to get propery by name: %v", err)
					continue
				}
				// If the user decides to ignore properties with nil values, we skip them
				if !m.config.IncludeNull && fieldValue == nil {
					continue
				}

				event.MetricSetFields.Put(fieldName, fieldValue)
			}
			report.Event(event)
		}
	}
	return nil
}
