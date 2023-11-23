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
)

func TestUTF16PtrToString(t *testing.T) {
	// Create a UTF-16 string
	sampleText := "This is a string test!"
	utf16Str, _ := syscall.UTF16FromString(sampleText)

	// Convert to *uint16
	ptr := &utf16Str[0]

	// Test the function
	result := UTF16PtrToString(ptr)
	assert.Equal(t, sampleText, result, "The converted string should match the original")
}

func TestUTF16AtOffsetToString(t *testing.T) {
	// Create a UTF-16 string
	sampleText := "This is a string test!"
	utf16Str, _ := syscall.UTF16FromString(sampleText)

	// Convert it to uintptr (simulate as if it's part of a larger struct)
	ptr := uintptr(unsafe.Pointer(&utf16Str[0]))

	// Test the function
	result := UTF16AtOffsetToString(ptr, 0)
	assert.Equal(t, sampleText, result, "The converted string should match the original")

	// Test with offset (skip the first character)
	offset := unsafe.Sizeof(utf16Str[0]) // Size of one UTF-16 character
	resultWithOffset := UTF16AtOffsetToString(ptr, offset)
	assert.Equal(t, sampleText[1:], resultWithOffset, "The converted string with offset should skip the first character")
}

func TestGUIDFromProviderName_EmptyName(t *testing.T) {
	guid, err := GUIDFromProviderName("")
	assert.EqualError(t, err, "empty provider name")
	assert.Equal(t, GUID{}, guid, "GUID should be empty for an empty provider name")
}

func TestGUIDFromProviderName_EmptyProviderList(t *testing.T) {
	// Backup and defer restoration of the original function
	originalFunc := EnumerateProvidersFunc
	defer func() { EnumerateProvidersFunc = originalFunc }()

	// Define a mock provider name and GUID for testing.
	mockProviderName := "NonExistentProvider"

	EnumerateProvidersFunc = func(pBuffer *ProviderEnumerationInfo, pBufferSize *uint32) error {
		// Empty list of providers
		*pBuffer = ProviderEnumerationInfo{
			NumberOfProviders:      0,
			TraceProviderInfoArray: [ANYSIZE_ARRAY]TraceProviderInfo{},
		}
		return nil
	}

	guid, err := GUIDFromProviderName(mockProviderName)
	assert.EqualError(t, err, "no providers found")
	assert.Equal(t, GUID{}, guid, "GUID should be empty when the provider is not found")
}

func TestGUIDFromProviderName_GUIDNotFound(t *testing.T) {
	// Backup and defer restoration of the original function
	originalFunc := EnumerateProvidersFunc
	defer func() { EnumerateProvidersFunc = originalFunc }()

	// Define a mock provider name and GUID for testing.
	mockProviderName := "NonExistentProvider"
	realProviderName := "ExistentProvider"
	mockGUID := GUID{Data1: 1234, Data2: 5678}

	EnumerateProvidersFunc = func(pBuffer *ProviderEnumerationInfo, pBufferSize *uint32) error {
		// Create and populate a buffer for the provider name
		utf16Name, _ := syscall.UTF16FromString(realProviderName)
		nameBuffer := make([]byte, len(utf16Name)*2)
		for i, v := range utf16Name {
			binary.LittleEndian.PutUint16(nameBuffer[i*2:], v)
		}

		// Create and populate the ProviderEnumerationInfo
		*pBuffer = ProviderEnumerationInfo{
			NumberOfProviders: 1,
			TraceProviderInfoArray: [ANYSIZE_ARRAY]TraceProviderInfo{
				{
					ProviderGuid:       mockGUID,
					ProviderNameOffset: uint32(uintptr(unsafe.Pointer(&nameBuffer[0]))),
				},
			},
		}
		return nil
	}

	guid, err := GUIDFromProviderName(mockProviderName)
	assert.EqualError(t, err, "unable to find GUID from provider name")
	assert.Equal(t, GUID{}, guid, "GUID should be empty when the provider is not found")
}

func TestGUIDFromProviderName_Success(t *testing.T) {
	// Backup and defer restoration of the original function
	originalFunc := EnumerateProvidersFunc
	defer func() { EnumerateProvidersFunc = originalFunc }()

	// Define a mock provider name and GUID for testing.
	mockProviderName := "MockProvider"
	mockGUID := GUID{Data1: 1234, Data2: 5678}

	EnumerateProvidersFunc = func(pBuffer *ProviderEnumerationInfo, pBufferSize *uint32) error {
		// Create and populate a buffer for the provider name
		utf16Name, _ := syscall.UTF16FromString(mockProviderName)
		nameBuffer := make([]byte, len(utf16Name)*2)
		for i, v := range utf16Name {
			binary.LittleEndian.PutUint16(nameBuffer[i*2:], v)
		}

		// Create and populate the ProviderEnumerationInfo
		*pBuffer = ProviderEnumerationInfo{
			NumberOfProviders: 1,
			TraceProviderInfoArray: [ANYSIZE_ARRAY]TraceProviderInfo{
				{
					ProviderGuid:       mockGUID,
					ProviderNameOffset: uint32(uintptr(unsafe.Pointer(&nameBuffer[0]))),
				},
			},
		}
		return nil
	}

	// Run the test
	guid, err := GUIDFromProviderName(mockProviderName)
	assert.NoError(t, err)
	assert.Equal(t, mockGUID, guid, "GUID should match the mock GUID")
}
