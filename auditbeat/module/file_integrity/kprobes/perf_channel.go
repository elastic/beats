package kprobes

import (
	"time"

	"github.com/elastic/beats/v7/auditbeat/module/file_integrity/kprobes/tracing"
)

type perfChannel interface {
	C() <-chan interface{}
	ErrC() <-chan error
	LostC() <-chan uint64
	Run() error
	Close() error
}

func newPerfChannel(probes map[tracing.Probe]tracing.AllocateFn, ringSizeExponent int, bufferSize int, pid int) (*tracing.PerfChannel, error) {
	tfs, err := tracing.NewTraceFS()
	if err != nil {
		return nil, err
	}

	pChannel, err := tracing.NewPerfChannel(
		tracing.WithTimestamp(),
		tracing.WithRingSizeExponent(ringSizeExponent),
		tracing.WithBufferSize(bufferSize),
		tracing.WithTID(pid),
		tracing.WithPollTimeout(200*time.Millisecond),
	)
	if err != nil {
		return nil, err
	}

	for probe, allocFn := range probes {
		_ = tfs.RemoveKProbe(probe)

		err := tfs.AddKProbe(probe)
		if err != nil {
			return nil, err
		}
		desc, err := tfs.LoadProbeFormat(probe)
		if err != nil {
			return nil, err
		}

		decoder, err := tracing.NewStructDecoder(desc, allocFn)
		if err != nil {
			return nil, err
		}

		if err := pChannel.MonitorProbe(desc, decoder); err != nil {
			return nil, err
		}
	}

	return pChannel, nil
}
