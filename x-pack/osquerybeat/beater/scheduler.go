// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
)

type QueryFunc func(context.Context, interface{}) error

// Scheduler executes queries either periodically or once depending on the query configuration
type Scheduler struct {
	ctx       context.Context
	inCh      chan []config.StreamConfig
	runners   map[string]*runner
	queryFunc QueryFunc
	log       *logp.Logger
}

func NewScheduler(ctx context.Context, queryFunc QueryFunc) *Scheduler {
	return &Scheduler{
		ctx:       ctx,
		inCh:      make(chan []config.StreamConfig, 1),
		runners:   make(map[string]*runner),
		queryFunc: queryFunc,
		log:       logp.NewLogger("scheduler"),
	}
}

func (s *Scheduler) Load(streams []config.StreamConfig) {
	select {
	case s.inCh <- streams:
	case <-s.ctx.Done():
	}
}

func (s *Scheduler) Run() {
LOOP:
	for {
		select {
		case streams := <-s.inCh:
			s.load(streams)
		case <-s.ctx.Done():
			s.stopRunners()
			s.log.Info("Exiting on context cancel")
			break LOOP
		}
	}
}

func (s *Scheduler) isCancelled() bool {
	select {
	case <-s.ctx.Done():
		return true
	default:
		return false
	}
}
func (s *Scheduler) stopRunners() {
	s.load(nil)
}

func (s *Scheduler) load(streams []config.StreamConfig) {
	var (
		once, repeating []config.StreamConfig
	)

	// Separate fire-once queries and repeating queries
	for _, stream := range streams {
		if stream.Interval == 0 {
			once = append(once, stream)
		} else {
			repeating = append(repeating, stream)
		}
	}

	// Cancel and remove the query runners that are not in the streams
	var ids []string
	for id, r := range s.runners {
		found := false
		for _, s := range repeating {
			if id == s.ID {
				found = true
				break
			}
		}
		if !found {
			r.stop()
			ids = append(ids, id)
		}
	}

	for _, id := range ids {
		delete(s.runners, id)
	}

	if s.isCancelled() {
		return
	}

	// Run queries that should be executed only one
	for _, q := range once {
		if s.isCancelled() {
			return
		}
		startRunner(s.ctx, q, q.Interval, s.queryFunc)
	}

	// Schedule interval queries
	for _, q := range repeating {
		if s.isCancelled() {
			return
		}
		if _, ok := s.runners[q.ID]; !ok {
			s.runners[q.ID] = startRunner(s.ctx, q, q.Interval, s.queryFunc)
		}
	}
}
