// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package etw

import (
	"testing"
	"time"

	"github.com/elastic/beats/v7/x-pack/libbeat/reader/etw"
	"gotest.tools/assert"

	"golang.org/x/sys/windows"
)

func Test_fillEventHeader(t *testing.T) {
	tests := []struct {
		name     string
		header   etw.EventHeader
		expected map[string]interface{}
	}{
		{
			name: "Test with Level 1 (Critical)",
			header: etw.EventHeader{
				Size:          100,
				HeaderType:    10,
				Flags:         20,
				EventProperty: 30,
				ThreadId:      40,
				ProcessId:     50,
				TimeStamp:     133516441890350000,
				ProviderId: windows.GUID{
					Data1: 0x12345678,
					Data2: 0x1234,
					Data3: 0x1234,
					Data4: [8]byte{0x12, 0x34, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc},
				},
				EventDescriptor: etw.EventDescriptor{
					Id:      60,
					Version: 70,
					Channel: 80,
					Level:   1, // Critical
					Opcode:  90,
					Task:    100,
					Keyword: 110,
				},
				Time: 120,
				ActivityId: windows.GUID{
					Data1: 0x12345678,
					Data2: 0x1234,
					Data3: 0x1234,
					Data4: [8]byte{0x12, 0x34, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc},
				},
			},
			expected: map[string]interface{}{
				"size":           100,
				"type":           10,
				"flags":          20,
				"event_property": 30,
				"thread_id":      40,
				"process_id":     50,
				"timestamp":      "2024-02-05T22:03:09.035Z",
				"provider_guid":  "{12345678-1234-1234-1234-123456789ABC}",
				"event_id":       60,
				"event_version":  70,
				"channel":        80,
				"level":          1,
				"severity":       "critical",
				"opcode":         90,
				"task":           100,
				"keyword":        110,
				"time":           120,
				"activity_guid":  "{12345678-1234-1234-1234-123456789ABC}",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := fillEventHeader(tt.header)
			assert.Equal(t, tt.expected["size"], header["size"])

		})
	}
}

func Test_convertFileTimeToGoTime(t *testing.T) {
	tests := []struct {
		name     string
		fileTime uint64
		want     time.Time
	}{
		{
			name:     "Windows epoch",
			fileTime: 0, // January 1, 1601 (Windows epoch)
			want:     time.Date(1601, 01, 01, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "Unix epoch",
			fileTime: 116444736000000000, // January 1, 1970 (Unix epoch)
			want:     time.Unix(0, 0),
		},
		{
			name:     "Actual date",
			fileTime: 133515900000000000, // February 05, 2023, 7:00:00 AM
			want:     time.Date(2023, 02, 05, 7, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertFileTimeToGoTime(tt.fileTime)
			if !got.Equal(tt.want) {
				t.Errorf("convertFileTimeToGoTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_fillEventMetadata(t *testing.T) {
	tests := []struct {
		name     string
		record   *etw.EventRecord
		session  *etw.Session
		cfg      config
		expected map[string]interface{}
	}{
		// Test Provider Name and GUID from config
		{
			name:   "TestProviderNameAndGUIDFromConfig",
			record: &etw.EventRecord{},
			session: &etw.Session{
				GUID: windows.GUID{},
				Name: "SessionName",
			},
			cfg: config{
				ProviderName: "TestProvider",
				ProviderGUID: "{12345678-1234-1234-1234-123456789abc}",
			},
			expected: map[string]interface{}{
				"ProviderName": "TestProvider",
				"ProviderGUID": "{12345678-1234-1234-1234-123456789abc}",
			},
		},
		// Test Provider GUID from session if not available in config
		{
			name:   "TestProviderGUIDFromSession",
			record: &etw.EventRecord{},
			session: &etw.Session{
				GUID: windows.GUID{
					Data1: 0x12345678,
					Data2: 0x1234,
					Data3: 0x1234,
					Data4: [8]byte{0x12, 0x34, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc},
				},
			},
			cfg: config{
				ProviderName: "TestProvider",
			},
			expected: map[string]interface{}{
				"ProviderName": "TestProvider",
				"ProviderGUID": "{12345678-1234-1234-1234-123456789abc}",
			},
		},
		// Test Logfile and Session Information
		{
			name:   "TestLogfileAndSessionInfo",
			record: &etw.EventRecord{},
			session: &etw.Session{
				GUID: windows.GUID{},
				Name: "SessionName",
			},
			cfg: config{
				Logfile:     "C:\\Logs\\test.log",
				Session:     "TestSession",
				SessionName: "DifferentSessionName",
			},
			expected: map[string]interface{}{
				"Logfile": "C:\\Logs\\test.log",
				"Session": "TestSession",
			},
		},
		// Test with nil EventRecord
		{
			name:     "TestWithNilEventRecord",
			record:   nil,
			session:  nil,
			cfg:      config{},
			expected: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fillEventMetadata(tt.record, tt.session, tt.cfg)
			assert.Equal(t, tt.expected, result, "fillEventMetadata() should match the expected output")
		})
	}
}
