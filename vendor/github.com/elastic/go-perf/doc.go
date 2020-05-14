// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package perf provides access to the Linux perf API.

Counting events

A Group represents a set of perf events measured together.

	var g perf.Group
	g.Add(perf.Instructions, perf.CPUCycles)

	hw, err := g.Open(targetpid, perf.AnyCPU)
	// ...
	gc, err := hw.MeasureGroup(func() { ... })

Attr configures an individual event.

	fa := &perf.Attr{
		CountFormat: perf.CountFormat{
			Running: true,
			ID:      true,
		},
	}
	perf.PageFaults.Configure(fa)

	faults, err := perf.Open(fa, perf.CallingThread, perf.AnyCPU, nil)
	// ...
	c, err := faults.Measure(func() { ... })

Sampling events

Overflow records are available once the MapRing method on Event is called:

	var ev perf.Event // initialized previously

	ev.MapRing()

	ev.Enable()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	for {
		rec, err := ev.ReadRecord(ctx)
		// process rec
	}

Tracepoints are also supported:

	wa := &perf.Attr{
		SampleFormat: perf.SampleFormat{
			Pid: true,
			Tid: true,
			IP:  true,
		},
	}
	wa.SetSamplePeriod(1)
	wa.SetWakeupEvents(1)
	wtp := perf.Tracepoint("syscalls", "sys_enter_write")
	wtp.Configure(wa)

	writes, err := perf.Open(wa, targetpid, perf.AnyCPU, nil)
	// ...
	c, err := writes.Measure(func() { ... })
	// ...
	fmt.Printf("saw %d writes\n", c.Value)

	rec, err := writes.ReadRecord(ctx)
	// ...
	sr, ok := rec.(*perf.SampleRecord)
	// ...
	fmt.Printf("pid = %d, tid = %d\n", sr.Pid, sr.Tid)

For more detailed information, see the examples, and man 2 perf_event_open.

NOTE: this package is experimental and does not yet offer compatibility
guarantees.
*/
package perf
