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

package udp

import (
	"fmt"
	"sync"
	"time"

	"github.com/mitchellh/hashstructure"
	"github.com/rcrowley/go-metrics"

	"github.com/elastic/beats/v7/filebeat/channel"
	"github.com/elastic/beats/v7/filebeat/harvester"
	"github.com/elastic/beats/v7/filebeat/input"
	"github.com/elastic/beats/v7/filebeat/inputsource"
	"github.com/elastic/beats/v7/filebeat/inputsource/udp"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/monitoring/inputmon"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/monitoring/adapter"
)

func init() {
	err := input.Register("udp", NewInput)
	if err != nil {
		panic(err)
	}
}

// Input defines a udp input to receive event on a specific host:port.
type Input struct {
	sync.Mutex
	udp     *udp.Server
	started bool
	outlet  channel.Outleter

	metrics *inputMetrics
}

// NewInput creates a new udp input
func NewInput(cfg *conf.C, outlet channel.Connector, context input.Context) (input.Input, error) {
	id, err := configID(cfg)
	if err != nil {
		return nil, err
	}
	out, err := outlet.Connect(cfg)
	if err != nil {
		return nil, err
	}

	config := defaultConfig
	if err = cfg.Unpack(&config); err != nil {
		return nil, err
	}

	forwarder := harvester.NewForwarder(out)
	metrics := newInputMetrics(id, config.Host, uint64(config.ReadBuffer))
	callback := func(data []byte, metadata inputsource.NetworkMetadata) {
		evt := beat.Event{
			Timestamp: time.Now(),
			Meta: mapstr.M{
				"truncated": metadata.Truncated,
			},
			Fields: mapstr.M{
				"message": string(data),
			},
		}
		if metadata.RemoteAddr != nil {
			evt.Fields["log"] = mapstr.M{
				"source": mapstr.M{
					"address": metadata.RemoteAddr.String(),
				},
			}
		}
		_ = forwarder.Send(evt)

		// This must be called after forwarder.Send to measure
		// the processing time metric.
		metrics.log(data, evt.Timestamp)
	}

	udp := udp.New(&config.Config, callback)

	return &Input{
		outlet:  out,
		udp:     udp,
		started: false,
		metrics: metrics,
	}, nil
}

func configID(config *conf.C) (string, error) {
	var tmp struct {
		ID string `config:"id"`
	}
	if err := config.Unpack(&tmp); err != nil {
		return "", fmt.Errorf("error extracting ID: %w", err)
	}
	if tmp.ID != "" {
		return tmp.ID, nil
	}

	var h map[string]interface{}
	_ = config.Unpack(&h)
	id, err := hashstructure.Hash(h, nil)
	if err != nil {
		return "", fmt.Errorf("can not compute ID from configuration: %w", err)
	}

	return fmt.Sprintf("%16X", id), nil
}

// Run starts and start the UDP server and read events from the socket
func (p *Input) Run() {
	p.Lock()
	defer p.Unlock()

	if !p.started {
		logp.Info("Starting UDP input")
		err := p.udp.Start()
		if err != nil {
			logp.Err("Error running harvester: %v", err)
		}
		p.started = true
	}
}

// Stop stops the UDP input
func (p *Input) Stop() {
	defer p.outlet.Close()
	p.Lock()
	defer p.Unlock()

	logp.Info("Stopping UDP input")
	p.udp.Stop()
	p.metrics.close()
	p.started = false
}

// Wait suspends the UDP input
func (p *Input) Wait() {
	p.Stop()
}

// inputMetrics handles the input's metric reporting.
type inputMetrics struct {
	unregister func()

	lastPacket time.Time

	device         *monitoring.String // name of the device being monitored
	packets        *monitoring.Uint   // number of packets processed
	bytes          *monitoring.Uint   // number of bytes processed
	bufferLen      *monitoring.Uint   // configured read buffer length
	arrivalPeriod  metrics.Sample     // histogram of the elapsed time between packet arrivals
	processingTime metrics.Sample     // histogram of the elapsed time between packet receipt and publication
}

// newInputMetrics returns an input metric for the UDP processor. If id is empty
// a nil inputMetric is returned.
func newInputMetrics(id, device string, buflen uint64) *inputMetrics {
	if id == "" {
		return nil
	}
	reg, unreg := inputmon.NewInputRegistry("udp", id+"::"+device, nil)
	out := &inputMetrics{
		unregister:     unreg,
		bufferLen:      monitoring.NewUint(reg, "udp_read_buffer_length"),
		device:         monitoring.NewString(reg, "device"),
		packets:        monitoring.NewUint(reg, "udp_packets"),
		bytes:          monitoring.NewUint(reg, "udp_bytes"),
		arrivalPeriod:  metrics.NewUniformSample(1024),
		processingTime: metrics.NewUniformSample(1024),
	}
	_ = adapter.NewGoMetrics(reg, "udp_arrival_period", adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.arrivalPeriod))
	_ = adapter.NewGoMetrics(reg, "udp_processing_time", adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.processingTime))

	out.device.Set(device)
	out.bufferLen.Set(buflen)

	return out
}

// log logs metric for the given packet.
func (m *inputMetrics) log(data []byte, timestamp time.Time) {
	if m == nil {
		return
	}
	m.processingTime.Update(time.Since(timestamp).Nanoseconds())
	m.packets.Add(1)
	m.bytes.Add(uint64(len(data)))
	if !m.lastPacket.IsZero() {
		m.arrivalPeriod.Update(timestamp.Sub(m.lastPacket).Nanoseconds())
	}
	m.lastPacket = timestamp
}

func (m *inputMetrics) close() {
	if m != nil {
		return
	}
	m.unregister()
}
