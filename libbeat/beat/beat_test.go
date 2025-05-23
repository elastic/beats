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

package beat

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
	"github.com/elastic/elastic-agent-libs/config"
)

type testManager struct {
	isUnpriv  bool
	mgmtMode  proto.AgentManagedMode
	isEnabled bool
}

func (tm testManager) UpdateStatus(_ status.Status, _ string) {}
func (tm testManager) Enabled() bool                          { return tm.isEnabled }
func (tm testManager) Start() error                           { return nil }
func (tm testManager) Stop()                                  {}
func (tm testManager) AgentInfo() client.AgentInfo {
	return client.AgentInfo{Unprivileged: tm.isUnpriv, ManagedMode: tm.mgmtMode}
}
func (tm testManager) SetStopCallback(_ func())            {}
func (tm testManager) CheckRawConfig(_ *config.C) error    { return nil }
func (tm testManager) RegisterAction(_ client.Action)      {}
func (tm testManager) UnregisterAction(_ client.Action)    {}
func (tm testManager) SetPayload(_ map[string]interface{}) {}
func (tm testManager) RegisterDiagnosticHook(_ string, _ string, _ string, _ string, _ client.DiagnosticHook) {
}

func TestUserAgentString(t *testing.T) {
	tests := []struct {
		beat             *Beat
		expectedComments []string
		name             string
	}{
		{
			name: "managed-unprivileged",
			beat: &Beat{Info: Info{Beat: "testbeat"},
				Manager: testManager{isEnabled: true, isUnpriv: true, mgmtMode: proto.AgentManagedMode_MANAGED}},
			expectedComments: []string{"Managed", "Unprivileged"},
		},
		{
			name: "managed-privileged",
			beat: &Beat{Info: Info{Beat: "testbeat"},
				Manager: testManager{isEnabled: true, isUnpriv: false, mgmtMode: proto.AgentManagedMode_MANAGED}},
			expectedComments: []string{"Managed"},
		},
		{
			name: "unmanaged-privileged",
			beat: &Beat{Info: Info{Beat: "testbeat"},
				Manager: testManager{isEnabled: true, isUnpriv: false, mgmtMode: proto.AgentManagedMode_STANDALONE}},
			expectedComments: []string{"Standalone"},
		},
		{
			name: "unmanaged-unprivileged",
			beat: &Beat{Info: Info{Beat: "testbeat"},
				Manager: testManager{isEnabled: true, isUnpriv: true, mgmtMode: proto.AgentManagedMode_STANDALONE}},
			expectedComments: []string{"Standalone", "Unprivileged"},
		},
		{
			name: "management-disabled",
			beat: &Beat{Info: Info{Beat: "testbeat"},
				Manager: testManager{isEnabled: false}},
			expectedComments: []string{},
		},
	}

	// User-Agent will take the form of
	// Elastic-testbeat/8.15.0 (linux; amd64; unknown; 0001-01-01 00:00:00 +0000 UTC; Standalone; Unprivileged)
	// the RFC (https://www.rfc-editor.org/rfc/rfc9110#name-user-agent) says the comment field can basically be anything,
	// but we put metadata in it, delimited by '; '
	uaReg := regexp.MustCompile(`Elastic-testbeat/([\d.]+) \(([\w-:+; ]+)\)`)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.beat.GenerateUserAgent()
			res := uaReg.FindAllStringSubmatch(test.beat.Info.UserAgent, -1)
			// check to make sure the regex passed, then verify the comments section
			require.NotEmpty(t, res[0])
			comments := strings.Split(res[0][2], "; ")

			for _, comment := range test.expectedComments {
				require.Contains(t, comments, comment)
			}
			// if no extra comment parts expected, check to make sure none have been added
			if len(test.expectedComments) == 0 {
				for _, comment := range []string{"Unprivileged", "Standalone", "Managed"} {
					require.NotContains(t, comments, comment)
				}
			}
		})
	}
}
