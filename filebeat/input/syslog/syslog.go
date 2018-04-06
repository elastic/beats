package main

import "time"

// SyslogMessage keeps an index reference to the orignal slices of bytes.
// and will lazy convert the byte values to concrete usable type.
type SyslogMessage struct {
	Message  []byte
	Hostname []byte
	Priority []byte
	Program  []byte
	Pid      []byte //int

	month  []byte
	day    []byte
	hour   []byte
	minute []byte
	second []byte
}

func newSyslogMessage() *SyslogMessage {
	return &SyslogMessage{}
}

func (s *SyslogMessage) Month(m []byte) {
	s.month = m
}

func (s *SyslogMessage) GetMonth() int {
	return 0
}

func (s *SyslogMessage) Day(d []byte) {
	s.day = d
}

func (s *SyslogMessage) Hour(h []byte) {
	s.hour = h
}

func (s *SyslogMessage) Minute(h []byte) {
	s.minute = h
}

func (s *SyslogMessage) Second(sec []byte) {
	s.second = sec
}

func (s SyslogMessage) Ts() (*time.Time, error) {
	// d := time.Date(time.Now().Year(), s.GetMonth())
	// // func Date(year int, month Month, day, hour, min, sec, nsec int, loc *Location) Time
	return nil, nil
}

func (s SyslogMessage) IsValid() bool {
	return true
}

// Severity int
// Facility int
// event.set("priority", 13)
// event.set("severity", 5)   # 13 & 7 == 5
// event.set("facility", 1)   # 13 >> 3 == 1
