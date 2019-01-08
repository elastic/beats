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

package publish

import (
	"net"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors"
	"github.com/elastic/beats/packetbeat/pb"
)

type TransactionPublisher struct {
	done      chan struct{}
	pipeline  beat.Pipeline
	canDrop   bool
	processor transProcessor
}

type transProcessor struct {
	ignoreOutgoing bool
	localIPs       []net.IP // TODO: Periodically update this list.
	localIPStrings []string // Deprecated. Use localIPs.
	name           string
}

var debugf = logp.MakeDebug("publish")

func NewTransactionPublisher(
	name string,
	pipeline beat.Pipeline,
	ignoreOutgoing bool,
	canDrop bool,
) (*TransactionPublisher, error) {
	addrs, err := common.LocalIPAddrs()
	if err != nil {
		return nil, err
	}
	var localIPs []net.IP
	var localIPStrings []string
	for _, addr := range addrs {
		if !addr.IsLoopback() {
			localIPs = append(localIPs, addr)
			localIPStrings = append(localIPStrings, addr.String())
		}
	}

	p := &TransactionPublisher{
		done:     make(chan struct{}),
		pipeline: pipeline,
		canDrop:  canDrop,
		processor: transProcessor{
			localIPs:       localIPs,
			localIPStrings: localIPStrings,
			name:           name,
			ignoreOutgoing: ignoreOutgoing,
		},
	}
	return p, nil
}

func (p *TransactionPublisher) Stop() {
	close(p.done)
}

func (p *TransactionPublisher) CreateReporter(
	config *common.Config,
) (func(beat.Event), error) {

	// load and register the module it's fields, tags and processors settings
	meta := struct {
		Event      common.EventMetadata    `config:",inline"`
		Processors processors.PluginConfig `config:"processors"`
	}{}
	if err := config.Unpack(&meta); err != nil {
		return nil, err
	}

	processors, err := processors.New(meta.Processors)
	if err != nil {
		return nil, err
	}

	clientConfig := beat.ClientConfig{
		EventMetadata: meta.Event,
		Processor:     processors,
	}
	if p.canDrop {
		clientConfig.PublishMode = beat.DropIfFull
	}

	client, err := p.pipeline.ConnectWith(clientConfig)
	if err != nil {
		return nil, err
	}

	// start worker, so post-processing and processor-pipeline
	// can work concurrently to sniffer acquiring new events
	ch := make(chan beat.Event, 3)
	go p.worker(ch, client)
	return func(event beat.Event) {
		select {
		case ch <- event:
		case <-p.done:
			ch = nil // stop serving more send requests
		}
	}, nil
}

func (p *TransactionPublisher) worker(ch chan beat.Event, client beat.Client) {
	for {
		select {
		case <-p.done:
			return
		case event := <-ch:
			pub, _ := p.processor.Run(&event)
			if pub != nil {
				client.Publish(*pub)
			}
		}
	}
}

func (p *transProcessor) Run(event *beat.Event) (*beat.Event, error) {
	if err := validateEvent(event); err != nil {
		logp.Warn("Dropping invalid event: %v", err)
		return nil, nil
	}

	if !p.normalizeTransAddr(event.Fields) {
		return nil, nil
	}

	if err := marshalPacketbeatFields(event, p.localIPs); err != nil {
		return nil, err
	}
	return event, nil
}

// filterEvent validates an event for common required fields with types.
// If event is to be filtered out the reason is returned as error.
func validateEvent(event *beat.Event) error {
	fields := event.Fields

	if event.Timestamp.IsZero() {
		return errors.New("missing '@timestamp'")
	}

	_, ok := fields["@timestamp"]
	if ok {
		return errors.New("duplicate '@timestamp' field from event")
	}

	t, ok := fields["type"]
	if !ok {
		return errors.New("missing 'type' field from event")
	}

	_, ok = t.(string)
	if !ok {
		return errors.New("invalid 'type' field from event")
	}

	return nil
}

func (p *transProcessor) normalizeTransAddr(event common.MapStr) bool {
	debugf("normalize address for: %v", event)

	var srcServer, dstServer string
	var process common.MapStr
	src, ok := event["src"].(*common.Endpoint)
	debugf("has src: %v", ok)
	if ok {
		delete(event, "src")

		// Check if it's outgoing transaction (as client).
		if p.IsPublisherIP(src.IP) {
			if p.ignoreOutgoing {
				// Duplicated transaction -> ignore it.
				debugf("Ignore duplicated transaction on: %s -> %s", srcServer, dstServer)
				return false
			}

			event.Put("network.direction", "outgoing")
		}

		var client common.MapStr
		client, process = makeEndpoint(p.name, src)
		event.DeepUpdate(common.MapStr{"client": client})
	}

	dst, ok := event["dst"].(*common.Endpoint)
	debugf("has dst: %v", ok)
	if ok {
		delete(event, "dst")

		var server common.MapStr
		server, process = makeEndpoint(p.name, dst)
		event.DeepUpdate(common.MapStr{"server": server})

		// Check if it's incoming transaction (as server).
		if p.IsPublisherIP(dst.IP) {
			event.Put("network.direction", "incoming")
		}
	}

	if len(process) > 0 {
		event.Put("process", process)
	}

	return true
}

func (p *transProcessor) IsPublisherIP(ip string) bool {
	for _, myip := range p.localIPStrings {
		if myip == ip {
			return true
		}
	}
	return false
}

// makeEndpoint builds a map containing the endpoint information. As a
// convenience it returns a reference to the process map that is contained in
// the endpoint map (for use in populating the top-level process field).
func makeEndpoint(shipperName string, endpoint *common.Endpoint) (m common.MapStr, process common.MapStr) {
	// address
	m = common.MapStr{
		"ip":   endpoint.IP,
		"port": endpoint.Port,
	}
	if endpoint.Domain != "" {
		m["domain"] = endpoint.Domain
	} else if shipperName != "" {
		if isLocal, err := common.IsLoopback(endpoint.IP); err == nil && isLocal {
			m["domain"] = shipperName
		}
	}

	// process
	if endpoint.PID > 0 {
		process := common.MapStr{
			"pid":        endpoint.PID,
			"ppid":       endpoint.PPID,
			"name":       endpoint.Name,
			"args":       endpoint.Args,
			"executable": endpoint.Exe,
			"start":      endpoint.StartTime,
		}
		if endpoint.CWD != "" {
			process["working_directory"] = endpoint.CWD
		}
		m["process"] = process
	}

	return m, process
}

func marshalPacketbeatFields(event *beat.Event, localIPs []net.IP) error {
	defer delete(event.Fields, pb.FieldsKey)

	fields, err := pb.GetFields(event.Fields)
	if err != nil || fields == nil {
		return err
	}

	if err := fields.ComputeValues(localIPs); err != nil {
		return err
	}

	return fields.MarshalMapStr(event.Fields)
}
