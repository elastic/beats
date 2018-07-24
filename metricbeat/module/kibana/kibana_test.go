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

package kibana

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/metricbeat/mb"
)

type MockReporterV2 struct {
	mb.ReporterV2
}

func (MockReporterV2) Event(event mb.Event) bool {
	return true
}

var currentErr error // This hack is necessary because the Error method below cannot receive the type *MockReporterV2

func (m MockReporterV2) Error(err error) bool {
	currentErr = err
	return true
}

func TestReportErrorForMissingField(t *testing.T) {
	field := "some.missing.field"
	r := MockReporterV2{}
	err := ReportErrorForMissingField(field, r)

	expectedError := fmt.Errorf("Could not find field '%v' in Kibana stats API response", field)
	assert.Equal(t, expectedError, err)
	assert.Equal(t, expectedError, currentErr)
}
