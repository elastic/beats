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
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/testing/integration"
)

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
	_, err = os.Create(filepath.Join(modules, "enabled-module.yml"))
	assert.NoError(t, err)
	_, err = os.Create(filepath.Join(modules, "disabled-module.yml.disabled"))
	assert.NoError(t, err)

	t.Run("Test modules list command", func(t *testing.T) {

		test := NewTest(t, TestOptions{
			Config: fmt.Sprintf(configTemplate, dir),
			Args:   []string{"modules", "list"},
		})

		test.ExpectOutput("Enabled:", "enabled-modue").ExpectOutput("Disabled:", "disabled-module")

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
