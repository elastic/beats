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

//go:build !integration
// +build !integration

package xml

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIncompleteXML(t *testing.T) {
	const xml = `
<person>
  <Name ID="123">John</Name>
`

	d := NewDecoder(strings.NewReader(xml))
	out, err := d.Decode()
	assert.Nil(t, out)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected EOF")
}

func TestLowercaseKeys(t *testing.T) {
	const xml = `
<person>
  <Name ID="123">John</Name>
</person>
`

	expected := map[string]interface{}{
		"person": map[string]interface{}{
			"name": map[string]interface{}{
				"#text": "John",
				"id":    "123",
			},
		},
	}

	d := NewDecoder(strings.NewReader(xml))
	d.LowercaseKeys()
	out, err := d.Decode()
	require.NoError(t, err)
	assert.Equal(t, expected, out)
}

func TestPrependHyphenToAttr(t *testing.T) {
	const xml = `
<person>
  <Name ID="123">John</Name>
</person>
`

	expected := map[string]interface{}{
		"person": map[string]interface{}{
			"Name": map[string]interface{}{
				"#text": "John",
				"-ID":   "123",
			},
		},
	}

	d := NewDecoder(strings.NewReader(xml))
	d.PrependHyphenToAttr()
	out, err := d.Decode()
	require.NoError(t, err)
	assert.Equal(t, expected, out)
}

func TestDecodeList(t *testing.T) {
	const xml = `
<people>
	<person>
	  <Name ID="123">John</Name>
	</person>
	<person>
	  <Name ID="456">Jane</Name>
	</person>
    <person>Foo</person>
</people>
`

	expected := map[string]interface{}{
		"people": map[string]interface{}{
			"person": []interface{}{
				map[string]interface{}{
					"Name": map[string]interface{}{
						"#text": "John",
						"ID":    "123",
					},
				},
				map[string]interface{}{
					"Name": map[string]interface{}{
						"#text": "Jane",
						"ID":    "456",
					},
				},
				"Foo",
			},
		},
	}

	d := NewDecoder(strings.NewReader(xml))
	out, err := d.Decode()
	require.NoError(t, err)
	assert.Equal(t, expected, out)
}

func TestEmptyElement(t *testing.T) {
	const xml = `
<people>
</people>
`

	expected := map[string]interface{}{
		"people": "",
	}

	d := NewDecoder(strings.NewReader(xml))
	out, err := d.Decode()
	require.NoError(t, err)
	assert.Equal(t, expected, out)
}

func TestDecode(t *testing.T) {
	type testCase struct {
		XML    string
		Output map[string]interface{}
	}

	tests := []testCase{
		{
			XML: `
			<catalog>
				<book seq="1">
					<author>William H. Gaddis</author>
					<title>The Recognitions</title>
					<review>One of the great seminal American novels of the 20th century.</review>
				</book>
			</catalog>`,
			Output: map[string]interface{}{
				"catalog": map[string]interface{}{
					"book": map[string]interface{}{
						"author": "William H. Gaddis",
						"review": "One of the great seminal American novels of the 20th century.",
						"seq":    "1",
						"title":  "The Recognitions"}}},
		},
		{
			XML: `
			<catalog>
				<book id="bk101">
					<author>Gambardella, Matthew</author>
					<title>XML Developer's Guide</title>
					<genre>Computer</genre>
					<price>44.95</price>
					<publish_date>2000-10-01</publish_date>
					<description>An in-depth look at creating applications with XML.</description>
				</book>
				<book id="bk102">
					<author>Ralls, Kim</author>
					<title>Midnight Rain</title>
					<genre>Fantasy</genre>
					<price>5.95</price>
					<publish_date>2000-12-16</publish_date>
					<description>A former architect battles corporate zombies, an evil sorceress, and her own childhood to become queen of the world.</description>
				</book>
			</catalog>`,
			Output: map[string]interface{}{
				"catalog": map[string]interface{}{
					"book": []interface{}{
						map[string]interface{}{
							"author":       "Gambardella, Matthew",
							"description":  "An in-depth look at creating applications with XML.",
							"genre":        "Computer",
							"id":           "bk101",
							"price":        "44.95",
							"publish_date": "2000-10-01",
							"title":        "XML Developer's Guide",
						},
						map[string]interface{}{
							"author":       "Ralls, Kim",
							"description":  "A former architect battles corporate zombies, an evil sorceress, and her own childhood to become queen of the world.",
							"genre":        "Fantasy",
							"id":           "bk102",
							"price":        "5.95",
							"publish_date": "2000-12-16",
							"title":        "Midnight Rain"}}}},
		},
		{
			XML: `
			<?xml version="1.0"?>
			<catalog>
				<book id="bk101">
					<author>Gambardella, Matthew</author>
					<title>XML Developer's Guide</title>
					<genre>Computer</genre>
					<price>44.95</price>
					<publish_date>2000-10-01</publish_date>
					<description>An in-depth look at creating applications with XML.</description>
				</book>
				<book id="bk102">
					<author>Ralls, Kim</author>
					<title>Midnight Rain</title>
					<genre>Fantasy</genre>
					<price>5.95</price>
					<publish_date>2000-12-16</publish_date>
					<description>A former architect battles corporate zombies, an evil sorceress, and her own childhood to become queen of the world.</description>
				</book>
			</catalog>`,
			Output: map[string]interface{}{
				"catalog": map[string]interface{}{
					"book": []interface{}{
						map[string]interface{}{
							"author":       "Gambardella, Matthew",
							"description":  "An in-depth look at creating applications with XML.",
							"genre":        "Computer",
							"id":           "bk101",
							"price":        "44.95",
							"publish_date": "2000-10-01",
							"title":        "XML Developer's Guide"},
						map[string]interface{}{
							"author":       "Ralls, Kim",
							"description":  "A former architect battles corporate zombies, an evil sorceress, and her own childhood to become queen of the world.",
							"genre":        "Fantasy",
							"id":           "bk102",
							"price":        "5.95",
							"publish_date": "2000-12-16",
							"title":        "Midnight Rain"}}}},
		},
		{
			XML: `
			<?xml version="1.0"?>
			<catalog>
				<book id="bk101">
					<author>Gambardella, Matthew</author>
					<title>XML Developer's Guide</title>
					<genre>Computer</genre>
					<price>44.95</price>
					<publish_date>2000-10-01</publish_date>
					<description>An in-depth look at creating applications with XML.</description>
				</book>
				<secondcategory>
					<paper id="bk102">
						<test2>Ralls, Kim</test2>
						<description>A former architect battles corporate zombies, an evil sorceress, and her own childhood to become queen of the world.</description>
					</paper>
				</secondcategory>
			</catalog>`,
			Output: map[string]interface{}{
				"catalog": map[string]interface{}{
					"book": map[string]interface{}{
						"author":       "Gambardella, Matthew",
						"description":  "An in-depth look at creating applications with XML.",
						"genre":        "Computer",
						"id":           "bk101",
						"price":        "44.95",
						"publish_date": "2000-10-01",
						"title":        "XML Developer's Guide"},
					"secondcategory": map[string]interface{}{
						"paper": map[string]interface{}{
							"description": "A former architect battles corporate zombies, an evil sorceress, and her own childhood to become queen of the world.",
							"id":          "bk102",
							"test2":       "Ralls, Kim"}}}},
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			d := NewDecoder(strings.NewReader(test.XML))
			d.LowercaseKeys()

			out, err := d.Decode()
			require.NoError(t, err)
			assert.EqualValues(t, test.Output, out)
		})
	}
}

func ExampleDecoder_Decode() {
	const xml = `
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
}
`
	dec := NewDecoder(strings.NewReader(xml))
	dec.LowercaseKeys()
	m, err := dec.Decode()
	if err != nil {
		return
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err = enc.Encode(m); err != nil {
		return
	}

	// Output:
	// {
	//   "event": {
	//     "eventdata": {
	//       "binary": "770069006E006C006F00670062006500610074002F0034000000",
	//       "data": [
	//         {
	//           "#text": "winlogbeat",
	//           "name": "param1"
	//         },
	//         {
	//           "#text": "running",
	//           "name": "param2"
	//         }
	//       ]
	//     },
	//     "processingerrordata": {
	//       "dataitemname": "shellId",
	//       "errorcode": "15005",
	//       "eventpayload": "68007400740070003A002F002F0073006300680065006D00610073002E006D006900630072006F0073006F00660074002E0063006F006D002F007700620065006D002F00770073006D0061006E002F0031002F00770069006E0064006F00770073002F007300680065006C006C002F0063006D0064000000"
	//     },
	//     "renderinginfo": {
	//       "channel": "Microsoft-Windows-WinRM/Operational",
	//       "culture": "en-US",
	//       "keywords": {
	//         "keyword": "Server"
	//       },
	//       "level": "Information",
	//       "message": "Creating WSMan shell on server with ResourceUri: %1",
	//       "opcode": "Info",
	//       "provider": "Microsoft-Windows-Windows Remote Management",
	//       "task": "Request handling"
	//     },
	//     "system": {
	//       "channel": "Microsoft-Windows-WinRM/Operational",
	//       "computer": "vagrant-2012-r2",
	//       "correlation": {
	//         "activityid": "{A066CCF1-8AB3-459B-B62F-F79F957A5036}",
	//         "relatedactivityid": "{85FC0930-9C49-42DA-804B-A7368104BD1B}"
	//       },
	//       "eventid": "91",
	//       "eventrecordid": "100",
	//       "execution": {
	//         "processid": "920",
	//         "threadid": "1152"
	//       },
	//       "keywords": "0x8020000000000000",
	//       "level": "4",
	//       "opcode": "0",
	//       "provider": {
	//         "eventsourcename": "Service Control Manager",
	//         "guid": "{a7975c8f-ac13-49f1-87da-5a984a4ab417}",
	//         "name": "Microsoft-Windows-WinRM"
	//       },
	//       "security": {
	//         "userid": "S-1-5-21-3541430928-2051711210-1391384369-1001"
	//       },
	//       "task": "9",
	//       "timecreated": {
	//         "systemtime": "2016-01-28T20:33:27.990735300Z"
	//       },
	//       "version": "1"
	//     },
	//     "userdata": {
	//       "eventxml": {
	//         "servername": "\\\\VAGRANT-2012-R2",
	//         "username": "vagrant",
	//         "xmlns": "Event_NS"
	//       }
	//     },
	//     "xmlns": "http://schemas.microsoft.com/win/2004/08/events/event"
	//   }
	// }
}
