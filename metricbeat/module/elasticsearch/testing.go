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

package elasticsearch

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/elastic/beats/metricbeat/mb"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"

	"github.com/stretchr/testify/assert"
)

// TestMapper tests mapping methods
func TestMapper(t *testing.T, glob string, mapper func(mb.ReporterV2, []byte) error) {
	files, err := filepath.Glob(glob)
	assert.NoError(t, err)
	// Makes sure glob matches at least 1 file
	assert.True(t, len(files) > 0)

	for _, f := range files {
		t.Run(f, func(t *testing.T) {
			input, err := ioutil.ReadFile(f)
			assert.NoError(t, err)

			reporter := &mbtest.CapturingReporterV2{}
			err = mapper(reporter, input)
			assert.NoError(t, err)
			assert.True(t, len(reporter.GetEvents()) >= 1)
			assert.Equal(t, 0, len(reporter.GetErrors()))
		})
	}
}

// TestMapperWithInfo tests mapping methods with Info fields
func TestMapperWithInfo(t *testing.T, glob string, mapper func(mb.ReporterV2, Info, []byte) error) {
	files, err := filepath.Glob(glob)
	assert.NoError(t, err)
	// Makes sure glob matches at least 1 file
	assert.True(t, len(files) > 0)

	info := Info{
		ClusterID:   "1234",
		ClusterName: "helloworld",
	}

	for _, f := range files {
		t.Run(f, func(t *testing.T) {
			input, err := ioutil.ReadFile(f)
			assert.NoError(t, err)

			reporter := &mbtest.CapturingReporterV2{}
			err = mapper(reporter, info, input)
			assert.NoError(t, err)
			assert.True(t, len(reporter.GetEvents()) >= 1)
			assert.Equal(t, 0, len(reporter.GetErrors()))
		})
	}
}
