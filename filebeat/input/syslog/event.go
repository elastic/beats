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
	"bytes"
	"math"
	"time"
)

const severityMask = 7
const facilityShift = 3

var month = map[string]time.Month{
	"Jan": time.January,
	"Feb": time.February,
	"Mar": time.March,
	"Apr": time.April,
	"May": time.May,
	"Jun": time.June,
	"Jul": time.July,
	"Aug": time.August,
	"Sep": time.September,
	"Oct": time.October,
	"Nov": time.November,
	"Dec": time.December,
}

var monthIndexed = []time.Month{
	0,
	time.January,
	time.February,
	time.March,
	time.April,
	time.May,
	time.June,
	time.July,
	time.August,
	time.September,
	time.October,
	time.November,
	time.December,
}

// event is a parsed syslog event, validation of the format is done at the parser level.
type event struct {
	message    string
	hostname   string //x
	priority   int
	program    string //x
	pid        int
	month      time.Month
	day        int
	hour       int
	minute     int
	second     int
	nanosecond int
	year       int
	loc        *time.Location
	sequence   int

	// RFC 5424
	version   int
	appName   string
	msgID     string
	processID string
	data      EventData
}

type EventData map[string]map[string]string

// newEvent() return a new event.
func newEvent() *event {
	return &event{
		priority: -1,
		pid:      -1,
		month:    -1,
		day:      -1,
		hour:     -1,
		minute:   -1,
		second:   -1,
		year:     time.Now().Year(),
	}
}

// SetTimeZone set the timezone offset from the string.
func (s *event) SetTimeZone(b []byte) {
	// We assume that we are in utc and ignore any other bytes after.
	// This can be followed by others bytes +00, +00:00 or +0000.
	if b[0] == 'Z' || b[0] == 'z' {
		s.loc = time.UTC
		return
	}

	d := 1
	if b[0] == '-' {
		d = -1
	}

	// +00 +00:00 or +0000
	// Use second value directly and don't use unecessary time.Duration.
	// Time.FixedZone accepts number of seconds.
	var h, m int
	switch len(b[1:]) {
	case 2:
		h = 3600 * bytesToInt(skipLeadZero(b[1:]))
		s.loc = time.FixedZone("", d*h)
	case 4:
		h = 3600 * bytesToInt(skipLeadZero(b[1:3]))
		m = 60 * bytesToInt(skipLeadZero(b[3:5]))
		s.loc = time.FixedZone("", d*(h+m))
	case 5:
		h = 3600 * bytesToInt(skipLeadZero(b[1:3]))
		m = 60 * bytesToInt(skipLeadZero(b[4:6]))
		s.loc = time.FixedZone("", d*(h+m))
	}
}

// SetMonthNumeric sets the month with a number.
func (s *event) SetMonthNumeric(b []byte) {
	s.month = monthIndexed[bytesToInt(skipLeadZero(b))]
}

// SetMonth sets the month.
func (s *event) SetMonth(b []byte) {
	var k string
	if len(b) > 3 {
		k = string(b[0:3])
	} else {
		k = string(b)
	}
	v, ok := month[k]
	if ok {
		s.month = v
	}
}

// Month returns the month.
func (s *event) Month() time.Month {
	return s.month
}

// SetDay sets the day as.
func (s *event) SetDay(b []byte) {
	s.day = bytesToInt(skipLeadZero(b))
}

// Day returns the day.
func (s *event) Day() int {
	return s.day
}

// SetHour sets the hour.
func (s *event) SetHour(b []byte) {
	s.hour = bytesToInt(skipLeadZero(b))
}

// Hour returns the hour.
func (s *event) Hour() int {
	return s.hour
}

// SetMinute sets the minute.
func (s *event) SetMinute(b []byte) {
	s.minute = bytesToInt(skipLeadZero(b))
}

// Minute return the minutes.
func (s *event) Minute() int {
	return s.minute
}

// SetSecond sets the second.
func (s *event) SetSecond(b []byte) {
	s.second = bytesToInt(skipLeadZero(b))
}

// Second returns the second.
func (s *event) Second() int {
	return s.second
}

// SetYear sets the current year.
func (s *event) SetYear(b []byte) {
	s.year = bytesToInt(b)
}

// Year returns the current year, since syslog events don't include that.
func (s *event) Year() int {
	return s.year
}

// SetMessage sets the message.
func (s *event) SetMessage(b []byte) {
	// remove BOM
	if b[0] == 0xef && b[1] == 0xbb && b[2] == 0xbf {
		s.message = string(b[3:])
	} else {
		s.message = string(b)
	}
}

// Message returns the message.
func (s *event) Message() string {
	return s.message
}

// SetPriority sets the priority.
func (s *event) SetPriority(priority []byte) {
	s.priority = bytesToInt(priority)
}

// Priority returns the priority.
func (s *event) Priority() int {
	return s.priority
}

// HasPriority returns if the priority was in original event.
func (s *event) HasPriority() bool {
	return s.priority > 0
}

// Severity returns the severity, will return -1 if priority is not set.
func (s *event) Severity() int {
	if !s.HasPriority() {
		return -1
	}
	return s.Priority() & severityMask
}

// Facility returns the facility, will return -1 if priority is not set.
func (s *event) Facility() int {
	if !s.HasPriority() {
		return -1
	}
	return s.Priority() >> facilityShift
}

// SetHostname sets the hostname.
func (s *event) SetHostname(b []byte) {
	s.hostname = string(b)
}

// Hostname returns the hostname.
func (s *event) Hostname() string {
	return string(s.hostname)
}

// SetProgram sets the programs as a byte slice.
func (s *event) SetProgram(b []byte) {
	s.program = string(b)
}

// Program returns the program name.
func (s *event) Program() string {
	return s.program
}

func (s *event) SetPid(b []byte) {
	s.pid = bytesToInt(b)
}

// Pid returns the pid.
func (s *event) Pid() int {
	return s.pid
}

// HasPid returns true if a pid is set.
func (s *event) HasPid() bool {
	return s.pid > 0
}

func (s *event) SetSequence(b []byte) {
	s.sequence = bytesToInt(b)
}

// Sequence returns the sequence number of the event when defined,
// otherwise return -1.
func (s *event) Sequence() int {
	return s.sequence
}

// SetNanoSecond sets the nanosecond.
func (s *event) SetNanosecond(b []byte) {
	// We assume that we receive a byte array representing a nanosecond, this might not be
	// always the case, so we have to pad it.
	if len(b) < 9 {
		s.nanosecond = bytesToInt(skipLeadZero(b)) * int(math.Pow10((9 - len(b))))
	} else {
		s.nanosecond = bytesToInt(skipLeadZero(b))
	}
}

// NanoSecond returns the nanosecond.
func (s *event) Nanosecond() int {
	return s.nanosecond
}

// SetVersion sets the version.
func (s *event) SetVersion(version []byte) {
	s.version = bytesToInt(version)
}

func (s *event) Version() int {
	return s.version
}

func (s *event) SetAppName(appname []byte) {
	s.appName = string(appname)
}

func (s *event) AppName() string {
	return s.appName
}

func (s *event) SetMsgID(msgID []byte) {
	s.msgID = string(msgID)
}

func (s *event) MsgID() string {
	return s.msgID
}

func (s *event) SetProcID(processID []byte) {
	s.processID = string(processID)
}

func (s *event) ProcID() string {
	return s.processID
}

// Timestamp return the timestamp in UTC.
func (s *event) Timestamp(timezone *time.Location) time.Time {
	var t *time.Location
	if s.loc == nil {
		t = timezone
	} else {
		t = s.loc
	}

	return time.Date(
		s.Year(),
		s.Month(),
		s.Day(),
		s.Hour(),
		s.Minute(),
		s.Second(),
		s.Nanosecond(),
		t,
	).UTC()
}

func (s *event) IsDataEmpty() bool {
	if s.data == nil {
		return true
	}
	return len(s.data) == 0
}

// IsValid returns true if the date and the message are present.
func (s *event) IsValid() bool {
	return s.day != -1 && s.hour != -1 && s.minute != -1 && s.second != -1 && (s.message != "" || !s.IsDataEmpty())
}

func (s *event) SetData(id string, key string, data []byte, start int, end int, bs []int) {
	var v string

	// param value escape
	// https://tools.ietf.org/html/rfc5424#section-6.3.3
	if len(bs) > 0 {
		buf := bytes.NewBufferString("")
		for _, i := range bs {
			buf.Write(data[start:i])
			start = i + 1
		}
		if start <= end {
			buf.Write(data[start:end])
		}
		v = buf.String()
	} else {
		v = string(data[start:end])
	}
	if element, ok := s.data[id]; ok {
		element[key] = v
	}
}

// BytesToInt takes a variable length of bytes and assume ascii chars and convert it to int, this is
// a simplified implementation of strconv.Atoi's fast path without error handling and remove the
// need to convert the byte array to string, we also assume that any errors are taken care at
// the parsing level.
func bytesToInt(b []byte) int {
	var i int
	for _, x := range b {
		i = i*10 + int(x-'0')
	}
	return i
}

func skipLeadZero(b []byte) []byte {
	if len(b) > 1 && b[0] == '0' {
		return b[1:]
	}
	return b
}
