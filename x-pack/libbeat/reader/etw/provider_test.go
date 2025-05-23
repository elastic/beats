// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

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

func TestUTF16AtOffsetToString(t *testing.T) {
	// Create a UTF-16 string
	sampleText := "This is a string test!"
	utf16Str, _ := syscall.UTF16FromString(sampleText)

	// Convert it to uintptr (simulate as if it's part of a larger struct)
	ptr := uintptr(unsafe.Pointer(&utf16Str[0]))

	// Test the function
	result := utf16AtOffsetToString(ptr, 0)
	assert.Equal(t, sampleText, result, "The converted string should match the original")

	// Test with offset (skip the first character)
	offset := unsafe.Sizeof(utf16Str[0]) // Size of one UTF-16 character
	resultWithOffset := utf16AtOffsetToString(ptr, offset)
	assert.Equal(t, sampleText[1:], resultWithOffset, "The converted string with offset should skip the first character")
}

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

		requiredSize := uint32(unsafe.Sizeof(ProviderEnumerationInfo{})) + uint32(unsafe.Sizeof(TraceProviderInfo{})) + uint32(nameSize)
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

		requiredSize := uint32(unsafe.Sizeof(ProviderEnumerationInfo{})) + uint32(unsafe.Sizeof(TraceProviderInfo{})) + uint32(nameSize)
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
