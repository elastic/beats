// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"strings"
	"time"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/filewatcher"
)

type periodic struct {
	log      *logger.Logger
	period   time.Duration
	done     chan struct{}
	watcher  *filewatcher.Watch
	emitter  emitterFunc
	discover discoverFunc
}

func (p *periodic) Start() error {
	go func() {
		if err := p.work(); err != nil {
			p.log.Debugf("Failed to read configuration, error: %s", err)
		}

		for {
			select {
			case <-p.done:
				break
			case <-time.After(p.period):
			}

			if err := p.work(); err != nil {
				p.log.Debugf("Failed to read configuration, error: %s", err)
			}
		}
	}()
	return nil
}

func (p *periodic) work() error {
	files, err := p.discover()
	if err != nil {
		return errors.New(err, "could not discover configuration files", errors.TypeConfig)
	}

	if len(files) == 0 {
		return ErrNoConfiguration
	}

	// Reset the state of the watched files
	p.watcher.Reset()

	p.log.Debugf("Adding %d file to watch", len(files))
	// Add any found files to the watchers
	for _, f := range files {
		p.watcher.Watch(f)
	}

	// Check for the following:
	// - Watching of new files.
	// - Files watched but some of them have changed.
	// - Files that we were watching but are not watched anymore.
	s, err := p.watcher.Update()
	if err != nil {
		return errors.New(err, "could not update the configuration states", errors.TypeConfig)
	}

	if s.NeedUpdate {
		p.log.Info("Configuration changes detected")
		if len(s.Unwatched) > 0 {
			p.log.Debugf("Unwatching %d files: %s", len(s.Unwatched), strings.Join(s.Unwatched, ", "))
		}

		if len(s.Updated) > 0 {
			p.log.Debugf("Updated %d files: %s", len(s.Updated), strings.Join(s.Updated, ", "))
		}

		if len(s.Unchanged) > 0 {
			p.log.Debugf("Unchanged %d files: %s", len(s.Unchanged), strings.Join(s.Updated, ", "))
		}

		err := readfiles(files, p.emitter)
		if err != nil {
			// assume something when really wrong and invalidate any cache
			// so we get a full new config on next tick.
			p.watcher.Invalidate()
			return errors.New(err, "could not emit configuration")
		}
	}

	p.log.Info("No configuration change")
	return nil
}

func (p *periodic) Stop() error {
	close(p.done)
	return nil
}

func newPeriodic(
	log *logger.Logger,
	period time.Duration,
	discover discoverFunc,
	emitter emitterFunc,
) *periodic {
	w, err := filewatcher.New(log, filewatcher.DefaultComparer)

	// this should not happen.
	if err != nil {
		panic(err)
	}

	return &periodic{
		log:      log,
		period:   period,
		done:     make(chan struct{}),
		watcher:  w,
		discover: discover,
		emitter:  emitter,
	}
}
