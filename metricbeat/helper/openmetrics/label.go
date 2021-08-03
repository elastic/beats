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

package openmetrics

// LabelMap defines the mapping from OpenMetrics label to a Metricbeat field
type LabelMap interface {
	// GetField returns the resulting field name
	GetField() string

	// IsKey returns true if the label is a key label
	IsKey() bool
}

// Label maps a OpenMetrics label to a Metricbeat field
func Label(field string) LabelMap {
	return &commonLabel{
		field: field,
		key:   false,
	}
}

// KeyLabel maps a OpenMetrics label to a Metricbeat field. The label is flagged as key.
// Metrics with the same tuple of key labels will be grouped in the same event.
func KeyLabel(field string) LabelMap {
	return &commonLabel{
		field: field,
		key:   true,
	}
}

type commonLabel struct {
	field string
	key   bool
}

// GetField returns the resulting field name
func (l *commonLabel) GetField() string {
	return l.field
}

// IsKey returns true if the label is a key label
func (l *commonLabel) IsKey() bool {
	return l.key
}
