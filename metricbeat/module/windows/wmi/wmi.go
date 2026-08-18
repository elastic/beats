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

	"github.com/elastic/beats/v9/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v9/metricbeat/mb"
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
	base.Logger().Warn(cfgwarn.Beta("The windows wmi metricset is beta."))

	config := NewDefaultConfig()
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	err := config.Validate()
	if err != nil {
		return nil, err
	}

	err = config.ValidateConnectionParameters()
	if err != nil {
		return nil, err
	}

	err = config.ApplyDefaultNamespaceToQueries(config.Namespace)
	if err != nil {
		return nil, err
	}

	config.BuildNamespaceQueryIndex()

	if config.WarningThreshold == 0 {
		config.WarningThreshold = base.Module().Config().Period
	}

	m := &MetricSet{
		BaseMetricSet: base,
		config:        config,
	}

	return m, nil
}

// This function handles the skip conditions
func (m *MetricSet) shouldSkipNilOrEmptyValue(propertyValue interface{}) bool {
	if propertyValue == nil {
		if !m.config.IncludeNullProperties {
			return true // Skip if it's nil and IncludeNullProperties is false
		}
	} else if stringValue, ok := propertyValue.(string); ok {
		if len(stringValue) == 0 && !m.config.IncludeEmptyStringProperties {
			return true // Skip if it's an empty string and IncludeEmptyStringProperties is false
		}
	}
	return false
}

func (m *MetricSet) reportError(report mb.ReporterV2, err error) {
	event := mb.Event{
		Error: err,
	}
	report.Event(event)
}

// Fetch method implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {

	sm := wmi.NewWmiSessionManager()
	defer sm.Dispose()

	// To optimize performance and reduce overhead, we create a single session
	// for each unique WMI namespace. This minimizes the number of session creations
	for namespace := range m.config.NamespaceQueryIndex {

		session, err := sm.GetSession(namespace, m.config.Host, m.config.Domain, m.config.User, m.config.Password)

		if err != nil {
			return fmt.Errorf("could not initialize session %w", err)
		}
		_, err = session.Connect()
		if err != nil {
			return fmt.Errorf("could not connect session in namespace '%s': %w", namespace, err)
		}
		defer session.Dispose()

		for i := range m.config.NamespaceQueryIndex[namespace] {

			// Get the queryConfig by reference to allow the initialization
			queryConfig := &m.config.NamespaceQueryIndex[namespace][i]

			// If we encountered an unrecoverable error before we do not attempt to perform the query again
			// We report the same error as in the first iteration
			if queryConfig.UnrecoverableError != nil {
				m.reportError(report, queryConfig.UnrecoverableError)
				continue
			}

			// We initialize the query lazily if not yet initialized
			if queryConfig.QueryStr == "" {
				err = m.initQuery(session, queryConfig)
				if err != nil {
					m.reportError(report, err)
					continue
				}
			}

			query := queryConfig.QueryStr

			rows, err := ExecuteGuardedQueryInstances(session, query, m.config.WarningThreshold, m.Logger())

			if err != nil {
				m.reportError(report, err)
				continue
			}

			defer wmi.CloseAllInstances(rows)

			if len(rows) == 0 {
				message := fmt.Sprintf(
					"The query '%s' did not return any results. "+
						"This can happen if the where clause is too restrictive, "+
						"but it might also indicate an invalid query. "+
						"Note: the class and property names are validated, but the where clause is not. "+
						"Ensure the full query is valid (e.g., using `Get-CimInstance -Query \"%s\" -Namespace \"%s\"`), "+
						"and check the WMI-Activity Operational Log for further details.",
					query, query, namespace,
				)
				m.Logger().Warn(message)
				m.reportError(report, fmt.Errorf("%s", message))
			}

			rowsToReport := len(rows)

			if m.config.MaxRowsPerQuery > 0 && rowsToReport > int(m.config.MaxRowsPerQuery) {
				m.Logger().Warnf("Got %d results, that exceeds the configured limit %d.", len(rows), (m.config.MaxRowsPerQuery))
				rowsToReport = int(m.config.MaxRowsPerQuery)
			}

			baseEvent := mb.Event{
				MetricSetFields: mapstr.M{
					"namespace": namespace,
					// Remote WMI is intentionally hidden, this will always be localhost
					// "host":      m.config.Host,
				},
			}

			if m.config.IncludeQueryClass {
				baseEvent.MetricSetFields.Put("query_class", queryConfig.Class)
			}

			if m.config.IncludeQueries {
				baseEvent.MetricSetFields.Put("query", query)
			}
			// Remote WMI is intentionally hidden, this will always be the empty string
			// if m.config.Domain != "" {
			// 	event.MetricSetFields.Put("domain", m.config.Domain)
			// }

			for _, instance := range rows[0:rowsToReport] {
				event := mb.Event{MetricSetFields: baseEvent.MetricSetFields.Clone()}

				// Add the instance class
				event.MetricSetFields.Put("class", instance.GetClassName())

				// Get only the required properties
				properties := queryConfig.Properties

				// If the Properties array is empty, we retrieve all available properties from the instance's class.
				// Note: due to inheritance, the instance's actual class may differ from the queried class
				if len(queryConfig.Properties) == 0 {
					properties = instance.GetClass().GetPropertiesNames()
				}

				for _, propertyName := range properties {
					propertyValue, err := instance.GetProperty(propertyName)
					if err != nil {
						m.Logger().Error("Unable to get propery by name: %v", err)
						continue
					}

					if m.shouldSkipNilOrEmptyValue(propertyValue) {
						continue
					}

					// The script API of WMI returns strings for uint64, sint64, datetime
					// Link: https://learn.microsoft.com/en-us/windows/win32/wmisdk/querying-wmi
					// As a user, I want to have the right CIM_TYPE in the final document
					//
					// Example: in the query: SELECT * FROM Win32_OperatingSystem
					// FreePhysicalMemory is a string, but it should be an uint64

					// We explicitly decide to perform a conversion on all fields because we do not support
					// the CIM_Object type as well
					convertFun, ok := queryConfig.WmiSchema.Get(instance.GetClassName(), propertyName)

					if !ok {
						convertFun, err = GetConvertFunction(instance, propertyName, m.Logger())
						if err != nil {
							m.Logger().Warnf("Skipping addition of property '%s'. Cannot fetch conversion function: '%v'", propertyName, err)
							continue
						}
						queryConfig.WmiSchema.Add(instance.GetClassName(), propertyName, convertFun)
					}
					finalValue, err := convertFun(propertyValue)
					if err != nil {
						m.Logger().Warnf("Skipping addition of property %s. Error during conversion: %v", propertyName, err)
						continue
					}
					event.MetricSetFields.Put(propertyName, finalValue)
				}
				report.Event(event)
			}
		}
	}
	return nil
}

// The WMI library does not differentiate between a genuinely empty result set and actual query errors.
// See this issue for more context: https://github.com/microsoft/wmi/issues/156
//
// To improve troubleshooting, we rule out the two most common causes early by validating
// the existence of the class and its required properties during the initial query.
func (m *MetricSet) initQuery(session WmiQueryInterface, queryConfig *QueryConfig) error {
	query := fmt.Sprintf("SELECT * FROM meta_class WHERE __Class = '%s'", queryConfig.Class)
	rows, err := ExecuteGuardedQueryInstances(session, query, m.config.WarningThreshold, m.Logger())

	if err != nil {
		return fmt.Errorf("could not execute the meta_class query '%s' with the error: '%w'. We will try in the next iteration", query, err)
	}

	defer wmi.CloseAllInstances(rows)

	err = errorOnClassDoesNotExist(rows, queryConfig.Class, queryConfig.Namespace)
	if err != nil {
		queryConfig.UnrecoverableError = err
		return err
	}

	instance := rows[0]

	err = validateQueryFields(instance, queryConfig, m.Logger())
	if err != nil {
		queryConfig.UnrecoverableError = err
		return err
	}

	wmiSchema, err := NewWMISchema(m.config.SchemaCacheSize)
	if err != nil {
		return fmt.Errorf("could not initialize wmi schema: %w", err)
	}
	queryConfig.WmiSchema = wmiSchema
	queryConfig.compileQuery()

	return nil
}
