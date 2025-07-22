// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package etw

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"reflect"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Helper function to create a UTF-16 multi-string byte buffer.
// Each string in the input slice is null-terminated, and the entire list
// is terminated by an additional null (effectively an empty string's null terminator).
// Format: Str1\0Str2\0\0 (where \0 is a UTF-16 U+0000 character)
func makeUTF16MultiStringBuffer(strs []string) []byte {
	var buf bytes.Buffer
	for _, s := range strs {
		utf16Sequence, err := windows.UTF16FromString(s) // Gives []uint16 for content
		if err != nil {
			panic(fmt.Sprintf("UTF16FromString failed for string '%s': %v", s, err))
		}
		for _, u16 := range utf16Sequence {
			err := binary.Write(&buf, binary.LittleEndian, u16)
			if err != nil {
				panic(fmt.Sprintf("binary.Write failed for u16 value of string '%s': %v", s, err))
			}
		}
		err = binary.Write(&buf, binary.LittleEndian, uint16(0)) // String's null terminator
		if err != nil {
			panic(fmt.Sprintf("binary.Write failed for string terminator after '%s': %v", s, err))
		}
	}
	err := binary.Write(&buf, binary.LittleEndian, uint16(0)) // List's terminating empty string's null terminator
	if err != nil {
		panic(fmt.Sprintf("binary.Write failed for final list terminator: %v", err))
	}
	return buf.Bytes()
}

func TestGetEventInfoMultiStringFromOffset(t *testing.T) {
	// This helper creates a buffer. `info` will point to its start.
	// `stringListOffset` will be the offset from `info` where the multi-string data begins.
	makeTestBed := func(teiFixedSize int, stringListToEmbed []string) (backingBuffer []byte, stringListOffset uint32) {
		multiStringBytes := makeUTF16MultiStringBuffer(stringListToEmbed)

		buffer := make([]byte, teiFixedSize+len(multiStringBytes))
		copy(buffer[teiFixedSize:], multiStringBytes)

		return buffer, uint32(teiFixedSize)
	}

	tests := []struct {
		name           string
		teiFixedSize   int      // Size of the simulated fixed part of TraceEventInfo
		stringList     []string // Strings to embed after the fixed part
		offsetToPass   uint32   // The offset value passed to the function under test
		expectedResult []string // Can be nil if the function is expected to return nil
	}{
		{
			name:           "info is nil",
			teiFixedSize:   0,   // Doesn't matter as info will be nil
			stringList:     nil, // No strings
			offsetToPass:   1,   // A non-zero offset
			expectedResult: nil, // Function returns nil if info is nil
		},
		{
			name:           "offset is 0 (with non-nil info)",
			teiFixedSize:   0, // Strings start right at info
			stringList:     []string{"Test"},
			offsetToPass:   0, // Function returns nil if offset is 0
			expectedResult: nil,
		},
		{
			name:           "Valid offset, empty list (double null at offset)",
			teiFixedSize:   int(unsafe.Sizeof(TraceEventInfo{})),
			stringList:     []string{}, // Creates only a double null list terminator
			offsetToPass:   uint32(unsafe.Sizeof(TraceEventInfo{})),
			expectedResult: nil, // Loop for str=="" breaks, results is nil initially
		},
		{
			name:           "Valid offset, single string",
			teiFixedSize:   int(unsafe.Sizeof(TraceEventInfo{})),
			stringList:     []string{"Hello"},
			offsetToPass:   uint32(unsafe.Sizeof(TraceEventInfo{})),
			expectedResult: []string{"Hello"},
		},
		{
			name:           "Valid offset, multiple strings",
			teiFixedSize:   int(unsafe.Sizeof(TraceEventInfo{})),
			stringList:     []string{"First", "Second", "Third"},
			offsetToPass:   uint32(unsafe.Sizeof(TraceEventInfo{})),
			expectedResult: []string{"First", "Second", "Third"},
		},
		{
			name:           "Strings with surrogate pairs",
			teiFixedSize:   int(unsafe.Sizeof(TraceEventInfo{})),
			stringList:     []string{"Hi😀", "Test"}, // 😀 is U+1F600
			offsetToPass:   uint32(unsafe.Sizeof(TraceEventInfo{})),
			expectedResult: []string{"Hi😀", "Test"},
		},
		{
			name:           "List with empty string element",
			teiFixedSize:   int(unsafe.Sizeof(TraceEventInfo{})),
			stringList:     []string{"Start", "", "End"},
			offsetToPass:   uint32(unsafe.Sizeof(TraceEventInfo{})),
			expectedResult: []string{"Start"},
		},
		{
			name:           "List starting with empty string",
			teiFixedSize:   int(unsafe.Sizeof(TraceEventInfo{})),
			stringList:     []string{"", "Middle", "End"},
			offsetToPass:   uint32(unsafe.Sizeof(TraceEventInfo{})),
			expectedResult: []string{},
		},
		// NOTE: Testing for out-of-bounds reads is difficult and dangerous with this unsafe function.
		// Such tests would likely cause a panic rather than a clean failure if the function
		// indeed reads out of bounds. The tests here assume the underlying memory (our backingBuffer)
		// is correctly formatted and large enough for the declared strings + terminators.
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var backingBufferForTest []byte

			if tt.name != "info is nil" {
				backingBufferForTest, _ = makeTestBed(tt.teiFixedSize, tt.stringList)
				_ = backingBufferForTest // Keep backingBuffer alive
			}

			got := getMultiStringFromBufferOffset(backingBufferForTest, tt.offsetToPass)

			// The user's function returns nil if the initial `results` slice is never appended to.
			// This happens for empty lists or if the first check `str == ""` is true.
			// `reflect.DeepEqual` considers a nil slice and an empty non-nil slice as different.
			// So, if `expectedResult` is `[]string{}`, we need to accept `nil` from `got`.
			if !((len(tt.expectedResult) == 0 && got == nil) || reflect.DeepEqual(got, tt.expectedResult)) {
				t.Errorf("getEventInfoMultiStringFromOffset(info, %d):\ngot:  %#v (%T)\nwant: %#v (%T)",
					tt.offsetToPass, got, got, tt.expectedResult, tt.expectedResult)
			}
		})
	}
}
