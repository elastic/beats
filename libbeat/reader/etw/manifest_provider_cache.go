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
	"regexp"
	"strings"
	"sync"
	"text/template"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/elastic/elastic-agent-libs/logp"
)

// manifestProviderCache stores metadata from a provider.
type manifestProviderCache struct {
	mutex sync.RWMutex
	log   *logp.Logger

	guid windows.GUID

	einfoCache        map[EventDescriptor]*cachedEventInfo
	keywords          map[uint64]*cachedProviderKeyword
	propertyMapsCache map[string]*cachedEventMapInfo
	failedDescriptors map[EventDescriptor]struct{}
}

func newManifestProviderCache(guid windows.GUID, log *logp.Logger) (*manifestProviderCache, error) {
	cache := &manifestProviderCache{
		guid:              guid,
		log:               log,
		einfoCache:        make(map[EventDescriptor]*cachedEventInfo),
		keywords:          make(map[uint64]*cachedProviderKeyword),
		propertyMapsCache: make(map[string]*cachedEventMapInfo),
		failedDescriptors: make(map[EventDescriptor]struct{}),
	}
	if err := cache.init(); err != nil {
		return nil, fmt.Errorf("failed to initialize manifest provider cache for %s: %w", guid, err)
	}
	log.Info("Initialized manifest-based provider cache")
	return cache, nil
}

func (cache *manifestProviderCache) init() error {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()

	if cache.guid == (windows.GUID{}) {
		return fmt.Errorf("provider cache not initialized with a GUID")
	}

	if err := cache.initKeywords(); err != nil {
		cache.log.Errorf("failed to initialize keywords for provider %s: %v", cache.guid, err)
	}

	descriptors, err := getProviderEventDescriptors(&cache.guid)
	if err != nil {
		cache.log.Errorf("failed to get event descriptors for provider %s: %v", cache.guid, err)
		return nil
	}

	for _, desc := range descriptors {
		r := &EventRecord{
			EventHeader: EventHeader{
				ProviderId:      cache.guid,
				EventDescriptor: desc,
			},
		}
		if err := cache.initEvent(r); err != nil {
			cache.log.Errorf("failed to initialize event for provider %s: %v", cache.guid, err)
		}
	}

	return nil
}

func (cache *manifestProviderCache) initKeywords() error {
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

func (cache *manifestProviderCache) getKeywords() map[uint64]*cachedProviderKeyword {
	cache.mutex.RLock()
	defer cache.mutex.RUnlock()
	return cache.keywords
}

func (cache *manifestProviderCache) initEvent(r *EventRecord) error {
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

	cached.EventMessageTemplate, err = compileEventMessageTemplate(cached.EventMessage)
	if err != nil {
		return fmt.Errorf("failed to compile event message template for %s: %w", cache.guid, err)
	}

	cached.Properties = getEventInfoProperties(buf, info)
	if err = initEventInfoMaps(cache.propertyMapsCache, cached.Properties, r); err != nil {
		return fmt.Errorf("failed to get event maps for %s: %w", cache.guid, err)
	}
	cache.einfoCache[r.EventHeader.EventDescriptor] = cached
	return nil
}

func (cache *manifestProviderCache) getPropertyMaps() map[string]*cachedEventMapInfo {
	cache.mutex.RLock()
	defer cache.mutex.RUnlock()
	return cache.propertyMapsCache
}

func (cache *manifestProviderCache) getEventInfo(r *EventRecord) (*cachedEventInfo, error) {
	cache.mutex.RLock()
	desc := r.EventHeader.EventDescriptor
	cached, found := cache.einfoCache[desc]
	if found {
		cache.mutex.RUnlock()
		return cached, nil
	}

	_, found = cache.failedDescriptors[desc]
	cache.mutex.RUnlock()
	if found {
		cache.log.Debugf("event descriptor %v not found in provider %s, returning nil", desc, cache.guid)
		return nil, ErrUnprocessableEvent
	}

	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	cached, found = cache.einfoCache[desc]
	if found {
		return cached, nil
	}
	// If not found, try to initialize the event info
	if err := cache.initEvent(r); err != nil {
		// we want to return the error only once to avoid log spam, so we cache the failed descriptor
		cache.failedDescriptors[desc] = struct{}{}
		return nil, fmt.Errorf("failed to initialize event info for event record %s: %w", r, err)
	}
	cached, found = cache.einfoCache[desc]
	if found {
		return cached, nil
	}
	return nil, ErrUnprocessableEvent
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

var dataIdxRegexp = regexp.MustCompile(`%(\d+)`)

func compileEventMessageTemplate(eventMessage string) (*template.Template, error) {
	// If there is no message or it does not contain any parameter indices, return nil.
	// This avoids unnecessary template parsing and compilation.
	if eventMessage == "" || !dataIdxRegexp.MatchString(eventMessage) {
		return nil, nil
	}

	// We want to replace all occurrences of %N with [[{{. $ N}}]]
	// where N is the parameter index.
	replacer := func(message string) string {
		return dataIdxRegexp.ReplaceAllString(message, `[[{{index . $1}}]]`)
	}

	processedMessage := replacer(eventMessage)

	// Set custom delimiters to avoid conflicts with Windows event message format
	tmpl, err := template.New("eventMessage").Delims("[[{{", "}}]]").Parse(processedMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to parse event message template: %w", err)
	}
	return tmpl, nil
}

func getStringFromBufferOffset(buf []byte, offset uint32) string {
	if offset == 0 || len(buf) == 0 {
		return ""
	}
	ptr := unsafe.Add(unsafe.Pointer(&buf[0]), offset)
	return strings.TrimSpace(windows.UTF16PtrToString((*uint16)(ptr)))
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
	if offset == 0 || len(buf) == 0 || int(offset) >= len(buf) {
		return nil
	}

	var results []string
	currentPtr := unsafe.Add(unsafe.Pointer(&buf[0]), offset)

	for {
		str := windows.UTF16PtrToString((*uint16)(currentPtr))
		if str == "" {
			break
		}
		results = append(results, strings.TrimSpace(str))

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

	// Determine if the property has a fixed length based on the flags
	var isFixedLen bool
	switch {
	case (prInfo.Flags & PropertyParamFixedLength) != 0:
		// Property has explicitly fixed length
		isFixedLen = true
	case prInfo.length() > 0:
		// Property has a non-zero length field (typically fixed length)
		isFixedLen = true
	case (prInfo.Flags & PropertyParamLength) != 0:
		// Length is specified by another property (variable length)
		isFixedLen = false
	default:
		// No length information or zero length (typically variable length)
		isFixedLen = false
	}

	// Determine if the property is an array based on flags and count
	var isArray bool
	if (prInfo.Flags & PropertyParamCount) != 0 {
		// Count is specified by another property (variable-size array)
		isArray = true
	} else if prInfo.count() > 1 {
		// Fixed-size array with count > 1
		isArray = true
	} else {
		// Single value (count == 0 or count == 1 without PropertyParamCount)
		isArray = false
	}

	detail := &cachedPropertyInfo{
		ParsedInfo: prInfo,
		Name:       propName,
		InType:     prInfo.inType(),
		OutType:    prInfo.outType(),
		MapName:    mapName,
		IsStruct:   (prInfo.Flags & PropertyStruct) != 0,
		IsArray:    isArray,
		IsFixedLen: isFixedLen,
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

		// Bounds checking to prevent reading beyond property array
		if uint32(lastIndex) > info.PropertyCount {
			lastIndex = uint16(info.PropertyCount) //nolint:gosec // This is a safe cast since we already checked the bounds
		}

		for i := startIndex; i < lastIndex; i++ {
			memberInfo := info.getEventPropertyInfoAtIndex(uint32(i))
			if memberInfo == nil {
				continue // Skip invalid property info
			}

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
		InfoBuf:                buf,
		ParsedInfo:             mapInfo,
		Name:                   mapName,
		IsValue:                (mapInfo.Flag&EventMapInfoFlagManifestValueMap) != 0 || (mapInfo.Flag&EventMapInfoFlagWBEMValueMap) != 0,
		IsBitmap:               (mapInfo.Flag&EventMapInfoFlagManifestBitMap) != 0 || (mapInfo.Flag&EventMapInfoFlagWBEMBitMap) != 0,
		IsPattern:              (mapInfo.Flag & EventMapInfoFlagManifestPatternMap) != 0,
		Entries:                make(map[any]*cachedMapEntry),
		formattedCachedEntries: make(map[string]formattedMapCacheEntry),
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
