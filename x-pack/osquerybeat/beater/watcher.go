// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
)

const watchFrequency = 10 * time.Second

type Watcher struct {
	log  *logp.Logger
	ppid int

	mx     sync.Mutex
	cancel context.CancelFunc
}

func NewWatcher(log *logp.Logger) *Watcher {
	w := &Watcher{
		log:  log,
		ppid: os.Getppid(),
	}
	return w
}

func (w *Watcher) Start() {
	go w.Run()
}

func (w *Watcher) Run() {
	w.mx.Lock()
	defer w.mx.Unlock()

	if w.cancel != nil {
		w.log.Debug("watcher is already running")
		return
	}

	var ctx context.Context
	ctx, w.cancel = context.WithCancel(context.Background())

	ticker := time.NewTicker(watchFrequency)
	defer ticker.Stop()

	f := func() {
		ppid := os.Getppid()
		if ppid != w.ppid {
			w.log.Errorf("orphaned osquerybeat, expected ppid: %v, found ppid: %v, quitting", w.ppid, ppid)
			os.Exit(1)
		}
	}

	for {
		select {
		case <-ticker.C:
			f()
		case <-ctx.Done():
			w.log.Info("exit watcher on context done")
		}
	}
}

func (w *Watcher) Close() {
	w.mx.Lock()
	defer w.mx.Unlock()

	if w.cancel != nil {
		w.cancel()
		w.cancel = nil
	}
}
