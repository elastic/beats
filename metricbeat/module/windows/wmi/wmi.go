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

//go:build windows

package wmi

import (
	"fmt"
	"time"

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

	err = config.ApplyDefaultNamespaceToQueries(config.Namespace)
	if err != nil {
		return nil, err
	}

	config.BuildNamespaceQueryIndex()

	if config.WarningThreshold == 0*time.Second {
		config.WarningThreshold = base.Module().Config().Period
	}

	m := &MetricSet{
		BaseMetricSet: base,
		config:        config,
	}

	return m, nil
}

// This function handles the skip conditions
func (m *MetricSet) shouldSkipNilOrEmptyValue(fieldValue interface{}) bool {
	if fieldValue == nil {
		if !m.config.IncludeNull {
			return true // Skip if it's nil and IncludeNull is false
		}
	} else if stringValue, ok := fieldValue.(string); ok {
		if len(stringValue) == 0 && !m.config.IncludeEmptyString {
			return true // Skip if it's an empty string and IncludeEmptyString is false
		}
	}
	return false
}

// Fetch method implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {

	sm := wmi.NewWmiSessionManager()
	defer sm.Dispose()

	// To optimize performance and reduce overhead, we create a single session
	// for each unique WMI namespace. This minimizes the number of session creations
	for namespace, queries := range m.config.NamespaceQueryIndex {

		session, err := sm.GetSession(namespace, m.config.Host, "", m.config.User, m.config.Password)

		if err != nil {
			return fmt.Errorf("could not initialize session %w", err)
		}
		_, err = session.Connect()
		if err != nil {
			return fmt.Errorf("could not connect session %w", err)
		}
		defer session.Dispose()

		// We create a shared conversion table for entries with the same schema.
		// This avoids repeatedly fetching the schema for each individual entry, improving efficiency.
		conversionTable := make(map[string]WmiStringConversionFunction)

		for _, queryConfig := range queries {

			query := queryConfig.QueryStr

			rows, err := ExecuteGuardedQueryInstances(session, query, m.config.WarningThreshold)

			if err != nil {
				logp.Warn("Could not execute query %v", err)
				continue
			}

			defer wmi.CloseAllInstances(rows)

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

				// Get only the required properties
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

					if m.shouldSkipNilOrEmptyValue(fieldValue) {
						continue
					}

					// The default case, we simply return what we got
					finalValue := fieldValue

					// The script API of WMI returns strings for uint64, sint64, datetime
					// Link: https://learn.microsoft.com/en-us/windows/win32/wmisdk/querying-wmi
					// As a user, I want to have the right CIM_TYPE in the final document
					//
					// Example: in the query: SELECT * FROM Win32_OperatingSystem
					// FreePhysicalMemory is a string, but it should be an uint64
					if RequiresExtraConversion(fieldValue) {
						convertFun, ok := conversionTable[fieldName]
						// If the function is not found let us fetch it and cache it
						if !ok {
							convertFun, err = GetConvertFunction(instance, fieldName)
							if err != nil {
								logp.Warn("Skipping addition of field %s: Unable to retrieve the conversion function: %v", fieldName, err)
								continue
							}
							conversionTable[fieldName] = convertFun
						}
						// Perform the conversion at this point it's safe to cast to string.
						convertedValue, err := convertFun(fieldValue.(string))
						if err != nil {
							logp.Warn("Skipping addition of field %s. Cannot convert: %v", fieldName, err)
							continue
						}
						finalValue = convertedValue
					}
					event.MetricSetFields.Put(fieldName, finalValue)
				}
				report.Event(event)
			}
		}
	}
	return nil
}
