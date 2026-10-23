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

package remote_write

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/prompb"

	serverhelper "github.com/elastic/beats/v7/metricbeat/helper/server"
	httpserver "github.com/elastic/beats/v7/metricbeat/helper/server/http"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/parse"
)

const legacyClosedMessage = "remote write metricset closed"

func init() {
	mb.Registry.MustAddMetricSet("prometheus", "remote_write",
		MetricSetBuilder(DefaultRemoteWriteEventsGeneratorFactory),
		mb.WithHostParser(parse.EmptyHostParser),
	)
}

// RemoteWriteEventsGenerator converts Prometheus Samples to a map of mb.Event
type RemoteWriteEventsGenerator interface {
	// Start must be called before using the generator
	Start()

	// GenerateEvents converts Prometheus Samples to a map of mb.Event
	GenerateEvents(metrics model.Samples) map[string]mb.Event

	// Stop must be called when the generator won't be used anymore
	Stop()
}

// RemoteWriteOwnerLoopRequired marks generators that require serialized batch processing.
type RemoteWriteOwnerLoopRequired interface {
	RequiresOwnerLoop() bool
}

// RequiresRemoteWriteOwnerLoop reports whether generator explicitly requires owner-loop processing.
func RequiresRemoteWriteOwnerLoop(generator RemoteWriteEventsGenerator) bool {
	required, ok := generator.(RemoteWriteOwnerLoopRequired)
	return ok && required.RequiresOwnerLoop()
}

// RemoteWriteEventsGeneratorFactory creates a RemoteWriteEventsGenerator when instanciating a metricset
type RemoteWriteEventsGeneratorFactory func(ms mb.BaseMetricSet, opts ...RemoteWriteEventsGeneratorOption) (RemoteWriteEventsGenerator, error)

type MetricSet struct {
	mb.BaseMetricSet
	server                 serverhelper.Server
	events                 chan mb.Event
	batches                chan batchSubmission
	promEventsGen          RemoteWriteEventsGenerator
	useOwnerLoop           bool
	eventGenMu             sync.Mutex
	eventGenStarted        bool
	eventGenClosed         bool
	maxCompressedBodyBytes int64
	maxDecodedBodyBytes    int64

	intakeMu         sync.RWMutex
	intakeCtx        context.Context
	intakeCancel     context.CancelFunc
	handlersInFlight atomic.Int32
}

// MetricSetBuilder returns a builder function for a new Prometheus remote_write metricset using
// the given namespace and event generator
func MetricSetBuilder(genFactory RemoteWriteEventsGeneratorFactory) func(base mb.BaseMetricSet) (mb.MetricSet, error) {
	return MetricSetBuilderWithConfig(genFactory, defaultConfig())
}

// MetricSetBuilderWithConfig returns a builder function for a new Prometheus remote_write metricset using
// the given namespace, event generator, and a base config that will be merged with module config
func MetricSetBuilderWithConfig(genFactory RemoteWriteEventsGeneratorFactory, baseConfig Config) func(base mb.BaseMetricSet) (mb.MetricSet, error) {
	return func(base mb.BaseMetricSet) (mb.MetricSet, error) {
		config := baseConfig
		err := base.Module().UnpackConfig(&config)
		if err != nil {
			return nil, err
		}

		promEventsGen, err := genFactory(base, WithCountMetrics(config.MetricsCount))
		if err != nil {
			return nil, err
		}

		m := &MetricSet{
			BaseMetricSet:          base,
			maxCompressedBodyBytes: config.MaxCompressedBodyBytes,
			maxDecodedBodyBytes:    config.MaxDecodedBodyBytes,
		}
		m.setPromEventsGenerator(promEventsGen)

		svc, err := httpserver.NewHttpServerWithHandler(base, m.handleFunc)
		if err != nil {
			return nil, err
		}
		m.server = svc

		return m, nil
	}
}

// setPromEventsGenerator replaces the generator and recomputes its flow before Run starts.
func (m *MetricSet) setPromEventsGenerator(generator RemoteWriteEventsGenerator) {
	m.eventGenMu.Lock()
	defer m.eventGenMu.Unlock()

	m.promEventsGen = generator
	m.useOwnerLoop = RequiresRemoteWriteOwnerLoop(generator)
	m.eventGenStarted = false
	m.eventGenClosed = false

	// Allocate only the intake channel used by the selected flow.
	if m.useOwnerLoop {
		m.events = nil
		m.batches = make(chan batchSubmission)
		return
	}

	m.events = make(chan mb.Event)
	m.batches = nil
}

func (m *MetricSet) Run(reporter mb.PushReporterV2) {
	if !m.useOwnerLoop {
		m.runEvents(reporter)
		return
	}
	m.runOwnerLoop(reporter)
}

// runEvents publishes events submitted through the events channel.
func (m *MetricSet) runEvents(reporter mb.PushReporterV2) {
	registerRunReadySignal(m)
	defer runReadySignals.Delete(m)

	if shouldStartHTTPServer(m) {
		_ = m.server.Start()
	}
	signalRunReady(m)

	for {
		select {
		case <-reporter.Done():
			if shouldStartHTTPServer(m) {
				m.server.Stop()
			}
			return
		case event := <-m.events:
			reporter.Event(event)
		}
	}
}

// This is the owner loop path used when the histogram assembler is enabled.
func (m *MetricSet) runOwnerLoop(reporter mb.PushReporterV2) {
	m.promEventsGen.Start()
	defer m.promEventsGen.Stop()

	registerRunReadySignal(m)
	defer runReadySignals.Delete(m)

	intakeCtx, intakeCancel := context.WithCancel(context.Background())
	m.intakeMu.Lock()
	m.intakeCtx = intakeCtx
	m.intakeCancel = intakeCancel
	m.intakeMu.Unlock()
	signalRunReady(m)
	defer intakeCancel()

	if shouldStartHTTPServer(m) {
		_ = m.server.Start()
	}

	go func() {
		<-reporter.Done()
		m.shutdownIntake()
		if shouldStartHTTPServer(m) {
			m.server.Stop()
		}
	}()

	tickFactory := tickSourceFactoryFor(m)
	var flushTicker tickSource
	var flushCh <-chan time.Time
	if flusher, ok := m.promEventsGen.(RemoteWriteFlushCapable); ok {
		if interval := flusher.NextFlushInterval(); interval > 0 {
			flushTicker = tickFactory(interval)
			flushCh = flushTicker.C()
		}
	}
	if flushTicker != nil {
		defer flushTicker.Stop()
	}

	for {
		var tickCh <-chan time.Time
		if flushCh != nil {
			tickCh = flushCh
		}

		select {
		case <-reporter.Done():
			m.shutdownOwnerLoop(reporter)
			return
		case sub := <-m.batches:
			m.processBatch(reporter, sub)
		case now := <-tickCh:
			if tickCh != nil {
				m.processFlush(reporter, now)
			}
		}
	}
}

// Close permanently closes legacy intake and stops its generator if an HTTP handler started it.
// Owner-loop generator lifecycle is wholly owned by Run so Close cannot race with batch processing.
func (m *MetricSet) Close() error {
	m.eventGenMu.Lock()
	defer m.eventGenMu.Unlock()
	if m.useOwnerLoop {
		return nil
	}
	m.eventGenClosed = true
	if m.eventGenStarted {
		m.promEventsGen.Stop()
		m.eventGenStarted = false
	}
	return nil
}

func writeBatchOutcome(writer http.ResponseWriter, outcome batchOutcome) {
	if outcome.statusCode == http.StatusAccepted {
		writer.WriteHeader(outcome.statusCode)
		return
	}
	if outcome.message != "" {
		http.Error(writer, outcome.message, outcome.statusCode)
		return
	}
	writer.WriteHeader(outcome.statusCode)
}

func (m *MetricSet) handleFunc(writer http.ResponseWriter, req *http.Request) {
	if !m.useOwnerLoop {
		if !m.startLegacyGenerator() {
			http.Error(writer, legacyClosedMessage, http.StatusServiceUnavailable)
			return
		}
	}

	samples, ok := m.decodeWriteRequest(writer, req)
	if !ok {
		return
	}

	if !m.useOwnerLoop {
		m.handleLegacyEvents(writer, req, samples)
		return
	}
	m.handleOwnerBatch(writer, req, samples)
}

func (m *MetricSet) startLegacyGenerator() bool {
	m.eventGenMu.Lock()
	defer m.eventGenMu.Unlock()
	if m.eventGenClosed {
		return false
	}
	if !m.eventGenStarted {
		m.promEventsGen.Start()
		m.eventGenStarted = true
	}
	return true
}

func (m *MetricSet) decodeWriteRequest(writer http.ResponseWriter, req *http.Request) (model.Samples, bool) {
	// Limit the size of the compressed request body to prevent resource exhaustion
	req.Body = http.MaxBytesReader(writer, req.Body, m.maxCompressedBodyBytes)

	compressed, err := io.ReadAll(req.Body)
	if err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			m.Logger().Warnf("Request body too large: exceeds %d bytes limit", m.maxCompressedBodyBytes)
			http.Error(writer, fmt.Sprintf("request body too large: exceeds %d bytes limit", m.maxCompressedBodyBytes), http.StatusRequestEntityTooLarge)
			return nil, false
		}
		m.Logger().Errorf("Read error %v", err)
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return nil, false
	}

	// Check decoded length before allocating memory to prevent
	decodedLen, err := snappy.DecodedLen(compressed)
	if err != nil {
		m.Logger().Errorf("Decoded length error: %v", err)
		http.Error(writer, "Decoded length error", http.StatusBadRequest)
		return nil, false
	}
	if int64(decodedLen) > m.maxDecodedBodyBytes {
		m.Logger().Warnf("Decoded length too large: %d bytes exceeds %d max decoded bytes limit (maxDecodedBodyBytes)", decodedLen, m.maxDecodedBodyBytes)
		http.Error(writer, fmt.Sprintf("decoded length too large: %d bytes exceeds %d max decoded bytes limit (maxDecodedBodyBytes)", decodedLen, m.maxDecodedBodyBytes), http.StatusRequestEntityTooLarge)
		return nil, false
	}

	reqBuf, err := snappy.Decode(nil, compressed)
	if err != nil {
		m.Logger().Errorf("Decode error %v", err)
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return nil, false
	}

	var protoReq prompb.WriteRequest
	if err := proto.Unmarshal(reqBuf, &protoReq); err != nil {
		m.Logger().Errorf("Unmarshal error %v", err)
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return nil, false
	}
	return protoToSamples(&protoReq), true
}

func (m *MetricSet) handleLegacyEvents(writer http.ResponseWriter, req *http.Request, samples model.Samples) {
	events := m.promEventsGen.GenerateEvents(samples)
	for _, event := range events {
		select {
		case <-req.Context().Done():
			return
		case m.events <- event:
		}
	}
	writer.WriteHeader(http.StatusAccepted)
}

func (m *MetricSet) handleOwnerBatch(writer http.ResponseWriter, req *http.Request, samples model.Samples) {
	m.handlersInFlight.Add(1)
	defer m.handlersInFlight.Add(-1)

	if intakeCtx := m.getIntakeCtx(); intakeCtx != nil {
		select {
		case <-intakeCtx.Done():
			http.Error(writer, "remote write intake stopped", http.StatusServiceUnavailable)
			return
		default:
		}
	}

	result := make(chan batchOutcome, 1)
	sub := batchSubmission{
		samples: samples,
		ctx:     req.Context(),
		result:  result,
	}

	submitted := make(chan bool, 1)
	go func() {
		sent := false
		select {
		case m.batches <- sub:
			sent = true
		case <-m.intakeDone():
		case <-req.Context().Done():
		}
		submitted <- sent
	}()

	select {
	case sent := <-submitted:
		if !sent {
			select {
			case <-req.Context().Done():
				return
			default:
				http.Error(writer, "remote write intake stopped", http.StatusServiceUnavailable)
				return
			}
		}
	case <-req.Context().Done():
		return
	case <-m.intakeDone():
		http.Error(writer, "remote write intake stopped", http.StatusServiceUnavailable)
		return
	}

	select {
	case outcome := <-result:
		writeBatchOutcome(writer, outcome)
	case <-req.Context().Done():
		return
	case <-m.intakeDone():
		select {
		case outcome := <-result:
			writeBatchOutcome(writer, outcome)
		case <-req.Context().Done():
		}
	}
}

func (m *MetricSet) shutdownIntake() {
	m.intakeMu.Lock()
	cancel := m.intakeCancel
	m.intakeMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (m *MetricSet) getIntakeCtx() context.Context {
	m.intakeMu.RLock()
	defer m.intakeMu.RUnlock()
	return m.intakeCtx
}

func (m *MetricSet) intakeDone() <-chan struct{} {
	ctx := m.getIntakeCtx()
	if ctx == nil {
		return intakeNeverClosed
	}
	return ctx.Done()
}

func protoToSamples(req *prompb.WriteRequest) model.Samples {
	var samples model.Samples
	for _, ts := range req.Timeseries {
		metric := make(model.Metric, len(ts.Labels))
		for _, l := range ts.Labels {
			metric[model.LabelName(l.Name)] = model.LabelValue(l.Value)
		}

		for _, s := range ts.Samples {
			samples = append(samples, &model.Sample{
				Metric:    metric,
				Value:     model.SampleValue(s.Value),
				Timestamp: model.Time(s.Timestamp),
			})
		}
	}
	return samples
}
