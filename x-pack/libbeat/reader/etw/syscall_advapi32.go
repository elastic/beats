// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package etw

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var _ unsafe.Pointer

var (
	advapi32 = windows.NewLazySystemDLL("advapi32.dll")
	// Controller
	startTraceW    = advapi32.NewProc("StartTraceW")
	enableTrace    = advapi32.NewProc("EnableTrace")    // Classic providers
	enableTraceEx  = advapi32.NewProc("EnableTraceEx")  // Manifest-based providers
	enableTraceEx2 = advapi32.NewProc("EnableTraceEx2") // Manifest-based providers and filtering
	controlTraceW  = advapi32.NewProc("ControlTraceW")
	// Consumer
	openTraceW   = advapi32.NewProc("OpenTraceW")
	processTrace = advapi32.NewProc("ProcessTrace")
	closeTrace   = advapi32.NewProc("CloseTrace")
)

type EventTrace struct {
	Header           EventTraceHeader
	InstanceId       uint32
	ParentInstanceId uint32
	ParentGuid       GUID
	MofData          uintptr
	MofLength        uint32
	UnionCtx         uint32
}

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

type WnodeHeader struct {
	BufferSize    uint32
	ProviderId    uint32
	Union1        uint64
	Union2        int64
	Guid          GUID
	ClientContext uint32
	Flags         uint32
}

// Used to enable a provider via EnableTraceEx2
type EnableTraceParameters struct {
	Version          uint32
	EnableProperty   uint32
	ControlFlags     uint32
	SourceId         GUID
	EnableFilterDesc *EventFilterDescriptor
	FilterDescrCount uint32
}

// Defines the filter data that a session passes
// to the provider's enable callback function
type EventFilterDescriptor struct {
	Ptr  uint64
	Size uint32
	Type uint32
}

type EventTraceLogfile struct {
	LogFileName    *uint16 // Logfile
	LoggerName     *uint16 // Real-time session
	CurrentTime    int64
	BuffersRead    uint32
	LogFileMode    uint32
	CurrentEvent   EventTrace
	LogfileHeader  TraceLogfileHeader
	BufferCallback uintptr // Optional: receives buffer-related statistics after delivered all the events
	BufferSize     uint32
	Filled         uint32
	EventsLost     uint32
	Callback       uintptr // Receive events (EventRecordCallback (TDH) or EventCallback) Tip: New code should use EventRecordCallback instead of EventCallback. The EventRecordCallback receives an EVENT_RECORD which contains more complete event information, can be used with decoding APIs such as TdhGetEventInformation, and has a context pointer that can be used by your callback. It is used since we set PROCESS_TRACE_MODE_EVENT_RECORD.
	IsKernelTrace  uint32
	Context        uintptr
}

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
	TimeZone           TimeZoneInformation
	BootTime           int64
	PerfFreq           int64
	StartTime          int64
	ReservedFlags      uint32
	BuffersLost        uint32
}

type TimeZoneInformation struct {
	Bias         int32
	StandardName [32]uint16
	StandardDate SystemTime
	StandardBias int32
	DaylightName [32]uint16
	DaylightDate SystemTime
	DaylighBias  int32
}

type FileTime struct {
	dwLowDateTime  uint32
	dwHighDateTime uint32
}

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

const (
	// Information levels
	TRACE_LEVEL_NONE        = 0
	TRACE_LEVEL_CRITICAL    = 1
	TRACE_LEVEL_FATAL       = 1
	TRACE_LEVEL_ERROR       = 2
	TRACE_LEVEL_WARNING     = 3
	TRACE_LEVEL_INFORMATION = 4
	TRACE_LEVEL_VERBOSE     = 5
	TRACE_LEVEL_RESERVED6   = 6
	TRACE_LEVEL_RESERVED7   = 7
	TRACE_LEVEL_RESERVED8   = 8
	TRACE_LEVEL_RESERVED9   = 9
)

const (
	EVENT_CONTROL_CODE_DISABLE_PROVIDER = 0
	EVENT_CONTROL_CODE_ENABLE_PROVIDER  = 1
	EVENT_CONTROL_CODE_CAPTURE_STATE    = 2
)

const (
	EVENT_TRACE_CONTROL_QUERY  = 0
	EVENT_TRACE_CONTROL_STOP   = 1
	EVENT_TRACE_CONTROL_UPDATE = 2
	EVENT_TRACE_CONTROL_FLUSH  = 3
)

const (
	PROCESS_TRACE_MODE_REAL_TIME     = 0x00000100
	PROCESS_TRACE_MODE_RAW_TIMESTAMP = 0x00001000
	PROCESS_TRACE_MODE_EVENT_RECORD  = 0x10000000
)

const INVALID_PROCESSTRACE_HANDLE = 0xFFFFFFFFFFFFFFFF

// https://learn.microsoft.com/en-us/windows/win32/debug/system-error-codes
const (
	ERROR_INVALID_HANDLE         syscall.Errno = 6
	ERROR_INVALID_PARAMETER      syscall.Errno = 87
	ERROR_INSUFFICIENT_BUFFER    syscall.Errno = 122
	ERROR_ALREADY_EXISTS         syscall.Errno = 183
	ERROR_NOT_FOUND              syscall.Errno = 1168
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

type GUID struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

// Function definitions

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

func _EnableTrace(enable uint32,
	enableFlag uint32,
	enableLevel uint32,
	controlGuid *GUID,
	traceHandle uintptr) error {
	r0, _, _ := enableTrace.Call(
		uintptr(enable),
		uintptr(enableFlag),
		uintptr(enableLevel),
		uintptr(unsafe.Pointer(controlGuid)),
		uintptr(traceHandle))
	if r0 == 0 {
		return nil
	}
	return syscall.Errno(r0)
}

func _EnableTraceEx(providerId *GUID,
	sourceId *GUID,
	traceHandle uintptr,
	controlCode uint32,
	level uint8,
	matchAnyKeyword uint64,
	matchAllKeyword uint64,
	timeout uint32,
	enableFilterDesc *EventFilterDescriptor) error {
	r0, _, _ := enableTraceEx.Call(
		uintptr(unsafe.Pointer(providerId)),
		uintptr(unsafe.Pointer(sourceId)),
		uintptr(traceHandle),
		uintptr(controlCode),
		uintptr(level),
		uintptr(matchAnyKeyword),
		uintptr(matchAllKeyword),
		uintptr(timeout),
		uintptr(unsafe.Pointer(enableFilterDesc)))
	if r0 == 0 {
		return nil
	}
	return syscall.Errno(r0)
}

func _EnableTraceEx2(traceHandle uintptr,
	providerId *GUID,
	isEnabled uint32,
	level uint8,
	matchAnyKeyword uint64,
	matchAllKeyword uint64,
	enableProperty uint32,
	enableParameters *EnableTraceParameters) error {
	r0, _, _ := enableTraceEx2.Call(
		uintptr(traceHandle),
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

func _ControlTrace(traceHandle uintptr,
	instanceName *uint16,
	properties *EventTraceProperties,
	controlCode uint32) error {
	r0, _, _ := controlTraceW.Call(
		uintptr(traceHandle),
		uintptr(unsafe.Pointer(instanceName)),
		uintptr(unsafe.Pointer(properties)),
		uintptr(controlCode))
	if r0 == 0 {
		return nil
	}
	return syscall.Errno(r0)
}

func _OpenTrace(logfile *EventTraceLogfile) (uint64, error) {
	r0, _, err := openTraceW.Call(
		uintptr(unsafe.Pointer(logfile)))
	if err.(syscall.Errno) == 0 {
		return uint64(r0), nil
	}
	return uint64(r0), err
}

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

func _CloseTrace(traceHandle uint64) error {
	r0, _, _ := closeTrace.Call(
		uintptr(traceHandle))
	if r0 == 0 {
		return nil
	}
	return syscall.Errno(r0)
}
