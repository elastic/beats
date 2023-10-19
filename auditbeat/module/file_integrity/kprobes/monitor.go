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
	done        chan bool
	pathMonitor pathTraverser
	perfChannel *tracing.PerfChannel
	addC        chan string
	errC        chan error
	log         *logp.Logger
	ctx         context.Context
	cancelFn    context.CancelFunc

	isExcludedPath func(path string) bool
}

func New(isRecursive bool, IsExcludedPath func(path string) bool) (*Monitor, error) {

	mCtx, cancelFn := context.WithCancel(context.TODO())

	tfs, err := tracing.NewTraceFS()
	if err != nil {
		return nil, err
	}

	if err := tfs.RemoveAllKProbes(); err != nil {
		return nil, err
	}

	validatedProbes, exec, err := getVerifiedProbes(context.TODO(), 5*time.Second)
	if err != nil {
		return nil, err
	}

	p, err := newPathMonitor(context.TODO(), exec, 0, isRecursive)
	if err != nil {
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
		return nil, err
	}

	for probe, allocFn := range validatedProbes {
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

		if err := channel.MonitorProbe(desc, decoder); err != nil {
			return nil, err
		}
	}

	return &Monitor{
		eventC:         make(chan MonitorEvent, 1),
		done:           nil,
		pathMonitor:    p,
		perfChannel:    channel,
		addC:           make(chan string),
		errC:           make(chan error),
		log:            logp.NewLogger("file_integrity"),
		ctx:            mCtx,
		cancelFn:       cancelFn,
		isExcludedPath: IsExcludedPath,
	}, nil
}

func (w *Monitor) Emit(ePath string, TID uint32, op uint32) error {
	for {
		select {
		case <-w.done:
			return nil

		case w.eventC <- MonitorEvent{
			Path: ePath,
			PID:  TID,
			Op:   op,
		}:
			return nil
		}
	}
}

func (w *Monitor) Add(path string) error {
	if w.done == nil {
		return nil
	}

	return w.pathMonitor.AddPathToMonitor(w.ctx, path)
}

func (w *Monitor) Close() error {
	var allErr error
	close(w.eventC)
	allErr = errors.Join(allErr, w.perfChannel.Close())
	allErr = errors.Join(allErr, w.pathMonitor.Close())
	w.cancelFn()
	return nil
}

func (w *Monitor) EventChannel() <-chan MonitorEvent {
	return w.eventC
}

func (w *Monitor) ErrorChannel() <-chan error {
	return w.errC
}

func (w *Monitor) Start() error {
	w.done = make(chan bool, 1)

	if err := w.perfChannel.Run(); err != nil {
		return err
	}

	eProc := newEventProcessor(w.pathMonitor, w)
	_ = eProc
	go func() {
		defer func() {
			closeErr := w.Close()
			if closeErr != nil {
				w.log.Warnf("error at closing watcher: %v", closeErr)
			}
		}()

		for {
			select {
			case <-w.done:
				return

			case e, ok := <-w.perfChannel.C():
				if !ok {
					w.errC <- fmt.Errorf("read invalid event from perf channel")
					return
				}

				switch eWithType := e.(type) {
				case *ProbeEvent:
					if err := eProc.process(context.TODO(), eWithType); err != nil {
						w.errC <- err
						return
					}
					continue
				default:
					w.errC <- errors.New("unexpected event type")
					return
				}

			case err := <-w.perfChannel.ErrC():
				w.errC <- err
				return

			case lost := <-w.perfChannel.LostC():
				err := fmt.Errorf("events lost %d", lost)
				w.errC <- err
				return
			}
		}
	}()

	return nil
}
