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

//go:build integration && !requirefips

package integration

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/go-ldap/ldap/v3"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
	"github.com/elastic/elastic-agent-autodiscover/docker"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

const translateguidCfg = `
filebeat.inputs:
  - type: filestream
    id: "test-translateguidCfg"
    file_identity.native: ~
    prospector.scanner.fingerprint.enabled: false
    paths:
      - %s

queue.mem:
  flush.min_events: 1
  flush.timeout: 0.1s

path.home: %s

output.file:
  path: ${path.home}
  filename: "output-file"

logging:
  metrics:
    enabled: false

processors:
  - add_fields:
      fields:
        guid: '%s'
  - translate_ldap_attribute:
      field: fields.guid
      target_field: fields.common_name
      ldap_address: 'ldap://localhost:1389'
      ldap_base_dn: 'dc=example,dc=org'
      ldap_bind_user: 'cn=admin,dc=example,dc=org'
      ldap_bind_password: 'adminpassword'
      ldap_search_attribute: 'entryUUID'
`

func TestTranslateGUIDWithLDAP(t *testing.T) {
	t.Skip("Flaky Test: https://github.com/elastic/beats/issues/42616")
	startOpenldapContainer(t)

	var entryUUID string
	require.Eventually(t, func() bool {
		var err error
		entryUUID, err = getLDAPUserEntryUUID()
		return err == nil
	}, 10*time.Second, time.Second)

	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)
	tempDir := filebeat.TempDir()

	// 1. Generate the log file path
	logFilePath := path.Join(tempDir, "log.log")
	integration.WriteLogFile(t, logFilePath, 1, false)

	// 2. Write configuration file and start Filebeat
	filebeat.WriteConfigFile(
		fmt.Sprintf(translateguidCfg, logFilePath, tempDir, entryUUID),
	)
	filebeat.Start()

	var outputFile string
	require.Eventually(t, func() bool {
		outputFiles, err := filepath.Glob(path.Join(tempDir, "output-file-*.ndjson"))
		if err != nil {
			return false
		}
		if len(outputFiles) != 1 {
			return false
		}
		outputFile = outputFiles[0]
		return true
	}, 10*time.Second, time.Second)

	// 3. Wait for the event with the expected translated guid
	filebeat.WaitFileContains(
		outputFile,
		fmt.Sprintf(`"fields":{"guid":"%s","common_name":["User1","user01"]}`, entryUUID),
		20*time.Second,
	)
}

func startOpenldapContainer(t *testing.T) {
	ctx := context.Background()
	c, err := docker.NewClient(client.DefaultDockerHost, nil, nil, logptest.NewTestingLogger(t, ""))
	if err != nil {
		t.Fatal(err)
	}

	reader, err := c.ImagePull(ctx, "bitnami/openldap:2", image.PullOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = io.Copy(os.Stdout, reader); err != nil {
		t.Fatal(err)
	}
	reader.Close()

	resp, err := c.ContainerCreate(ctx,
		&container.Config{
			Image: "bitnami/openldap:2",
			ExposedPorts: nat.PortSet{
				"1389/tcp": struct{}{},
			},
			Env: []string{
				"LDAP_URI=ldap://openldap:1389",
				"LDAP_BASE=dc=example,dc=org",
				"LDAP_BIND_DN=cn=admin,dc=example,dc=org",
				"LDAP_BIND_PASSWORD=adminpassword",
			},
		},
		&container.HostConfig{
			PortBindings: nat.PortMap{
				"1389/tcp": []nat.PortBinding{
					{
						HostIP:   "0.0.0.0",
						HostPort: "1389",
					},
				},
			},
		}, nil, nil, "")
	if err != nil {
		t.Fatal(err)
	}

	if err := c.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		defer c.Close()
		if err := c.ContainerRemove(ctx, resp.ID, container.RemoveOptions{RemoveVolumes: true, Force: true}); err != nil {
			t.Error(err)
		}
	})
}

func getLDAPUserEntryUUID() (string, error) {
	// Connect to the LDAP server
	l, err := ldap.DialURL("ldap://localhost:1389")
	if err != nil {
		return "", fmt.Errorf("failed to connect to LDAP server: %w", err)
	}
	defer l.Close()

	err = l.Bind("cn=admin,dc=example,dc=org", "adminpassword")
	if err != nil {
		return "", fmt.Errorf("failed to bind to LDAP server: %w", err)
	}

	searchRequest := ldap.NewSearchRequest(
		"dc=example,dc=org",
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 1, 0, false,
		"(cn=User1)", []string{"entryUUID"}, nil,
	)

	sr, err := l.Search(searchRequest)
	if err != nil {
		return "", fmt.Errorf("failed to execute search: %w", err)
	}

	// Process search results
	if len(sr.Entries) == 0 {
		return "", errors.New("no entries found for the specified username.")
	}
	entry := sr.Entries[0]
	entryUUID := entry.GetAttributeValue("entryUUID")
	if entryUUID == "" {
		return "", errors.New("entryUUID is empty")
	}
	return entryUUID, nil
}
