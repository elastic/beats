// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

//go:build linux

package kprobes

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-perf"
)

type MonitorEvent struct {
	Path string
	PID  uint32
	Op   uint32
}

type monitorEmitter struct {
	ctx    context.Context
	eventC chan<- MonitorEvent
}

func newMonitorEmitter(ctx context.Context, eventC chan MonitorEvent) *monitorEmitter {
	return &monitorEmitter{
		ctx:    ctx,
		eventC: eventC,
	}
}

func (m *monitorEmitter) Emit(ePath string, pid uint32, op uint32) error {
	select {
	case <-m.ctx.Done():
		return m.ctx.Err()

	case m.eventC <- MonitorEvent{
		Path: ePath,
		PID:  pid,
		Op:   op,
	}:
		return nil
	}
}

type Monitor struct {
	eventC      chan MonitorEvent
	pathMonitor *pTraverser
	perfChannel perfChannel
	errC        chan error
	eProc       *eProcessor
	log         *logp.Logger
	ctx         context.Context
	cancelFn    context.CancelFunc
	running     uint32
	isRecursive bool
	closeErr    error
}

func New(isRecursive bool) (*Monitor, error) {
	ctx := context.TODO()

	validatedProbes, exec, err := getVerifiedProbes(ctx, 5*time.Second)
	if err != nil {
		return nil, err
	}

	pChannel, err := newPerfChannel(validatedProbes, 10, 4096, perf.AllThreads)
	if err != nil {
		return nil, err
	}

	return newMonitor(ctx, isRecursive, pChannel, exec)
}

func newMonitor(ctx context.Context, isRecursive bool, pChannel perfChannel, exec executor) (*Monitor, error) {
	mCtx, cancelFunc := context.WithCancel(ctx)

	p, err := newPathMonitor(mCtx, exec, 0, isRecursive)
	if err != nil {
		cancelFunc()
		return nil, err
	}

	eventChannel := make(chan MonitorEvent, 512)
	eProc := newEventProcessor(p, newMonitorEmitter(mCtx, eventChannel), isRecursive)

	return &Monitor{
		eventC:      eventChannel,
		pathMonitor: p,
		perfChannel: pChannel,
		errC:        make(chan error, 1),
		eProc:       eProc,
		log:         logp.NewLogger("file_integrity"),
		ctx:         mCtx,
		cancelFn:    cancelFunc,
		isRecursive: isRecursive,
		closeErr:    nil,
	}, nil
}

func (w *Monitor) Add(path string) error {
	switch atomic.LoadUint32(&w.running) {
	case 0:
		return errors.New("monitor not started")
	case 2:
		return errors.New("monitor is closed")
	}

	return w.pathMonitor.AddPathToMonitor(w.ctx, path)
}

func (w *Monitor) Close() error {
	if !atomic.CompareAndSwapUint32(&w.running, 1, 2) {
		switch atomic.LoadUint32(&w.running) {
		case 0:
			// monitor hasn't started yet
			atomic.StoreUint32(&w.running, 2)
		default:
			return nil
		}
	}

	w.cancelFn()
	var allErr error
	allErr = errors.Join(allErr, w.pathMonitor.Close())
	allErr = errors.Join(allErr, w.perfChannel.Close())

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
	if !atomic.CompareAndSwapUint32(&w.running, 0, 1) {
		return errors.New("monitor already started")
	}

	if err := w.perfChannel.Run(); err != nil {
		if closeErr := w.Close(); closeErr != nil {
			w.log.Warnf("error at closing watcher: %v", closeErr)
		}
		return err
	}

	go func() {
		defer func() {
			close(w.eventC)
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
