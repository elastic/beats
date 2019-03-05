package dhcpv4

import (
	"fmt"
	"net"
)

// This option implements the server identifier option
// https://tools.ietf.org/html/rfc2132

// OptBroadcastAddress represents an option encapsulating the server identifier.
type OptBroadcastAddress struct {
	BroadcastAddress net.IP
}

// ParseOptBroadcastAddress returns a new OptBroadcastAddress from a byte
// stream, or error if any.
func ParseOptBroadcastAddress(data []byte) (*OptBroadcastAddress, error) {
	if len(data) < 2 {
		return nil, ErrShortByteStream
	}
	code := OptionCode(data[0])
	if code != OptionBroadcastAddress {
		return nil, fmt.Errorf("expected code %v, got %v", OptionBroadcastAddress, code)
	}
	length := int(data[1])
	if length != 4 {
		return nil, fmt.Errorf("unexepcted length: expected 4, got %v", length)
	}
	if len(data) < 6 {
		return nil, ErrShortByteStream
	}
	return &OptBroadcastAddress{BroadcastAddress: net.IP(data[2 : 2+length])}, nil
}

// Code returns the option code.
func (o *OptBroadcastAddress) Code() OptionCode {
	return OptionBroadcastAddress
}

// ToBytes returns a serialized stream of bytes for this option.
func (o *OptBroadcastAddress) ToBytes() []byte {
	ret := []byte{byte(o.Code()), byte(o.Length())}
	return append(ret, o.BroadcastAddress.To4()...)
}

// String returns a human-readable string.
func (o *OptBroadcastAddress) String() string {
	return fmt.Sprintf("Broadcast Address -> %v", o.BroadcastAddress.String())
}

// Length returns the length of the data portion (excluding option code an byte
// length).
func (o *OptBroadcastAddress) Length() int {
	return len(o.BroadcastAddress.To4())
}
