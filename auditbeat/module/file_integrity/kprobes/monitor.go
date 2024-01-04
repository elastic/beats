package kprobes

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/elastic/beats/v7/auditbeat/module/file_integrity/kprobes/tracing"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/pkoutsovasilis/go-perf"
)

type MonitorEvent struct {
	Path string
	PID  uint32
	Op   uint32
}

type Monitor struct {
	eventC      chan MonitorEvent
	pathMonitor pathTraverser
	perfChannel *tracing.PerfChannel
	addC        chan string
	errC        chan error
	eProc       eventProcessor
	log         *logp.Logger
	ctx         context.Context
	cancelFn    context.CancelFunc
	isRecursive bool
}

func New(isRecursive bool) (*Monitor, error) {

	tfs, err := tracing.NewTraceFS()
	if err != nil {
		return nil, err
	}

	if err := tfs.RemoveAllKProbes(); err != nil {
		return nil, err
	}

	mCtx, cancelFn := context.WithCancel(context.TODO())

	validatedProbes, exec, err := getVerifiedProbes(mCtx, 5*time.Second)
	if err != nil {
		cancelFn()
		return nil, err
	}

	p, err := newPathMonitor(mCtx, exec, 0, isRecursive)
	if err != nil {
		cancelFn()
		return nil, err
	}

	channel, err := tracing.NewPerfChannel(
		tracing.WithTimestamp(),
		tracing.WithRingSizeExponent(10),
		tracing.WithBufferSize(4096),
		tracing.WithTID(perf.AllThreads),
		tracing.WithPollTimeout(100*time.Millisecond),
	)
	if err != nil {
		cancelFn()
		return nil, err
	}

	for probe, allocFn := range validatedProbes {
		err := tfs.AddKProbe(probe)
		if err != nil {
			cancelFn()
			return nil, err
		}
		desc, err := tfs.LoadProbeFormat(probe)
		if err != nil {
			cancelFn()
			return nil, err
		}

		decoder, err := tracing.NewStructDecoder(desc, allocFn)
		if err != nil {
			cancelFn()
			return nil, err
		}

		if err := channel.MonitorProbe(desc, decoder); err != nil {
			cancelFn()
			return nil, err
		}
	}

	monitor := &Monitor{
		eventC:      make(chan MonitorEvent, 1),
		pathMonitor: p,
		perfChannel: channel,
		addC:        make(chan string),
		errC:        make(chan error),
		log:         logp.NewLogger("file_integrity"),
		ctx:         mCtx,
		cancelFn:    cancelFn,
		isRecursive: isRecursive,
	}

	monitor.eProc = newEventProcessor(monitor.pathMonitor, monitor, monitor.isRecursive)
	return monitor, nil
}

func (w *Monitor) Emit(ePath string, TID uint32, op uint32) error {
	select {
	case <-w.ctx.Done():
		return w.ctx.Err()

	case w.eventC <- MonitorEvent{
		Path: ePath,
		PID:  TID,
		Op:   op,
	}:
		return nil
	}
}

func (w *Monitor) Add(path string) error {
	if w.ctx.Err() != nil {
		return w.ctx.Err()
	}

	return w.pathMonitor.AddPathToMonitor(w.ctx, path)
}

func (w *Monitor) Close() error {
	w.cancelFn()

	var allErr error
	allErr = errors.Join(allErr, w.pathMonitor.Close())
	allErr = errors.Join(allErr, w.perfChannel.Close())
	close(w.eventC)
	return allErr
}

func (w *Monitor) EventChannel() <-chan MonitorEvent {
	return w.eventC
}

func (w *Monitor) ErrorChannel() <-chan error {
	return w.errC
}

func (w *Monitor) writeErr(err error) {
	select {
	case w.errC <- err:
	case <-w.ctx.Done():
	}
}

func (w *Monitor) Start() error {
	if err := w.perfChannel.Run(); err != nil {
		if closeErr := w.Close(); closeErr != nil {
			w.log.Warnf("error at closing watcher: %v", closeErr)
		}
		return err
	}

	go func() {
		defer func() {
			if closeErr := w.Close(); closeErr != nil {
				w.log.Warnf("error at closing watcher: %v", closeErr)
			}
		}()

		for {
			select {
			case <-w.ctx.Done():
				return

			case e, ok := <-w.perfChannel.C():
				if !ok {
					w.writeErr(fmt.Errorf("read invalid event from perf channel"))
					return
				}

				switch eWithType := e.(type) {
				case *ProbeEvent:
					if err := w.eProc.process(w.ctx, eWithType); err != nil {
						w.writeErr(err)
						return
					}
					continue
				default:
					w.writeErr(errors.New("unexpected event type"))
					return
				}

			case err := <-w.perfChannel.ErrC():
				w.writeErr(err)
				return

			case lost := <-w.perfChannel.LostC():
				w.writeErr(fmt.Errorf("events lost %d", lost))
				return

			case err := <-w.pathMonitor.ErrC():
				w.writeErr(err)
				return
			}
		}
	}()

	return nil
}
