package dhcpv4

import (
	"errors"
	"fmt"
)

// OptionGeneric is an option that only contains the option code and associated
// data. Every option that does not have a specific implementation will fall
// back to this option.
type OptionGeneric struct {
	OptionCode OptionCode
	Data       []byte
}

// ParseOptionGeneric parses a bytestream and creates a new OptionGeneric from
// it, or an error.
func ParseOptionGeneric(data []byte) (*OptionGeneric, error) {
	if len(data) == 0 {
		return nil, errors.New("invalid zero-length bytestream")
	}
	var (
		length     int
		optionData []byte
	)
	code := OptionCode(data[0])
	if code != OptionPad && code != OptionEnd {
		length = int(data[1])
		if len(data) < length+2 {
			return nil, fmt.Errorf("invalid data length: declared %v, actual %v", length, len(data))
		}
		optionData = data[2 : length+2]
	}
	return &OptionGeneric{OptionCode: code, Data: optionData}, nil
}

// Code returns the generic option code.
func (o OptionGeneric) Code() OptionCode {
	return o.OptionCode
}

// ToBytes returns a serialized generic option as a slice of bytes.
func (o OptionGeneric) ToBytes() []byte {
	ret := []byte{byte(o.OptionCode)}
	if o.OptionCode == OptionEnd || o.OptionCode == OptionPad {
		return ret
	}
	ret = append(ret, byte(o.Length()))
	ret = append(ret, o.Data...)
	return ret
}

// String returns a human-readable representation of a generic option.
func (o OptionGeneric) String() string {
	code, ok := OptionCodeToString[o.OptionCode]
	if !ok {
		code = "Unknown"
	}
	return fmt.Sprintf("%v -> %v", code, o.Data)
}

// Length returns the number of bytes comprising the data section of the option.
func (o OptionGeneric) Length() int {
	return len(o.Data)
}
