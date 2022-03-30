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
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	libxml "github.com/elastic/beats/v7/libbeat/common/encoding/xml"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/winlogbeat/sys"
)

// Debug selectors used in this package.
const (
	debugSelector = "winevent"
)

// Debug logging functions for this package.
var (
	debugf = logp.MakeDebug(debugSelector)
)

// Keyword Constants
const (
	keywordAuditFailure = 0x10000000000000
	keywordAuditSuccess = 0x20000000000000
)

// UnmarshalXML unmarshals the given XML into a new Event.
func UnmarshalXML(rawXML []byte) (Event, error) {
	var event Event
	decoder := xml.NewDecoder(libxml.NewSafeReader(rawXML))
	err := decoder.Decode(&event)
	return event, err
}

// Event holds the data from a log record.
type Event struct {
	// System
	Provider        Provider        `xml:"System>Provider"`
	EventIdentifier EventIdentifier `xml:"System>EventID"`
	Version         Version         `xml:"System>Version"`
	LevelRaw        uint8           `xml:"System>Level"`
	TaskRaw         uint16          `xml:"System>Task"`
	OpcodeRaw       *uint8          `xml:"System>Opcode,omitempty"`
	KeywordsRaw     HexInt64        `xml:"System>Keywords"`
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

func (e Event) Fields() common.MapStr {
	// Windows Log Specific data
	win := common.MapStr{}

	AddOptional(win, "channel", e.Channel)
	AddOptional(win, "event_id", fmt.Sprint(e.EventIdentifier.ID))
	AddOptional(win, "provider_name", e.Provider.Name)
	AddOptional(win, "record_id", e.RecordID)
	AddOptional(win, "task", e.Task)
	AddOptional(win, "computer_name", e.Computer)
	AddOptional(win, "keywords", e.Keywords)
	AddOptional(win, "opcode", e.Opcode)
	AddOptional(win, "provider_guid", e.Provider.GUID)
	AddOptional(win, "version", e.Version)
	AddOptional(win, "time_created", e.TimeCreated.SystemTime)

	if e.KeywordsRaw&keywordAuditFailure > 0 {
		_, _ = win.Put("outcome", "failure")
	} else if e.KeywordsRaw&keywordAuditSuccess > 0 {
		_, _ = win.Put("outcome", "success")
	}

	AddOptional(win, "level", strings.ToLower(e.Level))
	AddOptional(win, "message", sys.RemoveWindowsLineEndings(e.Message))

	if e.User.Identifier != "" {
		user := common.MapStr{
			"identifier": e.User.Identifier,
		}
		win["user"] = user
		AddOptional(user, "domain", e.User.Domain)
		AddOptional(user, "name", e.User.Name)
		AddOptional(user, "type", e.User.Type.String())
	}

	AddPairs(win, "event_data", e.EventData.Pairs)
	userData := AddPairs(win, "user_data", e.UserData.Pairs)
	AddOptional(userData, "xml_name", e.UserData.Name.Local)

	// Correlation
	AddOptional(win, "activity_id", e.Correlation.ActivityID)
	AddOptional(win, "related_activity_id", e.Correlation.RelatedActivityID)

	// Execution
	AddOptional(win, "kernel_time", e.Execution.KernelTime)
	AddOptional(win, "process.pid", e.Execution.ProcessID)
	AddOptional(win, "process.thread.id", e.Execution.ThreadID)
	AddOptional(win, "processor_id", e.Execution.ProcessorID)
	AddOptional(win, "processor_time", e.Execution.ProcessorTime)
	AddOptional(win, "session_id", e.Execution.SessionID)
	AddOptional(win, "user_time", e.Execution.UserTime)

	// Errors
	AddOptional(win, "error.code", e.RenderErrorCode)
	if len(e.RenderErr) == 1 {
		AddOptional(win, "error.message", e.RenderErr[0])
	} else {
		AddOptional(win, "error.message", e.RenderErr)
	}

	return win
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

// EventIdentifier is the identifier that the provider uses to identify a
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
			err = d.Skip()
			if err != nil {
				return err
			}
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

// Version contains the version number of the event's definition.
type Version uint8

// UnmarshalXML unmarshals the version number as an xsd:unsignedByte. Invalid
// values are ignored an no error is returned.
func (v *Version) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var s string
	if err := d.DecodeElement(&s, &start); err != nil {
		return err
	}

	version, err := strconv.ParseUint(s, 10, 8)
	if err != nil {
		return nil //nolint:nilerr // Ignore invalid version values.
	}

	*v = Version(version)
	return nil
}

type HexInt64 uint64

func (v *HexInt64) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var s string
	if err := d.DecodeElement(&s, &start); err != nil {
		return err
	}

	num, err := strconv.ParseUint(s, 0, 64)
	if err != nil {
		// Ignore invalid version values.
		return err
	}

	*v = HexInt64(num)
	return nil
}

// EnrichRawValuesWithNames adds the names associated with the raw system
// property values. It enriches the event with keywords, opcode, level, and
// task. The search order is defined in the EvtFormatMessage documentation.
func EnrichRawValuesWithNames(publisherMeta *WinMeta, event *Event) {
	// Keywords. Each bit in the value can represent a keyword.
	rawKeyword := int64(event.KeywordsRaw)

	if len(event.Keywords) == 0 {
		for mask, keyword := range defaultWinMeta.Keywords {
			if rawKeyword&mask != 0 {
				event.Keywords = append(event.Keywords, keyword)
				rawKeyword &^= mask
			}
		}
		if publisherMeta != nil {
			for mask, keyword := range publisherMeta.Keywords {
				if rawKeyword&mask != 0 {
					event.Keywords = append(event.Keywords, keyword)
					rawKeyword &^= mask
				}
			}
		}
	}

	var found bool
	if event.Opcode == "" {
		// Opcode (search in defaultWinMeta first).
		if event.OpcodeRaw != nil {
			event.Opcode, found = defaultWinMeta.Opcodes[*event.OpcodeRaw]
			if !found && publisherMeta != nil {
				event.Opcode = publisherMeta.Opcodes[*event.OpcodeRaw]
			}
		}
	}

	if event.Level == "" {
		// Level (search in defaultWinMeta first).
		event.Level, found = defaultWinMeta.Levels[event.LevelRaw]
		if !found && publisherMeta != nil {
			event.Level = publisherMeta.Levels[event.LevelRaw]
		}
	}

	if event.Task == "" {
		if publisherMeta != nil {
			// Task (fall-back to defaultWinMeta if not found).
			event.Task, found = publisherMeta.Tasks[event.TaskRaw]
			if !found {
				event.Task = defaultWinMeta.Tasks[event.TaskRaw]
			}
		} else {
			event.Task = defaultWinMeta.Tasks[event.TaskRaw]
		}
	}
}
