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

	"github.com/elastic/beats/v8/libbeat/beat"
	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/logp"
	"github.com/elastic/beats/v8/libbeat/processors"
	"github.com/elastic/beats/v8/packetbeat/pb"
)

type TransactionPublisher struct {
	done      chan struct{}
	pipeline  beat.Pipeline
	canDrop   bool
	processor transProcessor
}

type transProcessor struct {
	ignoreOutgoing   bool
	localIPs         []net.IP // TODO: Periodically update this list.
	internalNetworks []string
	name             string
}

var debugf = logp.MakeDebug("publish")

func NewTransactionPublisher(
	name string,
	pipeline beat.Pipeline,
	ignoreOutgoing bool,
	canDrop bool,
	internalNetworks []string,
) (*TransactionPublisher, error) {
	addrs, err := common.LocalIPAddrs()
	if err != nil {
		return nil, err
	}
	var localIPs []net.IP
	for _, addr := range addrs {
		if !addr.IsLoopback() {
			localIPs = append(localIPs, addr)
		}
	}

	p := &TransactionPublisher{
		done:     make(chan struct{}),
		pipeline: pipeline,
		canDrop:  canDrop,
		processor: transProcessor{
			localIPs:         localIPs,
			internalNetworks: internalNetworks,
			name:             name,
			ignoreOutgoing:   ignoreOutgoing,
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
		Index      string                  `config:"index"`
		Event      common.EventMetadata    `config:",inline"`
		Processors processors.PluginConfig `config:"processors"`
		KeepNull   bool                    `config:"keep_null"`
	}{}
	if err := config.Unpack(&meta); err != nil {
		return nil, err
	}

	processors, err := processors.New(meta.Processors)
	if err != nil {
		return nil, err
	}

	clientConfig := beat.ClientConfig{
		Processing: beat.ProcessingConfig{
			EventMetadata: meta.Event,
			Processor:     processors,
			KeepNull:      meta.KeepNull,
		},
	}
	if p.canDrop {
		clientConfig.PublishMode = beat.DropIfFull
	}
	if meta.Index != "" {
		clientConfig.Processing.Meta = common.MapStr{"raw_index": meta.Index}
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

	fields, err := MarshalPacketbeatFields(event, p.localIPs, p.internalNetworks)
	if err != nil {
		return nil, err
	}

	if fields != nil {
		if p.ignoreOutgoing && fields.Network.Direction == pb.Egress {
			debugf("Ignore outbound transaction on: %s -> %s",
				fields.Source.IP, fields.Destination.IP)
			return nil, nil
		}
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

// MarshalPacketbeatFields marshals data contained in the _packetbeat field
// into the event and removes the _packetbeat key.
func MarshalPacketbeatFields(event *beat.Event, localIPs []net.IP, internalNetworks []string) (*pb.Fields, error) {
	defer delete(event.Fields, pb.FieldsKey)

	fields, err := pb.GetFields(event.Fields)
	if err != nil || fields == nil {
		return nil, err
	}

	if err = fields.ComputeValues(localIPs, internalNetworks); err != nil {
		return nil, err
	}

	if err = fields.MarshalMapStr(event.Fields); err != nil {
		return nil, err
	}
	return fields, nil
}
