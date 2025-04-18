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

package cfgfile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

func TestGlobManagerInit(t *testing.T) {
	// Wrong settings return error
	logger := logptest.NewTestingLogger(t, "")
	manager, err := NewGlobManager("dir/*.yml", ".noyml", ".disabled", logger)
	assert.Error(t, err)
	assert.Nil(t, manager)
}

func TestGlobManager(t *testing.T) {
	// Create random temp directory
	dir := t.TempDir()

	// Prepare scenario:
	content := []byte("test\n")
	err := os.WriteFile(dir+"/config1.yml", content, 0644)
	assert.NoError(t, err)
	err = os.WriteFile(dir+"/config2.yml", content, 0644)
	assert.NoError(t, err)
	err = os.WriteFile(dir+"/config2-alt.yml.disabled", content, 0644)
	assert.NoError(t, err)
	err = os.WriteFile(dir+"/config3.yml.disabled", content, 0644)
	assert.NoError(t, err)

	// Init Glob Manager
	glob := dir + "/*.yml"
	logger := logptest.NewTestingLogger(t, "")
	manager, err := NewGlobManager(glob, ".yml", ".disabled", logger)
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, manager.Exists("config1"))
	assert.True(t, manager.Exists("config2"))
	assert.True(t, manager.Exists("config2-alt"))
	assert.True(t, manager.Exists("config3"))
	assert.False(t, manager.Exists("config4"))

	assert.True(t, manager.Enabled("config1"))
	assert.True(t, manager.Enabled("config2"))
	assert.False(t, manager.Enabled("config2-alt"))
	assert.False(t, manager.Enabled("config3"))

	assert.Equal(t, len(manager.ListEnabled()), 2)
	assert.Equal(t, len(manager.ListDisabled()), 2)

	// Test disable
	if err = manager.Disable("config2"); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, len(manager.ListEnabled()), 1)
	assert.Equal(t, len(manager.ListDisabled()), 3)

	enabled := manager.ListEnabled()
	assert.Equal(t, enabled[0].Name, "config1")
	assert.Equal(t, enabled[0].Enabled, true)

	// Test enable
	if err = manager.Enable("config3"); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, len(manager.ListEnabled()), 2)
	assert.Equal(t, len(manager.ListDisabled()), 2)

	disabled := manager.ListDisabled()
	assert.Equal(t, disabled[0].Name, "config2")
	assert.Equal(t, disabled[0].Enabled, false)
	assert.Equal(t, disabled[1].Name, "config2-alt")
	assert.Equal(t, disabled[1].Enabled, false)

	// Check correct files layout:
	files, err := filepath.Glob(dir + "/*")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, files, []string{
		filepath.Join(dir, "config1.yml"),
		filepath.Join(dir, "config2-alt.yml.disabled"),
		filepath.Join(dir, "config2.yml.disabled"),
		filepath.Join(dir, "config3.yml"),
	})
}

func TestCfgFileSorting(t *testing.T) {
	cfgFiles := byCfgFileDisplayNames{
		&CfgFile{
			"foo",
			"modules.d/foo.yml",
			false,
		},
		&CfgFile{
			"foo-variant",
			"modules.d/foo-variant.yml",
			false,
		},
		&CfgFile{
			"fox",
			"modules.d/fox.yml",
			false,
		},
	}

	assert.True(t, cfgFiles.Less(0, 1))
	assert.False(t, cfgFiles.Less(1, 0))
	assert.True(t, cfgFiles.Less(0, 2))
}
