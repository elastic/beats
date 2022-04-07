// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package protocol

import (
	"bytes"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v8/x-pack/filebeat/input/netflow/decoder/config"
	"github.com/elastic/beats/v8/x-pack/filebeat/input/netflow/decoder/record"
)

type testProto int

func (testProto) Version() uint16 {
	return 42
}

func (testProto) OnPacket(*bytes.Buffer, net.Addr) ([]record.Record, error) {
	return nil, nil
}

func (testProto) Start() error {
	return nil
}

func (testProto) Stop() error {
	return nil
}

func testFactory(value int) ProtocolFactory {
	return func(_ config.Config) Protocol {
		return testProto(value)
	}
}

func TestRegistry_Register(t *testing.T) {
	t.Run("valid protocol", func(t *testing.T) {
		registry := ProtocolRegistry{}
		err := registry.Register("my_proto", testFactory(0))
		assert.NoError(t, err)
	})
	t.Run("duplicate protocol", func(t *testing.T) {
		registry := ProtocolRegistry{}
		err := registry.Register("my_proto", testFactory(0))
		assert.NoError(t, err)
		err = registry.Register("my_proto", testFactory(1))
		assert.Error(t, err)
	})
}

func TestRegistry_Get(t *testing.T) {
	t.Run("valid protocol", func(t *testing.T) {
		registry := ProtocolRegistry{}
		err := registry.Register("my_proto", testFactory(0))
		assert.NoError(t, err)
		gen, err := registry.Get("my_proto")
		assert.NoError(t, err)
		assert.Equal(t, testProto(0), gen(config.Defaults()))
	})
	t.Run("two protocols", func(t *testing.T) {
		registry := ProtocolRegistry{}
		err := registry.Register("my_proto", testFactory(1))
		assert.NoError(t, err)
		err = registry.Register("other_proto", testFactory(2))
		assert.NoError(t, err)
		gen, err := registry.Get("my_proto")
		assert.NoError(t, err)
		assert.Equal(t, testProto(1), gen(config.Defaults()))
		gen, err = registry.Get("other_proto")
		assert.NoError(t, err)
		assert.Equal(t, testProto(2), gen(config.Defaults()))
	})
	t.Run("not registered", func(t *testing.T) {
		registry := ProtocolRegistry{}
		_, err := registry.Get("my_proto")
		assert.Error(t, err)
	})
}

func TestRegistry_All(t *testing.T) {
	protos := map[string]int{
		"proto1": 1,
		"proto2": 2,
		"proto3": 2,
	}
	registry := ProtocolRegistry{}
	for key, value := range protos {
		if err := registry.Register(key, testFactory(value)); err != nil {
			t.Fatal(err)
		}
	}
	names := registry.All()
	assert.Len(t, names, len(protos))
	for _, name := range names {
		_, found := protos[name]
		assert.True(t, found)
	}
}
