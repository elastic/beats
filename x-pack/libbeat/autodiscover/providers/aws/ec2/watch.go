// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package ec2

import (
	"context"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v8/libbeat/logp"
	awsauto "github.com/elastic/beats/v8/x-pack/libbeat/autodiscover/providers/aws"
)

type watcher struct {
	// gen tracks changes we increment the 'generation' of each entry in the map.
	gen          uint64
	fetcher      fetcher
	onStart      func(uuid string, lblMap *ec2Instance)
	onStop       func(uuid string)
	done         chan struct{}
	ticker       *time.Ticker
	period       time.Duration
	ec2Instances map[string]uint64
	logger       *logp.Logger
}

func newWatcher(
	fetcher fetcher,
	period time.Duration,
	onStart func(uuid string, instanceMap *ec2Instance),
	onStop func(uuid string)) *watcher {
	return &watcher{
		fetcher:      fetcher,
		onStart:      onStart,
		onStop:       onStop,
		done:         make(chan struct{}),
		ticker:       time.NewTicker(period),
		period:       period,
		ec2Instances: map[string]uint64{},
		logger:       logp.NewLogger("autodiscover-ec2-watcher"),
	}
}

func (w *watcher) start() {
	go w.forever()
}

func (w *watcher) stop() {
	close(w.done)
}

func (w *watcher) forever() {
	for {
		select {
		case <-w.done:
			w.ticker.Stop()
			return
		case <-w.ticker.C:
			err := w.once()
			if err != nil {
				logp.Error(errors.Wrap(err, "error while fetching AWS EC2s"))
			}
		}
	}
}

// once executes the watch loop a single time.
// This is mostly useful for testing.
func (w *watcher) once() error {
	ctx, cancelCtx := context.WithTimeout(context.Background(), w.period)
	defer cancelCtx() // Always cancel to avoid leak

	fetchedEC2s, err := w.fetcher.fetch(ctx)
	if err != nil {
		return err
	}
	w.logger.Debugf("fetched %d ec2 instances from AWS for autodiscover", len(fetchedEC2s))

	oldGen := w.gen
	w.gen++

	// Increment the generation of all EC2s returned by the API request
	for _, instance := range fetchedEC2s {
		instanceID := awsauto.SafeString(instance.ec2Instance.InstanceId)
		if _, exists := w.ec2Instances[instanceID]; !exists {
			if w.onStart != nil {
				w.onStart(instanceID, instance)
			}
		}
		w.ec2Instances[instanceID] = w.gen
	}

	// EC2s not seen in the API request get deleted
	for uuid, entryGen := range w.ec2Instances {
		if entryGen == oldGen {
			if w.onStop != nil {
				w.onStop(uuid)
				delete(w.ec2Instances, uuid)
			}
		}
	}

	return nil
}
