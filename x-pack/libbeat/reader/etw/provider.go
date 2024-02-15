// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package etw

import (
	"errors"
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// utf16AtOffsetToString converts a UTF-16 encoded string
// at a specific offset in a struct to a Go string.
func utf16AtOffsetToString(pstruct uintptr, offset uintptr) string {
	// Initialize a slice to store UTF-16 characters.
	out := make([]uint16, 0, 64)

	// Start reading at the given offset.
	wc := (*uint16)(unsafe.Pointer(pstruct + offset))

	// Iterate over the UTF-16 characters until a null terminator is encountered.
	for i := uintptr(2); *wc != 0; i += 2 {
		out = append(out, *wc)
		wc = (*uint16)(unsafe.Pointer(pstruct + offset + i))
	}

	// Convert the UTF-16 slice to a Go string and return.
	return syscall.UTF16ToString(out)
}

// guidFromProviderName searches for a provider by name and returns its GUID.
func guidFromProviderName(providerName string) (windows.GUID, error) {
	// Returns if the provider name is empty.
	if providerName == "" {
		return windows.GUID{}, fmt.Errorf("empty provider name")
	}

	var buf *ProviderEnumerationInfo
	size := uint32(1)

	// Attempt to retrieve provider information with a buffer that increases in size until it's sufficient.
	for {
		tmp := make([]byte, size)
		buf = (*ProviderEnumerationInfo)(unsafe.Pointer(&tmp[0]))
		if err := enumerateProvidersFunc(buf, &size); !errors.Is(err, ERROR_INSUFFICIENT_BUFFER) {
			break
		}
	}

	if buf.NumberOfProviders == 0 {
		return windows.GUID{}, fmt.Errorf("no providers found")
	}

	// Iterate through the list of providers to find a match by name.
	startProvEnumInfo := uintptr(unsafe.Pointer(buf))
	it := uintptr(unsafe.Pointer(&buf.TraceProviderInfoArray[0]))
	for i := uintptr(0); i < uintptr(buf.NumberOfProviders); i++ {
		pInfo := (*TraceProviderInfo)(unsafe.Pointer(it + i*unsafe.Sizeof(buf.TraceProviderInfoArray[0])))
		name := utf16AtOffsetToString(startProvEnumInfo, uintptr(pInfo.ProviderNameOffset))

		// If a match is found, return the corresponding GUID.
		if name == providerName {
			return pInfo.ProviderGuid, nil
		}
	}

	// No matching provider is found.
	return windows.GUID{}, fmt.Errorf("unable to find GUID from provider name")
}
