package appdash

import (
	"fmt"
	"runtime"
)

// memStats collects and prints some formatted *runtime.MemStats fields.
type memStats struct {
	// Number of collections occurring.
	Collections int

	indent int
}

// fmtBytes returns b (in bytes) as a nice human readable string.
func (m *memStats) fmtBytes(b uint64) string {
	var (
		kb uint64 = 1024
		mb uint64 = kb * 1024
		gb uint64 = mb * 1024
	)
	if b < kb {
		return fmt.Sprintf("%dB", b)
	}
	if b < mb {
		return fmt.Sprintf("%dKB", b/kb)
	}
	if b < gb {
		return fmt.Sprintf("%dMB", b/mb)
	}
	return fmt.Sprintf("%dGB", b/gb)
}

// repeat returns s repeated N times consecutively.
func (m *memStats) repeat(s string, n int) string {
	var v string
	for i := 0; i < n; i++ {
		v += s
	}
	return v
}

// logf invokes fmt.Printf but with m.indent spaces prefixed.
func (m *memStats) logf(format string, args ...interface{}) {
	fmt.Printf("%s%s", m.repeat(" ", m.indent), fmt.Sprintf(format, args...))
}

// logColumns logs the given rows as formatted (properly indented) columns.
func (m *memStats) logColumns(rows ...[]interface{}) {
	columnWidths := make([]int, len(rows[0]))
	for column := range rows[0] {
		for row := 0; row < len(rows); row++ {
			w := len(fmt.Sprintf("%v", rows[row][column]))
			if w > columnWidths[column] {
				columnWidths[column] = w
			}
		}
	}

	for _, row := range rows {
		m.logf("- ")
		for c, column := range row {
			w := len(fmt.Sprintf("%v", column))
			fmt.Printf("%v  %s", column, m.repeat(" ", columnWidths[c]-w))
		}
		fmt.Printf("\n")
	}
}

// Log should be called with a human-readable segment name (i.e. the segment of
// code whose memory perf is being tested).
func (m *memStats) Log(segment string) {
	var s runtime.MemStats
	runtime.ReadMemStats(&s)
	m.logf("\n\n[%s] %d-collections:\n", segment, m.Collections)
	m.indent += 2

	m.logf("General statistics\n")
	m.indent += 2
	m.logColumns(
		[]interface{}{"Alloc", m.fmtBytes(s.Alloc), "(allocated and still in use)"},
		[]interface{}{"TotalAlloc", m.fmtBytes(s.TotalAlloc), "(allocated (even if freed))"},
		[]interface{}{"Sys", m.fmtBytes(s.Sys), "(obtained from system (sum of XxxSys below))"},
		[]interface{}{"Lookups", s.Lookups, "(number of pointer lookups)"},
		[]interface{}{"Mallocs", s.Mallocs, "(number of mallocs)"},
		[]interface{}{"Frees", s.Frees, "(number of frees)"},
	)
	m.indent -= 2
	fmt.Printf("\n")

	m.logf("Main allocation heap statistics\n")
	m.indent += 2
	m.logColumns(
		[]interface{}{"HeapAlloc", m.fmtBytes(s.HeapAlloc), "(allocated and still in use)"},
		[]interface{}{"HeapSys", m.fmtBytes(s.HeapSys), "(obtained from system)"},
		[]interface{}{"HeapIdle", m.fmtBytes(s.HeapIdle), "(in idle spans)"},
		[]interface{}{"HeapInuse", m.fmtBytes(s.HeapInuse), "(in non-idle span)"},
		[]interface{}{"HeapReleased", m.fmtBytes(s.HeapReleased), "(released to the OS)"},
		[]interface{}{"HeapObjects", s.HeapObjects, "(total number of allocated objects)"},
	)
	m.indent -= 2
	fmt.Printf("\n")

	m.logf("Low-level fixed-size structure allocator statistics\n")
	m.indent += 2
	m.logColumns(
		[]interface{}{"StackInuse", m.fmtBytes(s.StackInuse), "(used by stack allocator right now)"},
		[]interface{}{"StackSys", m.fmtBytes(s.StackSys), "(obtained from system)"},
		[]interface{}{"MSpanInuse", m.fmtBytes(s.MSpanInuse), "(mspan structures / in use now)"},
		[]interface{}{"MSpanSys", m.fmtBytes(s.MSpanSys), "(obtained from system)"},
		[]interface{}{"MCacheInuse", m.fmtBytes(s.MCacheInuse), "(in use now))"},
		[]interface{}{"MCacheSys", m.fmtBytes(s.MCacheSys), "(mcache structures / obtained from system)"},
		[]interface{}{"BuckHashSys", m.fmtBytes(s.BuckHashSys), "(profiling bucket hash table / obtained from system)"},
		[]interface{}{"GCSys", m.fmtBytes(s.GCSys), "(GC metadata / obtained form system)"},
		[]interface{}{"OtherSys", m.fmtBytes(s.OtherSys), "(other system allocations)"},
	)
	fmt.Printf("\n")
	m.indent -= 2

	// TODO(slimsag): remaining GC fields may be useful later:
	/*
	   // Garbage collector statistics.
	   NextGC       uint64 // next collection will happen when HeapAlloc ≥ this amount
	   LastGC       uint64 // end time of last collection (nanoseconds since 1970)
	   PauseTotalNs uint64
	   PauseNs      [256]uint64 // circular buffer of recent GC pause durations, most recent at [(NumGC+255)%256]
	   PauseEnd     [256]uint64 // circular buffer of recent GC pause end times
	   NumGC        uint32
	   EnableGC     bool
	   DebugGC      bool

	   // Per-size allocation statistics.
	   // 61 is NumSizeClasses in the C code.
	   BySize [61]struct {
	           Size    uint32
	           Mallocs uint64
	           Frees   uint64
	   }
	*/
}
