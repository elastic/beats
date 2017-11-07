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

	rule.Buf = make([]byte, rule.BufLen)
	if _, err := reader.Read(rule.Buf); err != nil {
		return nil, err
	}

	return rule, nil
}
