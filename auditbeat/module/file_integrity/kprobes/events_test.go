package kprobes

import (
	"testing"
)

func BenchmarkEventAllocation(b *testing.B) {
	var p *ProbeEvent
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 10000; j++ {
			p = &ProbeEvent{}
			_ = p
			p = &ProbeEvent{MaskMonitor: 1}
			_ = p
			p = &ProbeEvent{MaskDelete: 1}
			_ = p
		}
	}
	_ = p
}

func BenchmarkEventPool(b *testing.B) {
	var p any
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 10000; j++ {
			p = allocProbeEvent()
			_ = p
			releaseProbeEvent(p.(*ProbeEvent))
			p = allocMonitorProbeEvent()
			_ = p
			releaseProbeEvent(p.(*ProbeEvent))
			p = allocDeleteProbeEvent()
			_ = p
			releaseProbeEvent(p.(*ProbeEvent))
		}
	}
	_ = p
}
