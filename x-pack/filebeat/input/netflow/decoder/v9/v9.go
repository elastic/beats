// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v9

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/netflow/decoder/config"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/netflow/decoder/protocol"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/netflow/decoder/record"
)

const (
	ProtocolName                      = "v9"
	LogPrefix                         = "[netflow-v9] "
	ProtocolID                 uint16 = 9
	MaxSequenceDifference             = 1000
	cacheCleanupInterval              = 30 * time.Second
	cacheCleanupEntryThreshold        = 10 * time.Second
)

type NetflowV9Protocol struct {
	ctx            context.Context
	cancel         context.CancelFunc
	decoder        Decoder
	logger         *logp.Logger
	Session        SessionMap
	timeout        time.Duration
	cache          *pendingTemplatesCache
	detectReset    bool
	shareTemplates bool
}

func init() {
	if err := protocol.Registry.Register(ProtocolName, New); err != nil {
		panic(err)
	}
}

func New(config config.Config) protocol.Protocol {
	logger := config.LogOutput().Named(LogPrefix)
	return NewProtocolWithDecoder(DecoderV9{Logger: logger, Fields: config.Fields()}, config, logger)
}

func NewProtocolWithDecoder(decoder Decoder, config config.Config, logger *logp.Logger) *NetflowV9Protocol {
	ctx, cancel := context.WithCancel(context.Background())
	pd := &NetflowV9Protocol{
		ctx:            ctx,
		cancel:         cancel,
		decoder:        decoder,
		logger:         logger,
		Session:        NewSessionMap(logger, config.ActiveSessionsMetric()),
		timeout:        config.ExpirationTimeout(),
		detectReset:    config.SequenceResetEnabled(),
		shareTemplates: config.ShareTemplatesEnabled(),
	}

	if config.Cache() {
		pd.cache = newPendingTemplatesCache()
	}

	return pd
}

func (*NetflowV9Protocol) Version() uint16 {
	return ProtocolID
}

func (p *NetflowV9Protocol) Start() error {
	if p.timeout != time.Duration(0) {
		go p.Session.CleanupLoop(p.timeout, p.ctx.Done())
	}

	if p.cache != nil {
		p.cache.start(p.ctx.Done(), cacheCleanupInterval, cacheCleanupEntryThreshold)
	}

	return nil
}

func (p *NetflowV9Protocol) Stop() error {
	p.cancel()
	if p.cache != nil {
		p.cache.wait()
	}
	return nil
}

func (p *NetflowV9Protocol) OnPacket(buf *bytes.Buffer, source net.Addr) (flows []record.Record, err error) {
	header, payload, numFlowSets, err := p.decoder.ReadPacketHeader(buf)
	if err != nil {
		p.logger.Debugf("Unable to read V9 header: %v", err)
		return nil, fmt.Errorf("error reading header: %w", err)
	}
	buf = payload

	sessionKey := MakeSessionKey(source, header.SourceID, p.shareTemplates)

	session := p.Session.GetOrCreate(sessionKey)
	remote := source.String()

	p.logger.Debugf("Packet from:%s src:%d seq:%d", remote, header.SourceID, header.SequenceNo)
	if p.detectReset {
		if prev, reset := session.CheckReset(header.SequenceNo); reset {
			p.logger.Debugf("Session %s reset (sequence=%d last=%d)", remote, header.SequenceNo, prev)
		}
	}

	for ; numFlowSets > 0; numFlowSets-- {
		set, err := p.decoder.ReadSetHeader(buf)
		if err != nil || set.IsPadding() {
			break
		}
		if buf.Len() < set.BodyLength() {
			p.logger.Debugf("FlowSet ID %+v overflows packet from %s", set, source)
			break
		}
		body := bytes.NewBuffer(buf.Next(set.BodyLength()))
		p.logger.Debugf("FlowSet ID %d length %d", set.SetID, set.BodyLength())

		f, err := p.parseSet(set.SetID, sessionKey, session, body)
		if err != nil {
			p.logger.Debugf("Error parsing set %d: %v", set.SetID, err)
			return nil, fmt.Errorf("error parsing set: %w", err)
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
	key SessionKey,
	session *SessionState,
	buf *bytes.Buffer) (flows []record.Record, err error,
) {
	if setID >= 256 {
		// Flow of Options record, lookup template and generate flows
		template := session.GetTemplate(setID)

		if template == nil {
			if p.cache != nil {
				p.cache.Add(key, buf)
			} else {
				p.logger.Debugf("No template for ID %d", setID)
			}
			return nil, nil
		}

		return template.Apply(buf, 0)
	}

	// Template sets
	templates, err := p.decoder.ReadTemplateSet(setID, buf)
	if err != nil {
		return nil, err
	}
	for _, template := range templates {
		session.AddTemplate(template)

		if p.cache == nil {
			continue
		}
		events := p.cache.GetAndRemove(key)
		for _, e := range events {
			f, err := template.Apply(e, 0)
			if err != nil {
				continue
			}
			flows = append(flows, f...)
		}
	}

	return flows, nil
}
