// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux
// +build linux

package tracing

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/joeshaw/multierror"
	"golang.org/x/sys/unix"

	"github.com/menderesk/go-perf"
)

var (
	// ErrUnsupported error indicates that perf_event_open is not available
	// in the current kernel.
	ErrUnsupported = errors.New("perf_event_open is not supported by this kernel")

	// ErrAlreadyRunning error is returned when a PerfChannel has already
	// started after a call to run.
	ErrAlreadyRunning = errors.New("channel already running")

	// ErrNotRunning error is returned by PerfChannel#Close when it has not been
	// started.
	ErrNotRunning = errors.New("channel not running")
)

type stream struct {
	decoder Decoder
	probeID int
}

// PerfChannel represents a channel to receive perf events.
type PerfChannel struct {
	done    chan struct{}
	sampleC chan interface{}
	errC    chan error
	lostC   chan uint64

	// one perf.Event per CPU
	events  []*perf.Event
	streams map[uint64]stream

	running uintptr
	wg      sync.WaitGroup
	cpus    CPUSet

	// Settings
	attr        perf.Attr
	mappedPages int
	pid         int
	pollTimeout time.Duration
	sizeSampleC int
	sizeErrC    int
	sizeLostC   int
	withTime    bool
}

// PerfChannelConf instances change the configuration of a perf channel.
type PerfChannelConf func(*PerfChannel) error

// Metadata struct contains the information stored in a trace event header.
type Metadata struct {
	StreamID  uint64
	CPU       uint64
	Timestamp uint64
	TID       uint32
	PID       uint32
	EventID   int
}

// NewPerfChannel creates a new perf channel in order to receive events from
// one or more probes.
func NewPerfChannel(cfg ...PerfChannelConf) (channel *PerfChannel, err error) {
	if !perf.Supported() {
		return nil, ErrUnsupported
	}

	// Defaults
	channel = &PerfChannel{
		sizeSampleC: 1024,
		sizeErrC:    8,
		sizeLostC:   64,
		mappedPages: 64,
		pollTimeout: time.Millisecond * 200,
		done:        make(chan struct{}, 0),
		streams:     make(map[uint64]stream),
		pid:         perf.AllThreads,
		attr: perf.Attr{
			Type:    perf.TracepointEvent,
			ClockID: unix.CLOCK_MONOTONIC,
			SampleFormat: perf.SampleFormat{
				Raw:      true,
				StreamID: true,
				Tid:      true,
				CPU:      true,
			},
		},
	}
	channel.attr.SetSamplePeriod(1)
	channel.attr.SetWakeupEvents(1)

	// Load the list of online CPUs from /sys/devices/system/cpu/online.
	// This is necessary in order to to install each kprobe on all online CPUs.
	//
	// Note:
	// There's currently no mechanism to adapt to CPUs being added or removed
	// at runtime (CPU hotplug).
	channel.cpus, err = NewCPUSetFromFile(OnlineCPUsPath)
	if err != nil {
		return nil, fmt.Errorf("error listing online CPUs: %w", err)
	}
	if channel.cpus.NumCPU() < 1 {
		return nil, errors.New("couldn't list online CPUs")
	}
	// Set configuration
	for _, fun := range cfg {
		if err = fun(channel); err != nil {
			return nil, err
		}
	}
	return channel, nil
}

// WithBufferSize configures the capacity of the channel used to pass tracing
// events (PerfChannel.C())
func WithBufferSize(size int) PerfChannelConf {
	return func(channel *PerfChannel) error {
		if size < 0 {
			return fmt.Errorf("bad size for sample channel: %d", size)
		}
		channel.sizeSampleC = size
		return nil
	}
}

// WithErrBufferSize configures the capacity of the channel used to pass errors.
// (PerfChannel.ErrC())
func WithErrBufferSize(size int) PerfChannelConf {
	return func(channel *PerfChannel) error {
		if size < 0 {
			return fmt.Errorf("bad size for err channel: %d", size)
		}
		channel.sizeErrC = size
		return nil
	}
}

// WithLostBufferSize configures the capacity of the channel used to pass lost
// event notifications (PerfChannel.LostC()).
func WithLostBufferSize(size int) PerfChannelConf {
	return func(channel *PerfChannel) error {
		if size < 0 {
			return fmt.Errorf("bad size for lost channel: %d", size)
		}
		channel.sizeLostC = size
		return nil
	}
}

// WithRingSizeExponent configures the size, in pages, of the ringbuffers used
// by the kernel to pass events to userspace. The final size will be 2^exp.
// There is one ringbuffer per CPU.
func WithRingSizeExponent(exp int) PerfChannelConf {
	return func(channel *PerfChannel) error {
		if exp < 0 || exp > 18 {
			return fmt.Errorf("bad exponent for ring buffer: %d", exp)
		}
		channel.mappedPages = 1 << uint(exp)
		return nil
	}
}

// WithTID configures the thread ID to monitor.
// By default it is `perf.AllThreads`, which means all running threads will be
// monitored. With this option the monitoring can be limited to a single thread.
func WithTID(pid int) PerfChannelConf {
	return func(channel *PerfChannel) error {
		if pid < -1 {
			return fmt.Errorf("bad thread ID (TID): %d", pid)
		}
		channel.pid = pid
		return nil
	}
}

// WithTimestamp enables the returned tracing events to be timestamped.
// This uses an internal kernel clock.
func WithTimestamp() PerfChannelConf {
	return func(channel *PerfChannel) error {
		channel.attr.SampleFormat.Time = true
		return nil
	}
}

// WithPollTimeout configures for how long the reader thread can block waiting
// for events. A higher value will use less CPU. A smaller value will cause
// the thread to respond faster to termination (Close() will return sooner)
// in exchange for using more CPU.
func WithPollTimeout(timeout time.Duration) PerfChannelConf {
	return func(channel *PerfChannel) error {
		channel.pollTimeout = timeout
		return nil
	}
}

// MonitorProbe associates a probe with the PerfChannel, so that events
// generated by this probe will be received. A probe is identified by its
// ProbeFormat. The Decoder is used to decode events from this probe and
// will determine the types and contents of the returned events.
func (c *PerfChannel) MonitorProbe(format ProbeFormat, decoder Decoder) error {
	c.attr.Config = uint64(format.ID)
	doGroup := len(c.events) > 0
	cpuList := c.cpus.AsList()
	for idx, cpu := range cpuList {
		var group *perf.Event
		var flags int
		if doGroup {
			group = c.events[idx]
			flags = unix.PERF_FLAG_FD_NO_GROUP | unix.PERF_FLAG_FD_OUTPUT
		}
		ev, err := perf.OpenWithFlags(&c.attr, c.pid, cpu, group, flags)
		if err != nil {
			return err
		}
		cid, err := ev.ID()
		if err != nil {
			return err
		}
		if len(format.Probe.Filter) > 0 {
			fd, err := ev.FD()
			if err != nil {
				return err
			}
			fbytes := []byte(format.Probe.Filter + "\x00")
			_, _, errNo := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), unix.PERF_EVENT_IOC_SET_FILTER, uintptr(unsafe.Pointer(&fbytes[0])))
			if errNo != 0 {
				return fmt.Errorf("unable to set filter '%s': %w", format.Probe.Filter, errNo)
			}
		}
		c.streams[cid] = stream{probeID: format.ID, decoder: decoder}
		c.events = append(c.events, ev)

		if !doGroup {
			if err := ev.MapRingNumPages(c.mappedPages); err != nil {
				return fmt.Errorf("perf channel mapring failed: %w", err)
			}
		}
	}
	return nil
}

// C returns the channel to read samples from.
func (c *PerfChannel) C() <-chan interface{} {
	return c.sampleC
}

// ErrC returns the channel to read errors from.
func (c *PerfChannel) ErrC() <-chan error {
	return c.errC
}

// LostC returns the channel to read lost samples notifications.
func (c *PerfChannel) LostC() <-chan uint64 {
	return c.lostC
}

// Run enables the configured probe and starts receiving perf events.
// sampleC is the channel where decoded perf events are received.
// errC is the channel where errors are received.
//
// The format of the received events depends on the Decoder used.
func (c *PerfChannel) Run() error {
	if !atomic.CompareAndSwapUintptr(&c.running, 0, 1) {
		return ErrAlreadyRunning
	}
	c.sampleC = make(chan interface{}, c.sizeSampleC)
	c.errC = make(chan error, c.sizeErrC)
	c.lostC = make(chan uint64, c.sizeLostC)

	for _, ev := range c.events {
		if err := ev.Enable(); err != nil {
			return fmt.Errorf("perf channel enable failed: %w", err)
		}
	}
	c.wg.Add(1)
	go c.channelLoop()
	return nil
}

// Close closes the channel.
func (c *PerfChannel) Close() error {
	if atomic.CompareAndSwapUintptr(&c.running, 1, 2) {
		close(c.done)
		c.wg.Wait()
		defer close(c.sampleC)
		defer close(c.errC)
		defer close(c.lostC)
	}
	var errs multierror.Errors
	for _, ev := range c.events {
		if err := ev.Disable(); err != nil {
			errs = append(errs, fmt.Errorf("failed to disable event channel: %w", err))
		}
		if err := ev.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close event channel: %w", err))
		}
	}
	return errs.Err()
}

// doneWrapperContext is a custom context.Context that is tailored to
// perf.Event.ReadRawRecord needs. It's used to avoid an expensive allocation
// before each call to ReadRawRecord while providing termination when
// the wrapped channel closes.
type doneWrapperContext <-chan struct{}

// Deadline always returns no deadline.
func (ctx doneWrapperContext) Deadline() (deadline time.Time, ok bool) {
	// No deadline
	return deadline, false
}

// Done returns the underlying done channel.
func (ctx doneWrapperContext) Done() <-chan struct{} {
	return ctx
}

// Err returns context.Canceled if the underlying done channel is closed.
func (ctx doneWrapperContext) Err() error {
	select {
	case <-ctx.Done():
		return context.Canceled
	default:
	}
	return nil
}

// Value always returns nil.
func (ctx doneWrapperContext) Value(key interface{}) interface{} {
	return nil
}

func makeMetadata(eventID int, record *perf.SampleRecord) Metadata {
	return Metadata{
		StreamID:  record.StreamID,
		Timestamp: record.Time,
		TID:       record.Tid,
		PID:       record.Pid,
		EventID:   eventID,
	}
}

func (c *PerfChannel) channelLoop() {
	defer c.wg.Done()
	ctx := doneWrapperContext(c.done)
	merger := newRecordMerger(c.events[:c.cpus.NumCPU()], c, c.pollTimeout)
	for {
		// Read the available event from all the monitored ring-buffers that
		// has the smallest timestamp.
		sample, ok := merger.nextSample(ctx)
		if !ok {
			// Close() called.
			return
		}
		// Locate the decoder associated to the source stream.
		stream := c.streams[sample.StreamID]
		if stream.decoder == nil {
			c.errC <- fmt.Errorf("no decoder for stream:%d", sample.StreamID)
			continue
		}
		// Decode the event
		meta := makeMetadata(stream.probeID, sample)
		output, err := stream.decoder.Decode(sample.Raw, meta)
		if err != nil {
			c.errC <- err
			continue
		}
		c.sampleC <- output
	}
}

// A recordMerger is used to read from a number of ring-buffers while trying to
// maintain the returned events in sorted order (by their Timestamp).
//
// As each individual ring-buffer is (usually) sorted, it's possible to read
// from them in order using a merge algorithm.
type recordMerger struct {
	evs     []*perf.Event
	records []*perf.SampleRecord
	channel *PerfChannel
	timeout time.Duration
}

func newRecordMerger(sources []*perf.Event, channel *PerfChannel, pollTimeout time.Duration) recordMerger {
	m := recordMerger{
		evs:     sources,
		channel: channel,
		timeout: pollTimeout,
	}
	m.records = make([]*perf.SampleRecord, len(sources))
	return m
}

// Reads the next in-order sample, blocking if necessary.
func (m *recordMerger) nextSample(ctx context.Context) (sr *perf.SampleRecord, ok bool) {
	for {
		// Return if the done channel is closed.
		select {
		case <-ctx.Done():
			return nil, false
		default:
		}
		// Fill the records slice with the oldest sample in each ring-buffer,
		// or nil if that ring-buffer is empty.
		// Selects the oldest sample that is available, if any.
		var selIdx int
		for i := 0; i < len(m.records); i++ {
			if m.records[i] == nil {
				if m.records[i], ok = m.readSampleNonBlock(m.evs[i], ctx); !ok {
					return nil, false
				}
			}
			if m.records[i] != nil && (sr == nil || sr.Time > m.records[i].Time) {
				sr = m.records[i]
				selIdx = i
			}
		}
		// If a sample is available, remove it from records and return it.
		if sr != nil {
			m.records[selIdx] = nil
			return sr, true
		}
		// No sample was available. Block until one of the ringbuffers has data.
		_, closed, err := pollAll(m.evs, m.timeout)
		if err != nil {
			m.channel.errC <- fmt.Errorf("poll failed: %w", err)
			return nil, false
		}
		// Some of the ring buffers closed. Report termination.
		if closed > 0 {
			return nil, false
		}
	}
}

func (m *recordMerger) readSampleNonBlock(ev *perf.Event, ctx context.Context) (sr *perf.SampleRecord, ok bool) {
	for ev.HasRecord() {
		rec, err := ev.ReadRecord(ctx)
		if ctx.Err() != nil {
			return nil, false
		}
		if err != nil {
			if err == perf.ErrBadRecord {
				m.channel.lostC <- ^uint64(0)
				continue
			}
			m.channel.errC <- err
			return nil, false
		}
		h := rec.Header()
		switch h.Type {
		case unix.PERF_RECORD_LOST:
			lost, ok := rec.(*perf.LostRecord)
			if !ok {
				m.channel.errC <- errors.New("PERF_RECORD_LOST is not a *perf.LostRecord")
				return nil, false
			}
			m.channel.lostC <- lost.Lost
			continue

		case unix.PERF_RECORD_SAMPLE:
			sample, ok := rec.(*perf.SampleRecord)
			if !ok {
				m.channel.errC <- errors.New("PERF_RECORD_SAMPLE is not a *perf.SampleRecord")
				return nil, false
			}
			return sample, true
		}
	}
	return nil, true
}

func pollAll(evs []*perf.Event, timeout time.Duration) (active int, closed int, err error) {
	pollfds := make([]unix.PollFd, len(evs))
	for idx, ev := range evs {
		fd, err := ev.FD()
		if err != nil {
			return 0, 0, errors.New("failed to get descriptor for perf event channel")
		}
		pollfds[idx] = unix.PollFd{Fd: int32(fd), Events: unix.POLLIN}
	}
	ts := unix.NsecToTimespec(timeout.Nanoseconds())

	for err = unix.EINTR; err == unix.EINTR; {
		_, err = unix.Ppoll(pollfds, &ts, nil)
	}
	if err != nil {
		return 0, 0, os.NewSyscallError("poll", err)
	}

	for _, fd := range pollfds {
		if fd.Revents&unix.POLLIN != 0 {
			active++
		}
		if fd.Revents&unix.POLLHUP != 0 {
			closed++
		}
	}
	return
}
