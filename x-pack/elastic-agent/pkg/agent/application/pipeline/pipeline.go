// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pipeline

import (
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/configrequest"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
)

// ConfigHandler is capable of handling configrequest.
type ConfigHandler interface {
	HandleConfig(configrequest.Request) error
	Close() error
	Shutdown()
}

// DefaultRK default routing keys until we implement the routing key / config matrix.
var DefaultRK = "DEFAULT"

// RoutingKey is used for routing as pipeline id.
type RoutingKey = string

// Dispatcher is an interace dispatching programs to correspongind stream
type Dispatcher interface {
	Dispatch(id string, grpProg map[RoutingKey][]program.Program) error
	Shutdown()
}

// StreamFunc creates a stream out of routing key.
type StreamFunc func(*logger.Logger, RoutingKey) (Stream, error)

// Stream is capable of executing configrequest change.
type Stream interface {
	Execute(configrequest.Request) error
	Close() error
	Shutdown()
}

// EmitterFunc emits configuration for processing.
type EmitterFunc func(*config.Config) error
