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

package node

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"

	"github.com/elastic/beats/v7/metricbeat/module/elasticsearch"
)

var info = elasticsearch.Info{
	ClusterID:   "1234",
	ClusterName: "helloworld",
}

func TestGetMappings(t *testing.T) {
	elasticsearch.TestMapperWithInfo(t, "./_meta/test/node.*.json", eventsMapping)
}

func TestInvalid(t *testing.T) {
	file := "./_meta/test/invalid.json"

	content, err := ioutil.ReadFile(file)
	require.NoError(t, err)

	reporter := &mbtest.CapturingReporterV2{}
	err = eventsMapping(reporter, info, content, true)
	require.Error(t, err)
}
