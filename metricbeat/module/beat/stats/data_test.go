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

package stats

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/metricbeat/module/beat"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
)

func TestEventMapping(t *testing.T) {

	files, err := filepath.Glob("./_meta/test/stats.*.json")
	require.NoError(t, err)

	info := beat.Info{
		UUID: "1234",
		Beat: "helloworld",
	}

	for _, f := range files {
		input, err := ioutil.ReadFile(f)
		require.NoError(t, err)

		reporter := &mbtest.CapturingReporterV2{}
		err = eventMapping(reporter, info, input)

		require.NoError(t, err, f)
		require.True(t, len(reporter.GetEvents()) >= 1, f)
		require.Equal(t, 0, len(reporter.GetErrors()), f)
	}
}
