// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build linux

package perf

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

// Special pid values for Open.
const (
	// CallingThread configures the event to measure the calling thread.
	CallingThread = 0

	// AllThreads configures the event to measure all threads on the
	// specified CPU.
	AllThreads = -1
)

// AnyCPU configures the specified process/thread to be measured on any CPU.
const AnyCPU = -1

// Event states.
const (
	eventStateUninitialized = 0
	eventStateOK            = 1
	eventStateClosed        = 2
)

// Event is an active perf event.
type Event struct {
	// state is the state of the event. See eventState* constants.
	state int32

	// perffd is the perf event file descriptor.
	perffd int

	// id is the unique event ID.
	id uint64

	// group contains other events in the event group, if this event is
	// an event group leader. The order is the order in which the events
	// were added to the group.
	group []*Event

	// groupByID maps group event IDs to the events themselves. The
	// reason why this mapping is needed is explained in ReadRecord.
	groupByID map[uint64]*Event

	// owned contains other events in the event group, which the caller
	// has no access to. The Event owns them all, Close closes them all.
	owned []*Event

	// a is the set of attributes the Event was configured with. It is
	// a clone of the original, save for the Label field, which may have
	// been set, if the original *Attr didn't set it.
	a *Attr

	// noReadRecord is true if ReadRecord is disabled for the event.
	// See SetOutput and ReadRecord.
	noReadRecord bool

	// ring is the (entire) memory mapped ring buffer.
	ring []byte

	// ringdata is the data region of the ring buffer.
	ringdata []byte

	// meta is the metadata page: &ring[0].
	meta *unix.PerfEventMmapPage

	// wakeupfd is an event file descriptor (see eventfd(2)). It is used to
	// unblock calls to ReadRawRecord when the associated context expires.
	wakeupfd int

	// pollreq communicates requests from ReadRawRecord to the poll goroutine
	// associated with the ring.
	pollreq chan pollreq

	// pollresp receives responses from the poll goroutine associated
	// with the ring, back to ReadRawRecord.
	pollresp chan pollresp

	// recordBuffer is used as storage for records returned by ReadRecord
	// and ReadRawRecord. This means memory for records returned from those
	// methods will be overwritten by successive calls.
	recordBuffer []byte
}

// Open opens the event configured by attr.
//
// The pid and cpu parameters specify which thread and CPU to monitor:
//
//     * if pid == CallingThread and cpu == AnyCPU, the event measures
//       the calling thread on any CPU
//
//     * if pid == CallingThread and cpu >= 0, the event measures
//       the calling thread only when running on the specified CPU
//
//     * if pid > 0 and cpu == AnyCPU, the event measures the specified
//       thread on any CPU
//
//     * if pid > 0 and cpu >= 0, the event measures the specified thread
//       only when running on the specified CPU
//
//     * if pid == AllThreads and cpu >= 0, the event measures all threads
//       on the specified CPU
//
//     * finally, the pid == AllThreads and cpu == AnyCPU setting is invalid
//
// If group is non-nil, the returned Event is made part of the group
// associated with the specified group Event.
func Open(a *Attr, pid, cpu int, group *Event) (*Event, error) {
	return open(a, pid, cpu, group, 0)
}

// OpenWithFlags is like Open but allows to specify additional flags to be
// passed to perf_event_open(2).
func OpenWithFlags(a *Attr, pid, cpu int, group *Event, flags int) (*Event, error) {
	return open(a, pid, cpu, group, flags)
}

// OpenCGroup is like Open, but activates per-container system-wide
// monitoring. If cgroupfs is mounted on /dev/cgroup, and the group to
// monitor is called "test", then cgroupfd must be a file descriptor opened
// on /dev/cgroup/test.
func OpenCGroup(a *Attr, cgroupfd, cpu int, group *Event) (*Event, error) {
	return open(a, cgroupfd, cpu, group, unix.PERF_FLAG_PID_CGROUP)
}

func open(a *Attr, pid, cpu int, group *Event, flags int) (*Event, error) {
	groupfd := -1
	if group != nil {
		if err := group.ok(); err != nil {
			return nil, err
		}
		groupfd = group.perffd
	}

	fd, err := perfEventOpen(a, pid, cpu, groupfd, flags)
	if err != nil {
		return nil, os.NewSyscallError("perf_event_open", err)
	}
	if err := unix.SetNonblock(fd, true); err != nil {
		unix.Close(fd)
		return nil, os.NewSyscallError("setnonblock", err)
	}

	// Clone the *Attr so the caller can't change it from under our feet.

	ac := new(Attr)
	*ac = *a // ok to copy since no slices
	if ac.Label == "" {
		evID := eventID{
			Type:   uint64(a.Type),
			Config: uint64(a.Config),
		}
		ac.Label = lookupLabel(evID).Name
	}

	ev := &Event{
		state:  eventStateOK,
		perffd: fd,
		a:      ac,
	}
	id, err := ev.ID()
	if err != nil {
		return nil, err
	}
	ev.id = id
	if group != nil {
		if group.groupByID == nil {
			group.groupByID = map[uint64]*Event{}
		}
		group.group = append(group.group, ev)
		group.groupByID[id] = ev
	}

	return ev, nil
}

// perfEventOpen wraps the perf_event_open system call with some additional
// logic around ensuring that file descriptors are marked close-on-exec.
func perfEventOpen(a *Attr, pid, cpu, groupfd, flags int) (fd int, err error) {
	sysAttr := a.sysAttr()
	cloexecFlags := flags | unix.PERF_FLAG_FD_CLOEXEC

	fd, err = unix.PerfEventOpen(sysAttr, pid, cpu, groupfd, cloexecFlags)
	switch err {
	case nil:
		return fd, nil
	case unix.EINVAL:
		// PERF_FLAG_FD_CLOEXEC is only available in Linux 3.14
		// and up, or in older kernels patched by distributions
		// with backported perf updates. If we got EINVAL, try again
		// without the flag, while holding syscall.ForkLock, following
		// the standard library pattern in net/sock_cloexec.go.
		syscall.ForkLock.RLock()
		defer syscall.ForkLock.RUnlock()

		fd, err = unix.PerfEventOpen(sysAttr, pid, cpu, groupfd, flags)
		if err == nil {
			unix.CloseOnExec(fd)
		}
		return fd, err
	default:
		return -1, err
	}
}

// DefaultNumPages is the number of pages used by MapRing. There is no
// fundamental logic to this number. We use it because that is what the perf
// tool does.
const DefaultNumPages = 128

// MapRing maps the ring buffer attached to the event into memory.
//
// This enables reading records via ReadRecord / ReadRawRecord.
func (ev *Event) MapRing() error {
	return ev.MapRingNumPages(DefaultNumPages)
}

// MapRingNumPages is like MapRing, but allows the caller to The size of
// the data portion of the ring is num pages. The total size of the ring
// is num+1 pages, because an additional metadata page is mapped before the
// data portion of the ring.
func (ev *Event) MapRingNumPages(num int) error {
	if err := ev.ok(); err != nil {
		return err
	}
	if ev.ring != nil {
		return nil
	}

	pgSize := unix.Getpagesize()
	size := (1 + num) * pgSize
	const prot = unix.PROT_READ | unix.PROT_WRITE
	const flags = unix.MAP_SHARED
	ring, err := unix.Mmap(ev.perffd, 0, size, prot, flags)
	if err != nil {
		return os.NewSyscallError("mmap", err)
	}

	meta := (*unix.PerfEventMmapPage)(unsafe.Pointer(&ring[0]))

	// Some systems do not fill in the data_offset and data_size fields
	// of the metadata page correctly: Centos 6.9 and Debian 8 have been
	// observed to do this. Try to detect this condition, and adjust
	// the values accordingly.
	if meta.Data_offset == 0 && meta.Data_size == 0 {
		atomic.StoreUint64(&meta.Data_offset, uint64(pgSize))
		atomic.StoreUint64(&meta.Data_size, uint64(num*pgSize))
	}

	ringdata := ring[meta.Data_offset:]

	wakeupfd, err := unix.Eventfd(0, unix.EFD_CLOEXEC|unix.EFD_NONBLOCK)
	if err != nil {
		return os.NewSyscallError("eventfd", err)
	}

	ev.ring = ring
	ev.meta = meta
	ev.ringdata = ringdata
	ev.wakeupfd = wakeupfd
	ev.pollreq = make(chan pollreq)
	ev.pollresp = make(chan pollresp)

	go ev.poll()

	return nil
}

func (ev *Event) ok() error {
	if ev == nil {
		return os.ErrInvalid
	}

	switch ev.state {
	case eventStateUninitialized:
		return os.ErrInvalid
	case eventStateOK:
		return nil
	default: // eventStateClosed
		return os.ErrClosed
	}
}

// FD returns the file descriptor associated with the event.
func (ev *Event) FD() (int, error) {
	if err := ev.ok(); err != nil {
		return -1, err
	}
	return ev.perffd, nil
}

// Measure disables the event, resets it, enables it, runs f, disables it again,
// then reads the Count associated with the event.
func (ev *Event) Measure(f func()) (Count, error) {
	if err := ev.Disable(); err != nil {
		return Count{}, err
	}
	if err := ev.Reset(); err != nil {
		return Count{}, err
	}
	if err := ev.Enable(); err != nil {
		return Count{}, err
	}

	f()

	if err := ev.Disable(); err != nil {
		return Count{}, err
	}
	return ev.ReadCount()
}

// MeasureGroup is like Measure, but for event groups.
func (ev *Event) MeasureGroup(f func()) (GroupCount, error) {
	if err := ev.Disable(); err != nil {
		return GroupCount{}, err
	}
	if err := ev.Reset(); err != nil {
		return GroupCount{}, err
	}
	if err := ev.Enable(); err != nil {
		return GroupCount{}, err
	}

	f()

	if err := ev.Disable(); err != nil {
		return GroupCount{}, err
	}
	return ev.ReadGroupCount()
}

// Enable enables the event.
func (ev *Event) Enable() error {
	if err := ev.ok(); err != nil {
		return err
	}
	err := ev.ioctlNoArg(unix.PERF_EVENT_IOC_ENABLE)
	return wrapIoctlError("PERF_EVENT_IOC_ENABLE", err)
}

// Disable disables the event. If ev is a group leader, Disable disables
// all events in the group.
func (ev *Event) Disable() error {
	if err := ev.ok(); err != nil {
		return err
	}
	err := ev.ioctlInt(unix.PERF_EVENT_IOC_DISABLE, 0)
	return wrapIoctlError("PERF_EVENT_IOC_DISABLE", err)
}

// TODO(acln): add support for PERF_IOC_FLAG_GROUP and for event followers
// to disable the entire group?

// Refresh adds delta to a counter associated with the event. This counter
// decrements every time the event overflows. Once the counter reaches zero,
// the event is disabled. Calling Refresh with delta == 0 is considered
// undefined behavior.
func (ev *Event) Refresh(delta int) error {
	if err := ev.ok(); err != nil {
		return err
	}
	err := ev.ioctlInt(unix.PERF_EVENT_IOC_REFRESH, uintptr(delta))
	return wrapIoctlError("PERF_EVENT_IOC_REFRESH", err)
}

// Reset resets the counters associated with the event.
func (ev *Event) Reset() error {
	if err := ev.ok(); err != nil {
		return err
	}
	err := ev.ioctlNoArg(unix.PERF_EVENT_IOC_RESET)
	return wrapIoctlError("PERF_EVENT_IOC_RESET", err)
}

// UpdatePeriod updates the overflow period for the event. On older kernels,
// the new period does not take effect until after the next overflow.
func (ev *Event) UpdatePeriod(p uint64) error {
	if err := ev.ok(); err != nil {
		return err
	}
	err := ev.ioctlPointer(unix.PERF_EVENT_IOC_PERIOD, unsafe.Pointer(&p))
	return wrapIoctlError("PERF_EVENT_IOC_PERIOD", err)
}

// SetOutput tells the kernel to send records to the specified
// target Event rather than ev.
//
// If target is nil, output from ev is ignored.
//
// Some restrictions apply:
//
// 1) Calling SetOutput on an *Event will fail with EINVAL if MapRing was
// called on that event previously. 2) If ev and target are not CPU-wide
// events, they must be on the same CPU. 3) If ev and target are CPU-wide
// events, they must refer to the same task. 4) ev and target must use the
// same clock.
//
// An additional restriction of the Go API also applies:
//
// In order to use ReadRecord on the target Event, the following settings on
// ev and target must match: Options.SampleIDAll, SampleFormat.Identifier,
// SampleFormat.IP, SampleFormat.Tid, SampleFormat.Time, SampleFormat.Addr,
// SampleFormat.ID, SampleFormat.StreamID. Furthermore, SampleFormat.StreamID
// must be set. SetOutput nevertheless succeeds even if this condition is
// not met, because callers can still use ReadRawRecord instead of ReadRecord.
func (ev *Event) SetOutput(target *Event) error {
	if err := ev.ok(); err != nil {
		return err
	}
	var targetfd int
	if target == nil {
		targetfd = -1
	} else {
		if err := target.ok(); err != nil {
			return err
		}
		if !target.canReadRecordFrom(ev) {
			target.noReadRecord = true
		}
		targetfd = target.perffd
	}
	err := ev.ioctlInt(unix.PERF_EVENT_IOC_SET_OUTPUT, uintptr(targetfd))
	return wrapIoctlError("PERF_EVENT_IOC_SET_OUTPUT", err)
}

// canReadRecordFrom returns a boolean indicating whether ev, as a leader,
// can read records produced by f, a follower.
func (ev *Event) canReadRecordFrom(f *Event) bool {
	lf := ev.a.SampleFormat
	ff := f.a.SampleFormat

	return lf.Identifier == ff.Identifier &&
		lf.IP == ff.IP &&
		lf.Tid == ff.Tid &&
		lf.Time == ff.Time &&
		lf.Addr == ff.Addr &&
		lf.ID == ff.ID &&
		lf.StreamID == ff.StreamID &&
		ff.StreamID
}

// BUG(acln): PERF_EVENT_IOC_SET_FILTER is not implemented

// ID returns the unique event ID value for ev.
func (ev *Event) ID() (uint64, error) {
	if err := ev.ok(); err != nil {
		return 0, err
	}
	var val uint64
	err := ev.ioctlPointer(unix.PERF_EVENT_IOC_ID, unsafe.Pointer(&val))
	return val, wrapIoctlError("PERF_EVENT_IOC_ID", err)
}

// SetBPF attaches a BPF program to ev, which must be a kprobe tracepoint
// event. progfd is the file descriptor associated with the BPF program.
func (ev *Event) SetBPF(progfd uint32) error {
	if err := ev.ok(); err != nil {
		return err
	}
	err := ev.ioctlInt(unix.PERF_EVENT_IOC_SET_BPF, uintptr(progfd))
	return wrapIoctlError("PERF_EVENT_IOC_SET_BPF", err)
}

// PauseOutput pauses the output from ev.
func (ev *Event) PauseOutput() error {
	if err := ev.ok(); err != nil {
		return err
	}
	err := ev.ioctlInt(unix.PERF_EVENT_IOC_PAUSE_OUTPUT, 1)
	return wrapIoctlError("PEF_EVENT_IOC_PAUSE_OUTPUT", err)
}

// ResumeOutput resumes output from ev.
func (ev *Event) ResumeOutput() error {
	if err := ev.ok(); err != nil {
		return err
	}
	err := ev.ioctlInt(unix.PERF_EVENT_IOC_PAUSE_OUTPUT, 0)
	return wrapIoctlError("PEF_EVENT_IOC_PAUSE_OUTPUT", err)
}

// QueryBPF queries the event for BPF program file descriptors attached to
// the same tracepoint as ev. max is the maximum number of file descriptors
// to return.
func (ev *Event) QueryBPF(max uint32) ([]uint32, error) {
	if err := ev.ok(); err != nil {
		return nil, err
	}
	buf := make([]uint32, 2+max)
	buf[0] = max
	err := ev.ioctlPointer(unix.PERF_EVENT_IOC_QUERY_BPF, unsafe.Pointer(&buf[0]))
	if err != nil {
		return nil, wrapIoctlError("PERF_EVENT_IOC_QUERY_BPF", err)
	}
	count := buf[1]
	fds := make([]uint32, count)
	copy(fds, buf[2:2+count])
	return fds, nil
}

// BUG(acln): PERF_EVENT_IOC_MODIFY_ATTRIBUTES is not implemented

func (ev *Event) ioctlNoArg(number int) error {
	return ev.ioctlInt(number, 0)
}

func (ev *Event) ioctlInt(number int, arg uintptr) error {
	_, _, e := unix.Syscall(unix.SYS_IOCTL, uintptr(ev.perffd), uintptr(number), arg)
	if e != 0 {
		return e
	}
	return nil
}

func (ev *Event) ioctlPointer(number uintptr, arg unsafe.Pointer) error {
	_, _, e := unix.Syscall(unix.SYS_IOCTL, uintptr(ev.perffd), number, uintptr(arg))
	if e != 0 {
		return e
	}
	return nil
}

func wrapIoctlError(ioctl string, err error) error {
	if err == nil {
		return nil
	}
	return &ioctlError{ioctl: ioctl, err: err}
}

type ioctlError struct {
	ioctl string
	err   error
}

func (e *ioctlError) Error() string {
	return fmt.Sprintf("%s: %v", e.ioctl, e.err)
}

func (e *ioctlError) Unwrap() error { return e.err }

// Close closes the event. Close must not be called concurrently with any
// other methods on the Event.
func (ev *Event) Close() error {
	if ev.ring != nil {
		close(ev.pollreq)
		<-ev.pollresp
		unix.Munmap(ev.ring)
		unix.Close(ev.wakeupfd)
	}

	for _, ev := range ev.owned {
		ev.Close()
	}

	ev.state = eventStateClosed
	return unix.Close(ev.perffd)
}

// Attr configures a perf event.
type Attr struct {
	// Label is a human readable label associated with the event.
	// For convenience, the Label is included in Count and GroupCount
	// measurements read from events.
	//
	// When an event is opened, if Label is the empty string, then a
	// Label is computed (if possible) based on the Type and Config
	// fields. Otherwise, if the Label user-defined (not the empty
	// string), it is included verbatim.
	//
	// For most events, the computed Label matches the label specified by
	// ``perf list'' for the same event (but see Bugs).
	Label string

	// Type is the major type of the event.
	Type EventType

	// Config is the type-specific event configuration.
	Config uint64

	// Sample configures the sample period or sample frequency for
	// overflow packets, based on Options.Freq: if Options.Freq is set,
	// Sample is interpreted as "sample frequency", otherwise it is
	// interpreted as "sample period".
	//
	// See also SetSample{Period,Freq}.
	Sample uint64

	// SampleFormat configures information requested in sample records,
	// on the memory mapped ring buffer.
	SampleFormat SampleFormat

	// CountFormat specifies the format of counts read from the
	// Event using ReadCount or ReadGroupCount. See the CountFormat
	// documentation for more details.
	CountFormat CountFormat

	// Options contains more fine grained event configuration.
	Options Options

	// Wakeup configures wakeups on the ring buffer associated with the
	// event. If Options.Watermark is set, Wakeup is interpreted as the
	// number of bytes before wakeup. Otherwise, it is interpreted as
	// "wake up every N events".
	//
	// See also SetWakeup{Events,Watermark}.
	Wakeup uint32

	// BreakpointType is the breakpoint type, if Type == BreakpointEvent.
	BreakpointType uint32

	// Config1 is used for events that need an extra register or otherwise
	// do not fit in the regular config field.
	//
	// For breakpoint events, Config1 is the breakpoint address.
	// For kprobes, it is the kprobe function. For uprobes, it is the
	// uprobe path.
	Config1 uint64

	// Config2 is a further extension of the Config1 field.
	//
	// For breakpoint events, it is the length of the breakpoint.
	// For kprobes, when the kprobe function is NULL, it is the address of
	// the kprobe. For both kprobes and uprobes, it is the probe offset.
	Config2 uint64

	// BranchSampleFormat specifies what branches to include in the
	// branch record, if SampleFormat.BranchStack is set.
	BranchSampleFormat BranchSampleFormat

	// SampleRegistersUser is the set of user registers to dump on samples.
	SampleRegistersUser uint64

	// SampleStackUser is the size of the user stack to  dump on samples.
	SampleStackUser uint32

	// ClockID is the clock ID to use with samples, if Options.UseClockID
	// is set.
	//
	// TODO(acln): What are the values for this? CLOCK_MONOTONIC and such?
	// Investigate. Can we choose a clock that can be compared to Go's
	// clock in a meaningful way? If so, should we add special support
	// for that?
	ClockID int32

	// SampleRegistersIntr is the set of register to dump for each sample.
	// See asm/perf_regs.h for details.
	SampleRegistersIntr uint64

	// AuxWatermark is the watermark for the aux area.
	AuxWatermark uint32

	// SampleMaxStack is the maximum number of frame pointers in a
	// callchain. The value must be < MaxStack().
	SampleMaxStack uint16
}

func (a Attr) sysAttr() *unix.PerfEventAttr {
	return &unix.PerfEventAttr{
		Type:               uint32(a.Type),
		Size:               uint32(unsafe.Sizeof(unix.PerfEventAttr{})),
		Config:             a.Config,
		Sample:             a.Sample,
		Sample_type:        a.SampleFormat.marshal(),
		Read_format:        a.CountFormat.marshal(),
		Bits:               a.Options.marshal(),
		Wakeup:             a.Wakeup,
		Bp_type:            a.BreakpointType,
		Ext1:               a.Config1,
		Ext2:               a.Config2,
		Branch_sample_type: a.BranchSampleFormat.marshal(),
		Sample_regs_user:   a.SampleRegistersUser,
		Sample_stack_user:  a.SampleStackUser,
		Clockid:            a.ClockID,
		Sample_regs_intr:   a.SampleRegistersIntr,
		Aux_watermark:      a.AuxWatermark,
		Sample_max_stack:   a.SampleMaxStack,
	}
}

// Configure implements the Configurator interface. It overwrites target
// with a. See also (*Group).Add.
func (a *Attr) Configure(target *Attr) error {
	*target = *a
	return nil
}

// SetSamplePeriod configures the sampling period for the event.
//
// It sets attr.Sample to p and disables a.Options.Freq.
func (a *Attr) SetSamplePeriod(p uint64) {
	a.Sample = p
	a.Options.Freq = false
}

// SetSampleFreq configures the sampling frequency for the event.
//
// It sets attr.Sample to f and enables a.Options.Freq.
func (a *Attr) SetSampleFreq(f uint64) {
	a.Sample = f
	a.Options.Freq = true
}

// SetWakeupEvents configures the event to wake up every n events.
//
// It sets a.Wakeup to n and disables a.Options.Watermark.
func (a *Attr) SetWakeupEvents(n uint32) {
	a.Wakeup = n
	a.Options.Watermark = false
}

// SetWakeupWatermark configures the number of bytes in overflow records
// before wakeup.
//
// It sets a.Wakeup to n and enables a.Options.Watermark.
func (a *Attr) SetWakeupWatermark(n uint32) {
	a.Wakeup = n
	a.Options.Watermark = true
}

// LookupEventType probes /sys/bus/event_source/devices/<device>/type
// for the EventType value associated with the specified PMU.
func LookupEventType(pmu string) (EventType, error) {
	path := filepath.Join("/sys/bus/event_source/devices", pmu, "type")
	et, err := readUint(path, 32)
	return EventType(et), err
}

// EventType is the overall type of a performance event.
type EventType uint32

// Supported event types.
const (
	HardwareEvent      EventType = unix.PERF_TYPE_HARDWARE
	SoftwareEvent      EventType = unix.PERF_TYPE_SOFTWARE
	TracepointEvent    EventType = unix.PERF_TYPE_TRACEPOINT
	HardwareCacheEvent EventType = unix.PERF_TYPE_HW_CACHE
	RawEvent           EventType = unix.PERF_TYPE_RAW
	BreakpointEvent    EventType = unix.PERF_TYPE_BREAKPOINT
)

// HardwareCounter is a hardware performance counter.
type HardwareCounter uint64

// Hardware performance counters.
const (
	CPUCycles             HardwareCounter = unix.PERF_COUNT_HW_CPU_CYCLES
	Instructions          HardwareCounter = unix.PERF_COUNT_HW_INSTRUCTIONS
	CacheReferences       HardwareCounter = unix.PERF_COUNT_HW_CACHE_REFERENCES
	CacheMisses           HardwareCounter = unix.PERF_COUNT_HW_CACHE_MISSES
	BranchInstructions    HardwareCounter = unix.PERF_COUNT_HW_BRANCH_INSTRUCTIONS
	BranchMisses          HardwareCounter = unix.PERF_COUNT_HW_BRANCH_MISSES
	BusCycles             HardwareCounter = unix.PERF_COUNT_HW_BUS_CYCLES
	StalledCyclesFrontend HardwareCounter = unix.PERF_COUNT_HW_STALLED_CYCLES_FRONTEND
	StalledCyclesBackend  HardwareCounter = unix.PERF_COUNT_HW_STALLED_CYCLES_BACKEND
	RefCPUCycles          HardwareCounter = unix.PERF_COUNT_HW_REF_CPU_CYCLES
)

var hardwareLabels = map[HardwareCounter]eventLabel{
	CPUCycles:             {Name: "cpu-cycles", Alias: "cycles"},
	Instructions:          {Name: "instructions"},
	CacheReferences:       {Name: "cache-references"},
	CacheMisses:           {Name: "cache-misses"},
	BranchInstructions:    {Name: "branch-instructions", Alias: "branches"},
	BranchMisses:          {Name: "branch-misses", Alias: "branch-misses"},
	BusCycles:             {Name: "bus-cycles"},
	StalledCyclesFrontend: {Name: "stalled-cycles-frontend", Alias: "idle-cycles-frontend"},
	StalledCyclesBackend:  {Name: "stalled-cycles-backend", Alias: "idle-cycles-backend"},
	RefCPUCycles:          {Name: "ref-cycles"},
}

func (hwc HardwareCounter) String() string {
	return hwc.eventLabel().Name
}

func (hwc HardwareCounter) eventLabel() eventLabel {
	return hardwareLabels[hwc]
}

// Configure configures attr to measure hwc. It sets the Label, Type, and
// Config fields on attr.
func (hwc HardwareCounter) Configure(attr *Attr) error {
	attr.Label = hwc.String()
	attr.Type = HardwareEvent
	attr.Config = uint64(hwc)
	return nil
}

// AllHardwareCounters returns a slice of all known hardware counters.
func AllHardwareCounters() []Configurator {
	return []Configurator{
		CPUCycles,
		Instructions,
		CacheReferences,
		CacheMisses,
		BranchInstructions,
		BranchMisses,
		BusCycles,
		StalledCyclesFrontend,
		StalledCyclesBackend,
		RefCPUCycles,
	}
}

// SoftwareCounter is a software performance counter.
type SoftwareCounter uint64

// Software performance counters.
const (
	CPUClock        SoftwareCounter = unix.PERF_COUNT_SW_CPU_CLOCK
	TaskClock       SoftwareCounter = unix.PERF_COUNT_SW_TASK_CLOCK
	PageFaults      SoftwareCounter = unix.PERF_COUNT_SW_PAGE_FAULTS
	ContextSwitches SoftwareCounter = unix.PERF_COUNT_SW_CONTEXT_SWITCHES
	CPUMigrations   SoftwareCounter = unix.PERF_COUNT_SW_CPU_MIGRATIONS
	MinorPageFaults SoftwareCounter = unix.PERF_COUNT_SW_PAGE_FAULTS_MIN
	MajorPageFaults SoftwareCounter = unix.PERF_COUNT_SW_PAGE_FAULTS_MAJ
	AlignmentFaults SoftwareCounter = unix.PERF_COUNT_SW_ALIGNMENT_FAULTS
	EmulationFaults SoftwareCounter = unix.PERF_COUNT_SW_EMULATION_FAULTS
	Dummy           SoftwareCounter = unix.PERF_COUNT_SW_DUMMY
	BPFOutput       SoftwareCounter = unix.PERF_COUNT_SW_BPF_OUTPUT
)

var softwareLabels = map[SoftwareCounter]eventLabel{
	CPUClock:        {Name: "cpu-clock"},
	TaskClock:       {Name: "task-clock"},
	PageFaults:      {Name: "page-faults", Alias: "faults"},
	ContextSwitches: {Name: "context-switches", Alias: "cs"},
	CPUMigrations:   {Name: "cpu-migrations", Alias: "migrations"},
	MinorPageFaults: {Name: "minor-faults"},
	MajorPageFaults: {Name: "major-faults"},
	AlignmentFaults: {Name: "alignment-faults"},
	EmulationFaults: {Name: "emulation-faults"},
	Dummy:           {Name: "dummy"},
	BPFOutput:       {Name: "bpf-output"},
}

func (swc SoftwareCounter) String() string {
	return swc.eventLabel().Name
}

func (swc SoftwareCounter) eventLabel() eventLabel {
	return softwareLabels[swc]
}

// Configure configures attr to measure swc. It sets attr.Type and attr.Config.
func (swc SoftwareCounter) Configure(attr *Attr) error {
	attr.Label = swc.eventLabel().Name
	attr.Type = SoftwareEvent
	attr.Config = uint64(swc)
	return nil
}

// AllSoftwareCounters returns a slice of all known software counters.
func AllSoftwareCounters() []Configurator {
	return []Configurator{
		CPUClock,
		TaskClock,
		PageFaults,
		ContextSwitches,
		CPUMigrations,
		MinorPageFaults,
		MajorPageFaults,
		AlignmentFaults,
		EmulationFaults,
		Dummy,
		BPFOutput,
	}
}

// Cache identifies a cache.
type Cache uint64

// Caches.
const (
	L1D  Cache = unix.PERF_COUNT_HW_CACHE_L1D
	L1I  Cache = unix.PERF_COUNT_HW_CACHE_L1I
	LL   Cache = unix.PERF_COUNT_HW_CACHE_LL
	DTLB Cache = unix.PERF_COUNT_HW_CACHE_DTLB
	ITLB Cache = unix.PERF_COUNT_HW_CACHE_ITLB
	BPU  Cache = unix.PERF_COUNT_HW_CACHE_BPU
	NODE Cache = unix.PERF_COUNT_HW_CACHE_NODE
)

// AllCaches returns a slice of all known cache types.
func AllCaches() []Cache {
	return []Cache{L1D, L1I, LL, DTLB, ITLB, BPU, NODE}
}

// CacheOp is a cache operation.
type CacheOp uint64

// Cache operations.
const (
	Read     CacheOp = unix.PERF_COUNT_HW_CACHE_OP_READ
	Write    CacheOp = unix.PERF_COUNT_HW_CACHE_OP_WRITE
	Prefetch CacheOp = unix.PERF_COUNT_HW_CACHE_OP_PREFETCH
)

// AllCacheOps returns a slice of all known cache operations.
func AllCacheOps() []CacheOp {
	return []CacheOp{Read, Write, Prefetch}
}

// CacheOpResult is the result of a cache operation.
type CacheOpResult uint64

// Cache operation results.
const (
	Access CacheOpResult = unix.PERF_COUNT_HW_CACHE_RESULT_ACCESS
	Miss   CacheOpResult = unix.PERF_COUNT_HW_CACHE_RESULT_MISS
)

// AllCacheOpResults returns a slice of all known cache operation results.
func AllCacheOpResults() []CacheOpResult {
	return []CacheOpResult{Access, Miss}
}

// A HardwareCacheCounter groups a cache, a cache operation, and an operation
// result. It measures the number of results for the specified op, on the
// specified cache.
type HardwareCacheCounter struct {
	Cache  Cache
	Op     CacheOp
	Result CacheOpResult
}

// Configure configures attr to measure hwcc. It sets attr.Type and attr.Config.
func (hwcc HardwareCacheCounter) Configure(attr *Attr) error {
	attr.Type = HardwareCacheEvent
	attr.Config = uint64(hwcc.Cache) | uint64(hwcc.Op<<8) | uint64(hwcc.Result<<16)
	return nil
}

// HardwareCacheCounters returns cache counters which measure the cartesian
// product of the specified caches, operations and results.
func HardwareCacheCounters(caches []Cache, ops []CacheOp, results []CacheOpResult) []Configurator {
	counters := make([]Configurator, 0, len(caches)*len(ops)*len(results))
	for _, cache := range caches {
		for _, op := range ops {
			for _, result := range results {
				c := HardwareCacheCounter{
					Cache:  cache,
					Op:     op,
					Result: result,
				}
				counters = append(counters, c)
			}
		}
	}
	return counters
}

// Tracepoint returns a Configurator for the specified category and event.
// The returned Configurator sets attr.Type and attr.Config.
func Tracepoint(category, event string) Configurator {
	return configuratorFunc(func(attr *Attr) error {
		cfg, err := LookupTracepointConfig(category, event)
		if err != nil {
			return err
		}

		attr.Label = fmt.Sprintf("%s:%s", category, event)
		attr.Type = TracepointEvent
		attr.Config = cfg

		return nil
	})
}

// LookupTracepointConfig probes
// /sys/kernel/debug/tracing/events/<category>/<event>/id for the Attr.Config
// value associated with the specified category and event.
func LookupTracepointConfig(category, event string) (uint64, error) {
	p := filepath.Join("/sys/kernel/debug/tracing/events", category, event, "id")
	return readUint(p, 64)
}

// Breakpoint returns a Configurator for a breakpoint event.
//
// typ is the type of the breakpoint.
//
// addr is the address of the breakpoint. For execution breakpoints, this
// is the memory address of the instruction of interest; for read and write
// breakpoints, it is the memory address of the memory location of interest.
//
// length is the length of the breakpoint being measured.
//
// The returned Configurator sets the Type, BreakpointType, Config1, and
// Config2 fields on attr.
func Breakpoint(typ BreakpointType, addr uint64, length BreakpointLength) Configurator {
	return configuratorFunc(func(attr *Attr) error {
		attr.Type = BreakpointEvent
		attr.BreakpointType = uint32(typ)
		attr.Config1 = addr
		attr.Config2 = uint64(length)

		return nil
	})
}

// BreakpointType is the type of a breakpoint.
type BreakpointType uint32

// Breakpoint types. Values are |-ed together. The combination of
// BreakpointTypeR or BreakpointTypeW with BreakpointTypeX is invalid.
const (
	BreakpointTypeEmpty BreakpointType = 0x0
	BreakpointTypeR     BreakpointType = 0x1
	BreakpointTypeW     BreakpointType = 0x2
	BreakpointTypeRW    BreakpointType = BreakpointTypeR | BreakpointTypeW
	BreakpointTypeX     BreakpointType = 0x4
)

// BreakpointLength is the length of the breakpoint being measured.
type BreakpointLength uint64

// Breakpoint length values.
const (
	BreakpointLength1 BreakpointLength = 1
	BreakpointLength2 BreakpointLength = 2
	BreakpointLength4 BreakpointLength = 4
	BreakpointLength8 BreakpointLength = 8
)

// ExecutionBreakpointLength returns the length of an execution breakpoint.
func ExecutionBreakpointLength() BreakpointLength {
	// TODO(acln): is this correct? The man page says to set this to
	// sizeof(long). Is sizeof(C long) == sizeof(Go uintptr) on all
	// platforms of interest?
	var x uintptr
	return BreakpointLength(unsafe.Sizeof(x))
}

// ExecutionBreakpoint returns a Configurator for an execution breakpoint
// at the specified address.
func ExecutionBreakpoint(addr uint64) Configurator {
	return Breakpoint(BreakpointTypeX, addr, ExecutionBreakpointLength())
}

// Options contains low level event configuration options.
type Options struct {
	// Disabled disables the event by default. If the event is in a
	// group, but not a group leader, this option has no effect, since
	// the group leader controls when events are enabled or disabled.
	Disabled bool

	// Inherit specifies that this counter should count events of child
	// tasks as well as the specified task. This only applies to new
	// children, not to any existing children at the time the counter
	// is created (nor to any new children of existing children).
	//
	// Inherit does not work with some combination of CountFormat options,
	// such as CountFormat.Group.
	Inherit bool

	// Pinned specifies that the counter should always be on the CPU if
	// possible. This bit applies only to hardware counters, and only
	// to group leaders. If a pinned counter canno be put onto the CPU,
	// then the counter goes into an error state, where reads return EOF,
	// until it is subsequently enabled or disabled.
	Pinned bool

	// Exclusive specifies that when this counter's group is on the CPU,
	// it should be the only group using the CPUs counters.
	Exclusive bool

	// ExcludeUser excludes events that happen in user space.
	ExcludeUser bool

	// ExcludeKernel excludes events that happen in kernel space.
	ExcludeKernel bool

	// ExcludeHypervisor excludes events that happen in the hypervisor.
	ExcludeHypervisor bool

	// ExcludeIdle disables counting while the CPU is idle.
	ExcludeIdle bool

	// The mmap bit enables generation of MmapRecord records for every
	// mmap(2) call that has PROT_EXEC set.
	Mmap bool

	// Comm enables tracking of process command name, as modified by
	// exec(2), prctl(PR_SET_NAME), as well as writing to /proc/self/comm.
	// If CommExec is also set, then the CommRecord records produced
	// can be queries using the WasExec method, to differentiate exec(2)
	// from the other ases.
	Comm bool

	// Freq configures the event to use sample frequency, rather than
	// sample period. See also Attr.Sample.
	Freq bool

	// InheritStat enables saving of event counts on context switch for
	// inherited tasks. InheritStat is only meaningful if Inherit is
	// also set.
	InheritStat bool

	// EnableOnExec configures the counter to be enabled automatically
	// after a call to exec(2).
	EnableOnExec bool

	// Task configures the event to include fork/exit notifications in
	// the ring buffer.
	Task bool

	// Watermark configures the ring buffer to issue an overflow
	// notification when the Wakeup boundary is crossed. If not set,
	// notifications happen after Wakeup samples. See also Attr.Wakeup.
	Watermark bool

	// PreciseIP controls the number of instructions between an event of
	// interest happening and the kernel being able to stop and record
	// the event.
	PreciseIP Skid

	// MmapData is the counterpart to Mmap. It enables generation of
	// MmapRecord records for mmap(2) calls that do not have PROT_EXEC
	// set.
	MmapData bool

	// SampleIDAll configures Tid, Time, ID, StreamID and CPU samples
	// to be included in non-Sample records.
	SampleIDAll bool

	// ExcludeHost configures only events happening inside a guest
	// instance (one that has executed a KVM_RUN ioctl) to be measured.
	ExcludeHost bool

	// ExcludeGuest is the opposite of ExcludeHost: it configures only
	// events outside a guest instance to be measured.
	ExcludeGuest bool

	// ExcludeKernelCallchain excludes kernel callchains.
	ExcludeKernelCallchain bool

	// ExcludeUserCallchain excludes user callchains.
	ExcludeUserCallchain bool

	// Mmap2 configures mmap(2) events to include inode data.
	Mmap2 bool

	// CommExec allows the distinction between process renaming
	// via exec(2) or via other means. See also Comm, and
	// (*CommRecord).WasExec.
	CommExec bool

	// UseClockID allows selecting which internal linux clock to use
	// when generating timestamps via the ClockID field.
	UseClockID bool

	// ContextSwitch enables the generation of SwitchRecord records,
	// and SwitchCPUWideRecord records when sampling in CPU-wide mode.
	ContextSwitch bool

	// writeBackward configures the kernel to write to the memory
	// mapped ring buffer backwards. This option is not supported by
	// package perf at the moment.
	writeBackward bool

	// Namespaces enables the generation of NamespacesRecord records.
	Namespaces bool
}

func (opt Options) marshal() uint64 {
	fields := []bool{
		opt.Disabled,
		opt.Inherit,
		opt.Pinned,
		opt.Exclusive,
		opt.ExcludeUser,
		opt.ExcludeKernel,
		opt.ExcludeHypervisor,
		opt.ExcludeIdle,
		opt.Mmap,
		opt.Comm,
		opt.Freq,
		opt.InheritStat,
		opt.EnableOnExec,
		opt.Task,
		opt.Watermark,
		false, false, // 2 bits for skid constraint
		opt.MmapData,
		opt.SampleIDAll,
		opt.ExcludeHost,
		opt.ExcludeGuest,
		opt.ExcludeKernelCallchain,
		opt.ExcludeUserCallchain,
		opt.Mmap2,
		opt.CommExec,
		opt.UseClockID,
		opt.ContextSwitch,
		opt.writeBackward,
		opt.Namespaces,
	}
	val := marshalBitwiseUint64(fields)

	const (
		skidlsb = 15
		skidmsb = 16
	)
	if opt.PreciseIP&0x01 != 0 {
		val |= 1 << skidlsb
	}
	if opt.PreciseIP&0x10 != 0 {
		val |= 1 << skidmsb
	}

	return val
}

// Supported returns a boolean indicating whether the host kernel supports
// the perf_event_open system call, which is a prerequisite for the operations
// of this package.
//
// Supported checks for the existence of a /proc/sys/kernel/perf_event_paranoid
// file, which is the canonical method for determining if a kernel supports
// perf_event_open(2).
func Supported() bool {
	_, err := os.Stat("/proc/sys/kernel/perf_event_paranoid")
	return err == nil
}

// MaxStack returns the maximum number of frame pointers in a recorded
// callchain. It reads the value from /proc/sys/kernel/perf_event_max_stack.
func MaxStack() (uint16, error) {
	max, err := readUint("/proc/sys/kernel/perf_event_max_stack", 16)
	return uint16(max), err
}

// fields is a collection of 32-bit or 64-bit fields.
type fields []byte

// uint64 decodes the next 64 bit field into v.
func (f *fields) uint64(v *uint64) {
	*v = *(*uint64)(unsafe.Pointer(&(*f)[0]))
	f.advance(8)
}

// uint64Cond decodes the next 64 bit field into v, if cond is true.
func (f *fields) uint64Cond(cond bool, v *uint64) {
	if cond {
		f.uint64(v)
	}
}

// uint32 decodes a pair of uint32s into a and b.
func (f *fields) uint32(a, b *uint32) {
	*a = *(*uint32)(unsafe.Pointer(&(*f)[0]))
	*b = *(*uint32)(unsafe.Pointer(&(*f)[4]))
	f.advance(8)
}

// uint32 decodes a pair of uint32s into a and b, if cond is true.
func (f *fields) uint32Cond(cond bool, a, b *uint32) {
	if cond {
		f.uint32(a, b)
	}
}

func (f *fields) uint32sizeBytes(b *[]byte) {
	size := *(*uint32)(unsafe.Pointer(&(*f)[0]))
	f.advance(4)
	data := make([]byte, size)
	copy(data, *f)
	f.advance(int(size))
	*b = data
}

func (f *fields) uint64sizeBytes(b *[]byte) {
	size := *(*uint64)(unsafe.Pointer(&(*f)[0]))
	f.advance(8)
	data := make([]byte, size)
	copy(data, *f)
	f.advance(int(size))
	*b = data
}

// duration decodes a duration into d.
func (f *fields) duration(d *time.Duration) {
	*d = *(*time.Duration)(unsafe.Pointer(&(*f)[0]))
	f.advance(8)
}

// string decodes a null-terminated string into s. The null terminator
// is not included in the string written to s.
func (f *fields) string(s *string) {
	for i := 0; i < len(*f); i++ {
		if (*f)[i] == 0 {
			*s = string((*f)[:i])
			if i+1 <= len(*f) {
				f.advance(i + 1)
			}
			return
		}
	}
}

// id decodes a SampleID based on the SampleFormat event was configured with,
// if cond is true.
func (f *fields) idCond(cond bool, id *SampleID, sfmt SampleFormat) {
	if !cond {
		return
	}
	f.uint32Cond(sfmt.Tid, &id.Pid, &id.Tid)
	f.uint64Cond(sfmt.Time, &id.Time)
	f.uint64Cond(sfmt.ID, &id.ID)
	f.uint64Cond(sfmt.StreamID, &id.StreamID)
	var reserved uint32
	f.uint32Cond(sfmt.CPU, &id.CPU, &reserved)
	f.uint64Cond(sfmt.Identifier, &id.Identifier)
}

// count decodes a Count into c.
func (f *fields) count(c *Count, cfmt CountFormat) {
	f.uint64(&c.Value)
	if cfmt.Enabled {
		f.duration(&c.Enabled)
	}
	if cfmt.Running {
		f.duration(&c.Running)
	}
	f.uint64Cond(cfmt.ID, &c.ID)
}

// groupCount decodes a GroupCount into gc.
func (f *fields) groupCount(gc *GroupCount, cfmt CountFormat) {
	var nr uint64
	f.uint64(&nr)
	if cfmt.Enabled {
		f.duration(&gc.Enabled)
	}
	if cfmt.Running {
		f.duration(&gc.Running)
	}
	gc.Values = make([]struct {
		Value, ID uint64
		Label     string
	}, nr)
	for i := 0; i < int(nr); i++ {
		f.uint64(&gc.Values[i].Value)
		f.uint64Cond(cfmt.ID, &gc.Values[i].ID)
	}
}

// advance advances through the fields by n bytes.
func (f *fields) advance(n int) {
	*f = (*f)[n:]
}

// marshalBitwiseUint64 marshals a set of bitwise flags into a
// uint64, LSB first.
func marshalBitwiseUint64(fields []bool) uint64 {
	var res uint64
	for shift, set := range fields {
		if set {
			res |= 1 << uint(shift)
		}
	}
	return res
}

// readUint reads an unsigned integer from the specified sys file.
// If readUint does not return an error, the returned integer is
// guaranteed to fit in the specified number of bits.
func readUint(sysfile string, bits int) (uint64, error) {
	content, err := ioutil.ReadFile(sysfile)
	if err != nil {
		return 0, err
	}
	content = bytes.TrimSpace(content)
	return strconv.ParseUint(string(content), 10, bits)
}

type eventLabel struct {
	Name, Alias string
}

func (el eventLabel) String() string {
	if el.Name == "" {
		return "unknown"
	}
	if el.Alias != "" {
		return fmt.Sprintf("%s OR %s", el.Name, el.Alias)
	}
	return el.Name
}

type eventID struct {
	Type, Config uint64
}

var eventLabels sync.Map // of eventID to eventLabel

func init() {
	type labeler interface {
		eventLabel() eventLabel
	}

	var events []Configurator
	events = append(events, AllHardwareCounters()...)
	events = append(events, AllSoftwareCounters()...)

	for _, cfg := range events {
		if l, ok := cfg.(labeler); ok {
			var a Attr
			cfg.Configure(&a)
			id := eventID{Type: uint64(a.Type), Config: a.Config}
			label := l.eventLabel()
			eventLabels.Store(id, label)
		}
	}
}

func lookupLabel(id eventID) eventLabel {
	v, ok := eventLabels.Load(id)
	if ok {
		return v.(eventLabel)
	}
	label := lookupLabelInSysfs(id)
	eventLabels.Store(id, label)
	return label
}

func lookupLabelInSysfs(id eventID) eventLabel {
	return eventLabel{}
}

// BUG(acln): generic Attr.Label lookup is not implemented
