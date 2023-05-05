// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package shipper

import (
	"time"

	"github.com/elastic/elastic-agent-libs/config"
)

// Instance represents all the config needed to start a single shipper input
// because a beat works fundamentally differently from the old shipper, we dont have to deal with async config that's being pieced together,
// this one config object recievewd on create has both the input and output config
type Instance struct {
	// config for the shipper's gRPC input
	Conn  ConnectionConfig `config:",inline"`
	Input InputConfig      `config:",inline"`
}

// ConnectionConfig is the shipper-relevant portion of the config received from input units
type ConnectionConfig struct {
	Server         string        `config:"server"`
	InitialTimeout time.Duration `config:"grpc_setup_timeout"`
	TLS            TLS           `config:"ssl"`
}

// TLS is TLS-specific shipper client settings
type TLS struct {
	CAs  []string `config:"certificate_authorities"`
	Cert string   `config:"certificate"`
	Key  string   `config:"key"`
}

// InputConfig represents the config for a shipper input. This is the complete config for that input, mirrored and sent to us.
// This is more or less the same as the the proto.UnitExpectedConfig type, but that doesn't have `config` struct tags,
// so for the sake of quick prototyping we're just (roughly) duplicating the structure here, minus any fields the shipper doesn't need (for now)
type InputConfig struct {
	ID         string     `config:"id"`
	Type       string     `config:"type"`
	Name       string     `config:"name"`
	DataStream DataStream `config:"data_stream"`
	// for now don't try to parse the streams,
	// once we have a better idea of how per-stream processors work, we can find a better way to unpack this
	Streams []Stream `config:"streams"`
}

// DataStream represents the datastream metadata from an input
type DataStream struct {
	Dataset   string `config:"dataset"`
	Type      string `config:"type"`
	Namespace string `config:"namespace"`
}

// Stream represents a single stream present inside an input.
// this field is largely unpredictable and varies by input type,
// we're just grabbing the fields the shipper needs.
type Stream struct {
	ID         string      `config:"id"`
	Processors []*config.C `config:"processors"`
	Index      string      `config:"index"`
}
