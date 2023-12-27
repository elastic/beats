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

package winevent

import (
	"encoding/json"
	"encoding/xml"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

const (
	allXML = `
<Event xmlns="http://schemas.microsoft.com/win/2004/08/events/event">
  <System>
    <Provider Name="Microsoft-Windows-WinRM" Guid="{a7975c8f-ac13-49f1-87da-5a984a4ab417}" EventSourceName="Service Control Manager"/>
    <EventID>91</EventID>
    <Version>1</Version>
    <Level>4</Level>
    <Task>9</Task>
    <Opcode>0</Opcode>
    <Keywords>0x8020000000000000</Keywords>
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

	mensajeEnEspañol = `
<Event xmlns="http://schemas.microsoft.com/win/2004/08/events/event">
  <System>
    <Provider Name="Microsoft-Windows-PowerShell" Guid="{b51b54b5-04a1-4c65-8760-04f9e2335cd5}" EventSourceName="Service Control Manager"/>
    <EventID>4103</EventID>
    <Version>1</Version>
    <Level>4</Level>
    <Task>9</Task>
    <Opcode>0</Opcode>
    <Keywords>0x8020000000000000</Keywords>
    <TimeCreated SystemTime="2016-01-28T20:33:27.990735300Z"/>
    <EventRecordID>100</EventRecordID>
    <Correlation ActivityID="{BE97B1E6-710B-47D7-8941-50AB2A2C757F}" RelatedActivityID="{EDAD163F-52D0-4E89-BEF8-0500EC6F08AF}" />
    <Execution ProcessID="920" ThreadID="1152"/>
    <Channel>Microsoft-Windows-WinRM/Operational</Channel>
    <Computer>vagrant-2012-r2</Computer>
    <Security UserID="S-1-5-21-4564564786-2382305473-342768465-7452"/>
  </System>
  <EventData>
    <Data Name="Nombre de script">C:\WINDOWS\Administrador\service-1\exec\Invoke.ps1</Data>
    <Data Name="Id. de host">1f3cf2e0-1ae4-4672-a138-e96befdf9586</Data>
    <Data Name="Id. de shell">Microsoft.PowerShell</Data>
    <Data Name="Id. de espacio de ejecución">91bb420c-d23e-4926-912b-6d11190df3fb</Data>
    <Data Name="Id. de canalización"></Data>
    <Data Name="Usuario">GENTE\persona</Data>
    <Data Name="Payload">CommandInvocation(Get-Date): "Get-Date"</Data>
    <Data Name="Versión de host">5.1.19041.1320</Data>
    <Data Name="Gravedad">Informational</Data>
    <Data Name="Tipo de comando">Cmdlet</Data>
    <Data Name="Nombre de host">ConsoleHost</Data>
    <Data Name="Número de secuencia">4174</Data>
    <Data Name="Versión del motor">5.1.19041.1320</Data>
    <Data Name="Aplicación host">C:\WINDOWS\System32\WindowsPowerShell\V1.0\PowerShell.exe -NoLogo -NoProfile -ExecutionPolicy Bypass -Command  exit 1</Data>
    <Data Name="Nombre de comando">Get-Dat</Data>
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
    <EventPayload>5463487987956</EventPayload>
  </ProcessingErrorData>
  <RenderingInfo Culture="en-US">
    <Message>CommandInvocation(Get-Date): "Get-Date"


Contexto:
 Gravedad = Informational
 Nombre de host = ConsoleHost
 Versión de host = 5.1.19041.1320
 Id. de host = 56995afd-2444-424e-871c-4c5513731a3b
 Aplicación host = C:\WINDOWS\System32\WindowsPowerShell\V1.0\PowerShell.exe -NoLogo -NoProfile -ExecutionPolicy Bypass -Command  exit 1
 Versión del motor = 5.1.19041.1320
 Id. de espacio de ejecución = 91bb420c-d23e-4926-912b-6d11190df3fb
 Id. de canalización = 1
 Nombre de comando = Get-Date
 Tipo de comando = Cmdlet
 Nombre de script = C:\WINDOWS\Administrador\service-1\exec\Invoke.ps1
 Ruta de acceso de comando =
 Número de secuencia = 4174
 Usuario = GENTE\persona
 Usuario conectado =
 Id. de shell = Microsoft.PowerShell


Datos de usuario:</Message>
    <Level>Information</Level>
    <Task>Request handling</Task>
    <Opcode>Para usar cuando la operación solo está ejecutando un método.</Opcode>
    <Channel>Microsoft-Windows-PowerShell/Operational</Channel>
    <Provider>Microsoft-Windows-PowerShell</Provider>
  </RenderingInfo>
</Event>
`
)

func TestXML(t *testing.T) {
	allXMLTimeCreated, _ := time.Parse(time.RFC3339Nano, "2016-01-28T20:33:27.990735300Z")

	tests := []struct {
		xml    string
		event  Event
		mapstr mapstr.M
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
				OpcodeRaw:       new(uint8), // The value in the XML is 0.
				KeywordsRaw:     0x8020000000000000,
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
				Version:                 1,
			},
		},
		{
			xml: mensajeEnEspañol,
			event: Event{
				Provider: Provider{
					Name:            "Microsoft-Windows-PowerShell",
					GUID:            "{b51b54b5-04a1-4c65-8760-04f9e2335cd5}",
					EventSourceName: "Service Control Manager",
				},
				EventIdentifier: EventIdentifier{
					ID: 0x1007,
				},
				Version:     0x1,
				LevelRaw:    0x4,
				TaskRaw:     0x9,
				OpcodeRaw:   new(uint8),
				KeywordsRaw: 0x8020000000000000,
				TimeCreated: TimeCreated{allXMLTimeCreated},
				RecordID:    0x64,
				Correlation: Correlation{
					ActivityID:        "{BE97B1E6-710B-47D7-8941-50AB2A2C757F}",
					RelatedActivityID: "{EDAD163F-52D0-4E89-BEF8-0500EC6F08AF}",
				},
				Execution: Execution{
					ProcessID: 0x398,
					ThreadID:  0x480,
				},
				Channel:  "Microsoft-Windows-WinRM/Operational",
				Computer: "vagrant-2012-r2",
				User: SID{
					Identifier: "S-1-5-21-4564564786-2382305473-342768465-7452",
				},
				EventData: EventData{
					Pairs: []KeyValue{
						{Key: "Nombre de script", Value: "C:\\WINDOWS\\Administrador\\service-1\\exec\\Invoke.ps1"},
						{Key: "Id. de host", Value: "1f3cf2e0-1ae4-4672-a138-e96befdf9586"},
						{Key: "Id. de shell", Value: "Microsoft.PowerShell"},
						{Key: "Id. de espacio de ejecución", Value: "91bb420c-d23e-4926-912b-6d11190df3fb"},
						{Key: "Id. de canalización"},
						{Key: "Usuario", Value: "GENTE\\persona"},
						{Key: "Payload", Value: "CommandInvocation(Get-Date): \"Get-Date\""},
						{Key: "Versión de host", Value: "5.1.19041.1320"},
						{Key: "Gravedad", Value: "Informational"},
						{Key: "Tipo de comando", Value: "Cmdlet"},
						{Key: "Nombre de host", Value: "ConsoleHost"},
						{Key: "Número de secuencia", Value: "4174"},
						{Key: "Versión del motor", Value: "5.1.19041.1320"},
						{Key: "Aplicación host", Value: "C:\\WINDOWS\\System32\\WindowsPowerShell\\V1.0\\PowerShell.exe -NoLogo -NoProfile -ExecutionPolicy Bypass -Command  exit 1"},
						{Key: "Nombre de comando", Value: "Get-Dat"},
						{Key: "Binary", Value: "770069006E006C006F00670062006500610074002F0034000000"},
					},
				},
				UserData: UserData{
					Name: xml.Name{
						Space: "Event_NS",
						Local: "EventXML",
					},
					Pairs: []KeyValue{
						{Key: "ServerName", Value: "\\\\VAGRANT-2012-R2"},
						{Key: "UserName", Value: "vagrant"},
					},
				},
				Message:                 "CommandInvocation(Get-Date): \"Get-Date\"\n\n\nContexto:\n Gravedad = Informational\n Nombre de host = ConsoleHost\n Versión de host = 5.1.19041.1320\n Id. de host = 56995afd-2444-424e-871c-4c5513731a3b\n Aplicación host = C:\\WINDOWS\\System32\\WindowsPowerShell\\V1.0\\PowerShell.exe -NoLogo -NoProfile -ExecutionPolicy Bypass -Command  exit 1\n Versión del motor = 5.1.19041.1320\n Id. de espacio de ejecución = 91bb420c-d23e-4926-912b-6d11190df3fb\n Id. de canalización = 1\n Nombre de comando = Get-Date\n Tipo de comando = Cmdlet\n Nombre de script = C:\\WINDOWS\\Administrador\\service-1\\exec\\Invoke.ps1\n Ruta de acceso de comando =\n Número de secuencia = 4174\n Usuario = GENTE\\persona\n Usuario conectado =\n Id. de shell = Microsoft.PowerShell\n\n\nDatos de usuario:",
				Level:                   "Information",
				Task:                    "Request handling",
				Opcode:                  "Para usar cuando la operación solo está ejecutando un método.",
				RenderErrorCode:         0x3a9d,
				RenderErrorDataItemName: "shellId",
			},
			mapstr: mapstr.M{
				"activity_id":   "{BE97B1E6-710B-47D7-8941-50AB2A2C757F}",
				"channel":       "Microsoft-Windows-WinRM/Operational",
				"computer_name": "vagrant-2012-r2",
				"error": mapstr.M{
					"code": uint32(0x3a9d),
				},
				"event_data": mapstr.M{
					"Aplicación host":             "C:\\WINDOWS\\System32\\WindowsPowerShell\\V1.0\\PowerShell.exe -NoLogo -NoProfile -ExecutionPolicy Bypass -Command  exit 1",
					"Binary":                      "770069006E006C006F00670062006500610074002F0034000000",
					"Gravedad":                    "Informational",
					"Id. de espacio de ejecución": "91bb420c-d23e-4926-912b-6d11190df3fb",
					"Id. de host":                 "1f3cf2e0-1ae4-4672-a138-e96befdf9586",
					"Id. de shell":                "Microsoft.PowerShell",
					"Nombre de comando":           "Get-Dat",
					"Nombre de host":              "ConsoleHost",
					"Nombre de script":            "C:\\WINDOWS\\Administrador\\service-1\\exec\\Invoke.ps1",
					"Número de secuencia":         "4174",
					"Payload":                     "CommandInvocation(Get-Date): \"Get-Date\"",
					"Tipo de comando":             "Cmdlet",
					"Usuario":                     "GENTE\\persona",
					"Versión de host":             "5.1.19041.1320",
					"Versión del motor":           "5.1.19041.1320",
				},
				"event_id": "4103",
				"level":    "information",
				"message":  "CommandInvocation(Get-Date): \"Get-Date\"\n\n\nContexto:\n Gravedad = Informational\n Nombre de host = ConsoleHost\n Versión de host = 5.1.19041.1320\n Id. de host = 56995afd-2444-424e-871c-4c5513731a3b\n Aplicación host = C:\\WINDOWS\\System32\\WindowsPowerShell\\V1.0\\PowerShell.exe -NoLogo -NoProfile -ExecutionPolicy Bypass -Command  exit 1\n Versión del motor = 5.1.19041.1320\n Id. de espacio de ejecución = 91bb420c-d23e-4926-912b-6d11190df3fb\n Id. de canalización = 1\n Nombre de comando = Get-Date\n Tipo de comando = Cmdlet\n Nombre de script = C:\\WINDOWS\\Administrador\\service-1\\exec\\Invoke.ps1\n Ruta de acceso de comando =\n Número de secuencia = 4174\n Usuario = GENTE\\persona\n Usuario conectado =\n Id. de shell = Microsoft.PowerShell\n\n\nDatos de usuario:",
				"opcode":   "Para usar cuando la operación solo está ejecutando un método.",
				"outcome":  "success",
				"process": mapstr.M{
					"pid": uint32(0x398),
					"thread": mapstr.M{
						"id": uint32(0x480),
					},
				},
				"provider_guid":       "{b51b54b5-04a1-4c65-8760-04f9e2335cd5}",
				"provider_name":       "Microsoft-Windows-PowerShell",
				"record_id":           uint64(0x64),
				"related_activity_id": "{EDAD163F-52D0-4E89-BEF8-0500EC6F08AF}",
				"task":                "Request handling",
				"time_created":        allXMLTimeCreated,
				"user": mapstr.M{
					"identifier": "S-1-5-21-4564564786-2382305473-342768465-7452",
				},
				"user_data": mapstr.M{
					"ServerName": "\\\\VAGRANT-2012-R2",
					"UserName":   "vagrant",
					"xml_name":   "EventXML",
				},
				"version": Version(0x1),
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
			mapstr: mapstr.M{
				"event_id":     "0",
				"time_created": time.Time{},
				"user_data": mapstr.M{
					"Id":       "{00000000-0000-0000-0000-000000000000}",
					"xml_name": "Operation_ClientFailure",
				},
			},
		},
	}

	for _, test := range tests {
		event, err := UnmarshalXML([]byte(test.xml))
		if err != nil {
			t.Error(err)
			continue
		}
		assert.Equal(t, test.event, event)
		if test.mapstr != nil {
			assert.Equal(t, test.mapstr, event.Fields())
		}

		if testing.Verbose() {
			json, err := json.MarshalIndent(event, "", "  ")
			if err != nil {
				t.Error(err)
			}
			t.Logf("%s", json)
		}
	}
}

// Tests that control characters other than CR and LF are escaped
// when the event is decoded.
func TestInvalidXML(t *testing.T) {
	evXML := strings.ReplaceAll(allXML, "%1", "\t&#xD;\n\x1b")
	ev, err := UnmarshalXML([]byte(evXML))
	assert.Equal(t, nil, err)
	assert.Equal(t, "Creating WSMan shell on server with ResourceUri: \t\r\n\\u001b", ev.Message)
}

// nonUnsignedIntVersion is an anonymized sample from a NetApp appliance that
// produces non-conforming data.
const nonUnsignedIntVersion = `
<Event xmlns="http://schemas.netapp.com/events/event">
  <System>
    <Provider Name="NetApp-Security-Auditing" Guid="{3CB2A168-FE19-4A4E-BDAD-DCF422F13473}"/>
    <EventID>4656</EventID>
    <EventName>Open Object</EventName>
    <Version>101.3</Version>
    <Source>CIFS</Source>
    <Level>0</Level>
    <Opcode>0</Opcode>
    <Keywords>0x8020000000000000</Keywords>
    <Result>Audit Success</Result>
    <TimeCreated SystemTime="2019-03-26T23:27:07.015494000Z"/>
    <Correlation/>
    <Channel>Security</Channel>
    <Computer>anvil/vs-anvil</Computer>
    <ComputerUUID>b1111111-2222-3444-4444-000000000000/91f49999-55fe-11e6-b525-00a098a5d936</ComputerUUID>
    <Security/>
  </System>
  <EventData>
    <Data Name="SubjectIP" IPVersion="4">192.168.1.2</Data>
    <Data Name="SubjectHostname" Source=""/>
    <Data Name="SubjectUnix" Uid="65534" Gid="65534" Local="false"/>
    <Data Name="SubjectUserSid">S-1-5-21-2770437333-1905999116-9999999999-1111</Data>
    <Data Name="SubjectUserIsLocal">false</Data>
    <Data Name="SubjectDomainName">DOMAIN</Data>
    <Data Name="SubjectUserName">john.doe</Data>
    <Data Name="ObjectServer">Security</Data>
    <Data Name="ObjectType">File</Data>
    <Data Name="HandleID">00000000000000;00;00000000;00000000</Data>
    <Data Name="ObjectName">(workshop_fg);/Some/Path/2020.jpg</Data>
    <Data Name="AccessList">%%4416 %%4417 %%4418 %%4419 %%4420 %%4423 %%4424 %%1538 </Data>
    <Data Name="AccessMask">8607</Data>
    <Data Name="DesiredAccess">Read Data; List Directory; Write Data; Add File; Append Data; Add Subdirectory; Read Extended Attributes; Write Extended Attributes; Read Attributes; Write Attributes; Read ACL; </Data>
    <Data Name="Attributes">Set Attributes; Create; Open a non-directory; </Data>
  </EventData>
</Event>
`

// TestInvalidVersion verifies that the reader will accept events where the the
// version number is not an unsigned byte as per the schema definition.
// Microsoft documentation defines version as:
//
//	<xs:element name="Version"
//	   type="unsignedByte"
//	/>
//
// But some event producers don't adhere to the schema. The value space of
// xsd:unsignedByte is the range of integers between 0 and 255 — the unsigned
// values that can fit in a word of 8 bits. Its lexical space allows an
// optional + sign and leading zeros before the significant digits.
//
// Reference: https://docs.microsoft.com/en-us/windows/win32/wes/schema-version-systempropertiestype-element
func TestInvalidVersion(t *testing.T) {
	ev, err := UnmarshalXML([]byte(nonUnsignedIntVersion))
	assert.NoError(t, err)
	assert.EqualValues(t, 0, ev.Version)
}

func BenchmarkXMLUnmarshal(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := UnmarshalXML([]byte(allXML))
		if err != nil {
			b.Fatal(err)
		}
	}
}
