// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package ipfix

import (
	"bytes"
	"encoding/hex"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/menderesk/beats/v7/x-pack/filebeat/input/netflow/decoder/config"
	"github.com/menderesk/beats/v7/x-pack/filebeat/input/netflow/decoder/fields"
	"github.com/menderesk/beats/v7/x-pack/filebeat/input/netflow/decoder/record"
	"github.com/menderesk/beats/v7/x-pack/filebeat/input/netflow/decoder/test"
	v9 "github.com/menderesk/beats/v7/x-pack/filebeat/input/netflow/decoder/v9"
)

func TestMessageWithOptions(t *testing.T) {
	rawString := "" +
		"000a01e45bf435e1000000a500000000000200480400001000080004000c0004" +
		"0001000400020004000a0004000e000400070002000b00020004000100060001" +
		"003c00010005000100200002003a000200160004001500040002004808000010" +
		"001b0010001c00100001000400020004000a0004000e000400070002000b0002" +
		"0004000100060001003c000100050001008b0002003a00020016000400150004" +
		"0003001e010000050001008f000400a000080130000201310002013200040100" +
		"00180000e9160000016731f277e100010001000000630400010ed83acd35d5da" +
		"354b0000002e0000000100000000000000000fb9005006100400000000006a53" +
		"cb3c6a53cb3c6f4de601d5da354b000000300000000100000000000000008022" +
		"005006180400000000006a53cb3c6a53cb3cd69bae4fd5da354b000000340000" +
		"000100000000000000007a51005006180400000000006a53cb3c6a53cb3cb9ae" +
		"3002d5da354b00000034000000010000000000000000e1e50050061804000000" +
		"00006a53cb3c6a53cb3cd83acd56d5da354b0000002e00000001000000000000" +
		"0000d317005006100400000000006a53cb3c6a53cb3cdbbb956bd5da354b0000" +
		"003c000000010000000000000000b235005006180400000000006a53cb3c6a53" +
		"cb3c0000"
	raw, err := hex.DecodeString(rawString)
	assert.NoError(t, err)

	captureTimeMillis, err := time.Parse(time.RFC3339, "2018-11-20T16:27:13.249Z")
	if !assert.NoError(t, err) {
		t.Fatal(err)
	}
	captureTime := time.Unix(captureTimeMillis.Unix(), 0).UTC()
	expected := record.Record{
		Type:      record.Options,
		Timestamp: captureTime,
		Fields: record.Map{
			"scope": record.Map{
				"meteringProcessId": uint64(59670),
			},
			"options": record.Map{
				"samplingPacketInterval":     uint64(1),
				"samplingPacketSpace":        uint64(99),
				"selectorAlgorithm":          uint64(1),
				"systemInitTimeMilliseconds": captureTimeMillis,
			},
		},
		Exporter: record.Map{
			"address":      "127.0.0.1:1234",
			"sourceId":     uint64(0),
			"timestamp":    captureTime,
			"uptimeMillis": uint64(0),
			"version":      uint64(10),
		},
	}
	proto := New(config.Defaults())
	flows, err := proto.OnPacket(bytes.NewBuffer(raw), test.MakeAddress(t, "127.0.0.1:1234"))
	assert.NoError(t, err)
	if assert.Len(t, flows, 7) {
		assert.Equal(t, record.Options, flows[0].Type)
		test.AssertRecordsEqual(t, expected, flows[0])
		for i := 1; i < len(flows); i++ {
			assert.Equal(t, record.Flow, flows[i].Type)
		}
	}
}

func TestOptionTemplates(t *testing.T) {
	addr := test.MakeAddress(t, "127.0.0.1:12345")
	key := v9.MakeSessionKey(addr, 1234)

	t.Run("Single options template", func(t *testing.T) {
		proto := New(config.Defaults())
		flows, err := proto.OnPacket(test.MakePacket([]uint16{
			// Header
			// Version, Length, Ts, SeqNo, Source
			10, 40, 11, 11, 22, 22, 0, 1234,
			// Set #1 (options template)
			3, 24, /*len of set*/
			999, 3 /*total field count */, 1, /*scope field count*/
			1, 4, // Fields
			2, 4,
			3, 4,
			0, // Padding
		}), addr)
		assert.NoError(t, err)
		assert.Empty(t, flows)

		ipfix, ok := proto.(*IPFixProtocol)
		assert.True(t, ok)
		v9proto := &ipfix.NetflowV9Protocol
		assert.Len(t, v9proto.Session.Sessions, 1)
		s, found := v9proto.Session.Sessions[key]
		assert.True(t, found)
		assert.Len(t, s.Templates, 1)
		opt := s.GetTemplate(999)
		assert.NotNil(t, opt)
		assert.Equal(t, 1, opt.ScopeFields)
	})

	t.Run("Multiple options template", func(t *testing.T) {
		proto := New(config.Defaults())
		raw := test.MakePacket([]uint16{
			// Header
			// Version, Count, Ts, SeqNo, Source
			10, 66, 11, 11, 22, 22, 0, 1234,
			// Set #1 (options template)
			3, 22 + 26, /*len of set*/
			999, 3 /*total field count*/, 2, /*scope field count*/
			1, 4, // Fields
			2, 4,
			3, 4,
			998, 5, 3,
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

		ipfix, ok := proto.(*IPFixProtocol)
		v9proto := &ipfix.NetflowV9Protocol
		assert.True(t, ok)
		assert.Len(t, v9proto.Session.Sessions, 1)
		s, found := v9proto.Session.Sessions[key]
		assert.True(t, found)
		assert.Len(t, s.Templates, 2)
		for _, id := range []uint16{998, 999} {
			opt := s.GetTemplate(id)
			assert.NotNil(t, opt)
			assert.True(t, opt.ScopeFields > 0)
		}
	})

	t.Run("records discarded", func(t *testing.T) {
		proto := New(config.Defaults())
		raw := test.MakePacket([]uint16{
			// Header
			// Version, Count, Ts, SeqNo, Source
			10, 24, 11, 11, 22, 22, 0, 1234,
			// Set #1 (options template)
			9998, 8, /*len of set*/
			1, 2,
		})
		flows, err := proto.OnPacket(raw, addr)
		assert.NoError(t, err)
		assert.Empty(t, flows)

		ipfix, ok := proto.(*IPFixProtocol)
		assert.True(t, ok)
		v9proto := &ipfix.NetflowV9Protocol

		assert.Len(t, v9proto.Session.Sessions, 1)
		s, found := v9proto.Session.Sessions[key]
		assert.True(t, found)
		assert.Len(t, s.Templates, 0)

		raw = test.MakePacket([]uint16{
			// Header
			// Version, Count, Ts, SeqNo, Source
			10, 30, 11, 11, 22, 22, 0, 1234,
			// Set #1 (options template)
			3, 14, /*len of set*/
			9998, 1, 1,
			3, 4,
		})
		flows, err = proto.OnPacket(raw, addr)
		assert.NoError(t, err)
		assert.Empty(t, flows)
		assert.Len(t, v9proto.Session.Sessions, 1)
		assert.Len(t, s.Templates, 1)
	})
}

func TestCustomFields(t *testing.T) {
	addr := test.MakeAddress(t, "127.0.0.1:12345")

	conf := config.Defaults()
	conf.WithCustomFields(fields.FieldDict{
		fields.Key{EnterpriseID: 0x12345678, FieldID: 33}: &fields.Field{Name: "customField", Decoder: fields.String},
	})
	assert.Contains(t, conf.Fields(), fields.Key{EnterpriseID: 0x12345678, FieldID: 33})
	proto := New(conf)
	flows, err := proto.OnPacket(test.MakePacket([]uint16{
		// Header
		// Version, Length, Ts, SeqNo, Source
		10, 42, 11, 11, 22, 22, 0, 1234,
		// Set #1 (record template)
		2, 26, /*len of set*/
		999, 3,
		1, 4, // Field 1
		2, 4, // Field 2
		// Field 3
		0x8000 | 33, 6,
		0x1234, 0x5678, // enterprise ID
		0, // Padding
	}), addr)
	assert.NoError(t, err)
	assert.Empty(t, flows)

	flows, err = proto.OnPacket(test.MakePacket([]uint16{
		// Header
		// Version, Length, Ts, SeqNo, Source
		10, 34, 11, 11, 22, 22, 0, 1234,
		// Set  (data record)
		999, 18, /*len of 999 record */
		0x0102, 0x0304, // field 1
		0x0506, 0x0708, // field 2
		// Field 3
		0x5465, 0x7374,
		0x4d65,
	}), addr)
	assert.NoError(t, err)
	assert.Len(t, flows, 1)
	assert.Contains(t, flows[0].Fields, "customField")
	assert.Equal(t, flows[0].Fields["customField"], "TestMe")
}
