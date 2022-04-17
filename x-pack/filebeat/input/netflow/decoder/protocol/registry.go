// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package protocol

import (
	"fmt"
	"strings"

	"github.com/menderesk/beats/v7/x-pack/filebeat/input/netflow/decoder/config"
)

// Registry is the global instance of the ProtocolRegistry. Protocol handlers
// must register themselves in this registry to be discoverable.
var Registry ProtocolRegistry = make(map[string]ProtocolFactory)

// ProtocolFactory is the type for a factory method that creates instances
// of a protocol.
type ProtocolFactory func(config config.Config) Protocol

// ProtocolRegistry allows protocols to be registered and be discovered by
// their protocol name.
type ProtocolRegistry map[string]ProtocolFactory

// Register registers a new protocol into the registry.
func (r ProtocolRegistry) Register(name string, factory ProtocolFactory) error {
	name = strings.ToLower(name)
	if _, exists := r[name]; exists {
		return fmt.Errorf("protocol '%s' already registered", name)
	}
	r[name] = factory
	return nil
}

// Get returns a ProtocolFactory for a registered protocol.
func (r ProtocolRegistry) Get(name string) (ProtocolFactory, error) {
	name = strings.ToLower(name)
	if generator, found := r[name]; found {
		return generator, nil
	}
	return nil, fmt.Errorf("protocol named '%s' not found", name)
}

// All returns a list of the registered protocol names.
func (r ProtocolRegistry) All() (names []string) {
	names = make([]string, 0, len(r))
	for proto := range r {
		names = append(names, proto)
	}
	return names
}
