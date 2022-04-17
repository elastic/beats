// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package ipfix

import (
	"log"

	"github.com/menderesk/beats/v7/x-pack/filebeat/input/netflow/decoder/config"
	"github.com/menderesk/beats/v7/x-pack/filebeat/input/netflow/decoder/protocol"
	v9 "github.com/menderesk/beats/v7/x-pack/filebeat/input/netflow/decoder/v9"
)

const (
	ProtocolName        = "ipfix"
	ProtocolID   uint16 = 10
	LogPrefix           = "[ipfix] "
)

type IPFixProtocol struct {
	v9.NetflowV9Protocol
}

var _ protocol.Protocol = (*IPFixProtocol)(nil)

func init() {
	protocol.Registry.Register(ProtocolName, New)
}

func New(config config.Config) protocol.Protocol {
	logger := log.New(config.LogOutput(), LogPrefix, 0)
	decoder := DecoderIPFIX{
		DecoderV9: v9.DecoderV9{Logger: logger, Fields: config.Fields()},
	}
	proto := &IPFixProtocol{
		NetflowV9Protocol: *v9.NewProtocolWithDecoder(decoder, config, logger),
	}
	return proto
}

func (*IPFixProtocol) Version() uint16 {
	return ProtocolID
}
