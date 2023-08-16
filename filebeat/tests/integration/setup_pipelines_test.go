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
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	cp "github.com/otiai10/copy"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
)

func TestSetupNoModules(t *testing.T) {
	cfg := `
filebeat:
output.elasticsearch:
  hosts:
    - %s
  username: %s
  password: %s
  allow_older_versions: true
setup.kibana:
  hosts: %s
  username: %s
  password: %s
`
	integration.EnsureESIsRunning(t)
	esURL := integration.GetESURL(t, "http")
	esPassword, ok := esURL.User.Password()
	require.True(t, ok, "ES didn't have a password")
	kURL, kUserInfo := integration.GetKibana(t)
	kPassword, ok := kUserInfo.Password()
	require.True(t, ok, "Kibana didn't have a password")

	filebeat := integration.NewBeat(t, "filebeat", "../../filebeat.test")

	err := cp.Copy("../../module", filepath.Join(filebeat.TempDir(), "module"))
	require.NoError(t, err, "error copying module directory")

	err = cp.Copy("../../modules.d", filepath.Join(filebeat.TempDir(), "modules.d"))
	require.NoError(t, err, "error copying modules.d directory")

	filebeat.WriteConfigFile(fmt.Sprintf(cfg, esURL.Host, esURL.User.Username(), esPassword, kURL.Host, kUserInfo.Username(), kPassword))
	filebeat.Start("setup")
	filebeat.WaitForLogs("Setup called, but no modules enabled.", 10*time.Second)
}

func TestSetupModulesNoFileset(t *testing.T) {
	cfg := `
filebeat.config:
  modules:
    enabled: true
    path: modules.d/*.yml
output.elasticsearch:
  hosts:
    - %s
  username: %s
  password: %s
  allow_older_versions: true
setup.kibana:
  hosts: %s
  username: %s
  password: %s
logging.level: debug
`
	integration.EnsureESIsRunning(t)
	esURL := integration.GetESURL(t, "http")
	esPassword, ok := esURL.User.Password()
	require.True(t, ok, "ES didn't have a password")
	kURL, kUserInfo := integration.GetKibana(t)
	kPassword, ok := kUserInfo.Password()
	require.True(t, ok, "Kibana didn't have a password")

	filebeat := integration.NewBeat(t, "filebeat", "../../filebeat.test")

	err := cp.Copy("../../module", filepath.Join(filebeat.TempDir(), "module"))
	require.NoError(t, err, "error copying module directory")

	err = cp.Copy("../../modules.d", filepath.Join(filebeat.TempDir(), "modules.d"))
	require.NoError(t, err, "error copying modules.d directory")

	filebeat.WriteConfigFile(fmt.Sprintf(cfg, esURL.Host, esURL.User.Username(), esPassword, kURL.Host, kUserInfo.Username(), kPassword))
	filebeat.Start("setup", "--pipelines")
	filebeat.WaitForLogs("Number of module configs found: 0", 10*time.Second)
}

func TestSetupModulesOneEnabled(t *testing.T) {
	cfg := `
filebeat.config:
  modules:
    enabled: true
    path: modules.d/*.yml
output.elasticsearch:
  hosts:
    - %s
  username: %s
  password: %s
  allow_older_versions: true
setup.kibana:
  hosts: %s
  username: %s
  password: %s
logging.level: debug
`
	integration.EnsureESIsRunning(t)
	esURL := integration.GetESURL(t, "http")
	esPassword, ok := esURL.User.Password()
	require.True(t, ok, "ES didn't have a password")
	kURL, kUserInfo := integration.GetKibana(t)
	kPassword, ok := kUserInfo.Password()
	require.True(t, ok, "Kibana didn't have a password")

	filebeat := integration.NewBeat(t, "filebeat", "../../filebeat.test")

	err := cp.Copy("../../module", filepath.Join(filebeat.TempDir(), "module"))
	require.NoError(t, err, "error copying module directory")

	err = cp.Copy("../../modules.d", filepath.Join(filebeat.TempDir(), "modules.d"))
	require.NoError(t, err, "error copying modules.d directory")

	filebeat.WriteConfigFile(fmt.Sprintf(cfg, esURL.Host, esURL.User.Username(), esPassword, kURL.Host, kUserInfo.Username(), kPassword))
	filebeat.Start("setup", "--pipelines", "--modules", "apache")
	filebeat.WaitForLogs("Exiting: module apache is configured but has no enabled filesets", 10*time.Second)
}

func TestSetupModulesOneEnabledOverride(t *testing.T) {
	cfg := `
filebeat.config:
  modules:
    enabled: true
    path: modules.d/*.yml
output.elasticsearch:
  hosts:
    - %s
  username: %s
  password: %s
  allow_older_versions: true
setup.kibana:
  hosts: %s
  username: %s
  password: %s
logging.level: debug
`
	integration.EnsureESIsRunning(t)
	esURL := integration.GetESURL(t, "http")
	esPassword, ok := esURL.User.Password()
	require.True(t, ok, "ES didn't have a password")
	kURL, kUserInfo := integration.GetKibana(t)
	kPassword, ok := kUserInfo.Password()
	require.True(t, ok, "Kibana didn't have a password")

	filebeat := integration.NewBeat(t, "filebeat", "../../filebeat.test")

	err := cp.Copy("../../module", filepath.Join(filebeat.TempDir(), "module"))
	require.NoError(t, err, "error copying module directory")

	err = cp.Copy("../../modules.d", filepath.Join(filebeat.TempDir(), "modules.d"))
	require.NoError(t, err, "error copying modules.d directory")

	filebeat.WriteConfigFile(fmt.Sprintf(cfg, esURL.Host, esURL.User.Username(), esPassword, kURL.Host, kUserInfo.Username(), kPassword))

	t.Cleanup(func() {
		delURL, err := url.Parse(esURL.String())
		if err != nil {
			t.Fatalf("could not parse ES url: %s", err)
		}
		delURL.Path = "/_ingest/pipeline/filebeat-*"
		ret, _, err := integration.HttpDo(t, http.MethodDelete, *delURL)
		if err != nil {
			t.Logf("error while deleting filebeat-* pipelines: %s", err)
		}
		if ret != http.StatusOK {
			t.Logf("status was %d while deleting filebeat-* pipelines", ret)
		}
	})
	filebeat.Start("setup", "--pipelines", "--modules", "apache", "--force-enable-module-filesets")
	filebeat.WaitForLogs("Elasticsearch pipeline loaded.", 10*time.Second)
	filebeat.WaitForLogs("Elasticsearch pipeline loaded.", 10*time.Second)
}
