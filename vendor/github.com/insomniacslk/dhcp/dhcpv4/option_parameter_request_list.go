package dhcpv4

import (
	"fmt"
	"strings"
)

// This option implements the parameter request list option
// https://tools.ietf.org/html/rfc2132

// OptParameterRequestList represents the parameter request list option.
type OptParameterRequestList struct {
	RequestedOpts []OptionCode
}

// ParseOptParameterRequestList returns a new OptParameterRequestList from a
// byte stream, or error if any.
func ParseOptParameterRequestList(data []byte) (*OptParameterRequestList, error) {
	// Should at least have code + length byte.
	if len(data) < 2 {
		return nil, ErrShortByteStream
	}
	code := OptionCode(data[0])
	if code != OptionParameterRequestList {
		return nil, fmt.Errorf("expected code %v, got %v", OptionParameterRequestList, code)
	}
	length := int(data[1])
	if len(data) < length+2 {
		return nil, ErrShortByteStream
	}
	var requestedOpts []OptionCode
	for _, opt := range data[2 : length+2] {
		requestedOpts = append(requestedOpts, OptionCode(opt))
	}
	return &OptParameterRequestList{RequestedOpts: requestedOpts}, nil
}

// Code returns the option code.
func (o *OptParameterRequestList) Code() OptionCode {
	return OptionParameterRequestList
}

// ToBytes returns a serialized stream of bytes for this option.
func (o *OptParameterRequestList) ToBytes() []byte {
	ret := []byte{byte(o.Code()), byte(o.Length())}
	for _, req := range o.RequestedOpts {
		ret = append(ret, byte(req))
	}
	return ret
}

// String returns a human-readable string for this option.
func (o *OptParameterRequestList) String() string {
	var optNames []string
	for _, ro := range o.RequestedOpts {
		if name, ok := OptionCodeToString[ro]; ok {
			optNames = append(optNames, name)
		} else {
			optNames = append(optNames, fmt.Sprintf("Unknown (%v)", ro))
		}
	}
	return fmt.Sprintf("Parameter Request List -> [%v]", strings.Join(optNames, ", "))
}

// Length returns the length of the data portion (excluding option code and byte
// for length, if any).
func (o *OptParameterRequestList) Length() int {
	return len(o.RequestedOpts)
}
