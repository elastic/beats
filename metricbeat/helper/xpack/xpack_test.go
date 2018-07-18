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

package xpack

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMakeMonitoringIndexName(t *testing.T) {
	today := time.Now().UTC().Format("2006.01.02")

	tests := []struct {
		Name     string
		Product  Product
		Expected string
	}{
		{
			"Elasticsearch monitoring index",
			Elasticsearch,
			fmt.Sprintf(".monitoring-es-6-mb-%v", today),
		},
		{
			"Kibana monitoring index",
			Kibana,
			fmt.Sprintf(".monitoring-kibana-6-mb-%v", today),
		},
		{
			"Logstash monitoring index",
			Logstash,
			fmt.Sprintf(".monitoring-logstash-6-mb-%v", today),
		},
		{
			"Beats monitoring index",
			Beats,
			fmt.Sprintf(".monitoring-beats-6-mb-%v", today),
		},
	}

	for _, test := range tests {
		name := fmt.Sprintf("Test naming %v", test.Name)
		t.Run(name, func(t *testing.T) {
			indexName := MakeMonitoringIndexName(test.Product)
			assert.Equal(t, test.Expected, indexName)
		})
	}
}
