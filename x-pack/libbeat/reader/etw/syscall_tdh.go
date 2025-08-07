// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package etw

import (
	"fmt"
	"sort"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	tdh                                  = windows.NewLazySystemDLL("tdh.dll")
	tdhEnumerateProviders                = tdh.NewProc("TdhEnumerateProviders")
	tdhEnumerateManifestProviderEvents   = tdh.NewProc("TdhEnumerateManifestProviderEvents")
	tdhEnumerateProviderFieldInformation = tdh.NewProc("TdhEnumerateProviderFieldInformation")
	tdhGetEventInformation               = tdh.NewProc("TdhGetEventInformation")
	tdhGetEventMapInformation            = tdh.NewProc("TdhGetEventMapInformation")
	tdhFormatProperty                    = tdh.NewProc("TdhFormatProperty")
	tdhGetProperty                       = tdh.NewProc("TdhGetProperty")
)

const anysizeArray = 1
const DEFAULT_PROPERTY_BUFFER_SIZE = 256

// https://learn.microsoft.com/en-us/windows/win32/api/tdh/ns-tdh-provider_enumeration_info
type ProviderEnumerationInfo struct {
	NumberOfProviders      uint32
	Reserved               uint32
	TraceProviderInfoArray [anysizeArray]TraceProviderInfo
}

// https://learn.microsoft.com/en-us/windows/win32/api/tdh/ns-tdh-trace_provider_info
type TraceProviderInfo struct {
	ProviderGuid       windows.GUID
	SchemaSource       uint32
	ProviderNameOffset uint32
}

// https://learn.microsoft.com/en-us/windows/win32/api/evntcons/ns-evntcons-event_record
type EventRecord struct {
	EventHeader       EventHeader
	BufferContext     EtwBufferContext
	ExtendedDataCount uint16
	UserDataLength    uint16
	ExtendedData      *EventHeaderExtendedDataItem
	UserData          uintptr // Event data
	UserContext       uintptr
}

// https://learn.microsoft.com/en-us/windows/win32/api/relogger/ns-relogger-event_header
const (
	EVENT_HEADER_FLAG_EXTENDED_INFO   = 0x0001
	EVENT_HEADER_FLAG_PRIVATE_SESSION = 0x0002
	EVENT_HEADER_FLAG_STRING_ONLY     = 0x0004
	EVENT_HEADER_FLAG_TRACE_MESSAGE   = 0x0008
	EVENT_HEADER_FLAG_NO_CPUTIME      = 0x0010
	EVENT_HEADER_FLAG_32_BIT_HEADER   = 0x0020
	EVENT_HEADER_FLAG_64_BIT_HEADER   = 0x0040
	EVENT_HEADER_FLAG_DECODE_GUID     = 0x0080
	EVENT_HEADER_FLAG_CLASSIC_HEADER  = 0x0100
	EVENT_HEADER_FLAG_PROCESSOR_INDEX = 0x0200
)

var flagMap = map[uint16]string{
	EVENT_HEADER_FLAG_EXTENDED_INFO:   "EXTENDED_INFO",
	EVENT_HEADER_FLAG_PRIVATE_SESSION: "PRIVATE_SESSION",
	EVENT_HEADER_FLAG_STRING_ONLY:     "STRING_ONLY",
	EVENT_HEADER_FLAG_TRACE_MESSAGE:   "TRACE_MESSAGE",
	EVENT_HEADER_FLAG_NO_CPUTIME:      "NO_CPUTIME",
	EVENT_HEADER_FLAG_32_BIT_HEADER:   "32_BIT_HEADER",
	EVENT_HEADER_FLAG_64_BIT_HEADER:   "64_BIT_HEADER",
	EVENT_HEADER_FLAG_DECODE_GUID:     "DECODE_GUID",
	EVENT_HEADER_FLAG_CLASSIC_HEADER:  "CLASSIC_HEADER",
	EVENT_HEADER_FLAG_PROCESSOR_INDEX: "PROCESSOR_INDEX",
}

// https://learn.microsoft.com/en-us/windows/win32/api/relogger/ns-relogger-event_header
type EventHeader struct {
	Size            uint16
	HeaderType      uint16
	Flags           uint16
	EventProperty   uint16
	ThreadId        uint32
	ProcessId       uint32
	TimeStamp       int64
	ProviderId      windows.GUID
	EventDescriptor EventDescriptor
	Time            int64
	ActivityId      windows.GUID
}

func (r *EventRecord) pointerSize() uint32 {
	if r.EventHeader.Flags&EVENT_HEADER_FLAG_32_BIT_HEADER == EVENT_HEADER_FLAG_32_BIT_HEADER {
		return 4
	}
	return 8
}

// https://learn.microsoft.com/en-us/windows/win32/api/evntprov/ns-evntprov-event_descriptor
type EventDescriptor struct {
	Id      uint16
	Version uint8
	Channel uint8
	Level   uint8
	Opcode  uint8
	Task    uint16
	Keyword uint64
}

// https://learn.microsoft.com/en-us/windows/desktop/api/relogger/ns-relogger-etw_buffer_context
type EtwBufferContext struct {
	Union    uint16
	LoggerId uint16
}

// https://learn.microsoft.com/en-us/windows/win32/api/evntcons/ns-evntcons-event_header_extended_data_item
type EventHeaderExtendedDataItem struct {
	Reserved1      uint16
	ExtType        uint16
	InternalStruct uint16
	DataSize       uint16
	DataPtr        uint64
}

func parseGUID(item *EventHeaderExtendedDataItem) any {
	guid := (*windows.GUID)(unsafe.Pointer(uintptr(item.DataPtr)))
	return guid.String()
}

func parseSID(item *EventHeaderExtendedDataItem) any {
	sid := (*windows.SID)(unsafe.Pointer(uintptr(item.DataPtr)))
	if sid != nil {
		return sid.String()
	}
	return "SID is nil"
}

func parseUint32(item *EventHeaderExtendedDataItem) any {
	return *(*uint32)(unsafe.Pointer(uintptr(item.DataPtr)))
}

func parseUint64(item *EventHeaderExtendedDataItem) any {
	return *(*uint64)(unsafe.Pointer(uintptr(item.DataPtr)))
}

func parseUint32Slice(item *EventHeaderExtendedDataItem) any {
	return unsafe.Slice((*uint32)(unsafe.Pointer(uintptr(item.DataPtr))), int(item.DataSize/4))
}

func parseUint64Slice(item *EventHeaderExtendedDataItem) any {
	return unsafe.Slice((*uint64)(unsafe.Pointer(uintptr(item.DataPtr))), int(item.DataSize/8))
}

func parseByteSlice(item *EventHeaderExtendedDataItem) any {
	return unsafe.Slice((*byte)(unsafe.Pointer(uintptr(item.DataPtr))), int(item.DataSize))
}

// https://learn.microsoft.com/en-us/windows/win32/api/tdh/ns-tdh-tdh_context
type TdhContext struct {
	ParameterValue uint32
	ParameterType  int32
	ParameterSize  uint32
}

// https://learn.microsoft.com/en-us/windows/win32/api/tdh/ns-tdh-trace_event_info
type TraceEventInfo struct {
	ProviderGUID                windows.GUID
	EventGUID                   windows.GUID
	EventDescriptor             EventDescriptor
	DecodingSource              DecodingSource
	ProviderNameOffset          uint32
	LevelNameOffset             uint32
	ChannelNameOffset           uint32
	KeywordsNameOffset          uint32
	TaskNameOffset              uint32
	OpcodeNameOffset            uint32
	EventMessageOffset          uint32
	ProviderMessageOffset       uint32
	BinaryXMLOffset             uint32
	BinaryXMLSize               uint32
	ActivityIDNameOffset        uint32
	RelatedActivityIDNameOffset uint32
	PropertyCount               uint32
	TopLevelPropertyCount       uint32
	Flags                       TemplateFlags
	EventPropertyInfoArray      [anysizeArray]EventPropertyInfo
}

func (info *TraceEventInfo) getEventPropertyInfoAtIndex(i uint32) *EventPropertyInfo {
	if i >= info.PropertyCount {
		return nil
	}

	// Compute the pointer to the i-th EventPropertyInfo safely,
	// simulating C-style flexible array access using offset arithmetic.
	eventPropertyInfoPtr := uintptr(unsafe.Pointer(info)) +
		unsafe.Offsetof(info.EventPropertyInfoArray) +
		uintptr(i)*unsafe.Sizeof(EventPropertyInfo{})

	return (*EventPropertyInfo)(unsafe.Pointer(eventPropertyInfoPtr))
}

// https://learn.microsoft.com/en-us/windows/win32/api/tdh/ns-tdh-provider_event_info
type ProviderEventInfo struct {
	NumberOfEvents        uint32
	Reserved              uint32
	EventDescriptorsArray [anysizeArray]EventDescriptor
}

// https://learn.microsoft.com/en-us/windows/win32/api/tdh/ns-tdh-provider_field_info
type ProviderFieldInfo struct {
	NameOffset        uint32
	DescriptionOffset uint32
	Value             uint64
}

// https://learn.microsoft.com/en-us/windows/win32/api/tdh/ns-tdh-provider_field_infoarray
type ProviderFieldInfoArray struct {
	NumberOfElements uint32
	FieldType        EventFieldType
	FieldInfoArray   [anysizeArray]ProviderFieldInfo
}

// https://learn.microsoft.com/en-us/windows/desktop/api/tdh/ns-tdh-event_property_info
type EventPropertyInfo struct {
	Flags      PropertyFlags
	NameOffset uint32
	TypeUnion  struct {
		u1 uint16
		u2 uint16
		u3 uint32
	}
	CountUnion  uint16
	LengthUnion uint16
	ResTagUnion uint32
}

func (i *EventPropertyInfo) count() uint16 {
	return i.CountUnion
}

func (i *EventPropertyInfo) length() uint16 {
	return i.LengthUnion
}

func (i *EventPropertyInfo) inType() uint16 {
	return i.TypeUnion.u1
}

func (i *EventPropertyInfo) outType() uint16 {
	return i.TypeUnion.u2
}

func (i *EventPropertyInfo) structStartIndex() uint16 {
	return i.inType()
}

func (i *EventPropertyInfo) numOfStructMembers() uint16 {
	return i.outType()
}

func (i *EventPropertyInfo) mapNameOffset() uint32 {
	return i.TypeUnion.u3
}

// TDH Input Types
// https://learn.microsoft.com/en-us/windows/win32/api/tdh/ne-tdh-_tdh_in_type
const (
	TdhIntypeNull                        = 0
	TdhIntypeUnicodeString               = 1
	TdhIntypeAnsiString                  = 2
	TdhIntypeInt8                        = 3
	TdhIntypeUint8                       = 4
	TdhIntypeInt16                       = 5
	TdhIntypeUint16                      = 6
	TdhIntypeInt32                       = 7
	TdhIntypeUint32                      = 8
	TdhIntypeInt64                       = 9
	TdhIntypeUint64                      = 10
	TdhIntypeFloat                       = 11
	TdhIntypeDouble                      = 12
	TdhIntypeBoolean                     = 13
	TdhIntypeBinary                      = 14
	TdhIntypeGuid                        = 15
	TdhIntypePointer                     = 16
	TdhIntypeFileTime                    = 17
	TdhIntypeSystemTime                  = 18
	TdhIntypeSid                         = 19
	TdhIntypeHexInt32                    = 20
	TdhIntypeHexInt64                    = 21
	TdhIntypeCountedString               = 300
	TdhIntypeCountedAnsiString           = 301
	TdhIntypeReversedCountedString       = 302
	TdhIntypeReversedCountedAnsiString   = 303
	TdhIntypeNonNullTerminatedString     = 304
	TdhIntypeNonNullTerminatedAnsiString = 305
	TdhIntypeUnicodeChar                 = 306
	TdhIntypeAnsiChar                    = 307
	TdhIntypeSizeT                       = 308
	TdhIntypeHexDump                     = 309
	TdhIntypeWbemsid                     = 310
)

// TDH Output Types
// https://learn.microsoft.com/en-us/windows/win32/api/tdh/ne-tdh-_tdh_out_type
const (
	TdhOuttypeNull                       = 0
	TdhOuttypeString                     = 1
	TdhOuttypeDatetime                   = 2
	TdhOuttypeByte                       = 3
	TdhOuttypeUnsignedByte               = 4
	TdhOuttypeShort                      = 5
	TdhOuttypeUnsignedShort              = 6
	TdhOuttypeInt                        = 7
	TdhOuttypeUnsignedInt                = 8
	TdhOuttypeLong                       = 9
	TdhOuttypeUnsignedLong               = 10
	TdhOuttypeFloat                      = 11
	TdhOuttypeDouble                     = 12
	TdhOuttypeBoolean                    = 13
	TdhOuttypeGuid                       = 14
	TdhOuttypeHexBinary                  = 15
	TdhOuttypeHexInt8                    = 16
	TdhOuttypeHexInt16                   = 17
	TdhOuttypeHexInt32                   = 18
	TdhOuttypeHexInt64                   = 19
	TdhOuttypePid                        = 20
	TdhOuttypeTid                        = 21
	TdhOuttypePort                       = 22
	TdhOuttypeIpv4                       = 23
	TdhOuttypeIpv6                       = 24
	TdhOuttypeSocketAddress              = 25
	TdhOuttypeCimDatetime                = 26
	TdhOuttypeEtwTime                    = 27
	TdhOuttypeXml                        = 28
	TdhOuttypeErrorCode                  = 29
	TdhOuttypeWin32Error                 = 30
	TdhOuttypeNtstatus                   = 31
	TdhOuttypeHresult                    = 32
	TdhOuttypeCultureInsensitiveDatetime = 33
	TdhOuttypeJson                       = 34
	TdhOuttypeUtf8                       = 35
	TdhOuttypePkcs7WithTypeInfo          = 36
	TdhOuttypeCodePointer                = 37
	TdhOuttypeDatetimeUtc                = 38
	TdhOuttypeReducedString              = 300
	TdhOuttypeNoPrint                    = 301
)

type DecodingSource int32

const (
	DecodingSourceXMLFile DecodingSource = 0
	DecodingSourceWbem    DecodingSource = 1
	DecodingSourceWPP     DecodingSource = 2
	DecodingSourceTlg     DecodingSource = 3
	DecodingSourceMax     DecodingSource = 4
)

type TemplateFlags int32

// https://learn.microsoft.com/en-us/windows/win32/api/tdh/ne-tdh-template_flags
const (
	TemplateEventData   = TemplateFlags(1)
	TemplateUserData    = TemplateFlags(2)
	TemplateControlGUID = TemplateFlags(4)
)

type PropertyFlags int32

// https://learn.microsoft.com/en-us/windows/win32/api/tdh/ne-tdh-property_flags
const (
	PropertyStruct           = PropertyFlags(0x1)
	PropertyParamLength      = PropertyFlags(0x2)
	PropertyParamCount       = PropertyFlags(0x4)
	PropertyWBEMXmlFragment  = PropertyFlags(0x8)
	PropertyParamFixedLength = PropertyFlags(0x10)
	PropertyParamFixedCount  = PropertyFlags(0x20)
	PropertyHasTags          = PropertyFlags(0x40)
	PropertyHasCustomSchema  = PropertyFlags(0x80)
)

// https://learn.microsoft.com/en-us/windows/win32/api/tdh/ns-tdh-event_map_info
type EventMapInfo struct {
	NameOffset    uint32
	Flag          MapFlags
	EntryCount    uint32
	Union         uint32
	MapEntryArray [anysizeArray]EventMapEntry
}

func (mi *EventMapInfo) mapEntryValueType() MapValueType {
	return MapValueType(mi.Union)
}

type MapValueType uint32

const (
	EventMapEntryValueTypeUlong MapValueType = iota
	EventMapEntryValueTypeString
)

type MapFlags uint32

const (
	EventMapInfoFlagManifestValueMap   = MapFlags(0x1)
	EventMapInfoFlagManifestBitMap     = MapFlags(0x2)
	EventMapInfoFlagManifestPatternMap = MapFlags(0x4)
	EventMapInfoFlagWBEMValueMap       = MapFlags(0x8)
	EventMapInfoFlagWBEMBitMap         = MapFlags(0x10)
	EventMapInfoFlagWBEMFlag           = MapFlags(0x20)
	EventMapInfoFlagWBEMNoMap          = MapFlags(0x40)
)

type EventFieldType uint32

// https://learn.microsoft.com/en-us/windows/win32/api/tdh/ne-tdh-event_field_type
const (
	EventKeywordInformation EventFieldType = iota
	EventLevelInformation
	EventChannelInformation
	EventTaskInformation
	EventOpcodeInformation
	EventInformationMax
)

// https://learn.microsoft.com/en-us/windows/win32/api/tdh/ns-tdh-event_map_entry
type EventMapEntry struct {
	OutputOffset uint32
	Union        uint32
}

func (me *EventMapEntry) value() uint32 {
	return me.Union
}

func (me *EventMapEntry) inputOffset() uint32 {
	return me.Union
}

// https://learn.microsoft.com/en-us/windows/desktop/api/tdh/ns-tdh-property_data_descriptor
type PropertyDataDescriptor struct {
	PropertyName unsafe.Pointer
	ArrayIndex   uint32
	Reserved     uint32
}

// enumerateProvidersFunc is used to replace the pointer to the function in unit tests
var enumerateProvidersFunc = _TdhEnumerateProviders

// https://learn.microsoft.com/en-us/windows/win32/api/tdh/nf-tdh-tdhenumerateproviders
func _TdhEnumerateProviders(
	pBuffer *ProviderEnumerationInfo,
	pBufferSize *uint32) error {
	r0, _, _ := tdhEnumerateProviders.Call(
		uintptr(unsafe.Pointer(pBuffer)),
		uintptr(unsafe.Pointer(pBufferSize)))
	if r0 == 0 {
		return nil
	}
	return syscall.Errno(r0)
}

// https://learn.microsoft.com/en-us/windows/win32/api/tdh/nf-tdh-tdhenumeratemanifestproviderevents
func _TdhEnumerateManifestProviderEvents(
	providerGUID *windows.GUID,
	pBuffer *ProviderEventInfo,
	pBufferSize *uint32) error {
	r0, _, _ := tdhEnumerateManifestProviderEvents.Call(
		uintptr(unsafe.Pointer(providerGUID)),
		uintptr(unsafe.Pointer(pBuffer)),
		uintptr(unsafe.Pointer(pBufferSize)))
	if r0 == 0 {
		return nil
	}
	return syscall.Errno(r0)
}

// https://learn.microsoft.com/en-us/windows/win32/api/tdh/nf-tdh-tdhenumerateproviderfieldinformation
func _TdhEnumerateProviderFieldInformation(
	providerGUID *windows.GUID,
	eventFieldType EventFieldType,
	pBuffer *ProviderFieldInfoArray,
	pBufferSize *uint32) error {
	r0, _, _ := tdhEnumerateProviderFieldInformation.Call(
		uintptr(unsafe.Pointer(providerGUID)),
		uintptr(eventFieldType),
		uintptr(unsafe.Pointer(pBuffer)),
		uintptr(unsafe.Pointer(pBufferSize)))
	if r0 == 0 {
		return nil
	}
	return syscall.Errno(r0)
}

// https://learn.microsoft.com/en-us/windows/win32/api/tdh/nf-tdh-tdhgeteventinformation
func _TdhGetEventInformation(pEvent *EventRecord,
	tdhContextCount uint32,
	pTdhContext *TdhContext,
	pBuffer *TraceEventInfo,
	pBufferSize *uint32) error {
	r0, _, _ := tdhGetEventInformation.Call(
		uintptr(unsafe.Pointer(pEvent)),
		uintptr(tdhContextCount),
		uintptr(unsafe.Pointer(pTdhContext)),
		uintptr(unsafe.Pointer(pBuffer)),
		uintptr(unsafe.Pointer(pBufferSize)))
	if r0 == 0 {
		return nil
	}
	return syscall.Errno(r0)
}

// https://learn.microsoft.com/en-us/windows/win32/api/tdh/nf-tdh-tdhformatproperty
func _TdhFormatProperty(
	eventInfo *TraceEventInfo,
	mapInfo *EventMapInfo,
	pointerSize uint32,
	propertyInType uint16,
	propertyOutType uint16,
	propertyLength uint16,
	userDataLength uint16,
	userData *byte,
	bufferSize *uint32,
	buffer *uint8,
	userDataConsumed *uint16) error {
	r0, _, _ := tdhFormatProperty.Call(
		uintptr(unsafe.Pointer(eventInfo)),
		uintptr(unsafe.Pointer(mapInfo)),
		uintptr(pointerSize),
		uintptr(propertyInType),
		uintptr(propertyOutType),
		uintptr(propertyLength),
		uintptr(userDataLength),
		uintptr(unsafe.Pointer(userData)),
		uintptr(unsafe.Pointer(bufferSize)),
		uintptr(unsafe.Pointer(buffer)),
		uintptr(unsafe.Pointer(userDataConsumed)))
	if r0 == 0 {
		return nil
	}
	return syscall.Errno(r0)
}

// https://learn.microsoft.com/en-us/windows/win32/api/tdh/nf-tdh-tdhgetproperty
func _TdhGetProperty(pEvent *EventRecord,
	tdhContextCount uint32,
	pTdhContext *TdhContext,
	propertyDataCount uint32,
	pPropertyData *PropertyDataDescriptor,
	bufferSize uint32,
	pBuffer *byte) error {
	r0, _, _ := tdhGetProperty.Call(
		uintptr(unsafe.Pointer(pEvent)),
		uintptr(tdhContextCount),
		uintptr(unsafe.Pointer(pTdhContext)),
		uintptr(propertyDataCount),
		uintptr(unsafe.Pointer(pPropertyData)),
		uintptr(bufferSize),
		uintptr(unsafe.Pointer(pBuffer)))
	if r0 == 0 {
		return nil
	}
	return syscall.Errno(r0)
}

// https://learn.microsoft.com/en-us/windows/win32/api/tdh/nf-tdh-tdhgeteventmapinformation
func _TdhGetEventMapInformation(pEvent *EventRecord,
	pMapName *uint16,
	pBuffer *EventMapInfo,
	pBufferSize *uint32) error {
	r0, _, _ := tdhGetEventMapInformation.Call(
		uintptr(unsafe.Pointer(pEvent)),
		uintptr(unsafe.Pointer(pMapName)),
		uintptr(unsafe.Pointer(pBuffer)),
		uintptr(unsafe.Pointer(pBufferSize)))
	if r0 == 0 {
		return nil
	}
	return syscall.Errno(r0)
}

// String returns a human-readable representation of the EventRecord
func (r *EventRecord) String() string {
	if r == nil {
		return "<nil EventRecord>"
	}

	return fmt.Sprintf("EventRecord{Provider: %s, ID: %d, Version: %d, Level: %d, Task: %d, Opcode: %d, Keyword: 0x%X, ProcessID: %d, ThreadID: %d, Timestamp: %d, UserDataLength: %d}",
		r.EventHeader.ProviderId.String(),
		r.EventHeader.EventDescriptor.Id,
		r.EventHeader.EventDescriptor.Version,
		r.EventHeader.EventDescriptor.Level,
		r.EventHeader.EventDescriptor.Task,
		r.EventHeader.EventDescriptor.Opcode,
		r.EventHeader.EventDescriptor.Keyword,
		r.EventHeader.ProcessId,
		r.EventHeader.ThreadId,
		r.EventHeader.TimeStamp,
		r.UserDataLength,
	)
}

// FlagsAsStrings returns a human-readable representation of EventHeader flags
func (h *EventHeader) FlagsAsStrings() []string {
	if h.Flags == 0 {
		return nil
	}

	var flags []string
	remainingFlags := h.Flags

	// Check each known flag
	for flagValue, flagName := range flagMap {
		if h.Flags&flagValue != 0 {
			flags = append(flags, flagName)
			remainingFlags &^= flagValue // Remove this flag from remaining
		}
	}

	// Add any unknown flags as hex
	if remainingFlags != 0 {
		flags = append(flags, fmt.Sprintf("UNKNOWN(0x%04X)", remainingFlags))
	}

	sort.Strings(flags)
	return flags
}
