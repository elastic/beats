// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package protocol

import (
	"bytes"
	"net"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/netflow/decoder/record"
)

// Protocol is the interface that NetFlow protocol parsers must conform to.
type Protocol interface {
	// Version returns the NetFlow version that this protocol implements.
	// The version number in packet headers is compared with this value to
	// select the appropriate protocol parser.
	Version() uint16

	// OnPacket is the main callback to decode network packets. It receives
	// the packet payload and the network source (address of the exporter)
	// and extracts any records contained in the packet.
	OnPacket(buf *bytes.Buffer, source net.Addr) ([]record.Record, error)

	// Start initializes the Protocol. This is necessary so that background
	// routines (i.e. to expire sessions) are required.
	Start() error

	// Stop stops any running goroutines and frees any other resources that
	// the protocol parser might be using.
	Stop() error
}
