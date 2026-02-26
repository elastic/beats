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

//go:build windows

package etw

import (
	"encoding/binary"
	"syscall"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"golang.org/x/sys/windows"
)

func TestGUIDFromProviderName_EmptyName(t *testing.T) {
	guid, err := guidFromProviderName("")
	assert.EqualError(t, err, "empty provider name")
	assert.Equal(t, windows.GUID{}, guid, "GUID should be empty for an empty provider name")
}

func TestGUIDFromProviderName_EmptyProviderList(t *testing.T) {
	// Defer restoration of the original function
	t.Cleanup(func() {
		enumerateProvidersFunc = _TdhEnumerateProviders
	})

	// Define a mock provider name and GUID for testing.
	mockProviderName := "NonExistentProvider"

	enumerateProvidersFunc = func(pBuffer *ProviderEnumerationInfo, pBufferSize *uint32) error {
		// Check if the buffer size is sufficient
		requiredSize := uint32(unsafe.Sizeof(ProviderEnumerationInfo{})) + uint32(unsafe.Sizeof(TraceProviderInfo{}))*0 // As there are no providers
		if *pBufferSize < requiredSize {
			// Set the size required and return the error
			*pBufferSize = requiredSize
			return ERROR_INSUFFICIENT_BUFFER
		}

		// Empty list of providers
		*pBuffer = ProviderEnumerationInfo{
			NumberOfProviders:      0,
			TraceProviderInfoArray: [anysizeArray]TraceProviderInfo{},
		}
		return nil
	}

	guid, err := guidFromProviderName(mockProviderName)
	assert.EqualError(t, err, "no providers found")
	assert.Equal(t, windows.GUID{}, guid, "GUID should be empty when the provider is not found")
}

func TestGUIDFromProviderName_GUIDNotFound(t *testing.T) {
	// Defer restoration of the original function
	t.Cleanup(func() {
		enumerateProvidersFunc = _TdhEnumerateProviders
	})

	// Define a mock provider name and GUID for testing.
	mockProviderName := "NonExistentProvider"
	realProviderName := "ExistentProvider"
	mockGUID := windows.GUID{Data1: 1234, Data2: 5678}

	enumerateProvidersFunc = func(pBuffer *ProviderEnumerationInfo, pBufferSize *uint32) error {
		// Convert provider name to UTF-16
		utf16ProviderName, _ := syscall.UTF16FromString(realProviderName)

		// Calculate size needed for the provider name string
		nameSize := (len(utf16ProviderName) + 1) * 2 // +1 for null-terminator

		requiredSize := uint32(unsafe.Sizeof(ProviderEnumerationInfo{}) + unsafe.Sizeof(TraceProviderInfo{}) + uintptr(nameSize))
		if *pBufferSize < requiredSize {
			*pBufferSize = requiredSize
			return ERROR_INSUFFICIENT_BUFFER
		}

		// Calculate the offset for the provider name
		// It's placed after ProviderEnumerationInfo and TraceProviderInfo
		nameOffset := unsafe.Sizeof(ProviderEnumerationInfo{}) + unsafe.Sizeof(TraceProviderInfo{})

		// Convert pBuffer to a byte slice starting at the calculated offset for the name
		byteBuffer := (*[1 << 30]byte)(unsafe.Pointer(pBuffer))[:]
		// Copy the UTF-16 encoded name into the buffer
		for i, char := range utf16ProviderName {
			binary.LittleEndian.PutUint16(byteBuffer[nameOffset+(uintptr(i)*2):], char)
		}

		// Create and populate the ProviderEnumerationInfo struct
		*pBuffer = ProviderEnumerationInfo{
			NumberOfProviders: 1,
			TraceProviderInfoArray: [anysizeArray]TraceProviderInfo{
				{
					ProviderGuid:       mockGUID,
					ProviderNameOffset: uint32(nameOffset),
				},
			},
		}
		return nil
	}

	guid, err := guidFromProviderName(mockProviderName)
	assert.EqualError(t, err, "unable to find GUID from provider name")
	assert.Equal(t, windows.GUID{}, guid, "GUID should be empty when the provider is not found")
}

func TestGUIDFromProviderName_Success(t *testing.T) {
	// Defer restoration of the original function
	t.Cleanup(func() {
		enumerateProvidersFunc = _TdhEnumerateProviders
	})

	// Define a mock provider name and GUID for testing.
	mockProviderName := "MockProvider"
	mockGUID := windows.GUID{Data1: 1234, Data2: 5678}

	enumerateProvidersFunc = func(pBuffer *ProviderEnumerationInfo, pBufferSize *uint32) error {
		// Convert provider name to UTF-16
		utf16ProviderName, _ := syscall.UTF16FromString(mockProviderName)

		// Calculate size needed for the provider name string
		nameSize := (len(utf16ProviderName) + 1) * 2 // +1 for null-terminator

		requiredSize := uint32(unsafe.Sizeof(ProviderEnumerationInfo{}) + unsafe.Sizeof(TraceProviderInfo{}) + uintptr(nameSize))
		if *pBufferSize < requiredSize {
			*pBufferSize = requiredSize
			return ERROR_INSUFFICIENT_BUFFER
		}

		// Calculate the offset for the provider name
		// It's placed after ProviderEnumerationInfo and TraceProviderInfo
		nameOffset := unsafe.Sizeof(ProviderEnumerationInfo{}) + unsafe.Sizeof(TraceProviderInfo{})

		// Convert pBuffer to a byte slice starting at the calculated offset for the name
		byteBuffer := (*[1 << 30]byte)(unsafe.Pointer(pBuffer))[:]
		// Copy the UTF-16 encoded name into the buffer
		for i, char := range utf16ProviderName {
			binary.LittleEndian.PutUint16(byteBuffer[nameOffset+(uintptr(i)*2):], char)
		}

		// Create and populate the ProviderEnumerationInfo struct
		*pBuffer = ProviderEnumerationInfo{
			NumberOfProviders: 1,
			TraceProviderInfoArray: [anysizeArray]TraceProviderInfo{
				{
					ProviderGuid:       mockGUID,
					ProviderNameOffset: uint32(nameOffset),
				},
			},
		}
		return nil
	}

	// Run the test
	guid, err := guidFromProviderName(mockProviderName)
	assert.NoError(t, err)
	assert.Equal(t, mockGUID, guid, "GUID should match the mock GUID")
}
