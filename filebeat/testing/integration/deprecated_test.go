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

//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/testing/integration"
)

func TestFilebeatDeprecated(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
	defer cancel()
	EnsureCompiled(ctx, t)

	reportOptions := integration.ReportOptions{
		PrintLinesOnFail:  10,
		PrintConfigOnFail: false,
	}

	t.Run("test invalid config with removed settings", func(t *testing.T) {

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		config := `
filebeat.config.modules:
  path: ${path.config}/modules.d/*.yml
  reload.enabled: true
output.console:
  enabled: true  
`

		test := NewTest(t, TestOptions{
			Config: config,
			Args:   []string{"-E", "filebeat.prospectors=anything", "-E", "filebeat.config.prospectors=anything"},
		})

		test.ExpectOutput(`setting 'filebeat.prospectors' has been removed`)
		test.ExpectOutput(`setting 'filebeat.config.prospectors' has been removed`)

		test.
			WithReportOptions(reportOptions).
			Start(ctx).
			Wait()
	})
}
