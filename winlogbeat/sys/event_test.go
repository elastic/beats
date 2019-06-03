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

package sys

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const allXML = `
<Event xmlns="http://schemas.microsoft.com/win/2004/08/events/event">
  <System>
    <Provider Name="Microsoft-Windows-WinRM" Guid="{a7975c8f-ac13-49f1-87da-5a984a4ab417}" EventSourceName="Service Control Manager"/>
    <EventID>91</EventID>
    <Version>0</Version>
    <Level>4</Level>
    <Task>9</Task>
    <Opcode>0</Opcode>
    <Keywords>0x4000000000000004</Keywords>
    <TimeCreated SystemTime="2016-01-28T20:33:27.990735300Z"/>
    <EventRecordID>100</EventRecordID>
    <Correlation ActivityID="{A066CCF1-8AB3-459B-B62F-F79F957A5036}" RelatedActivityID="{85FC0930-9C49-42DA-804B-A7368104BD1B}" />
    <Execution ProcessID="920" ThreadID="1152"/>
    <Channel>Microsoft-Windows-WinRM/Operational</Channel>
    <Computer>vagrant-2012-r2</Computer>
    <Security UserID="S-1-5-21-3541430928-2051711210-1391384369-1001"/>
  </System>
  <EventData>
    <Data Name="param1">winlogbeat</Data>
    <Data Name="param2">running</Data>
    <Binary>770069006E006C006F00670062006500610074002F0034000000</Binary>
  </EventData>
  <UserData>
    <EventXML xmlns="Event_NS">
      <ServerName>\\VAGRANT-2012-R2</ServerName>
      <UserName>vagrant</UserName>
    </EventXML>
  </UserData>
  <ProcessingErrorData>
    <ErrorCode>15005</ErrorCode>
    <DataItemName>shellId</DataItemName>
    <EventPayload>68007400740070003A002F002F0073006300680065006D00610073002E006D006900630072006F0073006F00660074002E0063006F006D002F007700620065006D002F00770073006D0061006E002F0031002F00770069006E0064006F00770073002F007300680065006C006C002F0063006D0064000000</EventPayload>
  </ProcessingErrorData>
  <RenderingInfo Culture="en-US">
    <Message>Creating WSMan shell on server with ResourceUri: %1</Message>
    <Level>Information</Level>
    <Task>Request handling</Task>
    <Opcode>Info</Opcode>
    <Channel>Microsoft-Windows-WinRM/Operational</Channel>
    <Provider>Microsoft-Windows-Windows Remote Management</Provider>
    <Keywords>
      <Keyword>Server</Keyword>
    </Keywords>
  </RenderingInfo>
</Event>
`

func TestXML(t *testing.T) {
	allXMLTimeCreated, _ := time.Parse(time.RFC3339Nano, "2016-01-28T20:33:27.990735300Z")

	var tests = []struct {
		xml   string
		event Event
	}{
		{
			xml: allXML,
			event: Event{
				Provider: Provider{
					Name:            "Microsoft-Windows-WinRM",
					GUID:            "{a7975c8f-ac13-49f1-87da-5a984a4ab417}",
					EventSourceName: "Service Control Manager",
				},
				EventIdentifier: EventIdentifier{ID: 91},
				LevelRaw:        4,
				TaskRaw:         9,
				TimeCreated:     TimeCreated{allXMLTimeCreated},
				RecordID:        100,
				Correlation:     Correlation{"{A066CCF1-8AB3-459B-B62F-F79F957A5036}", "{85FC0930-9C49-42DA-804B-A7368104BD1B}"},
				Execution:       Execution{ProcessID: 920, ThreadID: 1152},
				Channel:         "Microsoft-Windows-WinRM/Operational",
				Computer:        "vagrant-2012-r2",
				User:            SID{Identifier: "S-1-5-21-3541430928-2051711210-1391384369-1001"},
				EventData: EventData{
					Pairs: []KeyValue{
						{"param1", "winlogbeat"},
						{"param2", "running"},
						{"Binary", "770069006E006C006F00670062006500610074002F0034000000"},
					},
				},
				UserData: UserData{
					Name: xml.Name{
						Local: "EventXML",
						Space: "Event_NS",
					},
					Pairs: []KeyValue{
						{"ServerName", `\\VAGRANT-2012-R2`},
						{"UserName", "vagrant"},
					},
				},
				Message:                 "Creating WSMan shell on server with ResourceUri: %1",
				Level:                   "Information",
				Task:                    "Request handling",
				Opcode:                  "Info",
				Keywords:                []string{"Server"},
				RenderErrorCode:         15005,
				RenderErrorDataItemName: "shellId",
			},
		},
		{
			xml: `
<Event>
  <UserData>
    <Operation_ClientFailure xmlns='http://manifests.microsoft.com/win/2006/windows/WMI'>
      <Id>{00000000-0000-0000-0000-000000000000}</Id>
    </Operation_ClientFailure>
  </UserData>
</Event>
			`,
			event: Event{
				UserData: UserData{
					Name: xml.Name{
						Local: "Operation_ClientFailure",
						Space: "http://manifests.microsoft.com/win/2006/windows/WMI",
					},
					Pairs: []KeyValue{
						{"Id", "{00000000-0000-0000-0000-000000000000}"},
					},
				},
			},
		},
	}

	for _, test := range tests {
		event, err := UnmarshalEventXML([]byte(test.xml))
		if err != nil {
			t.Error(err)
			continue
		}
		assert.Equal(t, test.event, event)

		if testing.Verbose() {
			json, err := json.MarshalIndent(event, "", "  ")
			if err != nil {
				t.Error(err)
			}
			fmt.Println(string(json))
		}
	}
}

// Tests that control characters other than CR and LF are escaped
// when the event is decoded.
func TestInvalidXML(t *testing.T) {
	evXML := strings.Replace(allXML, "%1", "\t&#xD;\n\x1b", -1)
	ev, err := UnmarshalEventXML([]byte(evXML))
	assert.Equal(t, nil, err)
	assert.Equal(t, "Creating WSMan shell on server with ResourceUri: \t\r\n\\u001b", ev.Message)
}

func BenchmarkXMLUnmarshal(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := UnmarshalEventXML([]byte(allXML))
		if err != nil {
			b.Fatal(err)
		}
	}
}
