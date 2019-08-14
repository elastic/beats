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

package spool

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common/atomic"
	"github.com/elastic/beats/libbeat/publisher/queue"
	"github.com/elastic/go-txfile"
	"github.com/elastic/go-txfile/pq"
)

// Spool implements an on-disk queue.Queue.
type Spool struct {
	// producer/input support
	inCtx    *spoolCtx
	inBroker *inBroker

	// consumer/output support
	outCtx    *spoolCtx
	outBroker *outBroker

	queue *pq.Queue
	file  *txfile.File
}

type spoolCtx struct {
	logger logger
	wg     sync.WaitGroup
	active atomic.Bool
	done   chan struct{}
}

// Settings configure a new spool to be created.
type Settings struct {
	Mode os.FileMode

	File txfile.Options

	// Queue write buffer size. If a single event is bigger then the
	// write-buffer, the write-buffer will grow. In this case will the write
	// buffer be flushed and reset to its original size.
	WriteBuffer uint

	Eventer queue.Eventer

	WriteFlushTimeout time.Duration
	WriteFlushEvents  uint
	ReadFlushTimeout  time.Duration

	Codec codecID
}

const minInFlushTimeout = 100 * time.Millisecond
const minOutFlushTimeout = 0 * time.Millisecond

// NewSpool creates and initializes a new file based queue.
func NewSpool(logger logger, path string, settings Settings) (*Spool, error) {
	mode := settings.Mode
	if mode == 0 {
		mode = os.ModePerm
	}

	ok := false
	inCtx := newSpoolCtx(logger)
	outCtx := newSpoolCtx(logger)
	defer ifNotOK(&ok, inCtx.Close)
	defer ifNotOK(&ok, outCtx.Close)

	if info, err := os.Lstat(path); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	} else if runtime.GOOS != "windows" {
		perm := info.Mode().Perm()
		cfgPerm := settings.Mode.Perm()

		// check if file has permissions set, that must not be set via config
		if (perm | cfgPerm) != cfgPerm {
			return nil, fmt.Errorf("file permissions for '%v' must be more strict (required permissions: %v, actual permissions: %v)",
				path, cfgPerm, perm)
		}
	}

	f, err := txfile.Open(path, mode, settings.File)
	if err != nil {
		return nil, errors.Wrapf(err, "spool queue: failed to open file at path '%s'", path)
	}
	defer ifNotOK(&ok, ignoreErr(f.Close))

	queueDelegate, err := pq.NewStandaloneDelegate(f)
	if err != nil {
		return nil, err
	}

	spool := &Spool{
		inCtx:  inCtx,
		outCtx: outCtx,
	}

	queue, err := pq.New(queueDelegate, pq.Settings{
		WriteBuffer: settings.WriteBuffer,
		Flushed:     spool.onFlush,
		ACKed:       spool.onACK,
	})
	if err != nil {
		return nil, err
	}
	defer ifNotOK(&ok, ignoreErr(queue.Close))

	inFlushTimeout := settings.WriteFlushTimeout
	if inFlushTimeout < minInFlushTimeout {
		inFlushTimeout = minInFlushTimeout
	}
	inBroker, err := newInBroker(inCtx, settings.Eventer, queue, settings.Codec,
		inFlushTimeout, settings.WriteFlushEvents)
	if err != nil {
		return nil, err
	}

	outFlushTimeout := settings.ReadFlushTimeout
	if outFlushTimeout < minOutFlushTimeout {
		outFlushTimeout = minOutFlushTimeout
	}
	outBroker, err := newOutBroker(outCtx, queue, outFlushTimeout)
	if err != nil {
		return nil, err
	}

	ok = true
	spool.queue = queue
	spool.inBroker = inBroker
	spool.outBroker = outBroker
	spool.file = f
	return spool, nil
}

// Close shuts down the queue and closes the used file.
func (s *Spool) Close() error {
	// stop all workers (waits for all workers to be finished)
	s.outCtx.Close()
	s.inCtx.Close()

	// close queue (potentially flushing write buffer)
	err := s.queue.Close()

	// finally unmap and close file
	s.file.Close()

	return err
}

// BufferConfig returns the queue initial buffer settings.
func (s *Spool) BufferConfig() queue.BufferConfig {
	return queue.BufferConfig{Events: -1}
}

// Producer creates a new queue producer for publishing events.
func (s *Spool) Producer(cfg queue.ProducerConfig) queue.Producer {
	return s.inBroker.Producer(cfg)
}

// Consumer creates a new queue consumer for consuming and acking events.
func (s *Spool) Consumer() queue.Consumer {
	return s.outBroker.Consumer()
}

// onFlush is run whenever the queue signals it's write buffer being flushed.
// Flush events are forwarded to all workers.
// The onFlush callback is directly called by the queue writer (same go-routine)
// on Write or Flush operations.
func (s *Spool) onFlush(n uint) {
	s.inBroker.onFlush(n)
	s.outBroker.onFlush(n)
}

// onACK is run whenever the queue signals events being acked and removed from
// the queue.
// ACK events are forwarded to all workers.
func (s *Spool) onACK(events, pages uint) {
	s.inBroker.onACK(events, pages)
}

func newSpoolCtx(logger logger) *spoolCtx {
	return &spoolCtx{
		logger: logger,
		active: atomic.MakeBool(true),
		done:   make(chan struct{}),
	}
}

func (ctx *spoolCtx) Close() {
	if ctx.active.CAS(true, false) {
		close(ctx.done)
		ctx.wg.Wait()
	}
}

func (ctx *spoolCtx) Done() <-chan struct{} {
	return ctx.done
}

func (ctx *spoolCtx) Open() bool {
	return ctx.active.Load()
}

func (ctx *spoolCtx) Closed() bool {
	return !ctx.Open()
}

func (ctx *spoolCtx) Go(fn func()) {
	ctx.wg.Add(1)
	go func() {
		defer ctx.wg.Done()
		fn()
	}()
}

func ifNotOK(b *bool, fn func()) {
	if !(*b) {
		fn()
	}
}

func ignoreErr(fn func() error) func() {
	return func() { fn() }
}
