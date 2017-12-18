// +build darwin

// Package fsevents provides file system notifications on macOS.
package fsevents

import (
	"sync"
	"syscall"
	"time"
)

// CreateFlags for creating a New stream.
type CreateFlags uint32

// kFSEventStreamCreateFlag...
const (
	// use CoreFoundation types instead of raw C types (disabled)
	useCFTypes CreateFlags = 1 << iota

	// NoDefer sends events on the leading edge (for interactive applications).
	// By default events are delivered after latency seconds (for background tasks).
	NoDefer

	// WatchRoot for a change to occur to a directory along the path being watched.
	WatchRoot

	// IgnoreSelf doesn't send events triggered by the current process (macOS 10.6+).
	IgnoreSelf

	// FileEvents sends events about individual files, generating significantly
	// more events (macOS 10.7+) than directory level notifications.
	FileEvents
)

// EventFlags passed to the FSEventStreamCallback function.
type EventFlags uint32

// kFSEventStreamEventFlag...
const (
	// MustScanSubDirs indicates that events were coalesced hierarchically.
	MustScanSubDirs EventFlags = 1 << iota
	// UserDropped or KernelDropped is set alongside MustScanSubDirs
	// to help diagnose the problem.
	UserDropped
	KernelDropped

	// EventIDsWrapped indicates the 64-bit event ID counter wrapped around.
	EventIDsWrapped

	// HistoryDone is a sentinel event when retrieving events sinceWhen.
	HistoryDone

	// RootChanged indicates a change to a directory along the path being watched.
	RootChanged

	// Mount for a volume mounted underneath the path being monitored.
	Mount
	// Unmount event occurs after a volume is unmounted.
	Unmount

	// The following flags are only set when using FileEvents.

	ItemCreated
	ItemRemoved
	ItemInodeMetaMod
	ItemRenamed
	ItemModified
	ItemFinderInfoMod
	ItemChangeOwner
	ItemXattrMod
	ItemIsFile
	ItemIsDir
	ItemIsSymlink
)

// Event represents a single file system notification.
type Event struct {
	Path  string
	Flags EventFlags
	ID    uint64
}

// DeviceForPath returns the device ID for the specified volume.
func DeviceForPath(path string) (int32, error) {
	stat := syscall.Stat_t{}
	if err := syscall.Lstat(path, &stat); err != nil {
		return 0, err
	}
	return stat.Dev, nil
}

// EventStream is the primary interface to FSEvents
// You can provide your own event channel if you wish (or one will be
// created on Start).
//
//   es := &EventStream{Paths: []string{"/tmp"}, Flags: 0}
//   es.Start()
//   es.Stop()
//   ...
type EventStream struct {
	stream       FSEventStreamRef
	rlref        CFRunLoopRef
	hasFinalizer bool
	registryID   uintptr
	uuid         string

	Events  chan []Event
	Paths   []string
	Flags   CreateFlags
	EventID uint64
	Resume  bool
	Latency time.Duration
	// syscall represents this with an int32
	Device int32
}

// eventStreamRegistry is a lookup table for EventStream references passed to
// cgo. In Go 1.6+ passing a Go pointer to a Go pointer to cgo is not allowed.
// To get around this issue, we pass only an integer.
type eventStreamRegistry struct {
	sync.Mutex
	m      map[uintptr]*EventStream
	lastID uintptr
}

var registry = eventStreamRegistry{m: map[uintptr]*EventStream{}}

func (r *eventStreamRegistry) Add(e *EventStream) uintptr {
	r.Lock()
	defer r.Unlock()

	r.lastID++
	r.m[r.lastID] = e
	return r.lastID
}

func (r *eventStreamRegistry) Get(i uintptr) *EventStream {
	r.Lock()
	defer r.Unlock()

	return r.m[i]
}

func (r *eventStreamRegistry) Delete(i uintptr) {
	r.Lock()
	defer r.Unlock()

	delete(r.m, i)
}

// Start listening to an event stream.
func (es *EventStream) Start() {
	if es.Events == nil {
		es.Events = make(chan []Event)
	}

	// register eventstream in the local registry for later lookup
	// in C callback
	cbInfo := registry.Add(es)
	es.registryID = cbInfo
	if es.Device != 0 {
		es.uuid = GetDeviceUUID(es.Device)
	}
	es.start(es.Paths, cbInfo)
}

// Flush events that have occurred but haven't been delivered.
func (es *EventStream) Flush(sync bool) {
	flush(es.stream, sync)
}

// Stop listening to the event stream.
func (es *EventStream) Stop() {
	if es.stream != nil {
		stop(es.stream, es.rlref)
		es.stream = nil
	}

	// Remove eventstream from the registry
	registry.Delete(es.registryID)
	es.registryID = 0
}

// Restart listening.
func (es *EventStream) Restart() {
	es.Stop()
	es.Resume = true
	es.Start()
}
