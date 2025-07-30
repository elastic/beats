// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package etw

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

type eventRenderer struct {
	cache       providerCache
	r           *EventRecord
	data        []byte
	ptrSize     uint32
	bufferPools *bufferPools
}

// newEventRenderer initializes a new property parser for a given event record.
func newEventRenderer(cache providerCache, r *EventRecord, bufferPools *bufferPools) *eventRenderer {
	ptrSize := r.pointerSize()

	// Return a new eventRenderer instance initialized with event record data and metadata.
	return &eventRenderer{
		cache:       cache,
		r:           r,
		data:        unsafe.Slice((*uint8)(unsafe.Pointer(r.UserData)), r.UserDataLength),
		ptrSize:     ptrSize,
		bufferPools: bufferPools,
	}
}

func (p *eventRenderer) render() (RenderedEtwEvent, error) {
	eventInfo, err := p.cache.getEventInfo(p.r)
	if err != nil {
		return RenderedEtwEvent{}, fmt.Errorf("failed to get event info: %w", err)
	}

	event := RenderedEtwEvent{
		ProviderGUID:          eventInfo.ParsedInfo.ProviderGUID,
		ProviderName:          eventInfo.ProviderName,
		EventID:               p.r.EventHeader.EventDescriptor.Id,
		Version:               p.r.EventHeader.EventDescriptor.Version,
		Level:                 eventInfo.LevelName,
		LevelRaw:              p.r.EventHeader.EventDescriptor.Level,
		Task:                  eventInfo.TaskName,
		TaskRaw:               p.r.EventHeader.EventDescriptor.Task,
		Opcode:                eventInfo.OpcodeName,
		OpcodeRaw:             p.r.EventHeader.EventDescriptor.Opcode,
		KeywordsRaw:           p.r.EventHeader.EventDescriptor.Keyword,
		Channel:               eventInfo.ChannelName,
		Timestamp:             convertFileTimeToGoTime(uint64(p.r.EventHeader.TimeStamp)), //nolint:gosec // Timestamp is never going to be negative
		ProcessID:             p.r.EventHeader.ProcessId,
		ThreadID:              p.r.EventHeader.ThreadId,
		ActivityIDName:        eventInfo.ActivityIDName,
		RelatedActivityIDName: eventInfo.RelatedActivityIDName,
		ProviderMessage:       eventInfo.ProviderMessage,
		Properties:            make([]RenderedProperty, len(eventInfo.Properties)),
		propertiesMap:         make(map[string]int, len(eventInfo.Properties)),
		ExtendedData:          make([]RenderedExtendedData, int(p.r.ExtendedDataCount)),
		extendedDataMap:       make(map[string]int, int(p.r.ExtendedDataCount)),
	}

	// Set Active Keywords
	if event.KeywordsRaw != 0 {
		for keywordBitValue, cachedKw := range p.cache.getKeywords() {
			if (event.KeywordsRaw&keywordBitValue) == keywordBitValue && keywordBitValue != 0 { // Check if bit is set
				event.Keywords = append(event.Keywords, cachedKw.Name)
			}
		}
	}

	// Parse ExtendedData if present
	if p.r.ExtendedDataCount > 0 && p.r.ExtendedData != nil {
		for i := 0; i < int(p.r.ExtendedDataCount); i++ {
			item := (*EventHeaderExtendedDataItem)(unsafe.Pointer(
				uintptr(unsafe.Pointer(p.r.ExtendedData)) + uintptr(i)*unsafe.Sizeof(*p.r.ExtendedData),
			))
			event.ExtendedData[i] = renderExtendedData(item)
			event.extendedDataMap[event.ExtendedData[i].ExtType] = i // Map the extended data type to its index
		}
	}

	// Handle the case where the event only contains a string.
	if p.r.EventHeader.Flags == EVENT_HEADER_FLAG_STRING_ONLY {
		userDataBuf := uintptrToBytes(p.r.UserData, p.r.UserDataLength)
		event.EventMessage = getStringFromBufferOffset(userDataBuf, 0) // Convert the user data from UTF16 to string.
		return event, nil
	}

	for i, prop := range eventInfo.Properties {
		value, err := renderPropertyValue(p.cache, eventInfo, prop, p.r, p.ptrSize, &p.data, p.bufferPools)
		if err != nil {
			return RenderedEtwEvent{}, fmt.Errorf("failed to render property '%s': %w", prop.Name, err)
		}
		event.Properties[i] = RenderedProperty{
			Name:  prop.Name,
			Value: value,
		}
		event.propertiesMap[prop.Name] = i // Map the property name to its index
	}

	event.EventMessage = eventInfo.renderEventMessage(event.Properties)

	return event, nil
}

func renderPropertyValue(cache providerCache, eventInfo *cachedEventInfo, propInfo *cachedPropertyInfo, r *EventRecord, ptrSize uint32, buf *[]byte, bufferPools *bufferPools) (any, error) {
	// Determine the rendering strategy based on property type
	switch {
	case propInfo.IsStruct && propInfo.IsArray:
		return renderStructArray(cache, eventInfo, propInfo, r, ptrSize, buf, bufferPools)
	case propInfo.IsStruct && !propInfo.IsArray:
		return renderStruct(cache, eventInfo, propInfo, r, ptrSize, buf, bufferPools)
	case !propInfo.IsStruct && propInfo.IsArray:
		return renderSimpleArray(cache, eventInfo, propInfo, r, ptrSize, buf, bufferPools)
	default: // !propInfo.IsStruct && !propInfo.IsArray
		return renderSingleSimpleProperty(cache, eventInfo, propInfo, r, ptrSize, buf, bufferPools)
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

func renderStruct(cache providerCache, eventInfo *cachedEventInfo, propInfo *cachedPropertyInfo, r *EventRecord, ptrSize uint32, buf *[]byte, bufferPools *bufferPools) (map[string]any, error) {
	// Pre-allocate map with estimated capacity to reduce reallocations
	memberCount := len(propInfo.StructMembers)
	structure := make(map[string]any, memberCount)

	for _, sm := range propInfo.StructMembers {
		value, err := renderPropertyValue(cache, eventInfo, sm, r, ptrSize, buf, bufferPools)
		if err != nil {
			return nil, fmt.Errorf("failed to parse struct member '%s': %w", sm.Name, err)
		}
		structure[sm.Name] = value
	}
	return structure, nil
}

// renderStructArray handles arrays of structs
func renderStructArray(cache providerCache, eventInfo *cachedEventInfo, propInfo *cachedPropertyInfo, r *EventRecord, ptrSize uint32, buf *[]byte, bufferPools *bufferPools) (any, error) {
	count, err := getPropertyLength(eventInfo, propInfo, r)
	if err != nil {
		return nil, fmt.Errorf("failed to get array count: %w", err)
	}
	arraySize := int(count)

	if arraySize == 0 {
		return []any{}, nil
	}

	result := make([]any, arraySize)
	for j := 0; j < arraySize; j++ {
		value, err := renderStruct(cache, eventInfo, propInfo, r, ptrSize, buf, bufferPools)
		if err != nil {
			return nil, fmt.Errorf("failed to parse struct member %d: %w", j, err)
		}
		result[j] = value
	}
	return result, nil
}

// renderSimpleArray handles arrays of simple types (strings, integers, etc.)
func renderSimpleArray(cache providerCache, eventInfo *cachedEventInfo, propInfo *cachedPropertyInfo, r *EventRecord, ptrSize uint32, buf *[]byte, bufferPools *bufferPools) (any, error) {
	// Get the count for the array
	count, err := getPropertyLength(eventInfo, propInfo, r)
	if err != nil {
		return nil, fmt.Errorf("failed to get array count: %w", err)
	}

	if count == 0 {
		return []any{}, nil
	}

	// Set up map info for the array elements
	var mapInfo *EventMapInfo
	var cachedMapInfo *cachedEventMapInfo
	if propInfo.MapName != "" {
		if cached, found := cache.getPropertyMaps()[propInfo.MapName]; found {
			mapInfo = cached.ParsedInfo
			cachedMapInfo = cached
		}
	}

	result := make([]any, count)
	for i := uint32(0); i < count; i++ {
		// For each array element, process it as a single property
		value, err := renderSingleProperty(cache, eventInfo, propInfo, r, ptrSize, buf, mapInfo, cachedMapInfo, bufferPools)
		if err != nil {
			return nil, fmt.Errorf("failed to parse array element %d: %w", i, err)
		}
		result[i] = value
	}
	return result, nil
}

// renderSingleSimpleProperty handles a single simple property (not an array, not a struct)
func renderSingleSimpleProperty(cache providerCache, eventInfo *cachedEventInfo, propInfo *cachedPropertyInfo, r *EventRecord, ptrSize uint32, buf *[]byte, bufferPools *bufferPools) (any, error) {
	// Set up map info for the property
	var mapInfo *EventMapInfo
	var cachedMapInfo *cachedEventMapInfo
	if propInfo.MapName != "" {
		if cached, found := cache.getPropertyMaps()[propInfo.MapName]; found {
			mapInfo = cached.ParsedInfo
			cachedMapInfo = cached
		}
	}

	return renderSingleProperty(cache, eventInfo, propInfo, r, ptrSize, buf, mapInfo, cachedMapInfo, bufferPools)
}

func getPropertyLength(eventInfo *cachedEventInfo, propInfo *cachedPropertyInfo, r *EventRecord) (uint32, error) {
	// Check if the length of the property is defined by another property.
	if propInfo.hasLengthFromOtherProperty() {
		// Length is specified by another property - use the property index
		var dataDescriptor PropertyDataDescriptor
		lengthPropIndex := propInfo.getLengthPropertyIndex()
		nameProp := eventInfo.ParsedInfo.getEventPropertyInfoAtIndex(uint32(lengthPropIndex))
		if nameProp == nil {
			return 0, fmt.Errorf("property length is defined by another property, but no property found at index %d", lengthPropIndex)
		}
		// Read the property name that contains the length information.
		dataDescriptor.PropertyName = unsafe.Pointer(windows.StringToUTF16Ptr(getStringFromBufferOffset(eventInfo.InfoBuf, nameProp.NameOffset)))
		dataDescriptor.ArrayIndex = 0xFFFFFFFF
		// Retrieve the length from the specified property.
		return getLengthFromProperty(r, &dataDescriptor)
	}

	// Check if the count of the property is defined by another property.
	if propInfo.hasCountFromOtherProperty() {
		// Count is specified by another property - use the property index
		var dataDescriptor PropertyDataDescriptor
		countPropIndex := propInfo.getCountPropertyIndex()
		nameProp := eventInfo.ParsedInfo.getEventPropertyInfoAtIndex(uint32(countPropIndex))
		if nameProp == nil {
			return 0, fmt.Errorf("property count is defined by another property, but no property found at index %d", countPropIndex)
		}
		// Read the property name that contains the count information.
		dataDescriptor.PropertyName = unsafe.Pointer(windows.StringToUTF16Ptr(getStringFromBufferOffset(eventInfo.InfoBuf, nameProp.NameOffset)))
		dataDescriptor.ArrayIndex = 0xFFFFFFFF
		// Retrieve the count from the specified property.
		return getLengthFromProperty(r, &dataDescriptor)
	}

	// Check if this is a fixed length property
	if propInfo.IsFixedLen {
		return propInfo.getFixedLength(), nil
	}

	// Variable length property without explicit length reference
	// For null-terminated strings, return 0 to indicate length should be determined by null terminator
	if propInfo.isVariableLengthString() {
		return 0, nil // 0 indicates null-terminated string
	}

	// For other variable-length types, we might need to determine length dynamically
	// This is a fallback case that might need specific handling based on the data type
	return 0, nil
}

// getElementLength gets the length of an individual property element (not the array count)
// This is used when parsing individual elements within an array
func getElementLength(eventInfo *cachedEventInfo, propInfo *cachedPropertyInfo, r *EventRecord) (uint32, error) {
	// For array elements, we should NOT check for count from other properties
	// because that would give us the array count, not the element length

	// Check if the length of the property is defined by another property.
	if propInfo.hasLengthFromOtherProperty() {
		// Length is specified by another property - use the property index
		var dataDescriptor PropertyDataDescriptor
		lengthPropIndex := propInfo.getLengthPropertyIndex()
		nameProp := eventInfo.ParsedInfo.getEventPropertyInfoAtIndex(uint32(lengthPropIndex))
		if nameProp == nil {
			return 0, fmt.Errorf("property length is defined by another property, but no property found at index %d", lengthPropIndex)
		}
		// Read the property name that contains the length information.
		dataDescriptor.PropertyName = unsafe.Pointer(windows.StringToUTF16Ptr(getStringFromBufferOffset(eventInfo.InfoBuf, nameProp.NameOffset)))
		dataDescriptor.ArrayIndex = 0xFFFFFFFF
		// Retrieve the length from the specified property.
		return getLengthFromProperty(r, &dataDescriptor)
	}

	// Check if this is a fixed length property
	if propInfo.IsFixedLen {
		return propInfo.getFixedLength(), nil
	}

	// Variable length property without explicit length reference
	// For null-terminated strings, return 0 to indicate length should be determined by null terminator
	if propInfo.isVariableLengthString() {
		return 0, nil // 0 indicates null-terminated string
	}

	// For other variable-length types, we might need to determine length dynamically
	// This is a fallback case that might need specific handling based on the data type
	return 0, nil
}

var extendedDataHandlers = map[uint16]func(*EventHeaderExtendedDataItem) any{
	EVENT_HEADER_EXT_TYPE_RELATED_ACTIVITYID: parseGUID,
	EVENT_HEADER_EXT_TYPE_SID:                parseSID,
	EVENT_HEADER_EXT_TYPE_TS_ID:              parseUint32,
	EVENT_HEADER_EXT_TYPE_INSTANCE_INFO:      parseUint64,
	EVENT_HEADER_EXT_TYPE_STACK_TRACE32:      parseUint32Slice,
	EVENT_HEADER_EXT_TYPE_STACK_TRACE64:      parseUint64Slice,
	EVENT_HEADER_EXT_TYPE_PEBS_INDEX:         parseUint64,
	EVENT_HEADER_EXT_TYPE_PMC_COUNTERS:       parseUint64Slice,
	EVENT_HEADER_EXT_TYPE_PSM_KEY:            parseUint64,
	EVENT_HEADER_EXT_TYPE_EVENT_KEY:          parseUint64,
	EVENT_HEADER_EXT_TYPE_EVENT_SCHEMA_TL:    parseByteSlice,
	EVENT_HEADER_EXT_TYPE_PROV_TRAITS:        parseByteSlice,
	EVENT_HEADER_EXT_TYPE_PROCESS_START_KEY:  parseUint64,
	EVENT_HEADER_EXT_TYPE_QPC_DELTA:          parseUint64,
	EVENT_HEADER_EXT_TYPE_CONTAINER_ID:       parseGUID,
}

func renderExtendedData(item *EventHeaderExtendedDataItem) RenderedExtendedData {
	data := RenderedExtendedData{
		ExtType:    extTypeToStr(item.ExtType),
		ExtTypeRaw: item.ExtType,
		DataSize:   item.DataSize,
	}
	if handler, ok := extendedDataHandlers[item.ExtType]; ok {
		data.Data = handler(item)
	} else {
		data.Data = fmt.Sprintf("Unknown ExtType: %d", item.ExtType)
	}
	return data
}

// convertFileTimeToGoTime converts a Windows FileTime to a Go time.Time structure.
func convertFileTimeToGoTime(fileTime64 uint64) time.Time {
	// Define the offset between Windows epoch (1601) and Unix epoch (1970)
	const epochDifference = 116444736000000000
	if fileTime64 < epochDifference {
		// Time is before the Unix epoch, adjust accordingly
		return time.Time{}
	}

	fileTime := windows.Filetime{
		HighDateTime: uint32(fileTime64 >> 32),            //nolint:gosec // High part of the 64-bit FileTime
		LowDateTime:  uint32(fileTime64 & math.MaxUint32), //nolint:gosec // Low part of the 64-bit FileTime
	}

	return time.Unix(0, fileTime.Nanoseconds()).UTC()
}

// DescribeComplexProperty provides a string description of a complex or unsupported property type.
func describeComplexProperty(info *cachedPropertyInfo) string {
	return fmt.Sprintf(
		"Complex/Unsupported property: Name=%q Flags=0x%X InType=%d OutType=%d Count=%d Length=%d MapNameOffset=0x%X",
		info.Name,
		info.ParsedInfo.Flags,
		info.InType,
		info.OutType,
		info.Count,
		info.Length,
		info.ParsedInfo.mapNameOffset(),
	)
}

// cleanUnicodeFormattingChars removes Unicode formatting characters that
// Windows APIs sometimes insert for display purposes
func cleanUnicodeFormattingChars(s string) string {
	// Remove common Unicode formatting characters:
	// U+200E (Left-to-Right Mark)
	// U+200F (Right-to-Left Mark)
	// U+202A (Left-to-Right Embedding)
	// U+202B (Right-to-Left Embedding)
	// U+202C (Pop Directional Formatting)
	// U+202D (Left-to-Right Override)
	// U+202E (Right-to-Left Override)
	// U+2060 (Word Joiner)
	// U+FEFF (Zero Width No-Break Space)
	replacer := strings.NewReplacer(
		"\u200E", "", // Left-to-Right Mark
		"\u200F", "", // Right-to-Left Mark
		"\u202A", "", // Left-to-Right Embedding
		"\u202B", "", // Right-to-Left Embedding
		"\u202C", "", // Pop Directional Formatting
		"\u202D", "", // Left-to-Right Override
		"\u202E", "", // Right-to-Left Override
		"\u2060", "", // Word Joiner
		"\uFEFF", "", // Zero Width No-Break Space
	)
	return replacer.Replace(s)
}

func renderSingleProperty(cache providerCache, eventInfo *cachedEventInfo, propInfo *cachedPropertyInfo, r *EventRecord, ptrSize uint32, buf *[]byte, mapInfo *EventMapInfo, cachedMapInfo *cachedEventMapInfo, bufferPools *bufferPools) (any, error) {
	// Get the length of the individual property element (not the array count).
	propertyLength, err := getElementLength(eventInfo, propInfo, r)
	if err != nil {
		return "", fmt.Errorf("failed to get property length due to: %w", err)
	}

	// If a property is defined as a variable-length string (indicated by length 0)
	// but there is no data left in the buffer, it represents an empty string.
	// TdhFormatProperty can fail in this case, so we handle it explicitly.
	if propertyLength == 0 && propInfo.isVariableLengthString() && len(*buf) == 0 {
		return "", nil
	}

	// If we have a cached map, try to use it directly for simple numeric types
	if cachedMapInfo != nil && propertyLength > 0 && len(*buf) > 0 {
		if mappedValue, consumed, ok := cachedMapInfo.getFormattedMapEntry(propInfo, *buf, int(propertyLength)); ok {
			*buf = (*buf)[consumed:]
			return mappedValue, nil
		}
	}

	var userDataConsumed uint16

	// Set a default buffer size for formatted data.
	formattedDataSize := uint32(DEFAULT_PROPERTY_BUFFER_SIZE)

	formattedDataPtr := bufferPools.getBuffer(formattedDataSize)
	defer bufferPools.putBuffer(formattedDataPtr, formattedDataSize)

	formattedData := *formattedDataPtr

	// Retry loop to handle buffer size adjustments.
retryLoop:
	for {
		var dataPtr *uint8
		if len(*buf) > 0 {
			dataPtr = &(*buf)[0]
		}

		// Ensure buffer is large enough
		if len(formattedData) < int(formattedDataSize) {
			formattedData = make([]byte, formattedDataSize)
		}

		err := _TdhFormatProperty(
			eventInfo.ParsedInfo,
			mapInfo,
			ptrSize,
			propInfo.InType,
			propInfo.OutType,
			uint16(propertyLength),
			uint16(len(*buf)), //nolint:gosec // This is the length of the user data buffer
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
			// For very large properties, we need to allocate a new buffer
			// Don't use the pool for these exceptional cases to avoid memory bloat
			formattedData = make([]byte, formattedDataSize*2)
			formattedDataSize = uint32(len(formattedData)) //nolint:gosec // Update the size for the next iteration
			continue
		case errors.Is(err, ERROR_EVT_INVALID_EVENT_DATA):
			// Handle invalid event data error.
			// Discarding MapInfo allows us to access
			// at least the non-interpreted data.
			if mapInfo != nil {
				mapInfo = nil
				continue
			}
			fallthrough
		default:
			// Fallback: describe the complex/unsupported property
			return describeComplexProperty(propInfo), nil
		}
	}

	// Convert the formatted data to string
	result := windows.UTF16PtrToString((*uint16)(unsafe.Pointer(&formattedData[0])))

	// Cache the result if we have a map and successfully consumed data
	if cachedMapInfo != nil && userDataConsumed > 0 {
		cachedMapInfo.cacheFormattedMapEntry(propInfo, (*buf)[:userDataConsumed], result, int(userDataConsumed))
	}

	// Update the data slice to account for consumed data.
	*buf = (*buf)[userDataConsumed:]

	return cleanUnicodeFormattingChars(result), nil
}
