// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build linux

package perf

import (
	"errors"
	"fmt"
	"io"
	"os"
	"text/tabwriter"
	"time"

	"golang.org/x/sys/unix"
)

// Count is a measurement taken by an Event.
//
// The Value field is always present and populated.
//
// The Enabled field is populated if CountFormat.Enabled is set on the Event
// the Count was read from. Ditto for TimeRunning and ID.
//
// Label is set based on the Label field of the Attr associated with the
// event. See the documentation there for more details.
type Count struct {
	Value   uint64
	Enabled time.Duration
	Running time.Duration
	ID      uint64
	Label   string
}

func (c Count) String() string {
	if c.Label != "" {
		return fmt.Sprintf("%s = %d", c.Label, c.Value)
	}
	return fmt.Sprint(c.Value)
}

var errGroup = errors.New("calling ReadCount on group Event")

// ReadCount reads the measurement associated with ev. If the Event was
// configured with CountFormat.Group, ReadCount returns an error.
func (ev *Event) ReadCount() (Count, error) {
	var c Count
	if err := ev.ok(); err != nil {
		return c, err
	}
	if ev.a.CountFormat.Group {
		return c, errGroup
	}

	// TODO(acln): on x86, the rdpmc instruction can be used here,
	// instead of read(2), to reduce the number of system calls, and
	// improve the accuracy of measurements.
	//
	// Investigate this. It seems like this functionality may not always
	// be available, even on x86, but we can check for it explicitly
	// if the ring associated with ev is mapped into memory: see
	// cap_user_rdpmc on perf_event_mmap_page.
	buf := make([]byte, ev.a.CountFormat.readSize())
	_, err := unix.Read(ev.perffd, buf)
	if err != nil {
		return c, os.NewSyscallError("read", err)
	}

	f := fields(buf)
	f.count(&c, ev.a.CountFormat)
	c.Label = ev.a.Label

	return c, err
}

// GroupCount is a group of measurements taken by an Event group.
//
// Fields are populated as described in the Count documentation.
type GroupCount struct {
	Enabled time.Duration
	Running time.Duration
	Values  []struct {
		Value uint64
		ID    uint64
		Label string
	}
}

type errWriter struct {
	w   io.Writer
	err error // sticky
}

func (ew *errWriter) Write(b []byte) (int, error) {
	if ew.err != nil {
		return 0, ew.err
	}
	n, err := ew.w.Write(b)
	ew.err = err
	return n, err
}

// PrintValues prints a table of gc.Values to w.
func (gc GroupCount) PrintValues(w io.Writer) error {
	ew := &errWriter{w: w}

	tw := new(tabwriter.Writer)
	tw.Init(ew, 0, 8, 1, ' ', 0)

	if gc.Values[0].ID != 0 {
		fmt.Fprintln(tw, "label\tvalue\tID")
	} else {
		fmt.Fprintln(tw, "label\tvalue")
	}

	for _, v := range gc.Values {
		if v.ID != 0 {
			fmt.Fprintf(tw, "%s\t%d\t%d\n", v.Label, v.Value, v.ID)
		} else {
			fmt.Fprintf(tw, "%s\t%d\n", v.Label, v.Value)
		}
	}

	tw.Flush()
	return ew.err
}

var errNotGroup = errors.New("calling ReadGroupCount on non-group Event")

// ReadGroupCount reads the measurements associated with ev. If the Event
// was not configued with CountFormat.Group, ReadGroupCount returns an error.
func (ev *Event) ReadGroupCount() (GroupCount, error) {
	var gc GroupCount
	if err := ev.ok(); err != nil {
		return gc, err
	}
	if !ev.a.CountFormat.Group {
		return gc, errNotGroup
	}

	size := ev.a.CountFormat.groupReadSize(1 + len(ev.group))
	buf := make([]byte, size)
	_, err := unix.Read(ev.perffd, buf)
	if err != nil {
		return gc, os.NewSyscallError("read", err)
	}

	f := fields(buf)
	f.groupCount(&gc, ev.a.CountFormat)
	gc.Values[0].Label = ev.a.Label
	for i := 0; i < len(ev.group); i++ {
		gc.Values[i+1].Label = ev.group[i].a.Label
	}

	return gc, nil
}

// CountFormat configures the format of Count or GroupCount measurements.
//
// Enabled and Running configure the Event to include time enabled and
// time running measurements to the counts. Usually, these two values are
// equal. They may differ when events are multiplexed.
//
// If ID is set, a unique ID is assigned to the associated event. For a
// given event, this ID matches the ID reported by the (*Event).ID method.
//
// If Group is set, the Event measures a group of events together: callers
// must use ReadGroupCount. If Group is not set, the Event measures a single
// counter: callers must use ReadCount.
type CountFormat struct {
	Enabled bool
	Running bool
	ID      bool
	Group   bool
}

// readSize returns the buffer size required for a Count read. Assumes
// f.Group is not set.
func (f CountFormat) readSize() int {
	size := 8 // value is always set
	if f.Enabled {
		size += 8
	}
	if f.Running {
		size += 8
	}
	if f.ID {
		size += 8
	}
	return size
}

// groupReadSize returns the buffer size required for a GroupCount read.
// Assumes f.Group is set.
func (f CountFormat) groupReadSize(events int) int {
	hsize := 8 // the number of events is always set
	if f.Enabled {
		hsize += 8
	}
	if f.Running {
		hsize += 8
	}
	vsize := 8 // each event contains at least a value
	if f.ID {
		vsize += 8
	}
	return hsize + events*vsize
}

// marshal marshals the CountFormat into a uint64.
func (f CountFormat) marshal() uint64 {
	// Always keep this in sync with the type definition above.
	fields := []bool{
		f.Enabled,
		f.Running,
		f.ID,
		f.Group,
	}
	return marshalBitwiseUint64(fields)
}
