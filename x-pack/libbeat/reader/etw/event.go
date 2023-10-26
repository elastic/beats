// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package etw

import (
	"encoding/json"
	"fmt"
	"unicode/utf16"
	"unsafe"
)

// This buffer should receive stats from the buffer
func DefaultBufferCallback(etl *EventTraceLogfile) uintptr {
	// Todo: Collect and forward metrics
	// fmt.Printf("Buffers read: %d\n", etl.BuffersRead)
	// fmt.Printf("Buffer size: %d\n", etl.BufferSize)
	// fmt.Printf("Buffers written: %d\n", etl.LogfileHeader.BuffersWritten)
	// fmt.Printf("Buffers written: %d\n", etl.LogfileHeader.BuffersWritten)
	// fmt.Printf("Event losts: %d\n", etl.EventsLost)

	// return True (1) to continue the processing
	// return False (0) to stop processing events
	return 1
}

// Default callback to collect ETW events
func DefaultCallback(er *EventRecord) {
	if er == nil {
		fmt.Errorf("received null event record")
		return
	}

	event := make(map[string]interface{})
	event["Header"] = er.EventHeader

	if data, err := GetEventProperties(er); err == nil {
		event["EventProperties"] = data
	} else {
		fmt.Errorf("failed to read event properties: %s", err)
		return
	}

	// Other output
	jsonData, err := json.Marshal(event)
	if err != nil {
		fmt.Errorf("failed marshaling JSON event")
		return
	}
	fmt.Println(string(jsonData))

	return
}

// propertyParser is used for parsing properties from raw EVENT_RECORD structure.
type propertyParser struct {
	er      *EventRecord
	info    *TraceEventInfo
	data    []byte
	ptrSize uint32
}

func GetEventProperties(er *EventRecord) (map[string]interface{}, error) {
	if er.EventHeader.Flags == EVENT_HEADER_FLAG_STRING_ONLY {
		userDataPtr := (*uint16)(unsafe.Pointer(er.UserData))
		return map[string]interface{}{
			"_": UTF16PtrToString(userDataPtr),
		}, nil
	}

	p, err := newPropertyParser(er)
	if err != nil {
		return nil, fmt.Errorf("failed to parse event properties: %v", err)
	}

	// Now, loop through the properties (fields) of the event and format each one
	properties := make(map[string]interface{}, int(p.info.TopLevelPropertyCount))
	for i := 0; i < int(p.info.TopLevelPropertyCount); i++ {
		name := p.getPropertyName(i)
		value, err := p.getPropertyValue(i)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %q value: %v", name, err)
		}
		properties[name] = value
	}
	return properties, nil

}

func newPropertyParser(er *EventRecord) (*propertyParser, error) {
	info, err := getEventInformation(er)
	if err != nil {
		return nil, fmt.Errorf("failed to get event information: %v", err)
	}
	ptrSize := er.PointerSize()
	return &propertyParser{
		er:      er,
		info:    info,
		ptrSize: ptrSize,
		data:    unsafe.Slice((*uint8)(unsafe.Pointer(er.UserData)), er.UserDataLength),
	}, nil
}

func getEventInformation(er *EventRecord) (info *TraceEventInfo, err error) {
	// Get the size needed for the event information
	var bufSize uint32
	if err = _TdhGetEventInformation(er, 0, nil, nil, &bufSize); err == ERROR_INSUFFICIENT_BUFFER {
		// Allocate memory for TRACE_EVENT_INFO
		buff := make([]byte, bufSize)
		info = ((*TraceEventInfo)(unsafe.Pointer(&buff[0])))
		// Get the event information
		err = _TdhGetEventInformation(er, 0, nil, info, &bufSize)
	}
	if err != nil {
		return nil, fmt.Errorf("TdhGetEventInformation failed: %v", err)
	}
	return info, nil
}

// Returns a name of the event property structure
func (p *propertyParser) getPropertyName(i int) string {
	return createUTF16String(readPropertyName(p, i), ANYSIZE_ARRAY)
}

func readPropertyName(p *propertyParser, i int) unsafe.Pointer {
	return unsafe.Add(unsafe.Pointer(p.info), p.info.EventPropertyInfoArray[i].NameOffset)
}

// Creates UTF16 string from raw parts.
func createUTF16String(ptr unsafe.Pointer, length int) string {
	if length == 0 {
		return ""
	}
	chars := (*[ANYSIZE_ARRAY]uint16)(ptr)[:length:length]

	// Detect actual length of UTF-16 zero terminated string
	var fastEncode = true
	for i, v := range chars {
		if v == 0 {
			chars = chars[0:i]
			break
		}
		if v >= 0x800 {
			fastEncode = false
		}
	}
	if fastEncode {
		// Optimized variant for simple texts
		var bytes = make([]byte, 0, len(chars)*2)
		for _, v := range chars {
			// Encoding for UTF-8
			if v < 0x80 {
				bytes = append(bytes, uint8(v))
			} else {
				bytes = append(bytes, 0b11000000&uint8(v>>6), 0b10000000&uint8(v))
			}
		}
		return *(*string)(unsafe.Pointer(&bytes))
	}
	return string(utf16.Decode(chars))
}

// Retrieves a value of event property structure
func (p *propertyParser) getPropertyValue(i int) (interface{}, error) {
	propertyInfo := p.info.EventPropertyInfoArray[i]

	arraySize, err := p.getArraySize(propertyInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to get array size: %v", err)
	}

	result := make([]interface{}, arraySize)
	for j := 0; j < int(arraySize); j++ {
		var (
			value interface{}
			err   error
		)
		// We pass same idx to parse function. Actual returned values are controlled
		// by data pointers offsets.
		if (propertyInfo.Flags & PropertyStruct) == PropertyStruct {
			value, err = p.parseStruct(propertyInfo)
		} else {
			value, err = p.parseSimpleType(propertyInfo)
		}
		if err != nil {
			return nil, err
		}
		result[j] = value
	}
	if ((propertyInfo.Flags & PropertyParamCount) == PropertyParamCount) ||
		(propertyInfo.Count() > 1) {
		return result, nil
	}
	return result[0], nil
}

func (p *propertyParser) getArraySize(propertyInfo EventPropertyInfo) (uint32, error) {
	if (propertyInfo.Flags & PropertyParamCount) == PropertyParamCount {
		var dataDescriptor PropertyDataDescriptor
		// Use the countPropertyIndex member of the EVENT_PROPERTY_INFO structure
		// to locate the property that contains the size of the array.
		dataDescriptor.PropertyName = readPropertyName(p, int(propertyInfo.Count()))
		dataDescriptor.ArrayIndex = 0xFFFFFFFF
		return getLengthFromProperty(p.er, &dataDescriptor)
	} else {
		return uint32(propertyInfo.Count()), nil
	}
}

func getLengthFromProperty(er *EventRecord, dataDescriptor *PropertyDataDescriptor) (uint32, error) {
	var length uint32
	err := _TdhGetProperty(
		er,
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

// Extract fields from embedded structure at property
func (p *propertyParser) parseStruct(propertyInfo EventPropertyInfo) (map[string]interface{}, error) {
	startIndex := propertyInfo.StructStartIndex()
	lastIndex := startIndex + propertyInfo.NumOfStructMembers()

	structure := make(map[string]interface{}, (lastIndex - startIndex))
	for j := startIndex; j < lastIndex; j++ {
		name := p.getPropertyName(int(j))
		value, err := p.getPropertyValue(int(j))
		if err != nil {
			return nil, fmt.Errorf("failed parse field '%s' of complex property type: %v", name, err)
		}
		structure[name] = value
	}
	return structure, nil
}

// Parses a simple value wrapping TdhFormatProperty
func (p *propertyParser) parseSimpleType(propertyInfo EventPropertyInfo) (string, error) {
	var mapInfo *EventMapInfo
	if propertyInfo.MapNameOffset() > 0 {
		// Get map information
		var err error
		mapInfo, err = p.getMapInfo(propertyInfo)
		if err != nil {
			return "", fmt.Errorf("failed to get map information due to: %v", err)
		}
	}

	propertyLength, err := p.getPropertyLength(propertyInfo)
	if err != nil {
		return "", fmt.Errorf("failed to get property length due to: %v", err)
	}

	var userDataConsumed uint16

	// Initialize parse buffer with a size that should be sufficient for most properties.
	formattedDataSize := uint32(DEFAULT_PROPERTY_BUFFER_SIZE)
	formattedData := make([]byte, int(formattedDataSize))

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
			propertyInfo.InType(),
			propertyInfo.OutType(),
			uint16(propertyLength),
			uint16(len(p.data)),
			dataPtr,
			&formattedDataSize,
			&formattedData[0],
			&userDataConsumed,
		)

		switch err {
		case nil:
			break retryLoop

		case ERROR_INSUFFICIENT_BUFFER:
			formattedData = make([]byte, formattedDataSize)
			continue

		case ERROR_EVT_INVALID_EVENT_DATA:
			// Can happen if the MapInfo doesn't match the actual data, e.g pure ETW provider
			// works with the outdated WEL manifest. Discarding MapInfo allows us to access
			// at least the non-interpreted data.
			if mapInfo != nil {
				mapInfo = nil
				continue
			}
			fallthrough // Unknown error

		default:
			return "", fmt.Errorf("TdhFormatProperty failed: %v", err)
		}
	}
	p.data = p.data[userDataConsumed:]

	return createUTF16String(unsafe.Pointer(&formattedData[0]), int(formattedDataSize)), nil
}

// Retrieve the mapping between the field and the structure it represents
func (p *propertyParser) getMapInfo(propertyInfo EventPropertyInfo) (*EventMapInfo, error) {
	var mapSize uint32
	mapName := (*uint16)(unsafe.Add(unsafe.Pointer(p.info), propertyInfo.MapNameOffset()))

	// Get map info size
	err := _TdhGetEventMapInformation(
		p.er,
		mapName,
		nil,
		&mapSize,
	)
	switch err {
	case ERROR_NOT_FOUND:
		return nil, nil // No map info, return nicely
	case ERROR_INSUFFICIENT_BUFFER:
		// Resize the buffer and try again
	default:
		return nil, fmt.Errorf("TdhGetEventMapInformation failed to get size: %v", err)
	}

	// Get the map information
	buff := make([]byte, int(mapSize))
	mapInfo := ((*EventMapInfo)(unsafe.Pointer(&buff[0])))
	err = _TdhGetEventMapInformation(
		p.er,
		mapName,
		mapInfo,
		&mapSize,
	)
	if err != nil {
		return nil, fmt.Errorf("TdhGetEventMapInformation failed: %v", err)
	}

	if mapInfo.EntryCount == 0 {
		return nil, nil
	}
	return mapInfo, nil
}

// Returns an associated length of the property of TraceEventInfo
func (p *propertyParser) getPropertyLength(propertyInfo EventPropertyInfo) (uint32, error) {
	// If the property is a binary blob it can point to another property that defines the
	// blob's size. The PropertyParamLength flag tells you where the blob's size is defined.
	if (propertyInfo.Flags & PropertyParamLength) == PropertyParamLength {
		var dataDescriptor PropertyDataDescriptor
		dataDescriptor.PropertyName = readPropertyName(p, int(propertyInfo.Length()))
		dataDescriptor.ArrayIndex = 0xFFFFFFFF
		return getLengthFromProperty(p.er, &dataDescriptor)
	}

	// If the property is an IP V6 address, you must set the PropertyLength parameter to the size
	// of the IN6_ADDR structure:
	// https://docs.microsoft.com/en-us/windows/win32/api/tdh/nf-tdh-tdhformatproperty#remarks
	inType := propertyInfo.InType()
	outType := propertyInfo.OutType()
	if TdhIntypeBinary == inType && TdhOuttypeIpv6 == outType {
		return 16, nil
	}

	// If no special cases handled just return the length defined in the info.
	// In some cases, the length is 0. This can signify that we are dealing with a variable
	// length field such as a structure or a string.
	return uint32(propertyInfo.Length()), nil
}
