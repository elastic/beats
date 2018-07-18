// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package rule

import (
	"bytes"
	"encoding/binary"
	"io"

	"github.com/pkg/errors"
)

const (
	syscallBitmaskSize = 64 // AUDIT_BITMASK_SIZE
	maxFields          = 64 // AUDIT_MAX_FIELDS
)

var endianness = binary.LittleEndian

// WireFormat is the binary representation of a rule as used to exchange rules
// (commands) with the kernel.
type WireFormat []byte

// auditRuleData supports filter rules with both integer and string
// fields.  It corresponds with AUDIT_ADD_RULE, AUDIT_DEL_RULE and
// AUDIT_LIST_RULES requests.
// https://github.com/linux-audit/audit-kernel/blob/v3.15/include/uapi/linux/audit.h#L423-L437
type auditRuleData struct {
	Flags      filter
	Action     action
	FieldCount uint32
	Mask       [syscallBitmaskSize]uint32 // Syscalls affected.
	Fields     [maxFields]field
	Values     [maxFields]uint32
	FieldFlags [maxFields]operator
	BufLen     uint32 // Total length of buffer used for string fields.
	Buf        []byte // String fields.
}

func (r auditRuleData) toWireFormat() WireFormat {
	out := new(bytes.Buffer)
	binary.Write(out, endianness, r.Flags)
	binary.Write(out, endianness, r.Action)
	binary.Write(out, endianness, r.FieldCount)
	binary.Write(out, endianness, r.Mask)
	binary.Write(out, endianness, r.Fields)
	binary.Write(out, endianness, r.Values)
	binary.Write(out, endianness, r.FieldFlags)
	binary.Write(out, endianness, r.BufLen)
	out.Write(r.Buf)

	// Adding padding.
	if out.Len()%4 > 0 {
		out.Write(make([]byte, 4-(out.Len()%4)))
	}

	return out.Bytes()
}

func fromWireFormat(data WireFormat) (*auditRuleData, error) {
	var partialRule struct {
		Flags      filter
		Action     action
		FieldCount uint32
		Mask       [syscallBitmaskSize]uint32
		Fields     [maxFields]field
		Values     [maxFields]uint32
		FieldFlags [maxFields]operator
		BufLen     uint32
	}

	reader := bytes.NewReader(data)
	if err := binary.Read(reader, endianness, &partialRule); err != nil {
		return nil, errors.Wrap(err, "deserialization of rule data failed")
	}

	rule := &auditRuleData{
		Flags:      partialRule.Flags,
		Action:     partialRule.Action,
		FieldCount: partialRule.FieldCount,
		Mask:       partialRule.Mask,
		Fields:     partialRule.Fields,
		Values:     partialRule.Values,
		FieldFlags: partialRule.FieldFlags,
		BufLen:     partialRule.BufLen,
	}

	if reader.Len() < int(rule.BufLen) {
		return nil, io.ErrUnexpectedEOF
	}

	if rule.BufLen > 0 {
		rule.Buf = make([]byte, rule.BufLen)
		if _, err := reader.Read(rule.Buf); err != nil {
			return nil, errors.Wrap(err, "deserialization of buf failed")
		}
	}

	return rule, nil
}
