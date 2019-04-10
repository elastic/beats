// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v9

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/x-pack/filebeat/input/netflow/decoder/config"
	"github.com/elastic/beats/x-pack/filebeat/input/netflow/decoder/fields"
	"github.com/elastic/beats/x-pack/filebeat/input/netflow/decoder/test"
)

func TestNetflowV9Protocol_ID(t *testing.T) {
	assert.Equal(t, ProtocolID, New(config.Defaults()).Version())
}

func TestNetflowProtocol_New(t *testing.T) {
	proto := New(config.Defaults())

	assert.Nil(t, proto.Start())
	assert.Equal(t, uint16(9), proto.Version())
	assert.Nil(t, proto.Stop())
}

func TestOptionTemplates(t *testing.T) {
	addr := test.MakeAddress(t, "127.0.0.1:12345")
	key := MakeSessionKey(addr)

	t.Run("Single options template", func(t *testing.T) {
		proto := New(config.Defaults())
		flows, err := proto.OnPacket(test.MakePacket([]uint16{
			// Header
			// Version, Count, Uptime, Ts, SeqNo, Source
			9, 1, 11, 11, 22, 22, 33, 33, 0, 1234,
			// Set #1 (options template)
			1, 24, /*len of set*/
			999, 4 /*scope len*/, 8, /*opts len*/
			1, 4, // Fields
			2, 4,
			3, 4,
			0, // Padding
		}), addr)
		assert.NoError(t, err)
		assert.Empty(t, flows)

		v9proto, ok := proto.(*NetflowV9Protocol)
		assert.True(t, ok)

		assert.Len(t, v9proto.Session.Sessions, 1)
		s, found := v9proto.Session.Sessions[key]
		assert.True(t, found)
		assert.Len(t, s.Templates, 1)
		opt := s.GetTemplate(1234, 999)
		assert.NotNil(t, opt)
		assert.True(t, opt.ScopeFields > 0)
	})

	t.Run("Multiple options template", func(t *testing.T) {
		proto := New(config.Defaults())
		raw := test.MakePacket([]uint16{
			// Header
			// Version, Count, Uptime, Ts, SeqNo, Source
			9, 2, 11, 11, 22, 22, 33, 33, 0, 1234,
			// Set #1 (options template)
			1, 22 + 26, /*len of set*/
			999, 4 /*scope len*/, 8, /*opts len*/
			1, 4, // Fields
			2, 4,
			3, 4,
			998, 8, 12,
			1, 4,
			2, 2,
			3, 3,
			4, 1,
			5, 1,
			0,
		})
		flows, err := proto.OnPacket(raw, addr)
		assert.NoError(t, err)
		assert.Empty(t, flows)

		v9proto, ok := proto.(*NetflowV9Protocol)
		assert.True(t, ok)
		assert.Len(t, v9proto.Session.Sessions, 1)
		s, found := v9proto.Session.Sessions[key]
		assert.True(t, found)
		assert.Len(t, s.Templates, 2)
		for _, id := range []uint16{998, 999} {
			opt := s.GetTemplate(1234, id)
			assert.NotNil(t, opt)
			assert.True(t, opt.ScopeFields > 0)
		}
	})

	t.Run("records discarded", func(t *testing.T) {
		proto := New(config.Defaults())
		raw := test.MakePacket([]uint16{
			// Header
			// Version, Count, Uptime, Ts, SeqNo, Source
			9, 1, 11, 11, 22, 22, 33, 33, 0, 1234,
			// Set #1 (options template)
			9998, 8, /*len of set*/
			1, 2,
		})
		flows, err := proto.OnPacket(raw, addr)
		assert.NoError(t, err)
		assert.Empty(t, flows)

		v9proto, ok := proto.(*NetflowV9Protocol)
		assert.True(t, ok)

		assert.Len(t, v9proto.Session.Sessions, 1)
		s, found := v9proto.Session.Sessions[key]
		assert.True(t, found)
		assert.Len(t, s.Templates, 0)

		raw = test.MakePacket([]uint16{
			// Header
			// Version, Count, Uptime, Ts, SeqNo, Source
			9, 1, 11, 11, 22, 22, 33, 33, 0, 1234,
			// Set #1 (options template)
			1, 14, /*len of set*/
			9998, 4, 0,
			3, 4,
		})
		flows, err = proto.OnPacket(raw, addr)
		assert.NoError(t, err)
		assert.Empty(t, flows)
		assert.Len(t, v9proto.Session.Sessions, 1)
		assert.Len(t, s.Templates, 1)
	})
}

func TestSessionReset(t *testing.T) {
	addr := test.MakeAddress(t, "127.0.0.1:12345")
	templatePacket := []uint16{
		// Header
		// Version, Count, Uptime, Ts, SeqNo, Source
		9, 1, 11, 11, 22, 22, 33, 33, 0, 1234,
		// Set #1 (template)
		0, 20, /*len of set*/
		999, 3, /*len*/
		1, 4, // Fields
		2, 4,
		3, 4,
	}
	flowsPacket := []uint16{
		// Header
		// Version, Count, Uptime, Ts, SeqNo, Source
		9, 1, 11, 11, 22, 22, 00, 33, 0, 1234,
		// Set #1 (template)
		999, 16, /*len of set*/
		1, 1,
		2, 2,
		3, 3,
	}
	t.Run("Reset disabled", func(t *testing.T) {
		cfg := config.Defaults()
		cfg.WithSequenceResetEnabled(false).WithLogOutput(test.TestLogWriter{TB: t})
		proto := New(cfg)
		flows, err := proto.OnPacket(test.MakePacket(templatePacket), addr)
		assert.NoError(t, err)
		assert.Empty(t, flows)
		flows, err = proto.OnPacket(test.MakePacket(flowsPacket), addr)
		assert.NoError(t, err)
		assert.Len(t, flows, 1)
	})
	t.Run("Reset enabled", func(t *testing.T) {
		cfg := config.Defaults()
		cfg.WithSequenceResetEnabled(true).WithLogOutput(test.TestLogWriter{TB: t})
		proto := New(cfg)
		flows, err := proto.OnPacket(test.MakePacket(templatePacket), addr)
		assert.NoError(t, err)
		assert.Empty(t, flows)
		flows, err = proto.OnPacket(test.MakePacket(flowsPacket), addr)
		assert.NoError(t, err)
		assert.Empty(t, flows)
	})
}

func TestCustomFields(t *testing.T) {
	addr := test.MakeAddress(t, "127.0.0.1:12345")

	conf := config.Defaults()
	conf.WithCustomFields(fields.FieldDict{
		fields.Key{FieldID: 33333}: &fields.Field{Name: "customField", Decoder: fields.String},
	})
	assert.Contains(t, conf.Fields(), fields.Key{FieldID: 33333})
	proto := New(conf)
	flows, err := proto.OnPacket(test.MakePacket([]uint16{
		// Header
		// Version, Count, Uptime, Ts, SeqNo, Source
		9, 1, 11, 11, 22, 22, 33, 33, 0, 1234,
		// Set #1 (template)
		0, 20, /*len of set*/
		999, 3, /*len*/
		1, 4, // Fields
		2, 4,
		33333, 8,
	}), addr)
	assert.NoError(t, err)
	assert.Empty(t, flows)

	flows, err = proto.OnPacket(test.MakePacket([]uint16{
		// Header
		// Version, Count, Uptime, Ts, SeqNo, Source
		9, 1, 11, 11, 22, 22, 33, 34, 0, 1234,
		// Set #1 (template)
		999, 20, /*len of set*/
		1, 1,
		2, 2,
		0x4865, 0x6c6c,
		0x6f20, 0x3a29,
	}), addr)
	assert.NoError(t, err)
	assert.Len(t, flows, 1)
	assert.Contains(t, flows[0].Fields, "customField")
	assert.Equal(t, flows[0].Fields["customField"], "Hello :)")
}
