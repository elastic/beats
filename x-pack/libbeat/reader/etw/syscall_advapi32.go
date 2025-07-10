// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package etw

import (
	"errors"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	advapi32 = windows.NewLazySystemDLL("advapi32.dll")
	// Controller
	startTraceW    = advapi32.NewProc("StartTraceW")
	enableTraceEx2 = advapi32.NewProc("EnableTraceEx2") // Manifest-based providers and filtering
	controlTraceW  = advapi32.NewProc("ControlTraceW")
	// Consumer
	openTraceW   = advapi32.NewProc("OpenTraceW")
	processTrace = advapi32.NewProc("ProcessTrace")
	closeTrace   = advapi32.NewProc("CloseTrace")
)

// https://learn.microsoft.com/en-us/windows/win32/api/evntrace/ns-evntrace-event_trace
type EventTrace struct {
	Header           EventTraceHeader
	InstanceId       uint32
	ParentInstanceId uint32
	ParentGuid       windows.GUID
	MofData          uintptr
	MofLength        uint32
	UnionCtx         uint32
}

// https://learn.microsoft.com/en-us/windows/win32/api/evntrace/ns-evntrace-event_trace_header
type EventTraceHeader struct {
	Size      uint16
	Union1    uint16
	Union2    uint32
	ThreadId  uint32
	ProcessId uint32
	TimeStamp int64
	Union3    [16]byte
	Union4    uint64
}

// https://learn.microsoft.com/en-us/windows/win32/api/evntrace/ns-evntrace-event_trace_properties
type EventTraceProperties struct {
	Wnode               WnodeHeader
	BufferSize          uint32
	MinimumBuffers      uint32
	MaximumBuffers      uint32
	MaximumFileSize     uint32
	LogFileMode         uint32
	FlushTimer          uint32
	EnableFlags         uint32
	AgeLimit            int32
	NumberOfBuffers     uint32
	FreeBuffers         uint32
	EventsLost          uint32
	BuffersWritten      uint32
	LogBuffersLost      uint32
	RealTimeBuffersLost uint32
	LoggerThreadId      syscall.Handle
	LogFileNameOffset   uint32
	LoggerNameOffset    uint32
}

// https://learn.microsoft.com/en-us/windows/win32/etw/wnode-header
type WnodeHeader struct {
	BufferSize    uint32
	ProviderId    uint32
	Union1        uint64
	Union2        int64
	Guid          windows.GUID
	ClientContext uint32
	Flags         uint32
}

// Used to enable a provider via EnableTraceEx2
// https://learn.microsoft.com/en-us/windows/win32/api/evntrace/ns-evntrace-enable_trace_parameters
type EnableTraceParameters struct {
	Version          uint32
	EnableProperty   uint32
	ControlFlags     uint32
	SourceId         windows.GUID
	EnableFilterDesc *EventFilterDescriptor
	FilterDescrCount uint32
}

// Defines the filter data that a session passes
// to the provider's enable callback function
// https://learn.microsoft.com/en-us/windows/win32/api/evntprov/ns-evntprov-event_filter_descriptor
type EventFilterDescriptor struct {
	Ptr  uint64
	Size uint32
	Type uint32
}

// https://learn.microsoft.com/en-us/windows/win32/api/evntrace/ns-evntrace-event_trace_logfilew
type EventTraceLogfile struct {
	LogFileName    *uint16 // Logfile
	LoggerName     *uint16 // Real-time session
	CurrentTime    int64
	BuffersRead    uint32
	LogFileMode    uint32
	CurrentEvent   EventTrace
	LogfileHeader  TraceLogfileHeader
	BufferCallback uintptr
	BufferSize     uint32
	Filled         uint32
	EventsLost     uint32
	// Receive events (EventRecordCallback (TDH) or EventCallback)
	// Tip: New code should use EventRecordCallback instead of EventCallback.
	// The EventRecordCallback receives an EVENT_RECORD which contains
	// more complete event information
	Callback      uintptr
	IsKernelTrace uint32
	Context       uintptr
}

// https://learn.microsoft.com/en-us/windows/win32/api/evntrace/ns-evntrace-trace_logfile_header
type TraceLogfileHeader struct {
	BufferSize         uint32
	VersionUnion       uint32
	ProviderVersion    uint32
	NumberOfProcessors uint32
	EndTime            int64
	TimerResolution    uint32
	MaximumFileSize    uint32
	LogFileMode        uint32
	BuffersWritten     uint32
	Union1             [16]byte
	LoggerName         *uint16
	LogFileName        *uint16
	TimeZone           windows.Timezoneinformation
	BootTime           int64
	PerfFreq           int64
	StartTime          int64
	ReservedFlags      uint32
	BuffersLost        uint32
}

// https://learn.microsoft.com/en-us/windows/win32/api/minwinbase/ns-minwinbase-filetime
type FileTime struct {
	dwLowDateTime  uint32
	dwHighDateTime uint32
}

// https://learn.microsoft.com/en-us/windows/win32/api/minwinbase/ns-minwinbase-systemtime
type SystemTime struct {
	Year         uint16
	Month        uint16
	DayOfWeek    uint16
	Day          uint16
	Hour         uint16
	Minute       uint16
	Second       uint16
	Milliseconds uint16
}

// https://learn.microsoft.com/en-us/windows/win32/api/evntrace/nf-evntrace-enabletrace
const (
	TRACE_LEVEL_NONE        = 0
	TRACE_LEVEL_CRITICAL    = 1
	TRACE_LEVEL_FATAL       = 1
	TRACE_LEVEL_ERROR       = 2
	TRACE_LEVEL_WARNING     = 3
	TRACE_LEVEL_INFORMATION = 4
	TRACE_LEVEL_VERBOSE     = 5
)

// https://learn.microsoft.com/en-us/windows/win32/api/evntprov/nc-evntprov-penablecallback
const (
	EVENT_CONTROL_CODE_DISABLE_PROVIDER = 0
	EVENT_CONTROL_CODE_ENABLE_PROVIDER  = 1
	EVENT_CONTROL_CODE_CAPTURE_STATE    = 2
)

// https://learn.microsoft.com/en-us/windows/win32/api/evntrace/nf-evntrace-controltracea
const (
	EVENT_TRACE_CONTROL_QUERY  = 0
	EVENT_TRACE_CONTROL_STOP   = 1
	EVENT_TRACE_CONTROL_UPDATE = 2
	EVENT_TRACE_CONTROL_FLUSH  = 3
)

// https://learn.microsoft.com/en-us/windows/win32/api/evntrace/ns-evntrace-event_trace_logfilea
const (
	PROCESS_TRACE_MODE_REAL_TIME     = 0x00000100
	PROCESS_TRACE_MODE_RAW_TIMESTAMP = 0x00001000
	PROCESS_TRACE_MODE_EVENT_RECORD  = 0x10000000
)

const INVALID_PROCESSTRACE_HANDLE = 0xFFFFFFFFFFFFFFFF

// https://learn.microsoft.com/en-us/windows/win32/debug/system-error-codes
const (
	ERROR_ACCESS_DENIED          syscall.Errno = 5
	ERROR_INVALID_HANDLE         syscall.Errno = 6
	ERROR_BAD_LENGTH             syscall.Errno = 24
	ERROR_INVALID_PARAMETER      syscall.Errno = 87
	ERROR_INSUFFICIENT_BUFFER    syscall.Errno = 122
	ERROR_BAD_PATHNAME           syscall.Errno = 161
	ERROR_ALREADY_EXISTS         syscall.Errno = 183
	ERROR_NOT_FOUND              syscall.Errno = 1168
	ERROR_NO_SYSTEM_RESOURCES    syscall.Errno = 1450
	ERROR_TIMEOUT                syscall.Errno = 1460
	ERROR_WMI_INSTANCE_NOT_FOUND syscall.Errno = 4201
	ERROR_CTX_CLOSE_PENDING      syscall.Errno = 7007
	ERROR_EVT_INVALID_EVENT_DATA syscall.Errno = 15005
)

// https://learn.microsoft.com/en-us/windows/win32/etw/logging-mode-constants (to extend modes)
// https://learn.microsoft.com/en-us/windows-hardware/drivers/ddi/wmistr/ns-wmistr-_wnode_header (to extend flags)
const (
	WNODE_FLAG_ALL_DATA        = 0x00000001
	WNODE_FLAG_TRACED_GUID     = 0x00020000
	EVENT_TRACE_REAL_TIME_MODE = 0x00000100
)

// https://learn.microsoft.com/en-us/windows/win32/api/evntcons/ns-evntcons-event_header_extended_data_item
const (
	EVENT_HEADER_EXT_TYPE_RELATED_ACTIVITYID = 0x0001
	EVENT_HEADER_EXT_TYPE_SID                = 0x0002
	EVENT_HEADER_EXT_TYPE_TS_ID              = 0x0003
	EVENT_HEADER_EXT_TYPE_INSTANCE_INFO      = 0x0004
	EVENT_HEADER_EXT_TYPE_STACK_TRACE32      = 0x0005
	EVENT_HEADER_EXT_TYPE_STACK_TRACE64      = 0x0006
	EVENT_HEADER_EXT_TYPE_PEBS_INDEX         = 0x0007
	EVENT_HEADER_EXT_TYPE_PMC_COUNTERS       = 0x0008
	EVENT_HEADER_EXT_TYPE_PSM_KEY            = 0x0009
	EVENT_HEADER_EXT_TYPE_EVENT_KEY          = 0x000A
	EVENT_HEADER_EXT_TYPE_EVENT_SCHEMA_TL    = 0x000B
	EVENT_HEADER_EXT_TYPE_PROV_TRAITS        = 0x000C
	EVENT_HEADER_EXT_TYPE_PROCESS_START_KEY  = 0x000D
	EVENT_HEADER_EXT_TYPE_CONTROL_GUID       = 0x000E
	EVENT_HEADER_EXT_TYPE_QPC_DELTA          = 0x000F
	EVENT_HEADER_EXT_TYPE_CONTAINER_ID       = 0x0010
	EVENT_HEADER_EXT_TYPE_MAX                = 0x0011
)

func extTypeToStr(extType uint16) string {
	switch extType {
	case EVENT_HEADER_EXT_TYPE_RELATED_ACTIVITYID:
		return "RELATED_ACTIVITYID"
	case EVENT_HEADER_EXT_TYPE_SID:
		return "SID"
	case EVENT_HEADER_EXT_TYPE_TS_ID:
		return "TS_ID"
	case EVENT_HEADER_EXT_TYPE_INSTANCE_INFO:
		return "INSTANCE_INFO"
	case EVENT_HEADER_EXT_TYPE_STACK_TRACE32:
		return "STACK_TRACE32"
	case EVENT_HEADER_EXT_TYPE_STACK_TRACE64:
		return "STACK_TRACE64"
	case EVENT_HEADER_EXT_TYPE_PEBS_INDEX:
		return "PEBS_INDEX"
	case EVENT_HEADER_EXT_TYPE_PMC_COUNTERS:
		return "PMC_COUNTERS"
	case EVENT_HEADER_EXT_TYPE_PSM_KEY:
		return "PSM_KEY"
	case EVENT_HEADER_EXT_TYPE_EVENT_KEY:
		return "EVENT_KEY"
	case EVENT_HEADER_EXT_TYPE_EVENT_SCHEMA_TL:
		return "EVENT_SCHEMA_TL"
	case EVENT_HEADER_EXT_TYPE_PROV_TRAITS:
		return "PROV_TRAITS"
	case EVENT_HEADER_EXT_TYPE_PROCESS_START_KEY:
		return "PROCESS_START_KEY"
	case EVENT_HEADER_EXT_TYPE_QPC_DELTA:
		return "QPC_DELTA"
	case EVENT_HEADER_EXT_TYPE_CONTAINER_ID:
		return "CONTAINER_ID"
	default:
		return "(undefined)"
	}
}

// https://learn.microsoft.com/en-us/windows/win32/api/evntrace/ns-evntrace-enable_trace_parameters_v1
const (
	EVENT_ENABLE_PROPERTY_SID               = 0x00000001
	EVENT_ENABLE_PROPERTY_TS_ID             = 0x00000002
	EVENT_ENABLE_PROPERTY_STACK_TRACE       = 0x00000004
	EVENT_ENABLE_PROPERTY_PSM_KEY           = 0x00000008
	EVENT_ENABLE_PROPERTY_IGNORE_KEYWORD_0  = 0x00000010
	EVENT_ENABLE_PROPERTY_PROVIDER_GROUP    = 0x00000020
	EVENT_ENABLE_PROPERTY_ENABLE_KEYWORD_0  = 0x00000040
	EVENT_ENABLE_PROPERTY_PROCESS_START_KEY = 0x00000080
	EVENT_ENABLE_PROPERTY_EVENT_KEY         = 0x00000100
	EVENT_ENABLE_PROPERTY_EXCLUDE_INPRIVATE = 0x00000200
)

func computeStringEnableProperty(strEp []string) uint32 {
	var ep uint32
	for _, prop := range strEp {
		ep |= strToEnableProperty(prop)
	}
	return ep
}

func strToEnableProperty(str string) uint32 {
	switch str {
	case "EVENT_ENABLE_PROPERTY_SID":
		return EVENT_ENABLE_PROPERTY_SID
	case "EVENT_ENABLE_PROPERTY_TS_ID":
		return EVENT_ENABLE_PROPERTY_TS_ID
	case "EVENT_ENABLE_PROPERTY_STACK_TRACE":
		return EVENT_ENABLE_PROPERTY_STACK_TRACE
	case "EVENT_ENABLE_PROPERTY_PSM_KEY":
		return EVENT_ENABLE_PROPERTY_PSM_KEY
	case "EVENT_ENABLE_PROPERTY_IGNORE_KEYWORD_0":
		return EVENT_ENABLE_PROPERTY_IGNORE_KEYWORD_0
	case "EVENT_ENABLE_PROPERTY_PROVIDER_GROUP":
		return EVENT_ENABLE_PROPERTY_PROVIDER_GROUP
	case "EVENT_ENABLE_PROPERTY_ENABLE_KEYWORD_0":
		return EVENT_ENABLE_PROPERTY_ENABLE_KEYWORD_0
	case "EVENT_ENABLE_PROPERTY_PROCESS_START_KEY":
		return EVENT_ENABLE_PROPERTY_PROCESS_START_KEY
	case "EVENT_ENABLE_PROPERTY_EVENT_KEY":
		return EVENT_ENABLE_PROPERTY_EVENT_KEY
	case "EVENT_ENABLE_PROPERTY_EXCLUDE_INPRIVATE":
		return EVENT_ENABLE_PROPERTY_EXCLUDE_INPRIVATE
	default:
		return 0
	}
}

// The EnableProperty field of the ENABLE_TRACE_PARAMETERS

// Wrappers

// https://learn.microsoft.com/en-us/windows/win32/api/evntrace/nf-evntrace-starttracew
func _StartTrace(traceHandle *uintptr,
	instanceName *uint16,
	properties *EventTraceProperties) error {
	r0, _, _ := startTraceW.Call(
		uintptr(unsafe.Pointer(traceHandle)),
		uintptr(unsafe.Pointer(instanceName)),
		uintptr(unsafe.Pointer(properties)))
	if r0 == 0 {
		return nil
	}
	return syscall.Errno(r0)
}

// https://learn.microsoft.com/en-us/windows/win32/api/evntrace/nf-evntrace-enabletraceex2
func _EnableTraceEx2(traceHandle uintptr,
	providerId *windows.GUID,
	isEnabled uint32,
	level uint8,
	matchAnyKeyword uint64,
	matchAllKeyword uint64,
	enableProperty uint32,
	enableParameters *EnableTraceParameters) error {
	r0, _, _ := enableTraceEx2.Call(
		traceHandle,
		uintptr(unsafe.Pointer(providerId)),
		uintptr(isEnabled),
		uintptr(level),
		uintptr(matchAnyKeyword),
		uintptr(matchAllKeyword),
		uintptr(enableProperty),
		uintptr(unsafe.Pointer(enableParameters)))
	if r0 == 0 {
		return nil
	}
	return syscall.Errno(r0)
}

// https://learn.microsoft.com/en-us/windows/win32/api/evntrace/nf-evntrace-controltracew
func _ControlTrace(traceHandle uintptr,
	instanceName *uint16,
	properties *EventTraceProperties,
	controlCode uint32) error {
	r0, _, _ := controlTraceW.Call(
		traceHandle,
		uintptr(unsafe.Pointer(instanceName)),
		uintptr(unsafe.Pointer(properties)),
		uintptr(controlCode))
	if r0 == 0 {
		return nil
	}
	return syscall.Errno(r0)
}

// https://learn.microsoft.com/en-us/windows/win32/api/evntrace/nf-evntrace-opentracew
func _OpenTrace(logfile *EventTraceLogfile) (uint64, error) {
	r0, _, err := openTraceW.Call(
		uintptr(unsafe.Pointer(logfile)))
	var errno syscall.Errno
	if errors.As(err, &errno) && errno == 0 {
		return uint64(r0), nil
	}
	return uint64(r0), err
}

// https://learn.microsoft.com/en-us/windows/win32/api/evntrace/nf-evntrace-processtrace
func _ProcessTrace(handleArray *uint64,
	handleCount uint32,
	startTime *FileTime,
	endTime *FileTime) error {
	r0, _, _ := processTrace.Call(
		uintptr(unsafe.Pointer(handleArray)),
		uintptr(handleCount),
		uintptr(unsafe.Pointer(startTime)),
		uintptr(unsafe.Pointer(endTime)))
	if r0 == 0 {
		return nil
	}
	return syscall.Errno(r0)
}

// https://learn.microsoft.com/en-us/windows/win32/api/evntrace/nf-evntrace-closetrace
func _CloseTrace(traceHandle uint64) error {
	r0, _, _ := closeTrace.Call(
		uintptr(traceHandle))
	if r0 == 0 {
		return nil
	}
	return syscall.Errno(r0)
}
