// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows

// Package eventlog implements access to Windows event log.
//
package eventlog

import (
	"errors"
	"syscall"

	"golang.org/x/sys/windows"
)

// Log provides access to the system log.
type Log struct {
	Handle windows.Handle
}

// Open retrieves a handle to the specified event log.
func Open(source string) (*Log, error) {
	return OpenRemote("", source)
}

// OpenRemote does the same as Open, but on different computer host.
func OpenRemote(host, source string) (*Log, error) {
	if source == "" {
		return nil, errors.New("Specify event log source")
	}
	var s *uint16
	if host != "" {
		s = syscall.StringToUTF16Ptr(host)
	}
	h, err := windows.RegisterEventSource(s, syscall.StringToUTF16Ptr(source))
	if err != nil {
		return nil, err
	}
	return &Log{Handle: h}, nil
}

// Close closes event log l.
func (l *Log) Close() error {
	return windows.DeregisterEventSource(l.Handle)
}

// Report write an event message with event type etype and event ID eid to the
// end of event log l.
// etype should be one of Info, Success, Warning, Error, AuditSuccess, or AuditFailure.
// When EventCreate.exe is used, eid must be between 1 and 1000.
func (l *Log) Report(etype uint16, eid uint32, msgs []string) error {
	var msgPtrs []*uint16
	for _, msg := range msgs {
		msgPtrs = append(msgPtrs, syscall.StringToUTF16Ptr(msg))
	}
	var ptr **uint16
	if len(msgPtrs) > 0 {
		ptr = &msgPtrs[0]
	}
	return windows.ReportEvent(l.Handle, etype, 0, eid, 0, uint16(len(msgPtrs)), 0, ptr, nil)
}

// Success writes a success event msg with event id eid to the end of event log l.
// When EventCreate.exe is used, eid must be between 1 and 1000.
func (l *Log) Success(eid uint32, msg string) error {
	return l.Report(Success, eid, []string{msg})
}

// Info writes an information event msg with event id eid to the end of event log l.
// When EventCreate.exe is used, eid must be between 1 and 1000.
func (l *Log) Info(eid uint32, msg string) error {
	return l.Report(Info, eid, []string{msg})
}

// Warning writes an warning event msg with event id eid to the end of event log l.
// When EventCreate.exe is used, eid must be between 1 and 1000.
func (l *Log) Warning(eid uint32, msg string) error {
	return l.Report(Warning, eid, []string{msg})
}

// Error writes an error event msg with event id eid to the end of event log l.
// When EventCreate.exe is used, eid must be between 1 and 1000.
func (l *Log) Error(eid uint32, msg string) error {
	return l.Report(Error, eid, []string{msg})
}

// AuditSuccess writes an audit event msg with event id eid to the end of event log l.
// When EventCreate.exe is used, eid must be between 1 and 1000.
func (l *Log) AuditSuccess(eid uint32, msg string) error {
	return l.Report(AuditSuccess, eid, []string{msg})
}

// AuditFailure writes an audit event msg with event id eid to the end of event log l.
// When EventCreate.exe is used, eid must be between 1 and 1000.
func (l *Log) AuditFailure(eid uint32, msg string) error {
	return l.Report(AuditFailure, eid, []string{msg})
}
