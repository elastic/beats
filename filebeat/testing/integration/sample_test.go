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

package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/testing/integration"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/stretchr/testify/assert"
)

func TestFilebeat(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	EnsureCompiled(ctx, t)

	messagePrefix := "sample test message"
	fileCount := 5
	lineCount := 128

	reportOptions := integration.ReportOptions{
		PrintLinesOnFail:  10,
		PrintConfigOnFail: true,
	}

	t.Run("Filebeat starts and ingests files", func(t *testing.T) {
		configPlainTemplate := `
filebeat.inputs:
  - type: filestream
    id: "test-filestream"
    paths:
      - %s
# we want to check that all messages are ingested
# without using an external service, this is an easy way
output.console:
  enabled: true
`
		configGZIPTemplate := `
filebeat.inputs:
  - type: filestream
    id: test-filestream
    paths:
      - %s
    gzip_experimental: true

# we want to check that all messages are ingested
# without using an external service, this is an easy way
output.console:
  enabled: true
`
		// we can generate any amount of expectations
		// they are light-weight
		expectIngestedFiles := func(test Test, files []string) {
			// ensuring we ingest every line from every file
			for _, filename := range files {
				for i := 1; i <= lineCount; i++ {
					line := fmt.Sprintf("%s:%d", filepath.Base(filename), i)
					test.ExpectOutput(line)
				}
			}
		}

		tcs := map[string]struct {
			configTemplate     string
			GenerateLogFilesFn func(t *testing.T, files, lines int, generator LogGenerator) (path string, filenames []string)
		}{
			"plain": {
				configTemplate:     configPlainTemplate,
				GenerateLogFilesFn: GenerateLogFiles,
			},
			"GZIP": {
				configTemplate:     configGZIPTemplate,
				GenerateLogFilesFn: GenerateGZIPLogFiles,
			},
		}
		for name, tc := range tcs {
			t.Run(name, func(t *testing.T) {

				t.Run("plain text logs - unstructured log files", func(t *testing.T) {
					ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
					defer cancel()

					generator := NewPlainTextGenerator(messagePrefix)
					path, files := tc.GenerateLogFilesFn(t, fileCount, lineCount, generator)
					config := fmt.Sprintf(tc.configTemplate, path)
					test := NewTest(t, TestOptions{
						Config: config,
					})

					expectIngestedFiles(test, files)

					test.
						// we expect to read all generated files to EOF
						ExpectEOF(files...).
						WithReportOptions(reportOptions).
						// we should observe the start message of the Beat
						ExpectStart().
						// check that the first and the last line of the file get ingested
						Start(ctx).
						// wait until all the expectations are met
						// or we hit the timeout set by the context
						Wait()
				})

				t.Run("JSON logs - structured log files", func(t *testing.T) {
					ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
					defer cancel()

					generator := NewJSONGenerator(messagePrefix)
					path, files := tc.GenerateLogFilesFn(t, fileCount, lineCount, generator)
					config := fmt.Sprintf(tc.configTemplate, path)
					test := NewTest(t, TestOptions{
						Config: config,
					})

					expectIngestedFiles(test, files)

					test.
						ExpectEOF(files...).
						WithReportOptions(reportOptions).
						ExpectStart().
						Start(ctx).
						Wait()
				})
			})
		}
	})
	t.Run("Filebeat crashes due to incorrect config", func(t *testing.T) {
		t.Skip("Flaky test: https://github.com/elastic/beats/issues/42778")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// path items are required, this config is invalid
		config := `
filebeat.inputs:
  - type: filestream
    id: "test-filestream"
output.console:
  enabled: true
`
		test := NewTest(t, TestOptions{
			Config: config,
		})

		test.
			WithReportOptions(reportOptions).
			ExpectStart().
			ExpectOutput("Exiting: Failed to start crawler: starting input failed: error while initializing input: no path is configured").
			ExpectStop(1).
			Start(ctx).
			Wait()
	})
}

func TestFilebeatModuleCmd(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	EnsureCompiled(ctx, t)

	reportOptions := integration.ReportOptions{
		PrintLinesOnFail:  10,
		PrintConfigOnFail: false,
	}

	configTemplate := `
filebeat.config.modules:
  path: %s/modules.d/*.yml
  reload.enabled: true
`

	dir := t.TempDir()
	modules := filepath.Join(dir, "modules.d")
	err := os.MkdirAll(modules, 0777)
	if err != nil {
		t.Fatalf("failed to create a module directory: %v", err)
	}
	os.Create(filepath.Join(modules, "enabled-module.yml"))
	os.Create(filepath.Join(modules, "disabled-module.yml.disabled"))

	t.Run("Test modules list command", func(t *testing.T) {

		test := NewTest(t, TestOptions{
			Config: fmt.Sprintf(configTemplate, dir),
			Args:   []string{"modules", "list"},
		})

		test.ExpectOutput("Enabled:", "enabled-module").ExpectOutput("Disabled:", "disabled-module")

		test.
			WithReportOptions(reportOptions).
			Start(ctx).
			Wait()
	})

	t.Run("test module enable command", func(t *testing.T) {

		test := NewTest(t, TestOptions{
			Config: fmt.Sprintf(configTemplate, dir),
			Args:   []string{"modules", "enable", "disabled-module"},
		})

		// Enable one module
		test.ExpectOutput("Enabled disabled-module")

		test.
			WithReportOptions(reportOptions).
			Start(ctx).
			Wait()

		_, err := os.Stat(filepath.Join(modules, "disabled-module.yml.disabled"))
		assert.True(t, os.IsNotExist(err))
		_, err = os.Stat(filepath.Join(modules, "disabled-module.yml"))
		assert.Nil(t, err)
	})

	t.Run("enable multiple module at once", func(t *testing.T) {

		test := NewTest(t, TestOptions{
			Config: fmt.Sprintf(configTemplate, dir),
			Args:   []string{"modules", "enable", "disabled2", "disabled3"},
		})

		os.Create(filepath.Join(modules, "disabled2.yml.disabled"))
		os.Create(filepath.Join(modules, "disabled3.yml.disabled"))

		test.ExpectOutput("Enabled disabled2")
		test.ExpectOutput("Enabled disabled3")

		test.
			WithReportOptions(reportOptions).
			Start(ctx).
			Wait()

		_, err := os.Stat(filepath.Join(modules, "disabled2.yml.disabled"))
		assert.True(t, os.IsNotExist(err))
		_, err = os.Stat(filepath.Join(modules, "disabled2.yml"))
		assert.Nil(t, err)
		_, err = os.Stat(filepath.Join(modules, "disabled3.yml.disabled"))
		assert.True(t, os.IsNotExist(err))
		_, err = os.Stat(filepath.Join(modules, "disabled3.yml"))
		assert.Nil(t, err)
	})

	t.Run("test disable command ", func(t *testing.T) {

		test := NewTest(t, TestOptions{
			Config: fmt.Sprintf(configTemplate, dir),
			Args:   []string{"modules", "disable", "enabled-module"},
		})

		test.ExpectOutput("Disabled enabled-module")

		test.
			WithReportOptions(reportOptions).
			Start(ctx).
			Wait()

		_, err := os.Stat(filepath.Join(modules, "enabled-module.yml"))
		assert.True(t, os.IsNotExist(err))
		_, err = os.Stat(filepath.Join(modules, "enabled-module.yml.disabled"))
		assert.Nil(t, err)

	})

	t.Run("disable multiple module at once", func(t *testing.T) {

		test := NewTest(t, TestOptions{
			Config: fmt.Sprintf(configTemplate, dir),
			Args:   []string{"modules", "disable", "enabled2", "enabled3"},
		})

		os.Create(filepath.Join(modules, "enabled2.yml"))
		os.Create(filepath.Join(modules, "enabled3.yml"))

		test.ExpectOutput("Disabled enabled2")
		test.ExpectOutput("Disabled enabled3")

		test.
			WithReportOptions(reportOptions).
			Start(ctx).
			Wait()

		_, err := os.Stat(filepath.Join(modules, "enabled2.yml"))
		assert.True(t, os.IsNotExist(err))
		_, err = os.Stat(filepath.Join(modules, "enabled2.yml.disabled"))
		assert.Nil(t, err)
		_, err = os.Stat(filepath.Join(modules, "enabled3.yml"))
		assert.True(t, os.IsNotExist(err))
		_, err = os.Stat(filepath.Join(modules, "enabled3.yml.disabled"))
		assert.Nil(t, err)
	})

}

func TestFilebeatDeprecated(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	EnsureCompiled(ctx, t)

	messagePrefix := "sample test message"
	fileCount := 1
	lineCount := 1

	reportOptions := integration.ReportOptions{
		PrintLinesOnFail:  10,
		PrintConfigOnFail: false,
	}

	t.Run("check that harvesting works with deprecated input_type", func(t *testing.T) {

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		config := `
filebeat.inputs:
  - input_type: log
    id: "test-filestream"
    paths:
     - %s
    scan_frequency: 0.1s
    allow_deprecated_use: true
output.console:
  enabled: true
`
		generator := NewPlainTextGenerator(messagePrefix)
		path, file := GenerateLogFiles(t, fileCount, lineCount, generator)
		test := NewTest(t, TestOptions{
			Config: fmt.Sprintf(config, path),
		})

		line := fmt.Sprintf("%s:%d", filepath.Base(file[0]), 1)
		test.ExpectOutput(line)
		test.ExpectOutput("DEPRECATED: input_type input config is deprecated")

		test.
			WithReportOptions(reportOptions).
			ExpectStart().
			Start(ctx).
			Wait()
	})

	t.Run("check that harvesting works with deprecated input_type", func(t *testing.T) {

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

func TestCustomFields(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	EnsureCompiled(ctx, t)

	messagePrefix := "sample test message"
	fileCount := 1
	lineCount := 10

	reportOptions := integration.ReportOptions{
		PrintLinesOnFail:  10,
		PrintConfigOnFail: true,
	}

	generator := NewPlainTextGenerator(messagePrefix)
	path, file := GenerateLogFiles(t, fileCount, lineCount, generator)

	t.Run("tests that custom fields show up in the output dict and  agent.name defaults to hostname", func(t *testing.T) {

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		config := `
filebeat.inputs:
  - type: filestream
    id: "test-filestream"
    paths:
     - %s
    fields:
      hello: world
      number: 2
    file_identity.native: ~
    prospector.scanner.fingerprint.enabled: false	  
output.console:
  enabled: true
`

		test := NewTest(t, TestOptions{
			Config: fmt.Sprintf(config, path),
		})

		host, _ := os.Hostname()
		line := fmt.Sprintf("%s:%d", filepath.Base(file[0]), 1)

		test.ExpectJSONFields(mapstr.M{
			"message":       fmt.Sprintf("sample test message %s", line),
			"fields.number": float64(2),
			"fields.hello":  "world",
			"hostname":      host,
		})

		test.
			WithReportOptions(reportOptions).
			ExpectStart().
			Start(ctx).
			Wait()
	})

	t.Run("tests that custom fields show up in the output dict when fields_under_root: true", func(t *testing.T) {

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		config := `
filebeat.inputs:
  - type: filestream
    id: "test-filestream"
    paths:
     - %s
    fields_under_root: true
    fields:
      hello: world
      number: 2
    file_identity.native: ~
    prospector.scanner.fingerprint.enabled: false	  
output.console:
  enabled: true
`

		test := NewTest(t, TestOptions{
			Config: fmt.Sprintf(config, path),
		})

		line := fmt.Sprintf("%s:%d", filepath.Base(file[0]), 1)
		test.ExpectJSONFields(mapstr.M{
			"message": fmt.Sprintf("sample test message %s", line),
			"number":  float64(2),
			"hello":   "world",
		})

		test.
			WithReportOptions(reportOptions).
			ExpectStart().
			Start(ctx).
			Wait()
	})

	t.Run("tests that custom fields show up in the output dict when fields_under_root: true", func(t *testing.T) {

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		config := `
filebeat.inputs:
  - type: filestream
    id: "test-filestream"
    paths:
     - %s
    file_identity.native: ~
    prospector.scanner.fingerprint.enabled: false
shipper: testShipperName
output.console:
  enabled: true
`

		test := NewTest(t, TestOptions{
			Config: fmt.Sprintf(config, path),
		})

		line := fmt.Sprintf("%s:%d", filepath.Base(file[0]), 1)
		test.ExpectJSONFields(mapstr.M{
			"message":    fmt.Sprintf("sample test message %s", line),
			"host.name":  "testShipperName",
			"agent.name": "testShipperName",
		})

		test.
			WithReportOptions(reportOptions).
			ExpectStart().
			Start(ctx).
			Wait()
	})

}
