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

// +build !integration

package cluster_stats

import (
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/metricbeat/helper"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/elasticsearch"
)


func TestMapper(t *testing.T) {
	httpHelper, err := helper.NewHTTPFromConfig(helper.Config{
		ConnectTimeout: 30 * time.Second,
		Timeout:        30 * time.Second,
	}, mb.HostData{
		URI:          "http://localhost:9200",
		SanitizedURI: "http://localhost:9200",
		Host:         "http://localhost:9200",
	})
	if err != nil {
		t.Fatal(err)
	}

	f := func(r mb.ReporterV2, content []byte) error {
		return eventMapping(r, httpHelper, elasticsearch.Info{
			ClusterName: "test_cluster",
			ClusterID:   "12345",
			Version: struct {
				Number *common.Version `json:"number"`
			}{
				Number: &common.Version{
					Major:  7,
					Minor:  10,
					Bugfix: 0,
				},
			},
		}, content)
	}

	elasticsearch.TestMapper(t, "./_meta/test/cluster_stats.*.json", f)
}
