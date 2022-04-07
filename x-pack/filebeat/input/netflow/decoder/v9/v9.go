// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v9

import (
	"bytes"
	"log"
	"net"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v8/x-pack/filebeat/input/netflow/decoder/config"
	"github.com/elastic/beats/v8/x-pack/filebeat/input/netflow/decoder/protocol"
	"github.com/elastic/beats/v8/x-pack/filebeat/input/netflow/decoder/record"
)

const (
	ProtocolName                 = "v9"
	LogPrefix                    = "[netflow-v9] "
	ProtocolID            uint16 = 9
	MaxSequenceDifference        = 1000
)

type NetflowV9Protocol struct {
	decoder     Decoder
	logger      *log.Logger
	Session     SessionMap
	timeout     time.Duration
	done        chan struct{}
	detectReset bool
}

func init() {
	protocol.Registry.Register(ProtocolName, New)
}

func New(config config.Config) protocol.Protocol {
	logger := log.New(config.LogOutput(), LogPrefix, 0)
	return NewProtocolWithDecoder(DecoderV9{Logger: logger, Fields: config.Fields()}, config, logger)
}

func NewProtocolWithDecoder(decoder Decoder, config config.Config, logger *log.Logger) *NetflowV9Protocol {
	return &NetflowV9Protocol{
		decoder:     decoder,
		Session:     NewSessionMap(logger),
		logger:      logger,
		timeout:     config.ExpirationTimeout(),
		detectReset: config.SequenceResetEnabled(),
	}
}

func (*NetflowV9Protocol) Version() uint16 {
	return ProtocolID
}

func (p *NetflowV9Protocol) Start() error {
	p.done = make(chan struct{})
	if p.timeout != time.Duration(0) {
		go p.Session.CleanupLoop(p.timeout, p.done)
	}
	return nil
}

func (p *NetflowV9Protocol) Stop() error {
	if p.done != nil {
		close(p.done)
	}
	return nil
}

func (p *NetflowV9Protocol) OnPacket(buf *bytes.Buffer, source net.Addr) (flows []record.Record, err error) {
	header, payload, numFlowSets, err := p.decoder.ReadPacketHeader(buf)
	if err != nil {
		p.logger.Printf("Unable to read V9 header: %v", err)
		return nil, errors.Wrapf(err, "error reading header")
	}
	buf = payload

	session := p.Session.GetOrCreate(MakeSessionKey(source, header.SourceID))
	remote := source.String()

	p.logger.Printf("Packet from:%s src:%d seq:%d", remote, header.SourceID, header.SequenceNo)
	if p.detectReset {
		if prev, reset := session.CheckReset(header.SequenceNo); reset {
			p.logger.Printf("Session %s reset (sequence=%d last=%d)", remote, header.SequenceNo, prev)
		}
	}

	for ; numFlowSets > 0; numFlowSets-- {
		set, err := p.decoder.ReadSetHeader(buf)
		if err != nil || set.IsPadding() {
			break
		}
		if buf.Len() < set.BodyLength() {
			p.logger.Printf("FlowSet ID %+v overflows packet from %s", set, source)
			break
		}
		body := bytes.NewBuffer(buf.Next(set.BodyLength()))
		p.logger.Printf("FlowSet ID %d length %d", set.SetID, set.BodyLength())

		f, err := p.parseSet(set.SetID, session, body)
		if err != nil {
			p.logger.Printf("Error parsing set %d: %v", set.SetID, err)
			return nil, errors.Wrapf(err, "error parsing set")
		}
		flows = append(flows, f...)
	}
	metadata := header.ExporterMetadata(source)
	for idx := range flows {
		flows[idx].Exporter = metadata
		flows[idx].Timestamp = header.UnixSecs
	}
	return flows, nil
}

func (p *NetflowV9Protocol) parseSet(
	setID uint16,
	session *SessionState,
	buf *bytes.Buffer) (flows []record.Record, err error,
) {
	if setID >= 256 {
		// Flow of Options record, lookup template and generate flows
		if template := session.GetTemplate(setID); template != nil {
			return template.Apply(buf, 0)
		}
		p.logger.Printf("No template for ID %d", setID)
		return nil, nil
	}

	// Template sets
	templates, err := p.decoder.ReadTemplateSet(setID, buf)
	if err != nil {
		return nil, err
	}
	for _, template := range templates {
		session.AddTemplate(template)
	}
	return flows, nil
}
