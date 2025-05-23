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
	"path/filepath"
	"testing"
	"time"
)

func TestCAPinningGoodSHA(t *testing.T) {
	EnsureESIsRunning(t)
	esURL := GetESURL(t, "https")
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	caPath := filepath.Join(mockbeat.TempDir(), "../../../../", "testing", "environments", "docker", "elasticsearch", "pki", "ca", "ca.crt")
	cfg := `
mockbeat:
name:
logging:
  level: debug
queue.mem:
  events: 4096
  flush.min_events: 8
  flush.timeout: 0.1s
output.elasticsearch:
  hosts:
    - %s
  username: admin
  password: testing
  allow_older_versions: true
  ssl:
    verification_mode: certificate
    certificate_authorities: %s
    ca_sha256: FDFOtqdUyXZw74YgvAJUC+I67ED1WfcI1qK44Qy2WQM=
`
	mockbeat.WriteConfigFile(fmt.Sprintf(cfg, esURL.String(), caPath))
	mockbeat.Start()
	mockbeat.WaitForLogs("mockbeat start running.", 60*time.Second)
	mockbeat.WaitForLogs("doBulkRequest: 1 events have been sent", 60*time.Second)
}

func TestCAPinningBadSHA(t *testing.T) {
	EnsureESIsRunning(t)
	esURL := GetESURL(t, "https")
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	caPath := filepath.Join(mockbeat.TempDir(), "../../../../", "testing", "environments", "docker", "elasticsearch", "pki", "ca", "ca.crt")
	cfg := `
mockbeat:
name:
logging:
  level: debug
queue.mem:
  events: 4096
  flush.min_events: 8
  flush.timeout: 0.1s
output.elasticsearch:
  hosts:
    - %s
  username: admin
  password: testing
  allow_older_versions: true
  ssl:
    verification_mode: certificate
    certificate_authorities: %s
    ca_sha256: bad
`
	mockbeat.WriteConfigFile(fmt.Sprintf(cfg, esURL.String(), caPath))
	mockbeat.Start()
	mockbeat.WaitForLogs("mockbeat start running.", 60*time.Second)
	mockbeat.WaitForLogs("provided CA certificate pins doesn't match any of the certificate authorities used to validate the certificate", 60*time.Second)
}
