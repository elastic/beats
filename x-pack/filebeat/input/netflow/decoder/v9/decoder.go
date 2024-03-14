// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v9

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/netflow/decoder/fields"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/netflow/decoder/record"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/netflow/decoder/template"
)

const (
	TemplateFlowSetID    = 0
	TemplateOptionsSetID = 1
)

type Decoder interface {
	ReadPacketHeader(*bytes.Buffer) (PacketHeader, *bytes.Buffer, int, error)
	ReadSetHeader(*bytes.Buffer) (SetHeader, error)
	ReadTemplateSet(setID uint16, buf *bytes.Buffer) ([]*template.Template, error)
	ReadFieldDefinition(*bytes.Buffer) (field fields.Key, length uint16, err error)
	GetLogger() *log.Logger
	GetFields() fields.FieldDict
}

type DecoderV9 struct {
	Logger *log.Logger
	Fields fields.FieldDict
}

var _ Decoder = (*DecoderV9)(nil)

func (d DecoderV9) GetLogger() *log.Logger {
	return d.Logger
}

func (DecoderV9) ReadPacketHeader(buf *bytes.Buffer) (header PacketHeader, newBuf *bytes.Buffer, numFlowSets int, err error) {
	var data [20]byte
	n, err := buf.Read(data[:])
	if n != len(data) || err != nil {
		return header, buf, numFlowSets, io.EOF
	}
	header = PacketHeader{
		Version:    binary.BigEndian.Uint16(data[:2]),
		Count:      binary.BigEndian.Uint16(data[2:4]),
		SysUptime:  binary.BigEndian.Uint32(data[4:8]),
		UnixSecs:   time.Unix(int64(binary.BigEndian.Uint32(data[8:12])), 0).UTC(),
		SequenceNo: binary.BigEndian.Uint32(data[12:16]),
		SourceID:   binary.BigEndian.Uint32(data[16:20]),
	}
	return header, buf, int(header.Count), nil
}

func (DecoderV9) ReadSetHeader(buf *bytes.Buffer) (SetHeader, error) {
	var data [4]byte
	n, err := buf.Read(data[:])
	if n != len(data) || err != nil {
		return SetHeader{}, io.EOF
	}
	return SetHeader{
		SetID:  binary.BigEndian.Uint16(data[:2]),
		Length: binary.BigEndian.Uint16(data[2:4]),
	}, nil
}

func (d DecoderV9) ReadTemplateSet(setID uint16, buf *bytes.Buffer) ([]*template.Template, error) {
	switch setID {
	case TemplateFlowSetID:
		return ReadTemplateFlowSet(d, buf)
	case TemplateOptionsSetID:
		return d.ReadOptionsTemplateFlowSet(buf)
	default:
		return nil, fmt.Errorf("set id %d not supported", setID)
	}
}

func (d DecoderV9) ReadFieldDefinition(buf *bytes.Buffer) (field fields.Key, length uint16, err error) {
	var row [4]byte
	if n, err := buf.Read(row[:]); err != nil || n != len(row) {
		return field, length, io.EOF
	}
	field.FieldID = binary.BigEndian.Uint16(row[:2])
	length = binary.BigEndian.Uint16(row[2:])
	return field, length, nil
}

func (d DecoderV9) GetFields() fields.FieldDict {
	if f := d.Fields; f != nil {
		return f
	}
	return fields.GlobalFields
}

func ReadFields(d Decoder, buf *bytes.Buffer, count int) (record template.Template, err error) {
	knownFields := d.GetFields()
	logger := d.GetLogger()
	record.Fields = make([]template.FieldTemplate, count)
	for i := 0; i < count; i++ {
		key, length, err := d.ReadFieldDefinition(buf)
		if err != nil {
			return template.Template{}, io.EOF
		}
		field := template.FieldTemplate{
			Length: length,
		}
		if length == template.VariableLength {
			record.VariableLength = true
			record.Length += 1
		} else {
			record.Length += int(field.Length)
		}
		if fieldInfo, found := knownFields[key]; found {
			min, max := fieldInfo.Decoder.MinLength(), fieldInfo.Decoder.MaxLength()
			if length == template.VariableLength || min <= field.Length && field.Length <= max {
				field.Info = fieldInfo
			} else if logger != nil {
				logger.Printf("Size of field %s in template is out of bounds (size=%d, min=%d, max=%d)", fieldInfo.Name, field.Length, min, max)
			}
		} else if logger != nil {
			logger.Printf("Field %v in template not found", key)
		}
		record.Fields[i] = field
	}
	return record, nil
}

func ReadTemplateFlowSet(d Decoder, buf *bytes.Buffer) (templates []*template.Template, err error) {
	var row [4]byte
	for {
		if buf.Len() < 8 {
			return templates, nil
		}
		if n, err := buf.Read(row[:]); err != nil || n != len(row) {
			return nil, io.EOF
		}
		tID := binary.BigEndian.Uint16(row[:2])
		if tID < 256 {
			return nil, errors.New("invalid template id")
		}
		count := int(binary.BigEndian.Uint16(row[2:]))
		// Ignore empty template
		if count == 0 {
			continue
		}
		if buf.Len() < 2*count {
			return nil, io.EOF
		}
		recordTemplate, err := ReadFields(d, buf, count)
		if err != nil {
			break
		}
		recordTemplate.ID = tID
		templates = append(templates, &recordTemplate)
	}
	return templates, nil
}

func (d DecoderV9) ReadOptionsTemplateFlowSet(buf *bytes.Buffer) (templates []*template.Template, err error) {
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
		scopeLen := int(binary.BigEndian.Uint16(header[2:4]))
		optsLen := int(binary.BigEndian.Uint16(header[4:]))
		length := optsLen + scopeLen
		if buf.Len() < length {
			return nil, io.EOF
		}
		if (scopeLen+optsLen) == 0 || scopeLen&3 != 0 || optsLen&3 != 0 {
			return nil, fmt.Errorf("bad length for options template. scope=%d options=%d", scopeLen, optsLen)
		}
		template, err := ReadFields(d, buf, (scopeLen+optsLen)/4)
		if err != nil {
			return nil, err
		}
		template.ID = tID
		template.ScopeFields = scopeLen / 4
		template.IsOptions = true
		templates = append(templates, &template)
	}
	return templates, nil
}

type PacketHeader struct {
	Version, Count       uint16
	SysUptime            uint32
	UnixSecs             time.Time
	SequenceNo, SourceID uint32
}

type SetHeader struct {
	SetID, Length uint16
}

func (h SetHeader) BodyLength() int {
	if h.Length < 4 {
		return 0
	}
	return int(h.Length) - 4
}

func (h SetHeader) IsPadding() bool {
	return h.SetID == 0 && h.Length == 0
}

func (h PacketHeader) ExporterMetadata(source net.Addr) record.Map {
	return record.Map{
		"version":      uint64(h.Version),
		"timestamp":    h.UnixSecs,
		"uptimeMillis": uint64(h.SysUptime),
		"address":      source.String(),
		"sourceId":     uint64(h.SourceID),
	}
}
