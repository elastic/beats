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
	startOpenldapContainer(t)

	entryUUID := waitForLDAPUser(t, "User1")

	filebeat := integration.NewBeat(t, "filebeat", "../../filebeat.test")
	tempDir := filebeat.TempDir()

	logFilePath := path.Join(tempDir, "log.log")
	integration.WriteLogFile(t, logFilePath, 1, false)

	filebeat.WriteConfigFile(fmt.Sprintf(translateguidCfg, logFilePath, tempDir, entryUUID))
	filebeat.Start()

	outputFile := waitForOutputFile(t, tempDir)

	// Verify the GUID and translated common name are present (don't check field order)
	filebeat.WaitFileContains(outputFile, fmt.Sprintf(`"guid":"%s"`, entryUUID), 20*time.Second)
	filebeat.WaitFileContains(outputFile, `"common_name":["User1","user01"]`, 5*time.Second)
}

const translateMultipleCfg = `
filebeat.inputs:
  - type: filestream
    id: "test-translateMultipleCfg"
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
  - decode_json_fields:
      fields: ["message"]
      target: ""
  - translate_ldap_attribute:
      field: guid
      target_field: common_name
      ldap_address: 'ldap://localhost:1389'
      ldap_base_dn: 'dc=example,dc=org'
      ldap_bind_user: 'cn=admin,dc=example,dc=org'
      ldap_bind_password: 'adminpassword'
      ldap_search_attribute: 'entryUUID'
      ignore_missing: true
      ignore_failure: true
`

func TestTranslateGUIDWithMultipleCallsAndFailures(t *testing.T) {
	startOpenldapContainer(t)

	entryUUID1 := waitForLDAPUser(t, "User1")
	entryUUID2 := waitForLDAPUser(t, "User2")

	filebeat := integration.NewBeat(t, "filebeat", "../../filebeat.test")
	tempDir := filebeat.TempDir()
	logFilePath := path.Join(tempDir, "log.log")

	// Write multiple log entries with mixed valid and invalid GUIDs
	entries := []string{
		fmt.Sprintf(`{"guid":"%s","message":"valid entry 1"}`, entryUUID1),
		`{"guid":"00000000-0000-0000-0000-000000000000","message":"invalid entry 1"}`,
		fmt.Sprintf(`{"guid":"%s","message":"valid entry 2"}`, entryUUID2),
		`{"guid":"11111111-1111-1111-1111-111111111111","message":"invalid entry 2"}`,
		fmt.Sprintf(`{"guid":"%s","message":"valid entry 3"}`, entryUUID1),
		`{"guid":"22222222-2222-2222-2222-222222222222","message":"invalid entry 3"}`,
		fmt.Sprintf(`{"guid":"%s","message":"valid entry 4"}`, entryUUID2),
		`{"message":"no guid field"}`,
		fmt.Sprintf(`{"guid":"%s","message":"valid entry 5"}`, entryUUID1),
		`{"guid":"33333333-3333-3333-3333-333333333333","message":"invalid entry 4"}`,
	}

	logFile, err := os.Create(logFilePath)
	require.NoError(t, err)
	for _, entry := range entries {
		_, err := logFile.WriteString(entry + "\n")
		require.NoError(t, err)
	}
	logFile.Close()

	filebeat.WriteConfigFile(fmt.Sprintf(translateMultipleCfg, logFilePath, tempDir))
	filebeat.Start()

	outputFile := waitForOutputFile(t, tempDir)

	// Verify valid entries are processed with translated GUIDs
	filebeat.WaitFileContains(outputFile, fmt.Sprintf(`"guid":"%s"`, entryUUID1), 30*time.Second)
	filebeat.WaitFileContains(outputFile, `"common_name":["User1","user01"]`, 5*time.Second)
	filebeat.WaitFileContains(outputFile, fmt.Sprintf(`"guid":"%s"`, entryUUID2), 5*time.Second)
	filebeat.WaitFileContains(outputFile, `"common_name":["User2","user02"]`, 5*time.Second)

	// Verify invalid entries are also processed (processor didn't hang)
	filebeat.WaitFileContains(outputFile, `"guid":"00000000-0000-0000-0000-000000000000"`, 5*time.Second)
	filebeat.WaitFileContains(outputFile, `"guid":"33333333-3333-3333-3333-333333333333"`, 5*time.Second)
	filebeat.WaitFileContains(outputFile, `no guid field`, 5*time.Second)
}

const translateConcurrentCfg = `
filebeat.inputs:
  - type: filestream
    id: "test-translateConcurrentCfg-1"
    file_identity.native: ~
    prospector.scanner.fingerprint.enabled: false
    paths:
      - %s

  - type: filestream
    id: "test-translateConcurrentCfg-2"
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
  - decode_json_fields:
      fields: ["message"]
      target: ""
  - translate_ldap_attribute:
      field: guid
      target_field: common_name
      ldap_address: 'ldap://localhost:1389'
      ldap_base_dn: 'dc=example,dc=org'
      ldap_bind_user: 'cn=admin,dc=example,dc=org'
      ldap_bind_password: 'adminpassword'
      ldap_search_attribute: 'entryUUID'
      ignore_missing: true
      ignore_failure: true
`

func TestTranslateGUIDWithConcurrentCalls(t *testing.T) {
	startOpenldapContainer(t)

	entryUUID1 := waitForLDAPUser(t, "User1")
	entryUUID2 := waitForLDAPUser(t, "User2")

	filebeat := integration.NewBeat(t, "filebeat", "../../filebeat.test")
	tempDir := filebeat.TempDir()

	// Create two separate log files that will be read concurrently
	logFilePath1 := path.Join(tempDir, "log1.log")
	logFilePath2 := path.Join(tempDir, "log2.log")

	// Write entries to first log file
	entries1 := []string{
		fmt.Sprintf(`{"guid":"%s","message":"concurrent file 1 - entry 1"}`, entryUUID1),
		fmt.Sprintf(`{"guid":"%s","message":"concurrent file 1 - entry 2"}`, entryUUID2),
		fmt.Sprintf(`{"guid":"%s","message":"concurrent file 1 - entry 3"}`, entryUUID1),
		fmt.Sprintf(`{"guid":"%s","message":"concurrent file 1 - entry 4"}`, entryUUID2),
		fmt.Sprintf(`{"guid":"%s","message":"concurrent file 1 - entry 5"}`, entryUUID1),
	}

	// Write entries to second log file
	entries2 := []string{
		fmt.Sprintf(`{"guid":"%s","message":"concurrent file 2 - entry 1"}`, entryUUID2),
		fmt.Sprintf(`{"guid":"%s","message":"concurrent file 2 - entry 2"}`, entryUUID1),
		fmt.Sprintf(`{"guid":"%s","message":"concurrent file 2 - entry 3"}`, entryUUID2),
		fmt.Sprintf(`{"guid":"%s","message":"concurrent file 2 - entry 4"}`, entryUUID1),
		fmt.Sprintf(`{"guid":"%s","message":"concurrent file 2 - entry 5"}`, entryUUID2),
	}

	logFile1, err := os.Create(logFilePath1)
	require.NoError(t, err)
	for _, entry := range entries1 {
		_, err := logFile1.WriteString(entry + "\n")
		require.NoError(t, err)
	}
	logFile1.Close()

	logFile2, err := os.Create(logFilePath2)
	require.NoError(t, err)
	for _, entry := range entries2 {
		_, err := logFile2.WriteString(entry + "\n")
		require.NoError(t, err)
	}
	logFile2.Close()

	filebeat.WriteConfigFile(fmt.Sprintf(translateConcurrentCfg, logFilePath1, logFilePath2, tempDir))
	filebeat.Start()

	outputFile := waitForOutputFile(t, tempDir)

	// Verify entries from both files are processed correctly with translations
	filebeat.WaitFileContains(outputFile, `concurrent file 1 - entry 1`, 30*time.Second)
	filebeat.WaitFileContains(outputFile, `concurrent file 2 - entry 1`, 5*time.Second)

	// Verify both users' common names appear (proving concurrent lookups worked)
	filebeat.WaitFileContains(outputFile, `"common_name":["User1","user01"]`, 5*time.Second)
	filebeat.WaitFileContains(outputFile, `"common_name":["User2","user02"]`, 5*time.Second)

	// Verify multiple entries from each file were processed
	filebeat.WaitFileContains(outputFile, `concurrent file 1 - entry 5`, 5*time.Second)
	filebeat.WaitFileContains(outputFile, `concurrent file 2 - entry 5`, 5*time.Second)
}

const translateConcurrentWorkersCfg = `
filebeat.inputs:
  - type: filestream
    id: "test-translateConcurrentWorkersCfg"
    file_identity.native: ~
    prospector.scanner.fingerprint.enabled: false
    paths:
      - %s

queue.mem:
  flush.min_events: 1
  flush.timeout: 0.1s

# Enable multiple pipeline workers for concurrent processing
queue.mem.events: 4096
pipeline.workers: 4

path.home: %s

output.file:
  path: ${path.home}
  filename: "output-file"

logging:
  metrics:
    enabled: false

processors:
  - decode_json_fields:
      fields: ["message"]
      target: ""
  - translate_ldap_attribute:
      field: guid
      target_field: common_name
      ldap_address: 'ldap://localhost:1389'
      ldap_base_dn: 'dc=example,dc=org'
      ldap_bind_user: 'cn=admin,dc=example,dc=org'
      ldap_bind_password: 'adminpassword'
      ldap_search_attribute: 'entryUUID'
      ignore_missing: true
      ignore_failure: true
`

func TestTranslateGUIDWithConcurrentWorkersInSameInput(t *testing.T) {
	startOpenldapContainer(t)

	entryUUID1 := waitForLDAPUser(t, "User1")
	entryUUID2 := waitForLDAPUser(t, "User2")

	filebeat := integration.NewBeat(t, "filebeat", "../../filebeat.test")
	tempDir := filebeat.TempDir()
	logFilePath := path.Join(tempDir, "log.log")

	// Create a larger batch of entries that will be processed by multiple workers
	entries := []string{}
	for i := 1; i <= 20; i++ {
		// Alternate between User1 and User2 to ensure concurrent LDAP lookups
		if i%2 == 0 {
			entries = append(entries, fmt.Sprintf(`{"guid":"%s","message":"worker test entry %d"}`, entryUUID1, i))
		} else {
			entries = append(entries, fmt.Sprintf(`{"guid":"%s","message":"worker test entry %d"}`, entryUUID2, i))
		}
	}

	logFile, err := os.Create(logFilePath)
	require.NoError(t, err)
	for _, entry := range entries {
		_, err := logFile.WriteString(entry + "\n")
		require.NoError(t, err)
	}
	logFile.Close()

	filebeat.WriteConfigFile(fmt.Sprintf(translateConcurrentWorkersCfg, logFilePath, tempDir))
	filebeat.Start()

	outputFile := waitForOutputFile(t, tempDir)

	// Verify that multiple entries are processed with translations
	filebeat.WaitFileContains(outputFile, `worker test entry 1`, 30*time.Second)
	filebeat.WaitFileContains(outputFile, `worker test entry 20`, 10*time.Second)

	// Verify both users' common names appear (proving concurrent workers processed different events)
	filebeat.WaitFileContains(outputFile, `"common_name":["User1","user01"]`, 5*time.Second)
	filebeat.WaitFileContains(outputFile, `"common_name":["User2","user02"]`, 5*time.Second)

	// Verify multiple entries from different parts of the batch
	filebeat.WaitFileContains(outputFile, `worker test entry 5`, 5*time.Second)
	filebeat.WaitFileContains(outputFile, `worker test entry 10`, 5*time.Second)
	filebeat.WaitFileContains(outputFile, `worker test entry 15`, 5*time.Second)
}

func startOpenldapContainer(t *testing.T) {
	ctx := context.Background()
	c, err := docker.NewClient(client.DefaultDockerHost, nil, nil, logptest.NewTestingLogger(t, ""))
	if err != nil {
		t.Fatal(err)
	}

	reader, err := c.ImagePull(ctx, "osixia/openldap:1.5.0", image.PullOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = io.Copy(os.Stdout, reader); err != nil {
		t.Fatal(err)
	}
	reader.Close()

	resp, err := c.ContainerCreate(ctx,
		&container.Config{
			Image: "osixia/openldap:1.5.0",
			ExposedPorts: nat.PortSet{
				"389/tcp": struct{}{},
			},
			Env: []string{
				"LDAP_ORGANISATION=example",
				"LDAP_DOMAIN=example.org",
				"LDAP_ADMIN_PASSWORD=adminpassword",
			},
		},
		&container.HostConfig{
			PortBindings: nat.PortMap{
				"389/tcp": []nat.PortBinding{
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

	// Wait for LDAP to be ready and add test users
	require.Eventually(t, func() bool {
		err := addTestUserToLDAP()
		return err == nil
	}, 30*time.Second, time.Second, "Failed to add test users to LDAP")
}

func connectToLDAP() (*ldap.Conn, error) {
	l, err := ldap.DialURL("ldap://localhost:1389")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to LDAP server: %w", err)
	}

	if err = l.Bind("cn=admin,dc=example,dc=org", "adminpassword"); err != nil {
		l.Close()
		return nil, fmt.Errorf("failed to bind to LDAP server: %w", err)
	}

	return l, nil
}

func addTestUserToLDAP() error {
	l, err := connectToLDAP()
	if err != nil {
		return err
	}
	defer l.Close()

	users := []struct {
		cn, uid string
	}{
		{"User1", "user01"},
		{"User2", "user02"},
	}

	for _, user := range users {
		addRequest := ldap.NewAddRequest(fmt.Sprintf("cn=%s,dc=example,dc=org", user.cn), nil)
		addRequest.Attribute("objectClass", []string{"inetOrgPerson", "organizationalPerson", "person", "top"})
		addRequest.Attribute("cn", []string{user.cn, user.uid})
		addRequest.Attribute("sn", []string{user.cn})
		addRequest.Attribute("uid", []string{user.uid})

		if err := l.Add(addRequest); err != nil {
			return fmt.Errorf("failed to add test user %s: %w", user.cn, err)
		}
	}

	return nil
}

func getLDAPUserEntryUUID(username string) (string, error) {
	l, err := connectToLDAP()
	if err != nil {
		return "", err
	}
	defer l.Close()

	searchRequest := ldap.NewSearchRequest(
		"dc=example,dc=org",
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 1, 0, false,
		fmt.Sprintf("(cn=%s)", username), []string{"entryUUID"}, nil,
	)

	sr, err := l.Search(searchRequest)
	if err != nil {
		return "", fmt.Errorf("failed to execute search: %w", err)
	}

	if len(sr.Entries) == 0 {
		return "", fmt.Errorf("no entries found for username: %s", username)
	}

	entryUUID := sr.Entries[0].GetAttributeValue("entryUUID")
	if entryUUID == "" {
		return "", fmt.Errorf("entryUUID is empty for username: %s", username)
	}

	return entryUUID, nil
}

func waitForLDAPUser(t *testing.T, username string) string {
	var entryUUID string
	require.Eventually(t, func() bool {
		var err error
		entryUUID, err = getLDAPUserEntryUUID(username)
		return err == nil
	}, 10*time.Second, time.Second)
	return entryUUID
}

func waitForOutputFile(t *testing.T, tempDir string) string {
	var outputFile string
	require.Eventually(t, func() bool {
		outputFiles, err := filepath.Glob(path.Join(tempDir, "output-file-*.ndjson"))
		if err != nil || len(outputFiles) != 1 {
			return false
		}
		outputFile = outputFiles[0]
		return true
	}, 10*time.Second, time.Second)
	return outputFile
}
