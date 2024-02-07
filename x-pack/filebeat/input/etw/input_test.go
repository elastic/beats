// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package etw

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/x-pack/libbeat/reader/etw"

	"golang.org/x/sys/windows"
)

func Test_fillEventHeader(t *testing.T) {
	tests := []struct {
		name     string
		header   etw.EventHeader
		expected map[string]interface{}
	}{
		{
			name: "TestStandardHeader",
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
				"size":           uint16(100),
				"type":           uint16(10),
				"flags":          uint16(20),
				"event_property": uint16(30),
				"thread_id":      uint32(40),
				"process_id":     uint32(50),
				"timestamp":      "2024-02-05T22:03:09.035Z",
				"provider_guid":  "{12345678-1234-1234-1234-123456789ABC}",
				"event_id":       uint16(60),
				"event_version":  uint8(70),
				"channel":        uint8(80),
				"level":          uint8(1),
				"severity":       "critical",
				"opcode":         uint8(90),
				"task":           uint16(100),
				"keyword":        uint64(110),
				"time":           uint64(120),
				"activity_guid":  "{12345678-1234-1234-1234-123456789ABC}",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := fillEventHeader(tt.header)
			assert.Equal(t, tt.expected["size"], header["size"])
			assert.Equal(t, tt.expected["type"], header["type"])
			assert.Equal(t, tt.expected["flags"], header["flags"])
			assert.Equal(t, tt.expected["event_property"], header["event_property"])
			assert.Equal(t, tt.expected["thread_id"], header["thread_id"])
			assert.Equal(t, tt.expected["process_id"], header["process_id"])
			assert.Equal(t, tt.expected["provider_guid"], header["provider_guid"])
			assert.Equal(t, tt.expected["event_id"], header["event_id"])
			assert.Equal(t, tt.expected["event_version"], header["event_version"])
			assert.Equal(t, tt.expected["channel"], header["channel"])
			assert.Equal(t, tt.expected["level"], header["level"])
			assert.Equal(t, tt.expected["severity"], header["severity"])
			assert.Equal(t, tt.expected["opcode"], header["opcode"])
			assert.Equal(t, tt.expected["task"], header["task"])
			assert.Equal(t, tt.expected["keyword"], header["keyword"])
			assert.Equal(t, tt.expected["time"], header["time"])
			assert.Equal(t, tt.expected["activity_guid"], header["activity_guid"])
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
			name:     "TestZeroValue",
			fileTime: 0,
			want:     time.Time{},
		},
		{
			name:     "TestUnixEpoch",
			fileTime: 116444736000000000, // January 1, 1970 (Unix epoch)
			want:     time.Unix(0, 0),
		},
		{
			name:     "TestActualDate",
			fileTime: 133515900000000000, // February 05, 2024, 7:00:00 AM
			want:     time.Date(2024, 02, 05, 7, 0, 0, 0, time.UTC),
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
		session  *etw.Session
		cfg      config
		expected map[string]interface{}
	}{
		// Test Provider Name and GUID from config
		{
			name: "TestProviderNameAndGUIDFromConfig",
			session: &etw.Session{
				GUID: windows.GUID{},
				Name: "SessionName",
			},
			cfg: config{
				ProviderName: "TestProvider",
				ProviderGUID: "{12345678-1234-1234-1234-123456789ABC}",
			},
			expected: map[string]interface{}{
				"provider_name": "TestProvider",
				"provider_guid": "{12345678-1234-1234-1234-123456789ABC}",
			},
		},
		// Test Provider GUID from session if not available in config
		{
			name: "TestProviderGUIDFromSession",
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
				"provider_name": "TestProvider",
				"provider_guid": "{12345678-1234-1234-1234-123456789ABC}",
			},
		},
		// Test Logfile and Session Information
		{
			name: "TestLogfileAndSessionInfo",
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
				"logfile": "C:\\Logs\\test.log",
				"session": "TestSession",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fillEventMetadata(tt.session, tt.cfg)
			assert.Equal(t, tt.expected, result, "fillEventMetadata() should match the expected output")
		})
	}
}
