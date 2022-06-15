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

package syslog

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

const (
	severityMask  = 7
	facilityShift = 3
	utf8BOM       = "\ufeff"
)

var (
	severityLabels = []string{
		"Emergency",
		"Alert",
		"Critical",
		"Error",
		"Warning",
		"Notice",
		"Informational",
		"Debug",
	}
	facilityLabels = []string{
		"kernel",
		"user-level",
		"mail",
		"system",
		"security/authorization",
		"syslogd",
		"line printer",
		"network news",
		"UUCP",
		"clock",
		"security/authorization",
		"FTP",
		"NTP",
		"log audit",
		"log alert",
		"clock",
		"local0",
		"local1",
		"local2",
		"local3",
		"local4",
		"local5",
		"local6",
		"local7",
	}
)

// message is a syslog message.
type message struct {
	timestamp time.Time
	facility  int
	severity  int
	priority  int
	hostname  string
	msg       string
	process   string
	pid       string

	// RFC-5424 fields.
	msgID      string
	version    int
	rawSDValue string
}

// setTimestampRFC3339 sets the timestamp for this message using an RFC3339 timestamp (time.RFC3339Nano).
func (m *message) setTimestampRFC3339(v string) error {
	t, err := time.Parse(time.RFC3339Nano, v)
	if err == nil {
		m.timestamp = t
	}

	return err
}

// setTimestampBSD sets the timestamp for this message using a BSD-style timestamp (time.Stamp). Since these
// timestamps lack year and timezone information, the year will be derived from the current time (adjusted for
// loc) and the timezone will be provided by loc.
func (m *message) setTimestampBSD(v string, loc *time.Location) error {
	if loc == nil {
		loc = time.Local
	}
	t, err := time.ParseInLocation(time.Stamp, v, loc)
	if err == nil {
		t = t.AddDate(time.Now().In(loc).Year(), 0, 0)
		m.timestamp = t
	}

	return err
}

// setPriority sets the priority for this message. The facility and severity are
// derived from the priority and associated values are set.
func (m *message) setPriority(v string) error {
	priority, err := strconv.Atoi(v)
	if err != nil {
		return fmt.Errorf("invalid priority: %w", err)
	}

	// Range defined by RFC.
	if priority < 0 || 191 < priority {
		return ErrPriority
	}

	m.priority = stringToInt(v)
	m.facility = m.priority >> facilityShift
	m.severity = m.priority & severityMask

	return nil
}

// setHostname sets the hostname for this message. If the value is the "nil value" (-), the hostname will NOT be set.
func (m *message) setHostname(v string) {
	if v != "-" {
		m.hostname = v
	}
}

// setMsg sets the msg for this message. If the message includes a UTF-8 byte order mark (BOM),
// then it will be removed.
func (m *message) setMsg(v string) {
	m.msg = strings.TrimPrefix(v, utf8BOM)
}

// setTag sets the process for this message.
func (m *message) setTag(v string) {
	m.process = v
}

// setAppName sets the process for this message. If the value is the "nil value" (-), the process will NOT be set.
func (m *message) setAppName(v string) {
	if v != "-" {
		m.process = v
	}
}

// setContent sets the pid for this message.
func (m *message) setContent(v string) {
	m.pid = v
}

// setProcID sets the pid for this message. If the value is the "nil value" (-), the pid will NOT be set.
func (m *message) setProcID(v string) {
	if v != "-" {
		m.pid = v
	}
}

// setMsgID sets the msgID for this message. If the value is the "nil value" (-), the msgID will NOT be set.
func (m *message) setMsgID(v string) {
	if v != "-" {
		m.msgID = v
	}
}

// setVersion sets the version for this message.
func (m *message) setVersion(v string) error {
	version, err := strconv.Atoi(v)
	if err != nil {
		return fmt.Errorf("invalid version, expected an integer: %w", err)
	}

	m.version = version

	return nil
}

func (m *message) setRawSDValue(v string) {
	if v != "-" {
		m.rawSDValue = v
	}
}

// fields produces fields from the message.
func (m message) fields() mapstr.M {
	f := mapstr.M{}
	msg := m.msg

	// Syslog fields.
	if m.priority >= 0 {
		_, _ = f.Put("log.syslog.priority", m.priority)
		_, _ = f.Put("log.syslog.facility.code", m.facility)
		_, _ = f.Put("log.syslog.severity.code", m.severity)
		if v, ok := mapIndexToString(m.severity, severityLabels); ok {
			_, _ = f.Put("log.syslog.severity.name", v)
		}
		if v, ok := mapIndexToString(m.facility, facilityLabels); ok {
			_, _ = f.Put("log.syslog.facility.name", v)
		}
	}
	if m.process != "" {
		_, _ = f.Put("log.syslog.appname", m.process)
		if m.pid != "" {
			_, _ = f.Put("log.syslog.procid", m.pid)
		}
	}
	if m.hostname != "" {
		_, _ = f.Put("log.syslog.hostname", m.hostname)
	}
	if m.msgID != "" {
		_, _ = f.Put("log.syslog.msgid", m.msgID)
	}
	if m.version != 0 {
		_, _ = f.Put("log.syslog.version", strconv.Itoa(m.version))
	}
	if data := parseStructuredData(m.rawSDValue); data != nil {
		_, _ = f.Put("log.syslog.structured_data", data)
	} else {
		// Raw structured data value is prepended to the message field
		// if it could not be parsed properly. The message is not altered
		// if no structured data was extracted from the message (nil value was
		// used or message format is not RFC 5424).
		msg = joinStr(m.rawSDValue, m.msg, " ")
	}

	// Message field.
	if msg != "" {
		_, _ = f.Put("message", msg)
	}

	return f
}
