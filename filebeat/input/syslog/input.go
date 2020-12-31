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

package syslog

import (
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/filebeat/channel"
	"github.com/elastic/beats/v7/filebeat/harvester"
	"github.com/elastic/beats/v7/filebeat/input"
	"github.com/elastic/beats/v7/filebeat/inputsource"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/libbeat/logp"
)

// Parser is generated from a ragel state machine using the following command:
//go:generate ragel -Z -G2 parser.rl -o parser.go
//go:generate goimports -l -w parser.go

// Severity and Facility are derived from the priority, theses are the human readable terms
// defined in https://tools.ietf.org/html/rfc3164#section-4.1.1.
//
// Example:
// 2 => "Critical"
type mapper []string

var (
	severityLabels = mapper{
		"Emergency",
		"Alert",
		"Critical",
		"Error",
		"Warning",
		"Notice",
		"Informational",
		"Debug",
	}

	facilityLabels = mapper{
		"kernel",
		"user-level",
		"mail",
		"system",
		"security/authorization",
		"syslogd",
		"line printer",
		"network news",
		"UUCP",
		"clock",
		"security/authorization",
		"FTP",
		"NTP",
		"log audit",
		"log alert",
		"clock",
		"local0",
		"local1",
		"local2",
		"local3",
		"local4",
		"local5",
		"local6",
		"local7",
	}
)

func init() {
	err := input.Register("syslog", NewInput)
	if err != nil {
		panic(err)
	}
}

// Input define a syslog input
type Input struct {
	sync.Mutex
	started bool
	outlet  channel.Outleter
	server  inputsource.Network
	config  *config
	log     *logp.Logger
}

// NewInput creates a new syslog input
func NewInput(
	cfg *common.Config,
	outlet channel.Connector,
	context input.Context,
) (input.Input, error) {
	cfgwarn.Experimental("Syslog input type is used")

	log := logp.NewLogger("syslog")

	out, err := outlet.Connect(cfg)
	if err != nil {
		return nil, err
	}

	config := defaultConfig
	if err = cfg.Unpack(&config); err != nil {
		return nil, err
	}

	forwarder := harvester.NewForwarder(out)
	cb := func(data []byte, metadata inputsource.NetworkMetadata) {
		ev := parseAndCreateEvent(data, metadata, time.Local, log)
		forwarder.Send(ev)
	}

	server, err := factory(cb, config.Protocol)
	if err != nil {
		return nil, err
	}

	return &Input{
		outlet:  out,
		started: false,
		server:  server,
		config:  &config,
		log:     log,
	}, nil
}

// Run starts listening for Syslog events over the network.
func (p *Input) Run() {
	p.Lock()
	defer p.Unlock()

	if !p.started {
		p.log.Infow("Starting Syslog input", "protocol", p.config.Protocol.Name())
		err := p.server.Start()
		if err != nil {
			p.log.Error("Error starting the server", "error", err)
			return
		}
		p.started = true
	}
}

// Stop stops the syslog input.
func (p *Input) Stop() {
	defer p.outlet.Close()
	p.Lock()
	defer p.Unlock()

	if !p.started {
		return
	}

	p.log.Info("Stopping Syslog input")
	p.server.Stop()
	p.started = false
}

// Wait stops the syslog input.
func (p *Input) Wait() {
	p.Stop()
}

func createEvent(ev *event, metadata inputsource.NetworkMetadata, timezone *time.Location, log *logp.Logger) beat.Event {
	f := common.MapStr{
		"message": strings.TrimRight(ev.Message(), "\n"),
	}

	syslog := common.MapStr{}
	event := common.MapStr{}
	process := common.MapStr{}

	if ev.Hostname() != "" {
		f["hostname"] = ev.Hostname()
	}

	if ev.HasPid() {
		process["pid"] = ev.Pid()
	}

	if ev.Program() != "" {
		process["program"] = ev.Program()
	}

	if ev.HasPriority() {
		syslog["priority"] = ev.Priority()

		event["severity"] = ev.Severity()
		v, err := mapValueToName(ev.Severity(), severityLabels)
		if err != nil {
			log.Debugw("could not find severity label", "error", err)
		} else {
			syslog["severity_label"] = v
		}

		syslog["facility"] = ev.Facility()
		v, err = mapValueToName(ev.Facility(), facilityLabels)
		if err != nil {
			log.Debugw("could not find facility label", "error", err)
		} else {
			syslog["facility_label"] = v
		}
	}

	f["syslog"] = syslog
	f["event"] = event
	if len(process) > 0 {
		f["process"] = process
	}

	if ev.Sequence() != -1 {
		f["event.sequence"] = ev.Sequence()
	}

	return newBeatEvent(ev.Timestamp(timezone), metadata, f)
}

func parseAndCreateEvent(data []byte, metadata inputsource.NetworkMetadata, timezone *time.Location, log *logp.Logger) beat.Event {
	ev := newEvent()
	Parse(data, ev)
	if !ev.IsValid() {
		log.Errorw("can't parse event as syslog rfc3164", "message", string(data))
		return newBeatEvent(time.Now(), metadata, common.MapStr{
			"message": string(data),
		})
	}
	return createEvent(ev, metadata, time.Local, log)
}

func newBeatEvent(timestamp time.Time, metadata inputsource.NetworkMetadata, fields common.MapStr) beat.Event {
	event := beat.Event{
		Timestamp: timestamp,
		Meta: common.MapStr{
			"truncated": metadata.Truncated,
		},
		Fields: fields,
	}
	if metadata.RemoteAddr != nil {
		event.Fields.Put("log.source.address", metadata.RemoteAddr.String())
	}
	return event
}

func mapValueToName(v int, m mapper) (string, error) {
	if v < 0 || v >= len(m) {
		return "", errors.Errorf("value out of bound: %d", v)
	}
	return m[v], nil
}
