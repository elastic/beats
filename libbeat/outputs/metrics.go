package outputs

import "github.com/elastic/beats/libbeat/monitoring"

// Stats provides a common type used by outputs to report common events.
// The output events will update a set of unified output metrics in the
// underlying monitoring.Registry.
type Stats struct {
	//
	// Output event stats
	//
	batches *monitoring.Uint // total number of batches processed by output
	events  *monitoring.Uint // total number of events processed by output

	acked  *monitoring.Uint // total number of events ACKed by output
	failed *monitoring.Uint // total number of events failed in output
	active *monitoring.Uint // events sent and waiting for ACK/fail from output

	//
	// Output network connection stats
	//
	writeBytes  *monitoring.Uint // total amount of bytes written by output
	writeErrors *monitoring.Uint // total number of errors on write

	readBytes  *monitoring.Uint // total amount of bytes read
	readErrors *monitoring.Uint // total number of errors while waiting for response on output
}

func MakeStats(reg *monitoring.Registry) Stats {
	return Stats{
		batches: monitoring.NewUint(reg, "events.batches"),
		events:  monitoring.NewUint(reg, "events.total"),
		acked:   monitoring.NewUint(reg, "events.acked"),
		failed:  monitoring.NewUint(reg, "events.failed"),
		active:  monitoring.NewUint(reg, "events.active"),

		writeBytes:  monitoring.NewUint(reg, "write.bytes"),
		writeErrors: monitoring.NewUint(reg, "write.errors"),

		readBytes:  monitoring.NewUint(reg, "read.bytes"),
		readErrors: monitoring.NewUint(reg, "read.errors"),
	}
}

func (s *Stats) NewBatch(n int) {
	if s != nil {
		s.batches.Inc()
		s.events.Add(uint64(n))
		s.active.Add(uint64(n))
	}
}

func (s *Stats) Acked(n int) {
	if s != nil {
		s.acked.Add(uint64(n))
		s.active.Sub(uint64(n))
	}
}

func (s *Stats) Failed(n int) {
	if s != nil {
		s.failed.Add(uint64(n))
		s.active.Sub(uint64(n))
	}
}

func (s *Stats) Dropped(n int) {
	// number of dropped events (e.g. encoding failures)
	if s != nil {
		s.active.Sub(uint64(n))
	}
}

func (s *Stats) Cancelled(n int) {
	if s != nil {
		s.active.Sub(uint64(n))
	}
}

func (s *Stats) WriteError() {
	if s != nil {
		s.writeErrors.Inc()
	}
}

func (s *Stats) WriteBytes(n int) {
	if s != nil {
		s.writeBytes.Add(uint64(n))
	}
}

func (s *Stats) ReadError() {
	if s != nil {
		s.readErrors.Inc()
	}
}

func (s *Stats) ReadBytes(n int) {
	if s != nil {
		s.readBytes.Add(uint64(n))
	}
}
