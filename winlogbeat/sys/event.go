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
	"encoding/xml"
	"fmt"
	"time"
)

// UnmarshalEventXML unmarshals the given XML into a new Event.
func UnmarshalEventXML(rawXML []byte) (Event, error) {
	var event Event
	decoder := xml.NewDecoder(newXMLSafeReader(rawXML))
	err := decoder.Decode(&event)
	return event, err
}

// Event holds the data from a log record.
type Event struct {
	// System
	Provider        Provider        `xml:"System>Provider"`
	EventIdentifier EventIdentifier `xml:"System>EventID"`
	Version         uint8           `xml:"System>Version"`
	LevelRaw        uint8           `xml:"System>Level"`
	TaskRaw         uint16          `xml:"System>Task"`
	OpcodeRaw       uint8           `xml:"System>Opcode"`
	TimeCreated     TimeCreated     `xml:"System>TimeCreated"`
	RecordID        uint64          `xml:"System>EventRecordID"`
	Correlation     Correlation     `xml:"System>Correlation"`
	Execution       Execution       `xml:"System>Execution"`
	Channel         string          `xml:"System>Channel"`
	Computer        string          `xml:"System>Computer"`
	User            SID             `xml:"System>Security"`

	EventData EventData `xml:"EventData"`
	UserData  UserData  `xml:"UserData"`

	// RenderingInfo
	Message  string   `xml:"RenderingInfo>Message"`
	Level    string   `xml:"RenderingInfo>Level"`
	Task     string   `xml:"RenderingInfo>Task"`
	Opcode   string   `xml:"RenderingInfo>Opcode"`
	Keywords []string `xml:"RenderingInfo>Keywords>Keyword"`

	// ProcessingErrorData
	RenderErrorCode         uint32 `xml:"ProcessingErrorData>ErrorCode"`
	RenderErrorDataItemName string `xml:"ProcessingErrorData>DataItemName"`
	RenderErr               []string
}

// Provider identifies the provider that logged the event. The Name and GUID
// attributes are included if the provider used an instrumentation manifest to
// define its events; otherwise, the EventSourceName attribute is included if a
// legacy event provider (using the Event Logging API) logged the event.
type Provider struct {
	Name            string `xml:"Name,attr"`
	GUID            string `xml:"Guid,attr"`
	EventSourceName string `xml:"EventSourceName,attr"`
}

// Correlation contains activity identifiers that consumers can use to group
// related events together.
type Correlation struct {
	ActivityID        string `xml:"ActivityID,attr"`
	RelatedActivityID string `xml:"RelatedActivityID,attr"`
}

// Execution contains information about the process and thread that logged the
// event.
type Execution struct {
	ProcessID uint32 `xml:"ProcessID,attr"`
	ThreadID  uint32 `xml:"ThreadID,attr"`

	// Only available for events logged to an event tracing log file (.etl file).
	ProcessorID   uint32 `xml:"ProcessorID,attr"`
	SessionID     uint32 `xml:"SessionID,attr"`
	KernelTime    uint32 `xml:"KernelTime,attr"`
	UserTime      uint32 `xml:"UserTime,attr"`
	ProcessorTime uint32 `xml:"ProcessorTime,attr"`
}

// EventIdentifier is the identifer that the provider uses to identify a
// specific event type.
type EventIdentifier struct {
	Qualifiers uint16 `xml:"Qualifiers,attr"`
	ID         uint32 `xml:",chardata"`
}

// TimeCreated contains the system time of when the event was logged.
type TimeCreated struct {
	SystemTime time.Time
}

// UnmarshalXML unmarshals an XML dataTime string.
func (t *TimeCreated) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	attrs := struct {
		SystemTime string `xml:"SystemTime,attr"`
		RawTime    uint64 `xml:"RawTime,attr"`
	}{}

	err := d.DecodeElement(&attrs, &start)
	if err != nil {
		return err
	}

	if attrs.SystemTime != "" {
		// This works but XML dateTime is really ISO8601.
		t.SystemTime, err = time.Parse(time.RFC3339Nano, attrs.SystemTime)
	} else if attrs.RawTime != 0 {
		// The units for RawTime are not specified in the documentation. I think
		// it is only used in event tracing so this shouldn't be a problem.
		err = fmt.Errorf("failed to unmarshal TimeCreated RawTime='%d'", attrs.RawTime)
	}

	return err
}

// EventData contains the event data. The EventData section is used if the
// message provider template does not contain a UserData section.
type EventData struct {
	Pairs []KeyValue `xml:",any"`
}

// UserData contains the event data.
type UserData struct {
	Name  xml.Name
	Pairs []KeyValue
}

// UnmarshalXML unmarshals UserData XML.
func (u *UserData) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	// Assume that UserData has the same general key-value structure as
	// EventData does.
	in := struct {
		Pairs []KeyValue `xml:",any"`
	}{}

	// Read tokens until we find the first StartElement then unmarshal it.
	for {
		t, err := d.Token()
		if err != nil {
			return err
		}

		if se, ok := t.(xml.StartElement); ok {
			err = d.DecodeElement(&in, &se)
			if err != nil {
				return err
			}

			u.Name = se.Name
			u.Pairs = in.Pairs
			d.Skip()
			break
		}
	}

	return nil
}

// KeyValue is a key value pair of strings.
type KeyValue struct {
	Key   string
	Value string
}

// UnmarshalXML unmarshals an arbitrary XML element into a KeyValue. The key
// becomes the name of the element or value of the Name attribute if it exists.
// The value is the character data contained within the element.
func (kv *KeyValue) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	elem := struct {
		XMLName xml.Name
		Name    string `xml:"Name,attr"`
		Value   string `xml:",chardata"`
	}{}

	err := d.DecodeElement(&elem, &start)
	if err != nil {
		return err
	}

	kv.Key = elem.XMLName.Local
	if elem.Name != "" {
		kv.Key = elem.Name
	}
	kv.Value = elem.Value

	return nil
}
