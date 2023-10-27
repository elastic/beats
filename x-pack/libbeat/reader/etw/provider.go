// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package etw

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// UTF16PtrToString transforms a *uint16 to a Go string
func UTF16PtrToString(utf16 *uint16) string {
	return UTF16AtOffsetToString(uintptr(unsafe.Pointer(utf16)), 0)
}

// Todo: this function is too unclear
func UTF16AtOffsetToString(pstruct uintptr, offset uintptr) string {
	out := make([]uint16, 0, 64)
	wc := (*uint16)(unsafe.Pointer(pstruct + offset))
	for i := uintptr(2); *wc != 0; i += 2 {
		out = append(out, *wc)
		wc = (*uint16)(unsafe.Pointer(pstruct + offset + i))
	}
	return syscall.UTF16ToString(out)
}

// Looks at the available providers in the system
// It returns a GUID given a Provider name
func GUIDFromProviderName(providerName string) (GUID, error) {
	if providerName == "" {
		return GUID{}, fmt.Errorf("Empty provider name.")
	}

	var buf *ProviderEnumerationInfo
	size := uint32(1)
	// Todo: change if possible the structure of this for
	// maybe call first _TdhEnumerateProviders with size = 0 and second with the real size
	for {
		tmp := make([]byte, size)
		buf = (*ProviderEnumerationInfo)(unsafe.Pointer(&tmp[0]))
		if err := _TdhEnumerateProviders(buf, &size); err != ERROR_INSUFFICIENT_BUFFER {
			break
		}
	}

	startProvEnumInfo := uintptr(unsafe.Pointer(buf))
	it := uintptr(unsafe.Pointer(&buf.TraceProviderInfoArray[0]))
	for i := uintptr(0); i < uintptr(buf.NumberOfProviders); i++ {
		pInfo := (*TraceProviderInfo)(unsafe.Pointer(it + i*unsafe.Sizeof(buf.TraceProviderInfoArray[0])))
		winGUID := windows.GUID(pInfo.ProviderGuid)
		guid := winGUID.String()
		name := UTF16AtOffsetToString(startProvEnumInfo, uintptr(pInfo.ProviderNameOffset))

		if name == providerName {
			fmt.Printf("Found GUID '%s' from provider name '%s'", guid, name)
			return pInfo.ProviderGuid, nil
		}
	}

	return GUID{}, fmt.Errorf("Unable to find GUID from provider name '%s'", providerName)

}
