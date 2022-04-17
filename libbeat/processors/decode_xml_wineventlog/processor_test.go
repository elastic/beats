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

package decode_xml_wineventlog

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/menderesk/beats/v7/libbeat/beat"
	"github.com/menderesk/beats/v7/libbeat/common"
)

var testMessage = "<Event xmlns='http://schemas.microsoft.com/win/2004/08/events/event'><System><Provider Name='Microsoft-Windows-Security-Auditing' Guid='{54849625-5478-4994-a5ba-3e3b0328c30d}'/>" +
	"<EventID>4672</EventID><Version>0</Version><Level>0</Level><Task>12548</Task><Opcode>0</Opcode><Keywords>0x8020000000000000</Keywords><TimeCreated SystemTime='2021-03-23T09:56:13.137310000Z'/>" +
	"<EventRecordID>11303</EventRecordID><Correlation ActivityID='{ffb23523-1f32-0000-c335-b2ff321fd701}'/><Execution ProcessID='652' ThreadID='4660'/><Channel>Security</Channel><Computer>vagrant</Computer>" +
	"<Security/></System><EventData><Data Name='SubjectUserSid'>S-1-5-18</Data><Data Name='SubjectUserName'>SYSTEM</Data><Data Name='SubjectDomainName'>NT AUTHORITY</Data><Data Name='SubjectLogonId'>0x3e7</Data>" +
	"<Data Name='PrivilegeList'>SeAssignPrimaryTokenPrivilege\n\t\t\tSeTcbPrivilege\n\t\t\tSeSecurityPrivilege\n\t\t\tSeTakeOwnershipPrivilege\n\t\t\tSeLoadDriverPrivilege\n\t\t\tSeBackupPrivilege\n\t\t\t" +
	"SeRestorePrivilege\n\t\t\tSeDebugPrivilege\n\t\t\tSeAuditPrivilege\n\t\t\tSeSystemEnvironmentPrivilege\n\t\t\tSeImpersonatePrivilege\n\t\t\tSeDelegateSessionUserImpersonatePrivilege</Data></EventData>" +
	"<RenderingInfo Culture='en-US'><Message>Special privileges assigned to new logon.\n\nSubject:\n\tSecurity ID:\t\tS-1-5-18\n\tAccount Name:\t\tSYSTEM\n\tAccount Domain:\t\tNT AUTHORITY\n\tLogon ID:\t\t0x3E7\n\n" +
	"Privileges:\t\tSeAssignPrimaryTokenPrivilege\n\t\t\tSeTcbPrivilege\n\t\t\tSeSecurityPrivilege\n\t\t\tSeTakeOwnershipPrivilege\n\t\t\tSeLoadDriverPrivilege\n\t\t\tSeBackupPrivilege\n\t\t\tSeRestorePrivilege\n\t\t\t" +
	"SeDebugPrivilege\n\t\t\tSeAuditPrivilege\n\t\t\tSeSystemEnvironmentPrivilege\n\t\t\tSeImpersonatePrivilege\n\t\t\tSeDelegateSessionUserImpersonatePrivilege</Message><Level>Information</Level>" +
	"<Task>Special Logon</Task><Opcode>Info</Opcode><Channel>Security</Channel><Provider>Microsoft Windows security auditing.</Provider><Keywords><Keyword>Audit Success</Keyword></Keywords></RenderingInfo></Event>"

var testMessageOutput = common.MapStr{
	"event": common.MapStr{
		"action":   "Special Logon",
		"code":     "4672",
		"kind":     "event",
		"outcome":  "success",
		"provider": "Microsoft-Windows-Security-Auditing",
	},
	"host": common.MapStr{
		"name": "vagrant",
	},
	"log": common.MapStr{
		"level": "information",
	},
	"winlog": common.MapStr{
		"channel":       "Security",
		"outcome":       "success",
		"activity_id":   "{ffb23523-1f32-0000-c335-b2ff321fd701}",
		"level":         "information",
		"event_id":      "4672",
		"provider_name": "Microsoft-Windows-Security-Auditing",
		"record_id":     uint64(11303),
		"computer_name": "vagrant",
		"time_created": func() time.Time {
			t, _ := time.Parse(time.RFC3339Nano, "2021-03-23T09:56:13.137310000Z")
			return t
		}(),
		"opcode":        "Info",
		"provider_guid": "{54849625-5478-4994-a5ba-3e3b0328c30d}",
		"event_data": common.MapStr{
			"SubjectUserSid":    "S-1-5-18",
			"SubjectUserName":   "SYSTEM",
			"SubjectDomainName": "NT AUTHORITY",
			"SubjectLogonId":    "0x3e7",
			"PrivilegeList":     "SeAssignPrimaryTokenPrivilege\n\t\t\tSeTcbPrivilege\n\t\t\tSeSecurityPrivilege\n\t\t\tSeTakeOwnershipPrivilege\n\t\t\tSeLoadDriverPrivilege\n\t\t\tSeBackupPrivilege\n\t\t\tSeRestorePrivilege\n\t\t\tSeDebugPrivilege\n\t\t\tSeAuditPrivilege\n\t\t\tSeSystemEnvironmentPrivilege\n\t\t\tSeImpersonatePrivilege\n\t\t\tSeDelegateSessionUserImpersonatePrivilege",
		},
		"task": "Special Logon",
		"keywords": []string{
			"Audit Success",
		},
		"message": "Special privileges assigned to new logon.\n\nSubject:\n\tSecurity ID:\t\tS-1-5-18\n\tAccount Name:\t\tSYSTEM\n\tAccount Domain:\t\tNT AUTHORITY\n\tLogon ID:\t\t0x3E7\n\nPrivileges:\t\tSeAssignPrimaryTokenPrivilege\n\t\t\tSeTcbPrivilege\n\t\t\tSeSecurityPrivilege\n\t\t\tSeTakeOwnershipPrivilege\n\t\t\tSeLoadDriverPrivilege\n\t\t\tSeBackupPrivilege\n\t\t\tSeRestorePrivilege\n\t\t\tSeDebugPrivilege\n\t\t\tSeAuditPrivilege\n\t\t\tSeSystemEnvironmentPrivilege\n\t\t\tSeImpersonatePrivilege\n\t\t\tSeDelegateSessionUserImpersonatePrivilege",
		"process": common.MapStr{
			"pid": uint32(652),
			"thread": common.MapStr{
				"id": uint32(4660),
			},
		},
	},
	"message": "Special privileges assigned to new logon.\n\nSubject:\n\tSecurity ID:\t\tS-1-5-18\n\tAccount Name:\t\tSYSTEM\n\tAccount Domain:\t\tNT AUTHORITY\n\tLogon ID:\t\t0x3E7\n\nPrivileges:\t\tSeAssignPrimaryTokenPrivilege\n\t\t\tSeTcbPrivilege\n\t\t\tSeSecurityPrivilege\n\t\t\tSeTakeOwnershipPrivilege\n\t\t\tSeLoadDriverPrivilege\n\t\t\tSeBackupPrivilege\n\t\t\tSeRestorePrivilege\n\t\t\tSeDebugPrivilege\n\t\t\tSeAuditPrivilege\n\t\t\tSeSystemEnvironmentPrivilege\n\t\t\tSeImpersonatePrivilege\n\t\t\tSeDelegateSessionUserImpersonatePrivilege",
}

func TestProcessor(t *testing.T) {
	var testCases = []struct {
		description  string
		config       config
		Input        common.MapStr
		Output       common.MapStr
		error        bool
		errorMessage string
	}{
		{
			description: "Decodes properly with default config",
			config:      defaultConfig(),
			Input: common.MapStr{
				"message": testMessage,
			},
			Output: testMessageOutput,
		},
		{
			description: "Decodes without ECS",
			config: config{
				Field:         "message",
				OverwriteKeys: true,
				MapECSFields:  false,
				Target:        "winlog",
			},
			Input: common.MapStr{
				"message": testMessage,
			},
			Output: common.MapStr{
				"winlog":  testMessageOutput["winlog"],
				"message": testMessage,
			},
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.description, func(t *testing.T) {
			t.Parallel()

			f, err := newProcessor(test.config)
			require.NoError(t, err)

			event := &beat.Event{
				Fields: test.Input,
			}
			newEvent, err := f.Run(event)
			if !test.error {
				assert.NoError(t, err)
			} else {
				if assert.Error(t, err) {
					assert.Contains(t, err.Error(), test.errorMessage)
				}
			}
			assert.Equal(t, test.Output, newEvent.Fields)
		})
	}

	t.Run("supports metadata as a target", func(t *testing.T) {
		t.Parallel()

		config := defaultConfig()
		config.Field = "@metadata.message"
		config.Target = "@metadata.target"

		f, err := newProcessor(config)
		require.NoError(t, err)

		event := &beat.Event{
			Fields: common.MapStr{},
			Meta: common.MapStr{
				"message": testMessage,
			},
		}

		expMeta := common.MapStr{
			"message": testMessage,
			"target":  testMessageOutput["winlog"],
		}
		newEvent, err := f.Run(event)
		assert.NoError(t, err)
		assert.Equal(t, expMeta, newEvent.Meta)
		assert.Equal(t, event.Fields, newEvent.Fields)
	})
}
