package dhcpv4

import (
	"fmt"
	"net"
)

// This option implements the requested IP address option
// https://tools.ietf.org/html/rfc2132

// OptRequestedIPAddress represents an option encapsulating the server
// identifier.
type OptRequestedIPAddress struct {
	RequestedAddr net.IP
}

// ParseOptRequestedIPAddress returns a new OptServerIdentifier from a byte
// stream, or error if any.
func ParseOptRequestedIPAddress(data []byte) (*OptRequestedIPAddress, error) {
	if len(data) < 2 {
		return nil, ErrShortByteStream
	}
	code := OptionCode(data[0])
	if code != OptionRequestedIPAddress {
		return nil, fmt.Errorf("expected code %v, got %v", OptionRequestedIPAddress, code)
	}
	length := int(data[1])
	if length != 4 {
		return nil, fmt.Errorf("unexepcted length: expected 4, got %v", length)
	}
	if len(data) < 6 {
		return nil, ErrShortByteStream
	}
	return &OptRequestedIPAddress{RequestedAddr: net.IP(data[2 : 2+length])}, nil
}

// Code returns the option code.
func (o *OptRequestedIPAddress) Code() OptionCode {
	return OptionRequestedIPAddress
}

// ToBytes returns a serialized stream of bytes for this option.
func (o *OptRequestedIPAddress) ToBytes() []byte {
	ret := []byte{byte(o.Code()), byte(o.Length())}
	return append(ret, o.RequestedAddr.To4()...)
}

// String returns a human-readable string.
func (o *OptRequestedIPAddress) String() string {
	return fmt.Sprintf("Requested IP Address -> %v", o.RequestedAddr.String())
}

// Length returns the length of the data portion (excluding option code an byte
// length).
func (o *OptRequestedIPAddress) Length() int {
	return len(o.RequestedAddr.To4())
}
