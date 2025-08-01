// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package etw

import (
	"bytes"
	"fmt"
	"sync"
	"text/template"

	"golang.org/x/sys/windows"

	"github.com/elastic/elastic-agent-libs/logp"
)

type metadataCache struct {
	mutex         sync.RWMutex
	providerCache map[windows.GUID]providerCache
	log           *logp.Logger
	bufferPools   *bufferPools
}

func newMetadataCache(log *logp.Logger) *metadataCache {
	log = log.Named("metadata_cache")
	return &metadataCache{
		providerCache: make(map[windows.GUID]providerCache),
		log:           log,
		bufferPools:   newBufferPools(),
	}
}

func (cache *metadataCache) getProviderCache(guid windows.GUID) (providerCache, error) {
	effectiveGUID := findEffectiveGUID(guid)
	cache.mutex.RLock()
	provider, found := cache.providerCache[effectiveGUID]
	cache.mutex.RUnlock()
	if found {
		return provider, nil
	}
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	provider, found = cache.providerCache[effectiveGUID]
	if found {
		return provider, nil
	}
	// If not found, create a new provider cache
	provider, err := newProviderCache(effectiveGUID, cache.log)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider cache for %s: %w", effectiveGUID, err)
	}
	cache.providerCache[effectiveGUID] = provider
	return provider, nil
}

type providerCache interface {
	getEventInfo(r *EventRecord) (*cachedEventInfo, error)
	getKeywords() map[uint64]*cachedProviderKeyword
	getPropertyMaps() map[string]*cachedEventMapInfo
}

func newProviderCache(guid windows.GUID, log *logp.Logger) (providerCache, error) {
	log = log.Named("provider_cache").With("guid", guid)
	return newManifestProviderCache(guid, log)
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

	EventMessageTemplate *template.Template
}

func (info *cachedEventInfo) renderEventMessage(props []RenderedProperty) string {
	if info.EventMessageTemplate == nil {
		// No template to render, return the raw message
		return info.EventMessage
	}

	// Create a map for the template parameters
	paramMap := make(map[int]any)
	for i, p := range props {
		paramMap[i+1] = p.Value // Use 1-based index for template parameters
	}

	// Render the template with the provided parameters
	var result bytes.Buffer
	if err := info.EventMessageTemplate.Execute(&result, paramMap); err != nil {
		return info.EventMessage + " (template error: " + err.Error() + ")"
	}
	return result.String()
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

// hasLengthFromOtherProperty returns true if this property's length is specified by another property
func (info *cachedPropertyInfo) hasLengthFromOtherProperty() bool {
	return (info.ParsedInfo.Flags & PropertyParamLength) != 0
}

// hasCountFromOtherProperty returns true if this property's count is specified by another property
func (info *cachedPropertyInfo) hasCountFromOtherProperty() bool {
	return (info.ParsedInfo.Flags & PropertyParamCount) != 0
}

// getCountPropertyIndex returns the index of the property that contains the count for this property
// Only valid when hasCountFromOtherProperty() returns true
func (info *cachedPropertyInfo) getCountPropertyIndex() uint16 {
	return info.Count
}

// getLengthPropertyIndex returns the index of the property that contains the length for this property
// Only valid when hasLengthFromOtherProperty() returns true
func (info *cachedPropertyInfo) getLengthPropertyIndex() uint16 {
	return info.Length
}

// isVariableLengthString returns true if this is a variable-length string (Unicode or ANSI)
func (info *cachedPropertyInfo) isVariableLengthString() bool {
	return !info.IsFixedLen &&
		(info.InType == TdhIntypeUnicodeString || info.InType == TdhIntypeAnsiString)
}

// isIPv6Address returns true if this property represents an IPv6 address
func (info *cachedPropertyInfo) isIPv6Address() bool {
	return info.InType == TdhIntypeBinary && info.OutType == TdhOuttypeIpv6
}

// getFixedLength returns the fixed length for this property
// Only valid when IsFixedLen is true and not an IPv6 address
func (info *cachedPropertyInfo) getFixedLength() uint32 {
	if info.isIPv6Address() {
		return 16
	}
	return uint32(info.Length)
}

type cachedProviderKeyword struct {
	Name        string
	Description string
}

type cachedMapEntry struct {
	Value   any
	IsUlong bool // true if the value is a uint64, false if it's a string
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

	// Lazy cache for TdhFormatProperty results
	// Key is hex string of the raw buffer bytes, Value is the formatted result
	formattedCachedEntries map[string]formattedMapCacheEntry
	m                      sync.RWMutex
}

type formattedMapCacheEntry struct {
	Result   string
	Consumed int
}

func (mi *cachedEventMapInfo) getFormattedMapEntry(propInfo *cachedPropertyInfo, rawBuf []byte, length int) (string, int, bool) {
	key := fmt.Sprintf("%x", rawBuf[:length])
	mi.m.RLock()
	value, found := mi.formattedCachedEntries[key]
	mi.m.RUnlock()
	if found {
		return value.Result, value.Consumed, true
	}
	return "", 0, false
}

func (mi *cachedEventMapInfo) cacheFormattedMapEntry(propInfo *cachedPropertyInfo, rawBuf []byte, value string, length int) {
	key := fmt.Sprintf("%x", rawBuf[:length])
	mi.m.Lock()
	defer mi.m.Unlock()
	mi.formattedCachedEntries[key] = formattedMapCacheEntry{
		Result:   value,
		Consumed: length,
	}
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

// bufferPools provides centralized memory management for ETW event processing.
// It handles buffer pooling to optimize memory usage and reduce garbage collection pressure.
type bufferPools struct {
	// Buffer pools for different size categories
	smallBufferPool  sync.Pool // <= 256 bytes
	mediumBufferPool sync.Pool // <= 1024 bytes
	largeBufferPool  sync.Pool // <= 4096 bytes
}

// newBufferPools creates a new memory manager with properly initialized pools.
func newBufferPools() *bufferPools {
	return &bufferPools{
		smallBufferPool: sync.Pool{
			New: func() interface{} {
				buf := make([]byte, 256)
				return &buf
			},
		},
		mediumBufferPool: sync.Pool{
			New: func() interface{} {
				buf := make([]byte, 1024)
				return &buf
			},
		},
		largeBufferPool: sync.Pool{
			New: func() interface{} {
				buf := make([]byte, 4096)
				return &buf
			},
		},
	}
}

// getBuffer retrieves a buffer from the appropriate pool based on the requested size.
// The returned buffer should be returned to the pool using putBuffer.
func (mm *bufferPools) getBuffer(size uint32) *[]byte {
	switch {
	case size <= 256:
		buf, _ := mm.smallBufferPool.Get().(*[]byte)
		return buf
	case size <= 1024:
		buf, _ := mm.mediumBufferPool.Get().(*[]byte)
		return buf
	default:
		buf, _ := mm.largeBufferPool.Get().(*[]byte)
		return buf
	}
}

// putBuffer returns a buffer to the appropriate pool after clearing it.
func (mm *bufferPools) putBuffer(buf *[]byte, size uint32) {
	// Clear the buffer for reuse
	for i := range *buf {
		(*buf)[i] = 0
	}

	switch {
	case size <= 256:
		mm.smallBufferPool.Put(buf)
	case size <= 1024:
		mm.mediumBufferPool.Put(buf)
	default:
		mm.largeBufferPool.Put(buf)
	}
}
