// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package etw

import (
	"errors"
	"fmt"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

type RenderedEtwEvent struct {
	ProviderGUID      windows.GUID
	ProviderName      string
	EventID           uint16
	Version           uint8
	Level             string
	LevelRaw          uint8
	Task              string
	TaskRaw           uint16
	Opcode            string
	OpcodeRaw         uint8
	Keywords          []string
	KeywordsRaw       uint64
	Channel           string
	Timestamp         time.Time
	ProcessID         uint32
	ThreadID          uint32
	ActivityID        string
	RelatedActivityID string
	EventMessage      string
	ProviderMessage   string
	Properties        []RenderedProperty
	ExtendedData      []RenderedExtendedData
}

type RenderedProperty struct {
	Name  string
	Value any
}

type RenderedExtendedData struct {
	ExtType    string
	ExtTypeRaw uint16
	DataSize   uint16
	Data       any
}

// guidFromProviderName searches for a provider by name and returns its GUID.
func guidFromProviderName(providerName string) (windows.GUID, error) {
	// Returns if the provider name is empty.
	if providerName == "" {
		return windows.GUID{}, fmt.Errorf("empty provider name")
	}

	var err error
	var bufSize uint32
	var buf []byte
	var pEnum *ProviderEnumerationInfo
	if err = enumerateProvidersFunc(nil, &bufSize); errors.Is(err, ERROR_INSUFFICIENT_BUFFER) {
		buf = make([]byte, bufSize)
		pEnum = ((*ProviderEnumerationInfo)(unsafe.Pointer(&buf[0])))
		err = enumerateProvidersFunc(pEnum, &bufSize)
	}

	if pEnum.NumberOfProviders == 0 {
		return windows.GUID{}, fmt.Errorf("no providers found")
	}

	it := uintptr(unsafe.Pointer(&pEnum.TraceProviderInfoArray[0]))
	for i := uintptr(0); i < uintptr(pEnum.NumberOfProviders); i++ {
		pInfo := (*TraceProviderInfo)(unsafe.Pointer(it + i*unsafe.Sizeof(pEnum.TraceProviderInfoArray[0])))
		name := getStringFromBufferOffset(buf, pInfo.ProviderNameOffset)

		// If a match is found, return the corresponding GUID.
		if name == providerName {
			return pInfo.ProviderGuid, nil
		}
	}

	// No matching provider is found.
	return windows.GUID{}, fmt.Errorf("unable to find GUID from provider name")
}
