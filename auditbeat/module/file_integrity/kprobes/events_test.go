package kprobes

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_allocProbeEvents(t *testing.T) {
	p := allocProbeEvent()
	require.IsType(t, &ProbeEvent{}, p)

	releaseProbeEvent(nil)

	pE := p.(*ProbeEvent)
	require.Zero(t, pE.MaskMonitor)
	require.Zero(t, pE.MaskCreate)
	require.Zero(t, pE.MaskDelete)
	require.Zero(t, pE.MaskAttrib)
	require.Zero(t, pE.MaskModify)
	require.Zero(t, pE.MaskDir)
	require.Zero(t, pE.MaskMoveTo)
	require.Zero(t, pE.MaskMoveFrom)
	releaseProbeEvent(pE)

	p = allocDeleteProbeEvent()
	require.IsType(t, &ProbeEvent{}, p)

	pE = p.(*ProbeEvent)
	require.Zero(t, pE.MaskMonitor)
	require.Zero(t, pE.MaskCreate)
	require.Equal(t, pE.MaskDelete, uint32(1))
	require.Zero(t, pE.MaskAttrib)
	require.Zero(t, pE.MaskModify)
	require.Zero(t, pE.MaskDir)
	require.Zero(t, pE.MaskMoveTo)
	require.Zero(t, pE.MaskMoveFrom)
	releaseProbeEvent(pE)

	p = allocMonitorProbeEvent()
	require.IsType(t, &ProbeEvent{}, p)

	pE = p.(*ProbeEvent)
	require.Equal(t, pE.MaskMonitor, uint32(1))
	require.Zero(t, pE.MaskCreate)
	require.Zero(t, pE.MaskDelete)
	require.Zero(t, pE.MaskAttrib)
	require.Zero(t, pE.MaskModify)
	require.Zero(t, pE.MaskDir)
	require.Zero(t, pE.MaskMoveTo)
	require.Zero(t, pE.MaskMoveFrom)
	releaseProbeEvent(pE)
}

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
