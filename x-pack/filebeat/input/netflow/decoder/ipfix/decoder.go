// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package ipfix

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"time"

	"github.com/menderesk/beats/v7/x-pack/filebeat/input/netflow/decoder/fields"
	"github.com/menderesk/beats/v7/x-pack/filebeat/input/netflow/decoder/template"
	v9 "github.com/menderesk/beats/v7/x-pack/filebeat/input/netflow/decoder/v9"
)

const (
	TemplateFlowSetID           = 2
	TemplateOptionsSetID        = 3
	EnterpriseBit        uint16 = 0x8000
	SizeOfIPFIXHeader    uint16 = 16
)

type DecoderIPFIX struct {
	v9.DecoderV9
}

var _ v9.Decoder = (*DecoderIPFIX)(nil)

func (_ DecoderIPFIX) ReadPacketHeader(buf *bytes.Buffer) (header v9.PacketHeader, newBuf *bytes.Buffer, countRecords int, err error) {
	var data [SizeOfIPFIXHeader]byte
	n, err := buf.Read(data[:])
	if n != len(data) || err != nil {
		return header, buf, countRecords, io.EOF
	}
	header = v9.PacketHeader{
		Version:    binary.BigEndian.Uint16(data[:2]),
		Count:      binary.BigEndian.Uint16(data[2:4]),
		UnixSecs:   time.Unix(int64(binary.BigEndian.Uint32(data[4:8])), 0).UTC(),
		SequenceNo: binary.BigEndian.Uint32(data[8:12]),
		SourceID:   binary.BigEndian.Uint32(data[12:16]),
	}
	// In IPFIX, Count is length of packet
	if header.Count < SizeOfIPFIXHeader {
		return header, buf, countRecords, io.EOF
	}
	payloadLen := header.Count - SizeOfIPFIXHeader
	payload := buf.Next(int(payloadLen))
	if len(payload) < int(payloadLen) {
		return header, buf, countRecords, io.EOF
	}
	return header, bytes.NewBuffer(payload), math.MaxUint16, nil
}

func (d DecoderIPFIX) ReadTemplateSet(setID uint16, buf *bytes.Buffer) ([]*template.Template, error) {
	switch setID {
	case TemplateFlowSetID:
		return v9.ReadTemplateFlowSet(d, buf)
	case TemplateOptionsSetID:
		return d.ReadOptionsTemplateFlowSet(buf)
	default:
		return nil, fmt.Errorf("set id %d not supported", setID)
	}
}

func (d DecoderIPFIX) ReadFieldDefinition(buf *bytes.Buffer) (field fields.Key, length uint16, err error) {
	var row [4]byte
	if n, err := buf.Read(row[:]); err != nil || n != len(row) {
		return field, length, io.EOF
	}
	field.FieldID = binary.BigEndian.Uint16(row[:2])
	length = binary.BigEndian.Uint16(row[2:])
	if field.FieldID&EnterpriseBit != 0 {
		field.FieldID &= ^EnterpriseBit
		if n, err := buf.Read(row[:]); err != nil || n != len(row) {
			return field, length, io.EOF
		}
		field.EnterpriseID = binary.BigEndian.Uint32(row[:])
	}
	return field, length, nil
}

func (d DecoderIPFIX) ReadOptionsTemplateFlowSet(buf *bytes.Buffer) (templates []*template.Template, err error) {
	var header [6]byte
	for buf.Len() >= len(header) {
		if n, err := buf.Read(header[:]); err != nil || n < len(header) {
			if err == nil {
				err = io.EOF
			}
			return nil, err
		}
		tID := binary.BigEndian.Uint16(header[:2])
		if tID < 256 {
			return nil, errors.New("invalid template id")
		}
		totalCount := int(binary.BigEndian.Uint16(header[2:4]))
		scopeCount := int(binary.BigEndian.Uint16(header[4:]))
		if scopeCount > totalCount || scopeCount == 0 {
			return nil, fmt.Errorf("wrong counts in options template flowset: scope=%d total=%d", scopeCount, totalCount)
		}
		template, err := v9.ReadFields(d, buf, totalCount)
		if err != nil {
			return nil, err
		}
		template.ID = tID
		template.ScopeFields = scopeCount
		template.IsOptions = true
		templates = append(templates, &template)
	}
	return templates, nil
}
