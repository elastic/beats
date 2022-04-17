// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cloudfoundry

import (
	"context"
	"sync"

	loggregator "code.cloudfoundry.org/go-loggregator"
	"code.cloudfoundry.org/go-loggregator/conversion"
	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"

	"github.com/menderesk/beats/v7/libbeat/logp"
)

type RlpListenerCallbacks struct {
	HttpAccess      func(*EventHttpAccess)
	Log             func(*EventLog)
	Counter         func(*EventCounter)
	ValueMetric     func(*EventValueMetric)
	ContainerMetric func(*EventContainerMetric)
	Error           func(*EventError)
}

// RlpListener is a listener client that connects to the cloudfoundry loggregator.
type RlpListener struct {
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	rlpAddress string
	doer       *authTokenDoer
	shardID    string
	log        *logp.Logger
	callbacks  RlpListenerCallbacks
}

// newRlpListener returns default implementation for RLPClient
func newRlpListener(
	rlpAddress string,
	doer *authTokenDoer,
	shardID string,
	callbacks RlpListenerCallbacks,
	log *logp.Logger) *RlpListener {
	return &RlpListener{
		rlpAddress: rlpAddress,
		doer:       doer,
		shardID:    shardID,
		callbacks:  callbacks,
		log:        log,
	}
}

// Start receiving events through from loggregator.
func (c *RlpListener) Start(ctx context.Context) {
	c.log.Debugw("starting RLP listener.", "rlpAddress", c.rlpAddress)

	ops := []loggregator.RLPGatewayClientOption{loggregator.WithRLPGatewayHTTPClient(c.doer)}
	rlpClient := loggregator.NewRLPGatewayClient(c.rlpAddress, ops...)

	ctx, cancel := context.WithCancel(ctx)
	c.cancel = cancel
	l := &loggregator_v2.EgressBatchRequest{
		ShardId:   c.shardID,
		Selectors: c.getSelectors(),
	}
	es := rlpClient.Stream(ctx, l)

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		for {
			select {
			case <-ctx.Done():
				c.log.Debug("context done message at loggregator received.")
				return
			default:
				envelopes := es()
				for i := range envelopes {
					v1s := conversion.ToV1(envelopes[i])
					for _, v := range v1s {
						evt := EnvelopeToEvent(v)
						if evt.EventType() == EventTypeHttpAccess && c.callbacks.HttpAccess != nil {
							c.callbacks.HttpAccess(evt.(*EventHttpAccess))
						} else if evt.EventType() == EventTypeLog && c.callbacks.Log != nil {
							c.callbacks.Log(evt.(*EventLog))
						} else if evt.EventType() == EventTypeCounter && c.callbacks.Counter != nil {
							c.callbacks.Counter(evt.(*EventCounter))
						} else if evt.EventType() == EventTypeValueMetric && c.callbacks.ValueMetric != nil {
							c.callbacks.ValueMetric(evt.(*EventValueMetric))
						} else if evt.EventType() == EventTypeContainerMetric && c.callbacks.ContainerMetric != nil {
							c.callbacks.ContainerMetric(evt.(*EventContainerMetric))
						} else if evt.EventType() == EventTypeError && c.callbacks.Error != nil {
							c.callbacks.Error(evt.(*EventError))
						}
					}
				}
			}
		}
	}()
}

// Stop receiving events
func (c *RlpListener) Stop() {
	c.log.Debugw("stopping RLP listener.", "rlpAddress", c.rlpAddress)

	if c.cancel != nil {
		c.cancel()
	}
	c.wg.Wait()
}

func (c *RlpListener) Wait() {
	c.wg.Wait()
}

// getSelectors returns the server side selectors based on the callbacks defined on the listener.
func (c *RlpListener) getSelectors() []*loggregator_v2.Selector {
	selectors := make([]*loggregator_v2.Selector, 0)
	if c.callbacks.HttpAccess != nil {
		selectors = append(selectors, &loggregator_v2.Selector{
			Message: &loggregator_v2.Selector_Timer{
				Timer: &loggregator_v2.TimerSelector{},
			},
		})
	}
	if c.callbacks.Log != nil {
		selectors = append(selectors, &loggregator_v2.Selector{
			Message: &loggregator_v2.Selector_Log{
				Log: &loggregator_v2.LogSelector{},
			},
		})
	}
	if c.callbacks.Counter != nil {
		selectors = append(selectors, &loggregator_v2.Selector{
			Message: &loggregator_v2.Selector_Counter{
				Counter: &loggregator_v2.CounterSelector{},
			},
		})
	}
	if c.callbacks.ValueMetric != nil || c.callbacks.ContainerMetric != nil {
		selectors = append(selectors, &loggregator_v2.Selector{
			Message: &loggregator_v2.Selector_Gauge{
				Gauge: &loggregator_v2.GaugeSelector{},
			},
		})
	}
	if c.callbacks.Error != nil {
		selectors = append(selectors, &loggregator_v2.Selector{
			Message: &loggregator_v2.Selector_Event{
				Event: &loggregator_v2.EventSelector{},
			},
		})
	}
	return selectors
}
