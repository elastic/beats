// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build linux

package perf

import (
	"context"
	"errors"
	"fmt"
	"math/bits"
	"os"
	"sync/atomic"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

// ErrDisabled is returned from ReadRecord and ReadRawRecord if the event
// being monitored is attached to a different process, and that process
// exits. (since Linux 3.18)
var ErrDisabled = errors.New("perf: event disabled")

// ErrNoReadRecord is returned by ReadRecord when it is disabled on a
// group event, due to different configurations of the leader and follower
// events. See also (*Event).SetOutput.
var ErrNoReadRecord = errors.New("perf: ReadRecord disabled")

// ErrBadRecord is returned by ReadRecord when a read record can't be decoded.
var ErrBadRecord = errors.New("bad record received")

// ReadRecord reads and decodes a record from the ring buffer associated
// with ev.
//
// ReadRecord may be called concurrently with ReadCount or ReadGroupCount,
// but not concurrently with itself, ReadRawRecord, Close, or any other
// Event method.
//
// If another event's records were routed to ev via SetOutput, and the
// two events did not have compatible SampleFormat Options settings (see
// SetOutput documentation), ReadRecord returns ErrNoReadRecord.
func (ev *Event) ReadRecord(ctx context.Context) (Record, error) {
	if err := ev.ok(); err != nil {
		return nil, err
	}
	if ev.noReadRecord {
		return nil, ErrNoReadRecord
	}
	var raw RawRecord
	if err := ev.ReadRawRecord(ctx, &raw); err != nil {
		return nil, err
	}
	rec, err := newRecord(ev, raw.Header.Type)
	if err != nil {
		return nil, err
	}
	if err := rec.DecodeFrom(&raw, ev); err != nil {
		return nil, err
	}
	return rec, nil
}

// ReadRawRecord reads and decodes a raw record from the ring buffer
// associated with ev into rec. Callers must not retain rec.Data.
//
// ReadRawRecord may be called concurrently with ReadCount or ReadGroupCount,
// but not concurrently with itself, ReadRecord, Close or any other Event
// method.
func (ev *Event) ReadRawRecord(ctx context.Context, raw *RawRecord) error {
	if err := ev.ok(); err != nil {
		return err
	}
	if ev.ring == nil {
		return errors.New("perf: event ring not mapped")
	}

	// Fast path: try reading from the ring buffer first. If there is
	// a record there, we are done.
	if ev.readRawRecordNonblock(raw) {
		return nil
	}

	// If the context has a deadline, and that deadline is in the future,
	// use it to compute a timeout for ppoll(2). If the context is
	// expired, bail out immediately. Otherwise, the timeout is zero,
	// which means no timeout.
	var timeout time.Duration
	deadline, ok := ctx.Deadline()
	if ok {
		timeout = time.Until(deadline)
		if timeout <= 0 {
			<-ctx.Done()
			return ctx.Err()
		}
	}

	// Start a round of polling, then await results. Only one request
	// can be in flight at a time, and the whole request-response cycle
	// is owned by the current invocation of ReadRawRecord.
again:
	ev.pollreq <- pollreq{timeout: timeout}
	select {
	case <-ctx.Done():
		active := false
		err := ctx.Err()
		if err == context.Canceled {
			// Initiate active wakeup on ev.wakeupfd, and wait for
			// doPoll to return. doPoll might miss this signal,
			// but that's okay: see below.
			val := uint64(1)
			buf := (*[8]byte)(unsafe.Pointer(&val))[:]
			unix.Write(ev.wakeupfd, buf)
			active = true
		}
		<-ev.pollresp

		// We don't know if doPoll woke up due to our active wakeup
		// or because it timed out. It doesn't make a difference.
		// The important detail here is that doPoll does not touch
		// ev.wakeupfd (besides polling it for readiness). If we
		// initiated active wakeup, we must restore the event file
		// descriptor to quiescent state ourselves, in order to avoid
		// a spurious wakeup during the next round of polling.
		if active {
			var buf [8]byte
			unix.Read(ev.wakeupfd, buf[:])
		}
		return err
	case resp := <-ev.pollresp:
		if resp.err != nil {
			// Polling failed. Nothing to do but report the error.
			return resp.err
		}
		if resp.perfhup {
			// Saw POLLHUP on ev.perffd. See also the
			// documentation for ErrDisabled.
			return ErrDisabled
		}
		if !resp.perfready {
			// Here, we have not touched ev.wakeupfd, there
			// was no polling error, and ev.perffd is not
			// ready. Therefore, ppoll(2) must have timed out.
			//
			// The reason we are here is the following: doPoll
			// woke up, and immediately sent us a pollresp, which
			// won the race with <-ctx.Done(), such that this
			// select case fired. In any case, ctx is expired,
			// because we wouldn't be here otherwise.
			<-ctx.Done()
			return ctx.Err()
		}
		if !ev.readRawRecordNonblock(raw) {
			// It might happen that an overflow notification was
			// generated on the file descriptor, we observed it
			// as POLLIN, but there is still nothing new for us
			// to read in the ring buffer.
			//
			// This is because the notification is raised based
			// on the Attr.Wakeup and Attr.Options.Watermark
			// settings, rather than based on what events we've
			// seen already.
			//
			// For example, for an event with Attr.Wakeup == 1,
			// POLLIN will be indicated on the file descriptor
			// after the first event, regardless of whether we
			// have consumed it from the ring buffer or not.
			//
			// If we happen to see POLLIN with an empty ring
			// buffer, the only thing to do is to wait again.
			//
			// See also https://github.com/acln0/perfwakeup.
			goto again
		}
		return nil
	}
}

// HasRecord returns if there is a record available to be read from the ring.
func (ev *Event) HasRecord() bool {
	return atomic.LoadUint64(&ev.meta.Data_head) != atomic.LoadUint64(&ev.meta.Data_tail)
}

// resetRing advances the read pointer to the write pointer to discard all the
// data in the ring. This is done when bogus data is read from the ring.
func (ev *Event) resetRing() {
	atomic.StoreUint64(&ev.meta.Data_tail, atomic.LoadUint64(&ev.meta.Data_head))
}

// readRawRecordNonblock reads a raw record into rec, if one is available.
// Callers must not retain rec.Data. The boolean return value signals whether
// a record was actually found / written to rec.
func (ev *Event) readRawRecordNonblock(raw *RawRecord) bool {
	head := atomic.LoadUint64(&ev.meta.Data_head)
	tail := atomic.LoadUint64(&ev.meta.Data_tail)
	if head == tail {
		return false
	}

	// Make sure there is enough space the read a record header. Otherwise
	// consider the ring to be corrupted.
	const headerSize = uint64(unsafe.Sizeof(RecordHeader{}))
	avail := head - tail
	if avail < headerSize {
		ev.resetRing()
		return false
	}

	// Head and tail values only ever grow, so we must take their value
	// modulo the size of the data segment of the ring.
	start := tail % uint64(len(ev.ringdata))
	raw.Header = *(*RecordHeader)(unsafe.Pointer(&ev.ringdata[start]))
	end := (tail + uint64(raw.Header.Size)) % uint64(len(ev.ringdata))

	// Make sure there is enough space available to read the whole record.
	// Otherwise treat the ring as corrupted.
	msgLen := uint64(raw.Header.Size)
	if avail < msgLen || msgLen < headerSize {
		ev.resetRing()
		return false
	}

	// Reserve space to store this record out of the ring.
	if uint64(len(ev.recordBuffer)) < msgLen {
		ev.recordBuffer = make([]byte, msgLen)
	}
	// If the record wraps around the ring, we must allocate storage,
	// so that we can return a contiguous area of memory to the caller.
	if end < start {
		n := copy(ev.recordBuffer, ev.ringdata[start:])
		copy(ev.recordBuffer[n:], ev.ringdata[:int(raw.Header.Size)-n])
	} else {
		copy(ev.recordBuffer, ev.ringdata[start:end])
	}
	raw.Data = ev.recordBuffer[unsafe.Sizeof(raw.Header):msgLen]

	// Notify the kernel of the last record we've seen.
	atomic.AddUint64(&ev.meta.Data_tail, msgLen)
	return true
}

// poll services requests from ev.pollreq and sends responses on ev.pollresp.
func (ev *Event) poll() {
	defer close(ev.pollresp)

	for req := range ev.pollreq {
		ev.pollresp <- ev.doPoll(req)
	}
}

// doPoll executes one round of polling on ev.perffd and ev.wakeupfd.
//
// A req.timeout value of zero is interpreted as "no timeout". req.timeout
// must not be negative.
func (ev *Event) doPoll(req pollreq) pollresp {
	var timeout *unix.Timespec
	if req.timeout > 0 {
		ts := unix.NsecToTimespec(req.timeout.Nanoseconds())
		timeout = &ts
	}

	pollfds := []unix.PollFd{
		{Fd: int32(ev.perffd), Events: unix.POLLIN},
		{Fd: int32(ev.wakeupfd), Events: unix.POLLIN},
	}

again:
	_, err := unix.Ppoll(pollfds, timeout, nil)
	// TODO(acln): do we need to do this business at all? See #20400.
	if err == unix.EINTR {
		goto again
	}

	// If we are here and we have successfully woken up, it is for one
	// of four reasons: we got POLLIN on ev.perffd, we got POLLHUP on
	// ev.perffd (see ErrDisabled), the ppoll(2) timeout fired, or we
	// got POLLIN on ev.wakeupfd.
	//
	// Report if the perf fd is ready, if we saw POLLHUP, and any
	// errors except EINTR. The machinery is documented in more detail
	// in ReadRawRecord.
	return pollresp{
		perfready: pollfds[0].Revents&unix.POLLIN != 0,
		perfhup:   pollfds[0].Revents&unix.POLLHUP != 0,
		err:       os.NewSyscallError("ppoll", err),
	}
}

type pollreq struct {
	// timeout is the timeout for ppoll(2): zero means no timeout
	timeout time.Duration
}

type pollresp struct {
	// perfready indicates if the perf FD (ev.perffd) is ready.
	perfready bool

	// perfhup indicates if POLLUP was observed on ev.perffd.
	perfhup bool

	// err is the *os.SyscallError from ppoll(2).
	err error
}

// SampleFormat configures information requested in overflow packets.
type SampleFormat struct {
	// IP records the instruction pointer.
	IP bool

	// Tid records process and thread IDs.
	Tid bool

	// Time records a hardware timestamp.
	Time bool

	// Addr records an address, if applicable.
	Addr bool

	// Count records counter values for all events in a group, not just
	// the group leader.
	Count bool

	// Callchain records the stack backtrace.
	Callchain bool

	// ID records a unique ID for the opened event's group leader.
	ID bool

	// CPU records the CPU number.
	CPU bool

	// Period records the current sampling period.
	Period bool

	// StreamID returns a unique ID for the opened event. Unlike ID,
	// the actual ID is returned, not the group ID.
	StreamID bool

	// Raw records additional data, if applicable. Usually returned by
	// tracepoint events.
	Raw bool

	// BranchStack provides a record of recent branches, as provided by
	// CPU branch sampling hardware. See also Attr.BranchSampleFormat.
	BranchStack bool

	// UserRegisters records the current user-level CPU state (the
	// values in the process before the kernel was called). See also
	// Attr.SampleRegistersUser.
	UserRegisters bool

	// UserStack records the user level stack, allowing stack unwinding.
	UserStack bool

	// Weight records a hardware provided weight value that expresses
	// how costly the sampled event was.
	Weight bool

	// DataSource records the data source: where in the memory hierarchy
	// the data associated with the sampled instruction came from.
	DataSource bool

	// Identifier places the ID value in a fixed position in the record.
	Identifier bool

	// Transaction records reasons for transactional memory abort events.
	Transaction bool

	// IntrRegisters Records a subset of the current CPU register state.
	// Unlike UserRegisters, the registers will return kernel register
	// state if the overflow happened while kernel code is running. See
	// also Attr.SampleRegistersIntr.
	IntrRegisters bool

	PhysicalAddress bool
}

// TODO(acln): document SampleFormat.PhysicalAddress

// marshal packs the SampleFormat into a uint64.
func (sf SampleFormat) marshal() uint64 {
	// Always keep this in sync with the type definition above.
	fields := []bool{
		sf.IP,
		sf.Tid,
		sf.Time,
		sf.Addr,
		sf.Count,
		sf.Callchain,
		sf.ID,
		sf.CPU,
		sf.Period,
		sf.StreamID,
		sf.Raw,
		sf.BranchStack,
		sf.UserRegisters,
		sf.UserStack,
		sf.Weight,
		sf.DataSource,
		sf.Identifier,
		sf.Transaction,
		sf.IntrRegisters,
		sf.PhysicalAddress,
	}
	return marshalBitwiseUint64(fields)
}

// SampleID contains identifiers for when and where a record was collected.
//
// A SampleID is included in a Record if Options.SampleIDAll is set on the
// associated event. Fields are set according to SampleFormat options.
type SampleID struct {
	Pid        uint32
	Tid        uint32
	Time       uint64
	ID         uint64
	StreamID   uint64
	CPU        uint32
	_          uint32 // reserved
	Identifier uint64
}

// Record is the interface implemented by all record types.
type Record interface {
	Header() RecordHeader
	DecodeFrom(*RawRecord, *Event) error
}

// RecordType is the type of an overflow record.
type RecordType uint32

// Known record types.
const (
	RecordTypeMmap          RecordType = unix.PERF_RECORD_MMAP
	RecordTypeLost          RecordType = unix.PERF_RECORD_LOST
	RecordTypeComm          RecordType = unix.PERF_RECORD_COMM
	RecordTypeExit          RecordType = unix.PERF_RECORD_EXIT
	RecordTypeThrottle      RecordType = unix.PERF_RECORD_THROTTLE
	RecordTypeUnthrottle    RecordType = unix.PERF_RECORD_UNTHROTTLE
	RecordTypeFork          RecordType = unix.PERF_RECORD_FORK
	RecordTypeRead          RecordType = unix.PERF_RECORD_READ
	RecordTypeSample        RecordType = unix.PERF_RECORD_SAMPLE
	RecordTypeMmap2         RecordType = unix.PERF_RECORD_MMAP2
	RecordTypeAux           RecordType = unix.PERF_RECORD_AUX
	RecordTypeItraceStart   RecordType = unix.PERF_RECORD_ITRACE_START
	RecordTypeLostSamples   RecordType = unix.PERF_RECORD_LOST_SAMPLES
	RecordTypeSwitch        RecordType = unix.PERF_RECORD_SWITCH
	RecordTypeSwitchCPUWide RecordType = unix.PERF_RECORD_SWITCH_CPU_WIDE
	RecordTypeNamespaces    RecordType = unix.PERF_RECORD_NAMESPACES
)

func (rt RecordType) known() bool {
	return rt >= RecordTypeMmap && rt <= RecordTypeNamespaces
}

// RecordHeader is the header present in every overflow record.
type RecordHeader struct {
	Type RecordType
	Misc uint16
	Size uint16
}

// Header returns rh itself, so that types which embed a RecordHeader
// automatically implement a part of the Record interface.
func (rh RecordHeader) Header() RecordHeader { return rh }

// CPUMode returns the CPU mode in use when the sample happened.
func (rh RecordHeader) CPUMode() CPUMode {
	return CPUMode(rh.Misc & cpuModeMask)
}

// CPUMode is a CPU operation mode.
type CPUMode uint8

const cpuModeMask = 7

// Known CPU modes.
const (
	UnknownMode CPUMode = iota
	KernelMode
	UserMode
	HypervisorMode
	GuestKernelMode
	GuestUserMode
)

// RawRecord is a raw overflow record, read from the memory mapped ring
// buffer associated with an Event.
//
// Header is the 8 byte record header. Data contains the rest of the record.
type RawRecord struct {
	Header RecordHeader
	Data   []byte
}

func (raw RawRecord) fields() fields { return fields(raw.Data) }

var newRecordFuncs = [...]func(ev *Event) Record{
	RecordTypeMmap:          func(_ *Event) Record { return &MmapRecord{} },
	RecordTypeLost:          func(_ *Event) Record { return &LostRecord{} },
	RecordTypeComm:          func(_ *Event) Record { return &CommRecord{} },
	RecordTypeExit:          func(_ *Event) Record { return &ExitRecord{} },
	RecordTypeThrottle:      func(_ *Event) Record { return &ThrottleRecord{} },
	RecordTypeUnthrottle:    func(_ *Event) Record { return &UnthrottleRecord{} },
	RecordTypeFork:          func(_ *Event) Record { return &ForkRecord{} },
	RecordTypeRead:          newReadRecord,
	RecordTypeSample:        newSampleRecord,
	RecordTypeMmap2:         func(_ *Event) Record { return &Mmap2Record{} },
	RecordTypeAux:           func(_ *Event) Record { return &AuxRecord{} },
	RecordTypeItraceStart:   func(_ *Event) Record { return &ItraceStartRecord{} },
	RecordTypeLostSamples:   func(_ *Event) Record { return &LostSamplesRecord{} },
	RecordTypeSwitch:        func(_ *Event) Record { return &SwitchRecord{} },
	RecordTypeSwitchCPUWide: func(_ *Event) Record { return &SwitchCPUWideRecord{} },
	RecordTypeNamespaces:    func(_ *Event) Record { return &NamespacesRecord{} },
}

func newReadRecord(ev *Event) Record {
	if ev.a.CountFormat.Group {
		return &ReadGroupRecord{}
	}
	return &ReadRecord{}
}

func newSampleRecord(ev *Event) Record {
	if ev.a.CountFormat.Group {
		return &SampleGroupRecord{}
	}
	return &SampleRecord{}
}

// newRecord returns an empty Record of the given type, tailored for the
// specified Event.
func newRecord(ev *Event, rt RecordType) (Record, error) {
	if !rt.known() {
		return nil, fmt.Errorf("unknown record type %d", rt)
	}
	return newRecordFuncs[rt](ev), nil
}

// mmapDataBit is PERF_RECORD_MISC_MMAP_DATA
const mmapDataBit = 1 << 13

// MmapRecord (PERF_RECORD_MMAP) records PROT_EXEC mappings such that
// user-space IPs can be correlated to code.
type MmapRecord struct {
	RecordHeader
	Pid        uint32 // process ID
	Tid        uint32 // thread ID
	Addr       uint64 // address of the allocated memory
	Len        uint64 // length of the allocated memory
	PageOffset uint64 // page offset of the allocated memory
	Filename   string // describes backing of allocated memory
	SampleID
}

// DecodeFrom implements the Record.DecodeFrom method.
func (mr *MmapRecord) DecodeFrom(raw *RawRecord, ev *Event) error {
	mr.RecordHeader = raw.Header
	f := raw.fields()
	f.uint32(&mr.Pid, &mr.Tid)
	f.uint64(&mr.Addr)
	f.uint64(&mr.Len)
	f.uint64(&mr.PageOffset)
	f.string(&mr.Filename)
	f.idCond(ev.a.Options.SampleIDAll, &mr.SampleID, ev.a.SampleFormat)
	return nil
}

// Executable returns a boolean indicating whether the mapping is executable.
func (mr *MmapRecord) Executable() bool {
	// The data bit is set when the mapping is _not_ executable.
	return mr.RecordHeader.Misc&mmapDataBit == 0
}

// LostRecord (PERF_RECORD_LOST) indicates when events are lost.
type LostRecord struct {
	RecordHeader
	ID   uint64 // the unique ID for the lost events
	Lost uint64 // the number of lost events
	SampleID
}

// DecodeFrom implements the Record.DecodeFrom method.
func (lr *LostRecord) DecodeFrom(raw *RawRecord, ev *Event) error {
	lr.RecordHeader = raw.Header
	f := raw.fields()
	f.uint64(&lr.ID)
	f.uint64(&lr.Lost)
	f.idCond(ev.a.Options.SampleIDAll, &lr.SampleID, ev.a.SampleFormat)
	return nil
}

// CommRecord (PERF_RECORD_COMM) indicates a change in the process name.
type CommRecord struct {
	RecordHeader
	Pid     uint32 // process ID
	Tid     uint32 // threadID
	NewName string // the new name of the process
	SampleID
}

// DecodeFrom implements the Record.DecodeFrom method.
func (cr *CommRecord) DecodeFrom(raw *RawRecord, ev *Event) error {
	cr.RecordHeader = raw.Header
	f := raw.fields()
	f.uint32(&cr.Pid, &cr.Tid)
	f.string(&cr.NewName)
	f.idCond(ev.a.Options.SampleIDAll, &cr.SampleID, ev.a.SampleFormat)
	return nil
}

// commExecBit is PERF_RECORD_MISC_COMM_EXEC
const commExecBit = 1 << 13

// WasExec returns a boolean indicating whether a process name change
// was caused by an exec(2) system call.
func (cr *CommRecord) WasExec() bool {
	return cr.RecordHeader.Misc&(commExecBit) != 0
}

// ExitRecord (PERF_RECORD_EXIT) indicates a process exit event.
type ExitRecord struct {
	RecordHeader
	Pid  uint32 // process ID
	Ppid uint32 // parent process ID
	Tid  uint32 // thread ID
	Ptid uint32 // parent thread ID
	Time uint64 // time when the process exited
	SampleID
}

// DecodeFrom implements the Record.DecodeFrom method.
func (er *ExitRecord) DecodeFrom(raw *RawRecord, ev *Event) error {
	er.RecordHeader = raw.Header
	f := raw.fields()
	f.uint32(&er.Pid, &er.Ppid)
	f.uint32(&er.Tid, &er.Ptid)
	f.uint64(&er.Time)
	f.idCond(ev.a.Options.SampleIDAll, &er.SampleID, ev.a.SampleFormat)
	return nil
}

// ThrottleRecord (PERF_RECORD_THROTTLE) indicates a throttle event.
type ThrottleRecord struct {
	RecordHeader
	Time     uint64
	ID       uint64
	StreamID uint64
	SampleID
}

// DecodeFrom implements the Record.DecodeFrom method.
func (tr *ThrottleRecord) DecodeFrom(raw *RawRecord, ev *Event) error {
	tr.RecordHeader = raw.Header
	f := raw.fields()
	f.uint64(&tr.Time)
	f.uint64(&tr.ID)
	f.uint64(&tr.StreamID)
	f.idCond(ev.a.Options.SampleIDAll, &tr.SampleID, ev.a.SampleFormat)
	return nil
}

// UnthrottleRecord (PERF_RECORD_UNTHROTTLE) indicates an unthrottle event.
type UnthrottleRecord struct {
	RecordHeader
	Time     uint64
	ID       uint64
	StreamID uint64
	SampleID
}

// DecodeFrom implements the Record.DecodeFrom method.
func (ur *UnthrottleRecord) DecodeFrom(raw *RawRecord, ev *Event) error {
	ur.RecordHeader = raw.Header
	f := raw.fields()
	f.uint64(&ur.Time)
	f.uint64(&ur.ID)
	f.uint64(&ur.StreamID)
	f.idCond(ev.a.Options.SampleIDAll, &ur.SampleID, ev.a.SampleFormat)
	return nil
}

// ForkRecord (PERF_RECORD_FORK) indicates a fork event.
type ForkRecord struct {
	RecordHeader
	Pid  uint32 // process ID
	Ppid uint32 // parent process ID
	Tid  uint32 // thread ID
	Ptid uint32 // parent thread ID
	Time uint64 // time when the fork occurred
	SampleID
}

// DecodeFrom implements the Record.DecodeFrom method.
func (fr *ForkRecord) DecodeFrom(raw *RawRecord, ev *Event) error {
	fr.RecordHeader = raw.Header
	f := raw.fields()
	f.uint32(&fr.Pid, &fr.Ppid)
	f.uint32(&fr.Tid, &fr.Ptid)
	f.uint64(&fr.Time)
	f.idCond(ev.a.Options.SampleIDAll, &fr.SampleID, ev.a.SampleFormat)
	return nil
}

// ReadRecord (PERF_RECORD_READ) indicates a read event.
type ReadRecord struct {
	RecordHeader
	Pid   uint32 // process ID
	Tid   uint32 // thread ID
	Count Count  // count value
	SampleID
}

// DecodeFrom implements the Record.DecodeFrom method.
func (rr *ReadRecord) DecodeFrom(raw *RawRecord, ev *Event) error {
	rr.RecordHeader = raw.Header
	f := raw.fields()
	f.uint32(&rr.Pid, &rr.Tid)
	f.count(&rr.Count, ev.a.CountFormat)
	f.idCond(ev.a.Options.SampleIDAll, &rr.SampleID, ev.a.SampleFormat)
	return nil
}

// ReadGroupRecord (PERF_RECORD_READ) indicates a read event on a group event.
type ReadGroupRecord struct {
	RecordHeader
	Pid        uint32     // process ID
	Tid        uint32     // thread ID
	GroupCount GroupCount // group count values
	SampleID
}

// DecodeFrom implements the Record.DecodeFrom method.
func (rr *ReadGroupRecord) DecodeFrom(raw *RawRecord, ev *Event) error {
	rr.RecordHeader = raw.Header
	f := raw.fields()
	f.uint32(&rr.Pid, &rr.Tid)
	f.groupCount(&rr.GroupCount, ev.a.CountFormat)
	f.idCond(ev.a.Options.SampleIDAll, &rr.SampleID, ev.a.SampleFormat)
	return nil
}

// SampleRecord indicates a sample.
//
// All the fields up to and including Callchain represent ABI bits. All the
// fields starting with Data are non-ABI and have no compatibility guarantees.
//
// Fields on SampleRecord are set according to the SampleFormat the event
// was configured with. A boolean flag in SampleFormat typically enables
// the homonymous field in a SampleRecord.
type SampleRecord struct {
	RecordHeader
	Identifier uint64
	IP         uint64
	Pid        uint32
	Tid        uint32
	Time       uint64
	Addr       uint64
	ID         uint64
	StreamID   uint64
	CPU        uint32
	_          uint32 // reserved
	Period     uint64
	Count      Count
	Callchain  []uint64

	Raw                  []byte
	BranchStack          []BranchEntry
	UserRegisterABI      uint64
	UserRegisters        []uint64
	UserStack            []byte
	UserStackDynamicSize uint64
	Weight               uint64
	DataSource           DataSource
	Transaction          Transaction
	IntrRegisterABI      uint64
	IntrRegisters        []uint64
	PhysicalAddress      uint64
}

// DecodeFrom implements the Record.DecodeFrom method.
func (sr *SampleRecord) DecodeFrom(raw *RawRecord, ev *Event) error {
	sr.RecordHeader = raw.Header
	f := raw.fields()
	f.uint64Cond(ev.a.SampleFormat.Identifier, &sr.Identifier)
	f.uint64Cond(ev.a.SampleFormat.IP, &sr.IP)
	f.uint32Cond(ev.a.SampleFormat.Tid, &sr.Pid, &sr.Tid)
	f.uint64Cond(ev.a.SampleFormat.Time, &sr.Time)
	f.uint64Cond(ev.a.SampleFormat.Addr, &sr.Addr)
	f.uint64Cond(ev.a.SampleFormat.ID, &sr.ID)
	f.uint64Cond(ev.a.SampleFormat.StreamID, &sr.StreamID)

	// If we have a StreamID and it is different from our
	// own ID, then the output from the event we're interested
	// in was redirected to ev. We must switch to that event
	// in order to decode the sample.
	if ev.a.SampleFormat.StreamID {
		if sr.StreamID != ev.id {
			newev := ev.groupByID[sr.StreamID]
			if newev == nil {
				ev.resetRing()
				return ErrBadRecord
			}
			ev = newev
		}
	}

	var reserved uint32
	f.uint32Cond(ev.a.SampleFormat.CPU, &sr.CPU, &reserved)
	f.uint64Cond(ev.a.SampleFormat.Period, &sr.Period)
	if ev.a.SampleFormat.Count {
		f.count(&sr.Count, ev.a.CountFormat)
	}
	if ev.a.SampleFormat.Callchain {
		var nr uint64
		f.uint64(&nr)
		sr.Callchain = make([]uint64, nr)
		for i := 0; i < len(sr.Callchain); i++ {
			f.uint64(&sr.Callchain[i])
		}
	}
	if ev.a.SampleFormat.Raw {
		f.uint32sizeBytes(&sr.Raw)
	}
	if ev.a.SampleFormat.BranchStack {
		var nr uint64
		f.uint64(&nr)
		sr.BranchStack = make([]BranchEntry, nr)
		for i := 0; i < len(sr.BranchStack); i++ {
			var from, to, entry uint64
			f.uint64(&from)
			f.uint64(&to)
			f.uint64(&entry)
			sr.BranchStack[i].decode(from, to, entry)
		}
	}
	if ev.a.SampleFormat.UserRegisters {
		f.uint64(&sr.UserRegisterABI)
		num := bits.OnesCount64(ev.a.SampleRegistersUser)
		sr.UserRegisters = make([]uint64, num)
		for i := 0; i < len(sr.UserRegisters); i++ {
			f.uint64(&sr.UserRegisters[i])
		}
	}
	if ev.a.SampleFormat.UserStack {
		f.uint64sizeBytes(&sr.UserStack)
		if len(sr.UserStack) > 0 {
			f.uint64(&sr.UserStackDynamicSize)
		}
	}
	f.uint64Cond(ev.a.SampleFormat.Weight, &sr.Weight)
	if ev.a.SampleFormat.DataSource {
		var ds uint64
		f.uint64(&ds)
		sr.DataSource = DataSource(ds)
	}
	if ev.a.SampleFormat.Transaction {
		var tx uint64
		f.uint64(&tx)
		sr.Transaction = Transaction(tx)
	}
	if ev.a.SampleFormat.IntrRegisters {
		f.uint64(&sr.IntrRegisterABI)
		num := bits.OnesCount64(ev.a.SampleRegistersIntr)
		sr.IntrRegisters = make([]uint64, num)
		for i := 0; i < len(sr.IntrRegisters); i++ {
			f.uint64(&sr.IntrRegisters[i])
		}
	}
	f.uint64Cond(ev.a.SampleFormat.PhysicalAddress, &sr.PhysicalAddress)
	return nil
}

// exactIPBit is PERF_RECORD_MISC_EXACT_IP
const exactIPBit = 1 << 14

// ExactIP indicates that sr.IP points to the actual instruction that
// triggered the event. See also Options.PreciseIP.
func (sr *SampleRecord) ExactIP() bool {
	return sr.RecordHeader.Misc&exactIPBit != 0
}

// SampleGroupRecord indicates a sample from an event group.
//
// All the fields up to and including Callchain represent ABI bits. All the
// fields starting with Data are non-ABI and have no compatibility guarantees.
//
// Fields on SampleGroupRecord are set according to the RecordFormat the event
// was configured with. A boolean flag in RecordFormat typically enables the
// homonymous field in SampleGroupRecord.
type SampleGroupRecord struct {
	RecordHeader
	Identifier uint64
	IP         uint64
	Pid        uint32
	Tid        uint32
	Time       uint64
	Addr       uint64
	ID         uint64
	StreamID   uint64
	CPU        uint32
	_          uint32
	Period     uint64
	Count      GroupCount
	Callchain  []uint64

	Raw                  []byte
	BranchStack          []BranchEntry
	UserRegisterABI      uint64
	UserRegisters        []uint64
	UserStack            []byte
	UserStackDynamicSize uint64
	Weight               uint64
	DataSource           DataSource
	Transaction          Transaction
	IntrRegisterABI      uint64
	IntrRegisters        []uint64
	PhysicalAddress      uint64
}

// DecodeFrom implements the Record.DecodeFrom method.
func (sr *SampleGroupRecord) DecodeFrom(raw *RawRecord, ev *Event) error {
	sr.RecordHeader = raw.Header
	f := raw.fields()
	f.uint64Cond(ev.a.SampleFormat.Identifier, &sr.Identifier)
	f.uint64Cond(ev.a.SampleFormat.IP, &sr.IP)
	f.uint32Cond(ev.a.SampleFormat.Tid, &sr.Pid, &sr.Tid)
	f.uint64Cond(ev.a.SampleFormat.Time, &sr.Time)
	f.uint64Cond(ev.a.SampleFormat.Addr, &sr.Addr)
	f.uint64Cond(ev.a.SampleFormat.ID, &sr.ID)
	f.uint64Cond(ev.a.SampleFormat.StreamID, &sr.StreamID)

	// If we have a StreamID and it is different from our
	// own ID, then the output from the event we're interested
	// in was redirected to ev. We must switch to that event
	// in order to decode the sample.
	if ev.a.SampleFormat.StreamID {
		if sr.StreamID != ev.id {
			ev = ev.groupByID[sr.StreamID]
		}
	}

	var reserved uint32
	f.uint32Cond(ev.a.SampleFormat.CPU, &sr.CPU, &reserved)
	f.uint64Cond(ev.a.SampleFormat.Period, &sr.Period)
	if ev.a.SampleFormat.Count {
		f.groupCount(&sr.Count, ev.a.CountFormat)
	}
	if ev.a.SampleFormat.Callchain {
		var nr uint64
		f.uint64(&nr)
		sr.Callchain = make([]uint64, nr)
		for i := 0; i < len(sr.Callchain); i++ {
			f.uint64(&sr.Callchain[i])
		}
	}
	if ev.a.SampleFormat.Raw {
		f.uint32sizeBytes(&sr.Raw)
	}
	if ev.a.SampleFormat.BranchStack {
		var nr uint64
		f.uint64(&nr)
		sr.BranchStack = make([]BranchEntry, nr)
		for i := 0; i < len(sr.BranchStack); i++ {
			var from, to, entry uint64
			f.uint64(&from)
			f.uint64(&to)
			f.uint64(&entry)
			sr.BranchStack[i].decode(from, to, entry)
		}
	}
	if ev.a.SampleFormat.UserRegisters {
		f.uint64(&sr.UserRegisterABI)
		num := bits.OnesCount64(ev.a.SampleRegistersUser)
		sr.UserRegisters = make([]uint64, num)
		for i := 0; i < len(sr.UserRegisters); i++ {
			f.uint64(&sr.UserRegisters[i])
		}
	}
	if ev.a.SampleFormat.UserStack {
		f.uint64sizeBytes(&sr.UserStack)
		if len(sr.UserStack) > 0 {
			f.uint64(&sr.UserStackDynamicSize)
		}
	}
	f.uint64Cond(ev.a.SampleFormat.Weight, &sr.Weight)
	if ev.a.SampleFormat.DataSource {
		var ds uint64
		f.uint64(&ds)
		sr.DataSource = DataSource(ds)
	}
	if ev.a.SampleFormat.Transaction {
		var tx uint64
		f.uint64(&tx)
		sr.Transaction = Transaction(tx)
	}
	if ev.a.SampleFormat.IntrRegisters {
		f.uint64(&sr.IntrRegisterABI)
		num := bits.OnesCount64(ev.a.SampleRegistersIntr)
		sr.IntrRegisters = make([]uint64, num)
		for i := 0; i < len(sr.IntrRegisters); i++ {
			f.uint64(&sr.IntrRegisters[i])
		}
	}
	f.uint64Cond(ev.a.SampleFormat.PhysicalAddress, &sr.PhysicalAddress)
	return nil
}

// ExactIP indicates that sr.IP points to the actual instruction that
// triggered the event. See also Options.PreciseIP.
func (sr *SampleGroupRecord) ExactIP() bool {
	return sr.RecordHeader.Misc&exactIPBit != 0
}

// BranchEntry is a sampled branch.
type BranchEntry struct {
	From             uint64
	To               uint64
	Mispredicted     bool
	Predicted        bool
	InTransaction    bool
	TransactionAbort bool
	Cycles           uint16
	BranchType       BranchType
}

func (be *BranchEntry) decode(from, to, entry uint64) {
	*be = BranchEntry{
		From:             from,
		To:               to,
		Mispredicted:     entry&(1<<0) != 0,
		Predicted:        entry&(1<<1) != 0,
		InTransaction:    entry&(1<<2) != 0,
		TransactionAbort: entry&(1<<3) != 0,
		Cycles:           uint16((entry << 44) >> 48),
		BranchType:       BranchType((entry << 40) >> 44),
	}
}

// BranchType classifies a BranchEntry.
type BranchType uint8

// Branch classifications.
const (
	BranchTypeUnknown BranchType = iota
	BranchTypeConditional
	BranchTypeUnconditional
	BranchTypeIndirect
	BranchTypeCall
	BranchTypeIndirectCall
	BranchTypeReturn
	BranchTypeSyscall
	BranchTypeSyscallReturn
	BranchTypeConditionalCall
	BranchTypeConditionalReturn
)

// Mmap2Record (PERF_RECORD_MMAP2) includes extended information on mmap(2)
// calls returning executable mappings. It is similar to MmapRecord, but
// includes extra values, allowing unique identification of shared mappings.
type Mmap2Record struct {
	RecordHeader
	Pid             uint32 // process ID
	Tid             uint32 // thread ID
	Addr            uint64 // address of the allocated memory
	Len             uint64 // length of the allocated memory
	PageOffset      uint64 // page offset of the allocated memory
	MajorID         uint32 // major ID of the underlying device
	MinorID         uint32 // minor ID of the underlying device
	Inode           uint64 // inode number
	InodeGeneration uint64 // inode generation
	Prot            uint32 // protection information
	Flags           uint32 // flags information
	Filename        string // describes the backing of the allocated memory
	SampleID
}

// DecodeFrom implements the Record.DecodeFrom method.
func (mr *Mmap2Record) DecodeFrom(raw *RawRecord, ev *Event) error {
	mr.RecordHeader = raw.Header
	f := raw.fields()
	f.uint32(&mr.Pid, &mr.Tid)
	f.uint64(&mr.Addr)
	f.uint64(&mr.Len)
	f.uint64(&mr.PageOffset)
	f.uint32(&mr.MajorID, &mr.MinorID)
	f.uint64(&mr.Inode)
	f.uint64(&mr.InodeGeneration)
	f.uint32(&mr.Prot, &mr.Flags)
	f.string(&mr.Filename)
	f.idCond(ev.a.Options.SampleIDAll, &mr.SampleID, ev.a.SampleFormat)
	return nil
}

// Executable returns a boolean indicating whether the mapping is executable.
func (mr *Mmap2Record) Executable() bool {
	// The data bit is set when the mapping is _not_ executable.
	return mr.RecordHeader.Misc&mmapDataBit == 0
}

// AuxRecord (PERF_RECORD_AUX) reports that new data is available in the
// AUX buffer region.
type AuxRecord struct {
	RecordHeader
	Offset uint64  // offset in the AUX mmap region where the new data begins
	Size   uint64  // size of data made available
	Flags  AuxFlag // describes the update
	SampleID
}

// AuxFlag describes an update to a record in the AUX buffer region.
type AuxFlag uint64

// AuxFlag bits.
const (
	AuxTruncated AuxFlag = 0x01 // record was truncated to fit
	AuxOverwrite AuxFlag = 0x02 // snapshot from overwrite mode
	AuxPartial   AuxFlag = 0x04 // record contains gaps
	AuxCollision AuxFlag = 0x08 // sample collided with another
)

// DecodeFrom implements the Record.DecodeFrom method.
func (ar *AuxRecord) DecodeFrom(raw *RawRecord, ev *Event) error {
	ar.RecordHeader = raw.Header
	f := raw.fields()
	f.uint64(&ar.Offset)
	f.uint64(&ar.Size)
	var flag uint64
	f.uint64(&flag)
	ar.Flags = AuxFlag(flag)
	f.idCond(ev.a.Options.SampleIDAll, &ar.SampleID, ev.a.SampleFormat)
	return nil
}

// ItraceStartRecord (PERF_RECORD_ITRACE_START) indicates which process
// has initiated an instruction trace event, allowing tools to correlate
// instruction addresses in the AUX buffer with the proper executable.
type ItraceStartRecord struct {
	RecordHeader
	Pid uint32 // process ID of the thread starting an instruction trace
	Tid uint32 // thread ID of the thread starting an instruction trace
	SampleID
}

// DecodeFrom implements the Record.DecodeFrom method.
func (ir *ItraceStartRecord) DecodeFrom(raw *RawRecord, ev *Event) error {
	ir.RecordHeader = raw.Header
	f := raw.fields()
	f.uint32(&ir.Pid, &ir.Tid)
	f.idCond(ev.a.Options.SampleIDAll, &ir.SampleID, ev.a.SampleFormat)
	return nil
}

// LostSamplesRecord (PERF_RECORD_LOST_SAMPLES) indicates some number of
// samples that may have been lost, when using hardware sampling such as
// Intel PEBS.
type LostSamplesRecord struct {
	RecordHeader
	Lost uint64 // the number of potentially lost samples
	SampleID
}

// DecodeFrom implements the Record.DecodeFrom method.
func (lr *LostSamplesRecord) DecodeFrom(raw *RawRecord, ev *Event) error {
	lr.RecordHeader = raw.Header
	f := raw.fields()
	f.uint64(&lr.Lost)
	f.idCond(ev.a.Options.SampleIDAll, &lr.SampleID, ev.a.SampleFormat)
	return nil
}

// SwitchRecord (PERF_RECORD_SWITCH) indicates that a context switch has
// happened.
type SwitchRecord struct {
	RecordHeader
	SampleID
}

// DecodeFrom implements the Record.DecodeFrom method.
func (sr *SwitchRecord) DecodeFrom(raw *RawRecord, ev *Event) error {
	sr.RecordHeader = raw.Header
	f := raw.fields()
	f.idCond(ev.a.Options.SampleIDAll, &sr.SampleID, ev.a.SampleFormat)
	return nil
}

// switchOutBit is PERF_RECORD_MISC_SWITCH_OUT
const switchOutBit = 1 << 13

// switchOutPreemptBit is PERF_RECORD_MISC_SWITCH_OUT_PREEMPT
const switchOutPreemptBit = 1 << 14

// Out returns a boolean indicating whether the context switch was
// out of the current process, or into the current process.
func (sr *SwitchRecord) Out() bool {
	return sr.RecordHeader.Misc&switchOutBit != 0
}

// Preempted indicates whether the thread was preempted in TASK_RUNNING state.
func (sr *SwitchRecord) Preempted() bool {
	return sr.RecordHeader.Misc&switchOutPreemptBit != 0
}

// SwitchCPUWideRecord (PERF_RECORD_SWITCH_CPU_WIDE) indicates a context
// switch, but only occurs when sampling in CPU-wide mode. It provides
// information on the process being switched to / from.
type SwitchCPUWideRecord struct {
	RecordHeader
	Pid uint32
	Tid uint32
	SampleID
}

// DecodeFrom implements the Record.DecodeFrom method.
func (sr *SwitchCPUWideRecord) DecodeFrom(raw *RawRecord, ev *Event) error {
	sr.RecordHeader = raw.Header
	f := raw.fields()
	f.uint32(&sr.Pid, &sr.Tid)
	f.idCond(ev.a.Options.SampleIDAll, &sr.SampleID, ev.a.SampleFormat)
	return nil
}

// Out returns a boolean indicating whether the context switch was
// out of the current process, or into the current process.
func (sr *SwitchCPUWideRecord) Out() bool {
	return sr.RecordHeader.Misc&switchOutBit != 0
}

// Preempted indicates whether the thread was preempted in TASK_RUNNING state.
func (sr *SwitchCPUWideRecord) Preempted() bool {
	return sr.RecordHeader.Misc&switchOutPreemptBit != 0
}

// NamespacesRecord (PERF_RECORD_NAMESPACES) describes the namespaces of a
// process when it is created.
type NamespacesRecord struct {
	RecordHeader
	Pid        uint32
	Tid        uint32
	Namespaces []struct {
		Dev   uint64
		Inode uint64
	}
	SampleID
}

// TODO(acln): check out *_NS_INDEX in perf_event.h

// DecodeFrom implements the Record.DecodeFrom method.
func (nr *NamespacesRecord) DecodeFrom(raw *RawRecord, ev *Event) error {
	nr.RecordHeader = raw.Header
	f := raw.fields()
	f.uint32(&nr.Pid, &nr.Tid)
	var num uint64
	f.uint64(&num)
	nr.Namespaces = make([]struct{ Dev, Inode uint64 }, num)
	for i := 0; i < int(num); i++ {
		f.uint64(&nr.Namespaces[i].Dev)
		f.uint64(&nr.Namespaces[i].Inode)
	}
	f.idCond(ev.a.Options.SampleIDAll, &nr.SampleID, ev.a.SampleFormat)
	return nil
}

// Skid is an instruction pointer skid constraint.
type Skid int

// Supported Skid settings.
const (
	CanHaveArbitrarySkid Skid = 0
	MustHaveConstantSkid Skid = 1
	RequestedZeroSkid    Skid = 2
	MustHaveZeroSkid     Skid = 3
)

// BranchSampleFormat specifies what branches to include in a branch record.
type BranchSampleFormat struct {
	Privilege BranchSamplePrivilege
	Sample    BranchSample
}

func (b BranchSampleFormat) marshal() uint64 {
	return uint64(b.Privilege) | uint64(b.Sample)
}

// BranchSamplePrivilege speifies a branch sample privilege level. If a
// level is not set explicitly, the kernel will use the event's privilege
// level. Event and branch privilege levels do not have to match.
type BranchSamplePrivilege uint64

// Branch sample privilege values. Values should be |-ed together.
const (
	BranchPrivilegeUser       BranchSamplePrivilege = unix.PERF_SAMPLE_BRANCH_USER
	BranchPrivilegeKernel     BranchSamplePrivilege = unix.PERF_SAMPLE_BRANCH_KERNEL
	BranchPrivilegeHypervisor BranchSamplePrivilege = unix.PERF_SAMPLE_BRANCH_HV
)

// BranchSample specifies a type of branch to sample.
type BranchSample uint64

// Branch sample bits. Values should be |-ed together.
const (
	BranchSampleAny              BranchSample = unix.PERF_SAMPLE_BRANCH_ANY
	BranchSampleAnyCall          BranchSample = unix.PERF_SAMPLE_BRANCH_ANY_CALL
	BranchSampleAnyReturn        BranchSample = unix.PERF_SAMPLE_BRANCH_ANY_RETURN
	BranchSampleIndirectCall     BranchSample = unix.PERF_SAMPLE_BRANCH_IND_CALL
	BranchSampleAbortTransaction BranchSample = unix.PERF_SAMPLE_BRANCH_ABORT_TX
	BranchSampleInTransaction    BranchSample = unix.PERF_SAMPLE_BRANCH_IN_TX
	BranchSampleNoTransaction    BranchSample = unix.PERF_SAMPLE_BRANCH_NO_TX
	BranchSampleCond             BranchSample = unix.PERF_SAMPLE_BRANCH_COND
	BranchSampleCallStack        BranchSample = unix.PERF_SAMPLE_BRANCH_CALL_STACK
	BranchSampleIndirectJump     BranchSample = unix.PERF_SAMPLE_BRANCH_IND_JUMP
	BranchSampleCall             BranchSample = unix.PERF_SAMPLE_BRANCH_CALL
	BranchSampleNoFlags          BranchSample = unix.PERF_SAMPLE_BRANCH_NO_FLAGS
	BranchSampleNoCycles         BranchSample = unix.PERF_SAMPLE_BRANCH_NO_CYCLES
	BranchSampleSave             BranchSample = unix.PERF_SAMPLE_BRANCH_TYPE_SAVE
)

// DataSource records where in the memory hierarchy the data associated with
// a sampled instruction came from.
type DataSource uint64

// MemOp returns the recorded memory operation.
func (ds DataSource) MemOp() MemOp {
	return MemOp(ds >> memOpShift)
}

// MemLevel returns the recorded memory level.
func (ds DataSource) MemLevel() MemLevel {
	return MemLevel(ds >> memLevelShift)
}

// MemRemote returns the recorded remote bit.
func (ds DataSource) MemRemote() MemRemote {
	return MemRemote(ds >> memRemoteShift)
}

// MemLevelNumber returns the recorded memory level number.
func (ds DataSource) MemLevelNumber() MemLevelNumber {
	return MemLevelNumber(ds >> memLevelNumberShift)
}

// MemSnoopMode returns the recorded memory snoop mode.
func (ds DataSource) MemSnoopMode() MemSnoopMode {
	return MemSnoopMode(ds >> memSnoopModeShift)
}

// MemSnoopModeX returns the recorded extended memory snoop mode.
func (ds DataSource) MemSnoopModeX() MemSnoopModeX {
	return MemSnoopModeX(ds >> memSnoopModeXShift)
}

// MemLock returns the recorded memory lock mode.
func (ds DataSource) MemLock() MemLock {
	return MemLock(ds >> memLockShift)
}

// MemTLB returns the recorded TLB access mode.
func (ds DataSource) MemTLB() MemTLB {
	return MemTLB(ds >> memTLBShift)
}

// MemOp is a memory operation.
type MemOp uint8

// MemOp flag bits.
const (
	MemOpNA MemOp = 1 << iota
	MemOpLoad
	MemOpStore
	MemOpPrefetch
	MemOpExec

	memOpShift = 0
)

// MemLevel is a memory level.
type MemLevel uint32

// MemLevel flag bits.
const (
	MemLevelNA MemLevel = 1 << iota
	MemLevelHit
	MemLevelMiss
	MemLevelL1
	MemLevelLFB
	MemLevelL2
	MemLevelL3
	MemLevelLocalDRAM
	MemLevelRemoteDRAM1
	MemLevelRemoteDRAM2
	MemLevelRemoteCache1
	MemLevelRemoteCache2
	MemLevelIO
	MemLevelUncached

	memLevelShift = 5
)

// MemRemote indicates whether remote memory was accessed.
type MemRemote uint8

// MemRemote flag bits.
const (
	MemRemoteRemote MemRemote = 1 << iota

	memRemoteShift = 37
)

// MemLevelNumber is a memory level number.
type MemLevelNumber uint8

// MemLevelNumber flag bits.
const (
	MemLevelNumberL1 MemLevelNumber = iota
	MemLevelNumberL2
	MemLevelNumberL3
	MemLevelNumberL4

	MemLevelNumberAnyCache MemLevelNumber = iota + 0x0b
	MemLevelNumberLFB
	MemLevelNumberRAM
	MemLevelNumberPMem
	MemLevelNumberNA

	memLevelNumberShift = 33
)

// MemSnoopMode is a memory snoop mode.
type MemSnoopMode uint8

// MemSnoopMode flag bits.
const (
	MemSnoopModeNA MemSnoopMode = 1 << iota
	MemSnoopModeNone
	MemSnoopModeHit
	MemSnoopModeMiss
	MemSnoopModeHitModified

	memSnoopModeShift = 19
)

// MemSnoopModeX is an extended memory snoop mode.
type MemSnoopModeX uint8

// MemSnoopModeX flag bits.
const (
	MemSnoopModeXForward MemSnoopModeX = 0x01 // forward

	memSnoopModeXShift = 37
)

// MemLock is a memory locking mode.
type MemLock uint8

// MemLock flag bits.
const (
	MemLockNA     MemLock = 1 << iota // not available
	MemLockLocked                     // locked transaction

	memLockShift = 24
)

// MemTLB is a TLB access mode.
type MemTLB uint8

// MemTLB flag bits.
const (
	MemTLBNA   MemTLB = 1 << iota // not available
	MemTLBHit                     // hit level
	MemTLBMiss                    // miss level
	MemTLBL1
	MemTLBL2
	MemTLBWK // Hardware Walker
	MemTLBOS // OS fault handler

	memTLBShift = 26
)

// Transaction describes a transactional memory abort.
type Transaction uint64

// Transaction bits: values should be &-ed with Transaction values.
const (
	// Transaction Elision indicates an abort from an elision type
	// transaction (Intel CPU specific).
	TransactionElision Transaction = 1 << iota

	// TransactionGeneric indicates an abort from a generic transaction.
	TransactionGeneric

	// TransactionSync indicates a synchronous abort (related to the
	// reported instruction).
	TransactionSync

	// TransactionAsync indicates an asynchronous abort (unrelated to
	// the reported instruction).
	TransactionAsync

	// TransactionRetryable indicates whether retrying the transaction
	// may have succeeded.
	TransactionRetryable

	// TransactionConflict indicates an abort rue to memory conflicts
	// with other threads.
	TransactionConflict

	// TransactionWriteCapacity indicates an abort due to write capacity
	// overflow.
	TransactionWriteCapacity

	// TransactionReadCapacity indicates an abort due to read capacity
	// overflow.
	TransactionReadCapacity
)

// txnAbortMask is PERF_TXN_ABORT_MASK
const txnAbortMask = 0xffffffff

// txnAbortShift is PERF_TXN_ABORT_SHIFT
const txnAbortShift = 32

// UserAbortCode returns the user-specified abort code associated with
// the transaction.
func (txn Transaction) UserAbortCode() uint32 {
	return uint32((txn >> txnAbortShift) & txnAbortMask)
}

// TODO(acln): the latter part of this file is full of constants added
// ad-hoc, which use iota. These should probably be added to x/sys/unix
// instead, and used from there.
