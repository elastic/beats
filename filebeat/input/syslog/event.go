package syslog

import (
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
	loc        *time.Location
}

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
	}
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

// Year returns the current year, since syslog events don't include that.
func (s *event) Year() int {
	return time.Now().Year()
}

// SetMessage sets the message.
func (s *event) SetMessage(b []byte) {
	s.message = string(b)
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

// SetNanoSecond sets the nanosecond.
func (s *event) SetNanosecond(b []byte) {
	s.nanosecond = bytesToInt(skipLeadZero(b))
}

// NanoSecond returns the nanosecond.
func (s *event) Nanosecond() int {
	return s.nanosecond
}

// Timestamp return the timestamp in UTC.
func (s *event) Timestamp(timezone *time.Location) time.Time {
	return time.Date(
		s.Year(),
		s.Month(),
		s.Day(),
		s.Hour(),
		s.Minute(),
		s.Second(),
		s.Nanosecond(),
		timezone,
	).UTC()
}

// IsValid returns true if the date and the message are present.
func (s *event) IsValid() bool {
	return s.day != -1 && s.hour != -1 && s.minute != -1 && s.second != -1 && s.message != ""
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
		return b[1:len(b)]
	}
	return b
}
