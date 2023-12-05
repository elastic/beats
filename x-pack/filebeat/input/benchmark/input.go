// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package benchmark

import (
	"sync"
	"time"

	"github.com/rcrowley/go-metrics"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	stateless "github.com/elastic/beats/v7/filebeat/input/v2/input-stateless"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/monitoring/inputmon"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/monitoring/adapter"
)

const (
	inputName = "benchmark"
)

// Plugin registers the input
func Plugin() v2.Plugin {
	return v2.Plugin{
		Name:      inputName,
		Stability: feature.Experimental,
		Manager:   stateless.NewInputManager(configure),
	}
}

func configure(cfg *config.C) (stateless.Input, error) {
	bConf := defaultConfig
	if err := cfg.Unpack(&bConf); err != nil {
		return nil, err
	}
	return &benchmarkInput{cfg: bConf}, nil
}

// benchmarkInput is the main runtime object for the input
type benchmarkInput struct {
	cfg benchmarkConfig
}

// Name returns the name of the input
func (bi *benchmarkInput) Name() string {
	return inputName
}

// Test validates the configuration
func (bi *benchmarkInput) Test(ctx v2.TestContext) error {
	return bi.cfg.Validate()
}

// Run starts the data generation.
func (bi *benchmarkInput) Run(ctx v2.Context, publisher stateless.Publisher) error {
	var wg sync.WaitGroup
	metrics := newInputMetrics(ctx.ID)

	for i := uint8(0); i < bi.cfg.Threads; i++ {
		wg.Add(1)
		go func(thread uint8) {
			defer wg.Done()
			runThread(ctx, publisher, thread, bi.cfg, metrics)
		}(i)
	}
	wg.Wait()
	return ctx.Cancelation.Err()
}

func runThread(ctx v2.Context, publisher stateless.Publisher, thread uint8, cfg benchmarkConfig, metrics *inputMetrics) {
	ctx.Logger.Infof("starting benchmark input thread: %d", thread)
	defer ctx.Logger.Infof("stopping benchmark input thread: %d", thread)

	var line uint64
	var name uint64

	switch {
	case cfg.Count > 0:
		for {
			select {
			case <-ctx.Cancelation.Done():
				return
			default:
				publishEvt(publisher, cfg.Message, line, name, thread, metrics)
				line++
				if (line % cfg.Count) == 0 {
					return
				}
			}
		}
	case cfg.Eps > 0:
		ticker := time.NewTicker(1 * time.Second)
		pubChan := make(chan bool, int(cfg.Eps))
		for {
			select {
			case <-ctx.Cancelation.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				//don't want to block on filling doPublish channel
				//so only send as many as it can hold right now
				numToSend := cap(pubChan) - len(pubChan)
				for i := 0; i < numToSend; i++ {
					pubChan <- true
				}
			case <-pubChan:
				publishEvt(publisher, cfg.Message, line, name, thread, metrics)
				line++
				if line == 0 {
					name++
				}
			}
		}
	default:
		for {
			select {
			case <-ctx.Cancelation.Done():
				return
			default:
				publishEvt(publisher, cfg.Message, line, name, thread, metrics)
				line++
				if line == 0 {
					name++
				}
			}
		}
	}
	return
}

func publishEvt(publisher stateless.Publisher, msg string, line uint64, filename uint64, thread uint8, metrics *inputMetrics) {
	timestamp := time.Now()
	evt := beat.Event{
		Timestamp: timestamp,
		Fields: mapstr.M{
			"message":  msg,
			"line":     line,
			"filename": filename,
			"thread":   thread,
		},
	}
	publisher.Publish(evt)
	metrics.publishingTime.Update(time.Since(timestamp).Nanoseconds())
	metrics.eventsPublished.Add(1)
}

type inputMetrics struct {
	unregister func()

	eventsPublished *monitoring.Uint // number of events published
	publishingTime  metrics.Sample   // histogram of the elapsed times in nanoseconds (time of publisher.Publish)
}

// newInputMetrics returns an input metric for the benchmark processor.
func newInputMetrics(id string) *inputMetrics {
	reg, unreg := inputmon.NewInputRegistry(inputName, id, nil)
	out := &inputMetrics{
		unregister:      unreg,
		eventsPublished: monitoring.NewUint(reg, "events_published_total"),
		publishingTime:  metrics.NewUniformSample(1024),
	}

	_ = adapter.NewGoMetrics(reg, "publishing_time", adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.publishingTime))

	return out
}

func (m *inputMetrics) Close() {
	m.unregister()
}
