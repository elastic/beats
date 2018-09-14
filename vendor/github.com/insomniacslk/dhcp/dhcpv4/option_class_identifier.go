package dhcpv4

import (
	"fmt"
)

// This option implements the Class Identifier option
// https://tools.ietf.org/html/rfc2132

// OptClassIdentifier represents the DHCP message type option.
type OptClassIdentifier struct {
	Identifier string
}

// ParseOptClassIdentifier constructs an OptClassIdentifier struct from a sequence of
// bytes and returns it, or an error.
func ParseOptClassIdentifier(data []byte) (*OptClassIdentifier, error) {
	// Should at least have code and length
	if len(data) < 2 {
		return nil, ErrShortByteStream
	}
	code := OptionCode(data[0])
	if code != OptionClassIdentifier {
		return nil, fmt.Errorf("expected option %v, got %v instead", OptionClassIdentifier, code)
	}
	length := int(data[1])
	if len(data) < 2+length {
		return nil, ErrShortByteStream
	}
	return &OptClassIdentifier{Identifier: string(data[2 : 2+length])}, nil
}

// Code returns the option code.
func (o *OptClassIdentifier) Code() OptionCode {
	return OptionClassIdentifier
}

// ToBytes returns a serialized stream of bytes for this option.
func (o *OptClassIdentifier) ToBytes() []byte {
	return append([]byte{byte(o.Code()), byte(o.Length())}, []byte(o.Identifier)...)
}

// String returns a human-readable string for this option.
func (o *OptClassIdentifier) String() string {
	return fmt.Sprintf("Class Identifier -> %v", o.Identifier)
}

// Length returns the length of the data portion (excluding option code and byte
// for length, if any).
func (o *OptClassIdentifier) Length() int {
	return len(o.Identifier)
}
