// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package etw

import (
	"errors"
	"fmt"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

type RenderedEtwEvent struct {
	ProviderGUID          windows.GUID
	ProviderName          string
	EventID               uint16
	Version               uint8
	Level                 string
	LevelRaw              uint8
	Task                  string
	TaskRaw               uint16
	Opcode                string
	OpcodeRaw             uint8
	Keywords              []string
	KeywordsRaw           uint64
	Channel               string
	Timestamp             time.Time
	ProcessID             uint32
	ThreadID              uint32
	ActivityIDName        string
	RelatedActivityIDName string
	EventMessage          string
	ProviderMessage       string
	Properties            []RenderedProperty
	ExtendedData          []RenderedExtendedData
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

// ntKernelLoggerProviderGUID is the GUID for the NT Kernel Logger/System Trace Control.
var ntKernelLoggerProviderGUID windows.GUID

// kernelEventClasses maps a known kernel event class GUID to its descriptive name.
// The actual provider for these event classes is always the NTKernelLoggerProviderGUID.
var kernelEventClasses map[windows.GUID]string
var loadGUIDsOnce sync.Once

// findEffectiveGUID determines the actual logging provider GUID.
// If the input 'guid' is a known kernel event class GUID, it returns
// the NTKernelLoggerProviderGUID. If 'guid' is the NTKernelLoggerProviderGUID
// itself, it's returned directly. Otherwise, the input 'guid' is returned,
// assuming it's a specific provider (e.g., manifest-based).
func findEffectiveGUID(guid windows.GUID) windows.GUID {
	loadGUIDsOnce.Do(func() {
		newGuid := func(str string) windows.GUID {
			g, _ := windows.GUIDFromString(str)
			return g
		}
		ntKernelLoggerProviderGUID = newGuid("{9e814aad-3204-11d2-9a82-006008a86939}")
		kernelEventClasses = map[windows.GUID]string{
			newGuid("{68fdd900-4a3e-11d1-84f4-0000f80464e3}"): "EventTraceEvent",
			newGuid("{3d6fa8d0-fe05-11d0-9dda-00c04fd7ba7c}"): "Process",
			newGuid("{3d6fa8d1-fe05-11d0-9dda-00c04fd7ba7c}"): "Thread",
			newGuid("{3d6fa8d4-fe05-11d0-9dda-00c04fd7ba7c}"): "DiskIo",
			newGuid("{ae53722e-c863-11d2-8659-00c04fa321a1}"): "Registry",
			newGuid("{d837ca92-12b9-44a5-ad6a-3a65b3578aa8}"): "SplitIo",
			newGuid("{90cbdc39-4a3e-11d1-84f4-0000f80464e3}"): "FileIo",
			newGuid("{9a280ac0-c8e0-11d1-84e2-00c04fb998a2}"): "TcpIp",
			newGuid("{bf3a50c5-a9c9-4988-a005-2df0b7c80f80}"): "UdpIp",
			newGuid("{2cb15d1d-5fc1-11d2-abe1-00a0c911f518}"): "Image",
			newGuid("{3d6fa8d3-fe05-11d0-9dda-00c04fd7ba7c}"): "PageFault",
			newGuid("{ce1dbfb4-137e-4da6-87b0-3f59aa102cbc}"): "PerfInfo",
			newGuid("{def2fe46-7bd6-4b80-bd94-f57fe20d0ce3}"): "StackWalk",
			newGuid("{45d8cccd-539f-4b72-a8b7-5c683142609a}"): "ALPC",
			newGuid("{6a399ae0-4bc6-4de9-870b-3657f8947e7e}"): "Lost_Event",
			newGuid("{01853a65-418f-4f36-aefc-dc0f1d2fd235}"): "SystemConfig",
		}
	})
	// Check if the input GUID is one of the known kernel event class GUIDs
	if _, found := kernelEventClasses[guid]; found {
		// For these classes, the NT Kernel Logger is the actual provider
		return ntKernelLoggerProviderGUID
	}

	// Otherwise, the input GUID is assumed to be the effective provider GUID
	return guid
}
