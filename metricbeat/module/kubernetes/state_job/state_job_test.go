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

//go:build !integration
// +build !integration

package state_job

import (
	"testing"

	k "github.com/elastic/beats/v7/metricbeat/helper/kubernetes/ktest"

	"github.com/elastic/beats/v7/metricbeat/helper/prometheus/ptest"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	_ "github.com/elastic/beats/v7/metricbeat/module/kubernetes"
)

var files = []string{
	"../_meta/test/ksm.v2.4.2.plain",
	"../_meta/test/ksm.v2.5.0.plain",
	"../_meta/test/ksm.v2.6.0.plain",
	"../_meta/test/ksm.v2.7.0.plain",
}

const name = "state_job"

func TestEventMapping(t *testing.T) {
	ptest.TestMetricSet(t, "kubernetes", name, k.GetTestCases(files))
}

func TestData(t *testing.T) {
	mbtest.TestDataFiles(t, "kubernetes", name)
}

func TestMetricsFamily(t *testing.T) {
	k.TestStateMetricsFamily(t, files, mapping)
}
