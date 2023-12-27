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

package elastic

import (
	"fmt"
	"strings"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/version"

	"github.com/elastic/beats/v7/metricbeat/mb"
	conf "github.com/elastic/elastic-agent-libs/config"
)

// Product supported by X-Pack Monitoring
type Product int

const (
	// Elasticsearch product
	Elasticsearch Product = iota

	// Kibana product
	Kibana

	// Logstash product
	Logstash

	// Beats product
	Beats

	// Enterprise Search product
	EnterpriseSearch
)

func (p Product) xPackMonitoringIndexString() string {
	indexProductNames := []string{
		"es",
		"kibana",
		"logstash",
		"beats",
		"ent-search",
	}

	if int(p) < 0 || int(p) > len(indexProductNames) {
		panic("Unknown product")
	}

	return indexProductNames[p]
}

func (p Product) String() string {
	productNames := []string{
		"elasticsearch",
		"kibana",
		"logstash",
		"beats",
		"enterprisesearch",
	}

	if int(p) < 0 || int(p) > len(productNames) {
		panic("Unknown product")
	}

	return productNames[p]
}

// MakeXPackMonitoringIndexName method returns the name of the monitoring index for
// a given product { elasticsearch, kibana, logstash, beats }
func MakeXPackMonitoringIndexName(product Product) string {
	const version = "8"

	return fmt.Sprintf(".monitoring-%v-%v-mb", product.xPackMonitoringIndexString(), version)
}

// ReportErrorForMissingField reports and returns an error message for the given
// field being missing in API response received from a given product
func ReportErrorForMissingField(field string, product Product, r mb.ReporterV2) error {
	err := MakeErrorForMissingField(field, product)
	r.Error(err)
	return err
}

// MakeErrorForMissingField returns an error message for the given field being missing in an API
// response received from a given product
func MakeErrorForMissingField(field string, product Product) error {
	return fmt.Errorf("Could not find field '%v' in %v API response", field, strings.Title(product.String()))
}

// IsFeatureAvailable returns whether a feature is available in the current product version
func IsFeatureAvailable(currentProductVersion, featureAvailableInProductVersion *version.V) bool {
	return !currentProductVersion.LessThan(featureAvailableInProductVersion)
}

// ReportAndLogError reports and logs the given error
func ReportAndLogError(err error, r mb.ReporterV2, l *logp.Logger) {
	r.Error(err)
	l.Error(err)
}

// FixTimestampField converts the given timestamp field in the given map from a float64 to an
// int, so that it is not serialized in scientific notation in the event. This is because
// Elasticsearch cannot accepts scientific notation to represent millis-since-epoch values
// for it's date fields: https://github.com/elastic/elasticsearch/pull/36691
func FixTimestampField(m mapstr.M, field string) error {
	v, err := m.GetValue(field)
	if err == mapstr.ErrKeyNotFound {
		return nil
	}
	if err != nil {
		return err
	}

	switch vv := v.(type) {
	case float64:
		_, err := m.Put(field, int(vv))
		return err
	}
	return nil
}

// NewModule returns a new Elastic stack module with the appropriate metricsets configured.
func NewModule(base *mb.BaseModule, xpackEnabledMetricsets []string, optionalXpackMetricsets []string, logger *logp.Logger) (*mb.BaseModule, error) {
	moduleName := base.Name()

	config := struct {
		XPackEnabled bool `config:"xpack.enabled"`
	}{}
	if err := base.UnpackConfig(&config); err != nil {
		return nil, fmt.Errorf("could not unpack configuration for module %v: %w", moduleName, err)
	}

	// No special configuration is needed if xpack.enabled != true
	if !config.XPackEnabled {
		return base, nil
	}

	var raw mapstr.M
	if err := base.UnpackConfig(&raw); err != nil {
		return nil, fmt.Errorf("could not unpack configuration for module %v: %w", moduleName, err)
	}

	// Ensure all required metricsets are enabled when xpack.enabled == true, and add any additional which are optional
	cfgdMetricsets, err := raw.GetValue("metricsets")
	metricsets := xpackEnabledMetricsets
	if err == nil && cfgdMetricsets != nil {
		// Type cast the metricsets to a slice of strings
		cfgdMetricsetsSlice := cfgdMetricsets.([]interface{})
		cfgdMetricsetsStrings := make([]string, len(cfgdMetricsetsSlice))
		for i := range cfgdMetricsetsSlice {
			cfgdMetricsetsStrings[i] = cfgdMetricsetsSlice[i].(string)
		}

		// Add any optional metricsets which are not already configured
		for _, cfgdMs := range cfgdMetricsetsStrings {
			found := false
			for _, ms := range optionalXpackMetricsets {
				if ms == cfgdMs {
					found = true
					break
				}
			}

			if found {
				metricsets = append(metricsets, cfgdMs)
			}
		}

	}

	raw["metricsets"] = metricsets

	newConfig, err := conf.NewConfigFrom(raw)
	if err != nil {
		return nil, fmt.Errorf("could not create new configuration for module %v: %w", moduleName, err)
	}

	newModule, err := base.WithConfig(*newConfig)
	if err != nil {
		return nil, fmt.Errorf("could not reconfigure module %v: %w", moduleName, err)
	}

	logger.Debugf("Configuration for module %v modified because xpack.enabled was set to true", moduleName)

	return newModule, nil
}
