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

package kibana_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/metricbeat/module/kibana"

	// Make sure metricsets are registered in mb.Registry
	_ "github.com/elastic/beats/v7/metricbeat/module/kibana/stats"
)

func TestIsStatsAPIAvailable(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"6.3.1", false},
		{"6.4.0", true},
		{"6.5.0", true},
		{"7.0.0-alpha1", true},
	}

	for _, test := range tests {
		actual := kibana.IsStatsAPIAvailable(common.MustNewVersion(test.input))
		require.Equal(t, test.expected, actual)
	}
}

func TestIsUsageExcludable(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"6.3.1", false},
		{"6.4.0", false},
		{"6.5.0", false},
		{"6.7.2", true},
		{"6.8.16", true},
		{"7.0.0-alpha1", false},
		{"7.0.0", false},
		{"7.0.1", true},
		{"7.0.2", true},
		{"7.5.0", true},
		{"7.16.2", true},
	}

	for _, test := range tests {
		actual := kibana.IsUsageExcludable(common.MustNewVersion(test.input))
		require.Equal(t, test.expected, actual)
	}
}
