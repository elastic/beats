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

	"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
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
)

func (p Product) xPackMonitoringIndexString() string {
	indexProductNames := []string{
		"es",
		"kibana",
		"logstash",
		"beats",
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
	}

	if int(p) < 0 || int(p) > len(productNames) {
		panic("Unknown product")
	}

	return productNames[p]
}

// MakeXPackMonitoringIndexName method returns the name of the monitoring index for
// a given product { elasticsearch, kibana, logstash, beats }
func MakeXPackMonitoringIndexName(product Product) string {
	const version = "7"

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
	return fmt.Errorf("Could not find field '%v' in %v stats API response", field, strings.Title(product.String()))
}

// IsFeatureAvailable returns whether a feature is available in the current product version
func IsFeatureAvailable(currentProductVersion, featureAvailableInProductVersion *common.Version) bool {
	return !currentProductVersion.LessThan(featureAvailableInProductVersion)
}

// ReportAndLogError reports and logs the given error
func ReportAndLogError(err error, r mb.ReporterV2, l *logp.Logger) {
	r.Error(err)
	l.Error(err)
}
