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
	"errors"
	"fmt"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

// providerCache stores metadata from a publisher.
type providerCache struct {
	mutex sync.RWMutex

	guid windows.GUID

	einfoCache        map[EventDescriptor]*cachedEventInfo
	keywords          map[uint64]*cachedProviderKeyword
	propertyMapsCache map[string]*cachedEventMapInfo
}

func newProviderCache(guid windows.GUID) (*providerCache, error) {
	cache := &providerCache{
		guid:              guid,
		einfoCache:        make(map[EventDescriptor]*cachedEventInfo),
		keywords:          make(map[uint64]*cachedProviderKeyword),
		propertyMapsCache: make(map[string]*cachedEventMapInfo),
	}
	if err := cache.init(); err != nil {
		return nil, fmt.Errorf("failed to initialize provider cache for %s: %w", guid, err)
	}
	return cache, nil
}

func (cache *providerCache) init() error {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()

	if cache.guid == (windows.GUID{}) {
		return fmt.Errorf("provider cache not initialized with a GUID")
	}

	if err := cache.initKeywords(); err != nil {
		return fmt.Errorf("failed to initialize keywords for provider %s: %w", cache.guid, err)
	}

	descriptors, err := getProviderEventDescriptors(&cache.guid)
	if err != nil {
		return fmt.Errorf("failed to get event descriptors for provider %s: %w", cache.guid, err)
	}

	for _, desc := range descriptors {
		r := &EventRecord{
			EventHeader: EventHeader{
				ProviderId:      cache.guid,
				EventDescriptor: desc,
			},
		}
		if err := cache.initEvent(r); err != nil {
			return fmt.Errorf("failed to initialize event for provider %s: %w", cache.guid, err)
		}
	}

	return nil
}

func (cache *providerCache) initKeywords() error {
	var err error
	var bufSize uint32
	var buf []byte
	var pfInfoArr *ProviderFieldInfoArray
	if err = _TdhEnumerateProviderFieldInformation(&cache.guid, EventKeywordInformation, nil, &bufSize); errors.Is(err, ERROR_INSUFFICIENT_BUFFER) {
		buf = make([]byte, bufSize)
		pfInfoArr = ((*ProviderFieldInfoArray)(unsafe.Pointer(&buf[0])))
		err = _TdhEnumerateProviderFieldInformation(&cache.guid, EventKeywordInformation, pfInfoArr, &bufSize)
	}

	if err != nil {
		return fmt.Errorf("TdhEnumerateProviderFieldInformation failed: %w", err)
	}

	if pfInfoArr.NumberOfElements == 0 {
		return fmt.Errorf("no keywords found for provider %s", cache.guid)
	}

	it := uintptr(unsafe.Pointer(&pfInfoArr.FieldInfoArray[0]))
	for i := uint32(0); i < pfInfoArr.NumberOfElements; i++ {
		pfInfo := *(*ProviderFieldInfo)(unsafe.Pointer(it + uintptr(i)*unsafe.Sizeof(pfInfoArr.FieldInfoArray[0])))
		cache.keywords[pfInfo.Value] = &cachedProviderKeyword{
			Name:        getStringFromBufferOffset(buf, pfInfo.NameOffset),
			Description: getStringFromBufferOffset(buf, pfInfo.DescriptionOffset),
		}
	}
	return nil
}

func (cache *providerCache) initEvent(r *EventRecord) error {
	var err error
	var bufSize uint32
	var buf []byte
	var info *TraceEventInfo
	if err = _TdhGetEventInformation(r, 0, nil, nil, &bufSize); errors.Is(err, ERROR_INSUFFICIENT_BUFFER) {
		buf = make([]byte, bufSize)
		info = ((*TraceEventInfo)(unsafe.Pointer(&buf[0])))
		err = _TdhGetEventInformation(r, 0, nil, info, &bufSize)
	}

	if err != nil {
		return fmt.Errorf("TdhGetEventInformation failed: %w", err)
	}

	cached := &cachedEventInfo{
		InfoBuf:               buf,
		ParsedInfo:            info,
		ProviderName:          getStringFromBufferOffset(buf, info.ProviderNameOffset),
		LevelName:             getStringFromBufferOffset(buf, info.LevelNameOffset),
		ChannelName:           getStringFromBufferOffset(buf, info.ChannelNameOffset),
		KeywordsNames:         getMultiStringFromBufferOffset(buf, info.KeywordsNameOffset),
		TaskName:              getStringFromBufferOffset(buf, info.TaskNameOffset),
		OpcodeName:            getStringFromBufferOffset(buf, info.OpcodeNameOffset),
		EventMessage:          getStringFromBufferOffset(buf, info.EventMessageOffset),
		ProviderMessage:       getStringFromBufferOffset(buf, info.ProviderMessageOffset),
		BinaryXML:             getEventInfoBinaryXML(info),
		ActivityIDName:        getStringFromBufferOffset(buf, info.ActivityIDNameOffset),
		RelatedActivityIDName: getStringFromBufferOffset(buf, info.RelatedActivityIDNameOffset),
	}
	cached.Properties = getEventInfoProperties(buf, info)
	if err = initEventInfoMaps(cache.propertyMapsCache, cached.Properties, r); err != nil {
		return fmt.Errorf("failed to get event maps for %s: %w", cache.guid, err)
	}
	cache.einfoCache[r.EventHeader.EventDescriptor] = cached
	return nil
}

func (cache *providerCache) getEventInfo(r *EventRecord) (*cachedEventInfo, error) {
	cache.mutex.RLock()
	desc := r.EventHeader.EventDescriptor
	cached, found := cache.einfoCache[desc]
	cache.mutex.RUnlock()
	if found {
		return cached, nil
	}

	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	cached, found = cache.einfoCache[desc]
	if found {
		return cached, nil
	}
	// If not found, try to initialize the event info
	if err := cache.initEvent(r); err != nil {
		return nil, fmt.Errorf("failed to initialize event info for %s: %w", cache.guid, err)
	}
	cached, found = cache.einfoCache[desc]
	if found {
		return cached, nil
	}
	return nil, fmt.Errorf("event descriptor %v not found in provider %s", desc, cache.guid)
}

func getProviderEventDescriptors(guid *windows.GUID) ([]EventDescriptor, error) {
	var err error
	var bufSize uint32
	var buf []byte
	var prInfo *ProviderEventInfo
	if err = _TdhEnumerateManifestProviderEvents(guid, nil, &bufSize); errors.Is(err, ERROR_INSUFFICIENT_BUFFER) {
		buf = make([]byte, bufSize)
		prInfo = ((*ProviderEventInfo)(unsafe.Pointer(&buf[0])))
		err = _TdhEnumerateManifestProviderEvents(guid, prInfo, &bufSize)
	}

	if err != nil {
		return nil, fmt.Errorf("TdhEnumerateManifestProviderEvents failed: %w", err)
	}

	if prInfo.NumberOfEvents == 0 {
		return nil, errors.New("no events found for provider")
	}

	descriptors := make([]EventDescriptor, prInfo.NumberOfEvents)
	it := uintptr(unsafe.Pointer(&prInfo.EventDescriptorsArray[0]))
	for i := uint32(0); i < prInfo.NumberOfEvents; i++ {
		descriptors[i] = *(*EventDescriptor)(unsafe.Pointer(it + uintptr(i)*unsafe.Sizeof(prInfo.EventDescriptorsArray[0])))
	}
	return descriptors, nil
}

type cachedEventInfo struct {
	// Buffer containing the TraceEventInfo structure and its data
	// Keep this to prevent ParsedInfo and Properties.ParsedInfo
	// underlying data from being garbage collected
	InfoBuf    []byte
	ParsedInfo *TraceEventInfo // Parsed TraceEventInfo structure

	ProviderName          string
	LevelName             string
	ChannelName           string
	KeywordsNames         []string
	TaskName              string
	OpcodeName            string
	EventMessage          string
	ProviderMessage       string
	BinaryXML             string
	ActivityIDName        string
	RelatedActivityIDName string

	Properties []*cachedPropertyInfo
}

type cachedPropertyInfo struct {
	ParsedInfo *EventPropertyInfo

	Name       string
	InType     uint16
	OutType    uint16
	MapName    string
	IsStruct   bool
	IsArray    bool
	IsFixedLen bool
	Count      uint16
	Length     uint16

	StructMembers []*cachedPropertyInfo // Only if IsStruct is true
}

type cachedProviderKeyword struct {
	Name        string
	Description string
}

type cachedMapEntry struct {
	Value   any
	IsUlong bool // true if the value is a uint64, false if it's a string
}

func (e *cachedMapEntry) getStringValue() string {
	if e.IsUlong {
		return fmt.Sprintf("%d", e.Value)
	}
	if str, ok := e.Value.(string); ok {
		return str
	}
	return fmt.Sprintf("%v", e.Value)
}

func (e *cachedMapEntry) getUlongValue() uint32 {
	v, ok := e.Value.(uint32)
	if !e.IsUlong || !ok {
		return 0
	}
	return v
}

type cachedEventMapInfo struct {
	// Buffer containing the EventMapInfo structure and its data
	// Keep this to prevent ParsedInfo from being garbage collected
	InfoBuf    []byte
	ParsedInfo *EventMapInfo

	Name      string
	IsBitmap  bool
	IsValue   bool
	IsPattern bool
	// For ValueMap: key is the numeric value from EVENT_MAP_ENTRY.Value
	// For BitMap: key is the bit value (e.g., 0x1, 0x2, 0x4) from EVENT_MAP_ENTRY.Value
	// For PatternMap: key is the pattern value from EVENT_MAP_ENTRY.InputOffset as a string
	Entries map[any]*cachedMapEntry
}

func getStringFromBufferOffset(buf []byte, offset uint32) string {
	if offset == 0 || len(buf) == 0 {
		return ""
	}
	ptr := unsafe.Add(unsafe.Pointer(&buf[0]), offset)
	return windows.UTF16PtrToString((*uint16)(ptr))
}

func getEventInfoBinaryXML(info *TraceEventInfo) string {
	if info == nil || info.BinaryXMLOffset == 0 || info.BinaryXMLSize == 0 {
		return ""
	}
	var result string
	ptr := unsafe.Add(unsafe.Pointer(info), info.BinaryXMLOffset)
	data := unsafe.Slice((*byte)(ptr), info.BinaryXMLSize)
	if isPrintable(data) {
		result = string(data)
	} else {
		result = fmt.Sprintf("0x%X", data)
	}
	return result
}

func getMultiStringFromBufferOffset(buf []byte, offset uint32) []string {
	if offset == 0 || len(buf) == 0 || offset >= uint32(len(buf)) {
		return nil
	}

	var results []string
	currentPtr := unsafe.Add(unsafe.Pointer(&buf[0]), offset)

	for {
		str := windows.UTF16PtrToString((*uint16)(currentPtr))
		if str == "" {
			break
		}
		results = append(results, str)

		utf16Chars, _ := windows.UTF16FromString(str)
		advanceByBytes := (len(utf16Chars) + 1) * 2 // +1 for the null terminator, *2 for bytes
		currentPtr = unsafe.Add(currentPtr, advanceByBytes)
		if uintptr(currentPtr) >= uintptr(unsafe.Add(unsafe.Pointer(&buf[0]), len(buf))) {
			break // Prevent reading beyond the buffer
		}
	}
	return results
}

func getEventInfoProperties(buf []byte, info *TraceEventInfo) []*cachedPropertyInfo {
	if info.PropertyCount == 0 {
		return nil
	}
	infos := make([]*cachedPropertyInfo, info.TopLevelPropertyCount)
	for i := uint32(0); i < info.TopLevelPropertyCount; i++ {
		prInfo := info.getEventPropertyInfoAtIndex(i)
		infos[i] = parseEventPropertyInfo(buf, info, prInfo)
	}
	return infos
}

func parseEventPropertyInfo(buf []byte, info *TraceEventInfo, prInfo *EventPropertyInfo) *cachedPropertyInfo {
	propName := getStringFromBufferOffset(buf, prInfo.NameOffset)
	mapName := ""
	if prInfo.mapNameOffset() != 0 {
		mapName = getStringFromBufferOffset(buf, prInfo.mapNameOffset())
	}

	detail := &cachedPropertyInfo{
		ParsedInfo: prInfo,
		Name:       propName,
		InType:     prInfo.inType(),
		OutType:    prInfo.outType(),
		MapName:    mapName,
		IsStruct:   (prInfo.Flags & PropertyStruct) != 0,
		IsArray:    (prInfo.Flags&PropertyParamCount) != 0 || prInfo.count() > 1,
		IsFixedLen: (prInfo.Flags & PropertyParamFixedLength) != 0,
		Count:      prInfo.count(),
		Length:     prInfo.length(),
	}

	if !detail.IsStruct && !detail.IsArray {
		// If it's not a struct or array, we can return early
		return detail
	}

	if detail.IsStruct {
		startIndex := prInfo.structStartIndex()
		lastIndex := startIndex + prInfo.numOfStructMembers()
		for i := startIndex; i < lastIndex; i++ {
			memberInfo := info.getEventPropertyInfoAtIndex(uint32(i))
			member := parseEventPropertyInfo(buf, info, memberInfo)
			if member != nil {
				detail.StructMembers = append(detail.StructMembers, member)
			}
		}
	}

	return detail
}

func initEventInfoMaps(cache map[string]*cachedEventMapInfo, props []*cachedPropertyInfo, r *EventRecord) error {
	for _, prInfo := range props {
		if prInfo == nil || prInfo.MapName == "" {
			continue
		}

		if _, found := cache[prInfo.MapName]; found {
			// Already cached
			continue
		}

		if len(prInfo.StructMembers) > 0 {
			if err := initEventInfoMaps(cache, prInfo.StructMembers, r); err != nil {
				return fmt.Errorf("failed to get event maps for struct member %s: %w", prInfo.Name, err)
			}
		}

		pMapName := windows.StringToUTF16Ptr(prInfo.MapName)
		var mapInfo *EventMapInfo
		var mapBuf []byte
		var bufSize uint32
		var err error
		if err = _TdhGetEventMapInformation(r, pMapName, nil, &bufSize); errors.Is(err, ERROR_INSUFFICIENT_BUFFER) {
			mapBuf = make([]byte, bufSize)
			mapInfo = ((*EventMapInfo)(unsafe.Pointer(&mapBuf[0])))
			err = _TdhGetEventMapInformation(r, pMapName, mapInfo, &bufSize)
		}

		if err != nil {
			return fmt.Errorf("TdhGetEventMapInformation failed: %w", err)
		}

		cache[prInfo.MapName] = parseEventMapBuffer(prInfo.MapName, mapBuf, mapInfo)
	}

	return nil
}

func parseEventMapBuffer(mapName string, buf []byte, mapInfo *EventMapInfo) *cachedEventMapInfo {
	parsed := &cachedEventMapInfo{
		InfoBuf:    buf,
		ParsedInfo: mapInfo,
		Name:       mapName,
		IsValue:    (mapInfo.Flag&EventMapInfoFlagManifestValueMap) != 0 || (mapInfo.Flag&EventMapInfoFlagWBEMValueMap) != 0,
		IsBitmap:   (mapInfo.Flag&EventMapInfoFlagManifestBitMap) != 0 || (mapInfo.Flag&EventMapInfoFlagWBEMBitMap) != 0,
		IsPattern:  (mapInfo.Flag & EventMapInfoFlagManifestPatternMap) != 0,
		Entries:    make(map[any]*cachedMapEntry),
	}

	if mapInfo.EntryCount == 0 {
		return parsed
	}

	it := uintptr(unsafe.Pointer(&mapInfo.MapEntryArray[0]))
	for i := uint32(0); i < mapInfo.EntryCount; i++ {
		mapEntry := (*EventMapEntry)(unsafe.Pointer(it + uintptr(i)*unsafe.Sizeof(EventMapEntry{})))
		var entryKey any
		switch {
		case parsed.IsPattern:
			entryKey = getStringFromBufferOffset(buf, mapEntry.inputOffset())
		default:
			entryKey = mapEntry.value()
		}
		var entryValue any
		switch {
		case mapInfo.mapEntryValueType() == EventMapEntryValueTypeString:
			entryValue = getStringFromBufferOffset(buf, mapEntry.OutputOffset)
		case mapInfo.mapEntryValueType() == EventMapEntryValueTypeUlong:
			entryValue = mapEntry.value()
		}

		parsed.Entries[entryKey] = &cachedMapEntry{
			Value:   entryValue,
			IsUlong: mapInfo.mapEntryValueType() == EventMapEntryValueTypeUlong,
		}
	}
	return parsed
}

// isPrintable checks if the data is likely UTF-8/ASCII printable
func isPrintable(data []byte) bool {
	for _, b := range data {
		if b < 0x09 || (b > 0x0D && b < 0x20) {
			return false
		}
	}
	return true
}
