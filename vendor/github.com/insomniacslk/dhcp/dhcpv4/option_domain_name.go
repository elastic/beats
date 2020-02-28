package dhcpv4

import "fmt"

// This option implements the server domani name option
// https://tools.ietf.org/html/rfc2132

// OptDomainName represents an option encapsulating the server identifier.
type OptDomainName struct {
	DomainName string
}

// ParseOptDomainName returns a new OptDomainName from a byte
// stream, or error if any.
func ParseOptDomainName(data []byte) (*OptDomainName, error) {
	if len(data) < 2 {
		return nil, ErrShortByteStream
	}
	code := OptionCode(data[0])
	if code != OptionDomainName {
		return nil, fmt.Errorf("expected code %v, got %v", OptionDomainName, code)
	}
	length := int(data[1])
	if len(data) < 2+length {
		return nil, ErrShortByteStream
	}
	return &OptDomainName{DomainName: string(data[2 : 2+length])}, nil
}

// Code returns the option code.
func (o *OptDomainName) Code() OptionCode {
	return OptionDomainName
}

// ToBytes returns a serialized stream of bytes for this option.
func (o *OptDomainName) ToBytes() []byte {
	return append([]byte{byte(o.Code()), byte(o.Length())}, []byte(o.DomainName)...)
}

// String returns a human-readable string.
func (o *OptDomainName) String() string {
	return fmt.Sprintf("Domain Name -> %v", o.DomainName)
}

// Length returns the length of the data portion (excluding option code an byte
// length).
func (o *OptDomainName) Length() int {
	return len(o.DomainName)
}
