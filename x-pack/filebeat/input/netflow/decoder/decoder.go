// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package decoder

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"sync"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/netflow/decoder/config"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/netflow/decoder/protocol"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/netflow/decoder/record"
)

// Decoder is a NetFlow decoder that accepts network packets from an Exporter
// and returns the NetFlow records contained in them.
type Decoder struct {
	mutex   sync.Mutex
	protos  map[uint16]protocol.Protocol
	started bool
	logger  log.Logger
}

// NewDecoder returns a new NetFlow decoder configured using the passed
// configuration.
func NewDecoder(config *config.Config) (*Decoder, error) {
	decoder := &Decoder{
		protos: make(map[uint16]protocol.Protocol, len(config.Protocols())),
	}
	for _, protoName := range config.Protocols() {
		factory, err := protocol.Registry.Get(protoName)
		if err != nil {
			return nil, err
		}
		proto := factory(*config)
		decoder.protos[proto.Version()] = proto
	}
	return decoder, nil
}

// Start will start some necessary background tasks in the decoder, mainly for
// session and template expiration in NetFlow 9 and IPFIX.
func (p *Decoder) Start() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.started {
		return errors.New("already started")
	}

	for _, proto := range p.protos {
		if err := proto.Start(); err != nil {
			p.stop()
			return errors.Wrapf(err, "failed to start protocol version %d", proto.Version())
		}
	}
	p.started = true
	return nil
}

// Stop will stop any background tasks running withing the decoder.
func (p *Decoder) Stop() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if !p.started {
		return errors.New("already stopped")
	}
	p.started = false
	return p.stop()
}

// Read will process a NetFlow packet received from the network.
// source is the address for the NetFlow exporter that sent the packet.
// It returns the (possibly empty) list of records extracted from the packet.
func (p *Decoder) Read(buf *bytes.Buffer, source net.Addr) (records []record.Record, err error) {
	if buf.Len() < 2 {
		return nil, io.EOF
	}
	version := binary.BigEndian.Uint16(buf.Bytes()[:2])

	handler, exists := p.protos[version]
	if !exists {
		return nil, fmt.Errorf("netflow protocol version %d not supported", version)
	}
	return handler.OnPacket(buf, source)
}

// NewConfig returns a new configuration structure to be passed to NewDecoder.
func NewConfig() *config.Config {
	cfg := config.Defaults()
	return &cfg
}

func (p *Decoder) stop() error {
	for _, proto := range p.protos {
		if err := proto.Stop(); err != nil {
			p.logger.Printf("Error stopping protocol %d: %v", proto.Version(), err)
		}
	}
	return nil
}
