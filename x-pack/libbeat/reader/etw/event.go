// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package etw

import (
	"errors"
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

// propertyParser is used for parsing properties from raw EVENT_RECORD structures.
type propertyParser struct {
	r       *EventRecord
	info    *TraceEventInfo
	data    []byte
	ptrSize uint32
}

// GetEventProperties extracts and returns properties from an ETW event record.
func GetEventProperties(r *EventRecord) (map[string]interface{}, error) {
	// Handle the case where the event only contains a string.
	if r.EventHeader.Flags == EVENT_HEADER_FLAG_STRING_ONLY {
		userDataPtr := (*uint16)(unsafe.Pointer(r.UserData))
		return map[string]interface{}{
			"_": utf16AtOffsetToString(uintptr(unsafe.Pointer(userDataPtr)), 0), // Convert the user data from UTF16 to string.
		}, nil
	}

	// Initialize a new property parser for the event record.
	p, err := newPropertyParser(r)
	if err != nil {
		return nil, fmt.Errorf("failed to parse event properties: %w", err)
	}

	// Iterate through each property of the event and format it
	properties := make(map[string]interface{}, int(p.info.TopLevelPropertyCount))
	for i := 0; i < int(p.info.TopLevelPropertyCount); i++ {
		name := p.getPropertyName(i)
		value, err := p.getPropertyValue(i)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %q value: %w", name, err)
		}
		properties[name] = value
	}

	return properties, nil
}

// newPropertyParser initializes a new property parser for a given event record.
func newPropertyParser(r *EventRecord) (*propertyParser, error) {
	info, err := getEventInformation(r)
	if err != nil {
		return nil, fmt.Errorf("failed to get event information: %w", err)
	}
	ptrSize := r.pointerSize()
	// Return a new propertyParser instance initialized with event record data and metadata.
	return &propertyParser{
		r:       r,
		info:    info,
		ptrSize: ptrSize,
		data:    unsafe.Slice((*uint8)(unsafe.Pointer(r.UserData)), r.UserDataLength),
	}, nil
}

// getEventPropertyInfoAtIndex looks for the EventPropertyInfo object at a specified index.
func (info *TraceEventInfo) getEventPropertyInfoAtIndex(i uint32) *EventPropertyInfo {
	if i < info.PropertyCount {
		// Calculate the address of the first element in EventPropertyInfoArray.
		eventPropertyInfoPtr := uintptr(unsafe.Pointer(&info.EventPropertyInfoArray[0]))
		// Adjust the pointer to point to the i-th EventPropertyInfo element.
		eventPropertyInfoPtr += uintptr(i) * unsafe.Sizeof(EventPropertyInfo{})

		return ((*EventPropertyInfo)(unsafe.Pointer(eventPropertyInfoPtr)))
	}
	return nil
}

// getEventInformation retrieves detailed metadata about an event record.
func getEventInformation(r *EventRecord) (info *TraceEventInfo, err error) {
	// Initially call TdhGetEventInformation to get the required buffer size.
	var bufSize uint32
	if err = _TdhGetEventInformation(r, 0, nil, nil, &bufSize); errors.Is(err, ERROR_INSUFFICIENT_BUFFER) {
		// Allocate enough memory for TRACE_EVENT_INFO based on the required size.
		buff := make([]byte, bufSize)
		info = ((*TraceEventInfo)(unsafe.Pointer(&buff[0])))
		// Retrieve the event information into the allocated buffer.
		err = _TdhGetEventInformation(r, 0, nil, info, &bufSize)
	}

	// Check for errors in retrieving the event information.
	if err != nil {
		return nil, fmt.Errorf("TdhGetEventInformation failed: %w", err)
	}

	return info, nil
}

// getPropertyName retrieves the name of the i-th event property in the event record.
func (p *propertyParser) getPropertyName(i int) string {
	// Convert the UTF16 property name to a Go string.
	namePtr := readPropertyName(p, i)
	return windows.UTF16PtrToString((*uint16)(namePtr))
}

// readPropertyName gets the pointer to the property name in the event information structure.
func readPropertyName(p *propertyParser, i int) unsafe.Pointer {
	// Calculate the pointer to the property name using its offset in the event property array.
	return unsafe.Add(unsafe.Pointer(p.info), p.info.getEventPropertyInfoAtIndex(uint32(i)).NameOffset)
}

// getPropertyValue retrieves the value of a specified event property.
func (p *propertyParser) getPropertyValue(i int) (interface{}, error) {
	propertyInfo := p.info.getEventPropertyInfoAtIndex(uint32(i))

	// Determine the size of the property array.
	arraySize, err := p.getArraySize(*propertyInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to get array size: %w", err)
	}

	// Initialize a slice to hold the property values.
	result := make([]interface{}, arraySize)
	for j := 0; j < int(arraySize); j++ {
		var (
			value interface{}
			err   error
		)
		// Parse the property value based on its type (simple or structured).
		if (propertyInfo.Flags & PropertyStruct) == PropertyStruct {
			value, err = p.parseStruct(*propertyInfo)
		} else {
			value, err = p.parseSimpleType(*propertyInfo)
		}
		if err != nil {
			return nil, err
		}
		result[j] = value
	}

	// Return the entire result set or the single value, based on the property count.
	if ((propertyInfo.Flags & PropertyParamCount) == PropertyParamCount) ||
		(propertyInfo.count() > 1) {
		return result, nil
	}
	return result[0], nil
}

// getArraySize calculates the size of an array property within an event.
func (p *propertyParser) getArraySize(propertyInfo EventPropertyInfo) (uint32, error) {
	// Check if the property's count is specified by another property.
	if (propertyInfo.Flags & PropertyParamCount) == PropertyParamCount {
		var dataDescriptor PropertyDataDescriptor
		// Locate the property containing the array size using the countPropertyIndex.
		dataDescriptor.PropertyName = readPropertyName(p, int(propertyInfo.count()))
		dataDescriptor.ArrayIndex = 0xFFFFFFFF
		// Retrieve the length of the array from the specified property.
		return getLengthFromProperty(p.r, &dataDescriptor)
	} else {
		// If the array size is directly specified, return it.
		return uint32(propertyInfo.count()), nil
	}
}

// getLengthFromProperty retrieves the length of a property from an event record.
func getLengthFromProperty(r *EventRecord, dataDescriptor *PropertyDataDescriptor) (uint32, error) {
	var length uint32
	// Call TdhGetProperty to get the length of the property specified by the dataDescriptor.
	err := _TdhGetProperty(
		r,
		0,
		nil,
		1,
		dataDescriptor,
		uint32(unsafe.Sizeof(length)),
		(*byte)(unsafe.Pointer(&length)),
	)
	if err != nil {
		return 0, err
	}
	return length, nil
}

// parseStruct extracts and returns the fields from an embedded structure within a property.
func (p *propertyParser) parseStruct(propertyInfo EventPropertyInfo) (map[string]interface{}, error) {
	// Determine the start and end indexes of the structure members within the property info.
	startIndex := propertyInfo.structStartIndex()
	lastIndex := startIndex + propertyInfo.numOfStructMembers()

	// Initialize a map to hold the structure's fields.
	structure := make(map[string]interface{}, (lastIndex - startIndex))
	// Iterate through each member of the structure.
	for j := startIndex; j < lastIndex; j++ {
		name := p.getPropertyName(int(j))
		value, err := p.getPropertyValue(int(j))
		if err != nil {
			return nil, fmt.Errorf("failed parse field '%s' of complex property type: %w", name, err)
		}
		structure[name] = value // Add the field to the structure map.
	}

	return structure, nil
}

// parseSimpleType parses a simple property type using TdhFormatProperty.
func (p *propertyParser) parseSimpleType(propertyInfo EventPropertyInfo) (string, error) {
	var mapInfo *EventMapInfo
	if propertyInfo.mapNameOffset() > 0 {
		// If failed retrieving the map information, returns on error
		var err error
		mapInfo, err = p.getMapInfo(propertyInfo)
		if err != nil {
			return "", fmt.Errorf("failed to get map information due to: %w", err)
		}
	}

	// Get the length of the property.
	propertyLength, err := p.getPropertyLength(propertyInfo)
	if err != nil {
		return "", fmt.Errorf("failed to get property length due to: %w", err)
	}

	var userDataConsumed uint16

	// Set a default buffer size for formatted data.
	formattedDataSize := uint32(DEFAULT_PROPERTY_BUFFER_SIZE)
	formattedData := make([]byte, int(formattedDataSize))

	// Retry loop to handle buffer size adjustments.
retryLoop:
	for {
		var dataPtr *uint8
		if len(p.data) > 0 {
			dataPtr = &p.data[0]
		}
		err := _TdhFormatProperty(
			p.info,
			mapInfo,
			p.ptrSize,
			propertyInfo.inType(),
			propertyInfo.outType(),
			uint16(propertyLength),
			uint16(len(p.data)),
			dataPtr,
			&formattedDataSize,
			&formattedData[0],
			&userDataConsumed,
		)

		switch {
		case err == nil:
			// If formatting is successful, break out of the loop.
			break retryLoop
		case errors.Is(err, ERROR_INSUFFICIENT_BUFFER):
			// Increase the buffer size if it's insufficient.
			formattedData = make([]byte, formattedDataSize)
			continue
		case errors.Is(err, ERROR_EVT_INVALID_EVENT_DATA):
			// Handle invalid event data error.
			// Discarding MapInfo allows us to access
			// at least the non-interpreted data.
			if mapInfo != nil {
				mapInfo = nil
				continue
			}
			return "", fmt.Errorf("TdhFormatProperty failed: %w", err) // Handle unknown error
		default:
			return "", fmt.Errorf("TdhFormatProperty failed: %w", err)
		}
	}
	// Update the data slice to account for consumed data.
	p.data = p.data[userDataConsumed:]

	// Convert the formatted data to string and return.
	return windows.UTF16PtrToString((*uint16)(unsafe.Pointer(&formattedData[0]))), nil
}

// getMapInfo retrieves mapping information for a given property.
func (p *propertyParser) getMapInfo(propertyInfo EventPropertyInfo) (*EventMapInfo, error) {
	var mapSize uint32
	// Get the name of the map from the property info.
	mapName := (*uint16)(unsafe.Add(unsafe.Pointer(p.info), propertyInfo.mapNameOffset()))

	// First call to get the required size of the map info.
	err := _TdhGetEventMapInformation(p.r, mapName, nil, &mapSize)
	switch {
	case errors.Is(err, ERROR_NOT_FOUND):
		// No mapping information available. This is not an error.
		return nil, nil
	case errors.Is(err, ERROR_INSUFFICIENT_BUFFER):
		// Resize the buffer and try again.
	default:
		return nil, fmt.Errorf("TdhGetEventMapInformation failed to get size: %w", err)
	}

	// Allocate buffer and retrieve the actual map information.
	buff := make([]byte, int(mapSize))
	mapInfo := ((*EventMapInfo)(unsafe.Pointer(&buff[0])))
	err = _TdhGetEventMapInformation(p.r, mapName, mapInfo, &mapSize)
	if err != nil {
		return nil, fmt.Errorf("TdhGetEventMapInformation failed: %w", err)
	}

	if mapInfo.EntryCount == 0 {
		return nil, nil // No entries in the map.
	}

	return mapInfo, nil
}

// getPropertyLength returns the length of a specific property within TraceEventInfo.
func (p *propertyParser) getPropertyLength(propertyInfo EventPropertyInfo) (uint32, error) {
	// Check if the length of the property is defined by another property.
	if (propertyInfo.Flags & PropertyParamLength) == PropertyParamLength {
		var dataDescriptor PropertyDataDescriptor
		// Read the property name that contains the length information.
		dataDescriptor.PropertyName = readPropertyName(p, int(propertyInfo.length()))
		dataDescriptor.ArrayIndex = 0xFFFFFFFF
		// Retrieve the length from the specified property.
		return getLengthFromProperty(p.r, &dataDescriptor)
	}

	inType := propertyInfo.inType()
	outType := propertyInfo.outType()
	// Special handling for properties representing IPv6 addresses.
	// https://docs.microsoft.com/en-us/windows/win32/api/tdh/nf-tdh-tdhformatproperty#remarks
	if TdhIntypeBinary == inType && TdhOuttypeIpv6 == outType {
		// Return the fixed size of an IPv6 address.
		return 16, nil
	}

	// Default case: return the length as defined in the property info.
	// Note: A length of 0 can indicate a variable-length field (e.g., structure, string).
	return uint32(propertyInfo.length()), nil
}
