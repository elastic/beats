package dhcpv4

import (
	"bytes"
	"errors"
	"fmt"
)

// ErrShortByteStream is an error that is thrown any time a short byte stream is
// detected during option parsing.
var ErrShortByteStream = errors.New("short byte stream")

// ErrZeroLengthByteStream is an error that is thrown any time a zero-length
// byte stream is encountered.
var ErrZeroLengthByteStream = errors.New("zero-length byte stream")

// MagicCookie is the magic 4-byte value at the beginning of the list of options
// in a DHCPv4 packet.
var MagicCookie = []byte{99, 130, 83, 99}

// OptionCode is a single byte representing the code for a given Option.
type OptionCode byte

// Option is an interface that all DHCP v4 options adhere to.
type Option interface {
	Code() OptionCode
	ToBytes() []byte
	Length() int
	String() string
}

// ParseOption parses a sequence of bytes as a single DHCPv4 option, returning
// the specific option structure or error, if any.
func ParseOption(data []byte) (Option, error) {
	if len(data) == 0 {
		return nil, errors.New("invalid zero-length DHCPv4 option")
	}
	var (
		opt Option
		err error
	)
	switch OptionCode(data[0]) {
	case OptionDHCPMessageType:
		opt, err = ParseOptMessageType(data)
	case OptionParameterRequestList:
		opt, err = ParseOptParameterRequestList(data)
	case OptionRequestedIPAddress:
		opt, err = ParseOptRequestedIPAddress(data)
	case OptionServerIdentifier:
		opt, err = ParseOptServerIdentifier(data)
	case OptionBroadcastAddress:
		opt, err = ParseOptBroadcastAddress(data)
	case OptionMaximumDHCPMessageSize:
		opt, err = ParseOptMaximumDHCPMessageSize(data)
	case OptionClassIdentifier:
		opt, err = ParseOptClassIdentifier(data)
	case OptionDomainName:
		opt, err = ParseOptDomainName(data)
	case OptionDomainNameServer:
		opt, err = ParseOptDomainNameServer(data)
	case OptionVendorIdentifyingVendorClass:
		opt, err = ParseOptVIVC(data)
	default:
		opt, err = ParseOptionGeneric(data)
	}
	if err != nil {
		return nil, err
	}
	return opt, nil
}

// OptionsFromBytes parses a sequence of bytes until the end and builds a list
// of options from it. The sequence must contain the Magic Cookie. Returns an
// error if any invalid option or length is found.
func OptionsFromBytes(data []byte) ([]Option, error) {
	if len(data) < len(MagicCookie) {
		return nil, errors.New("invalid options: shorter than 4 bytes")
	}
	if !bytes.Equal(data[:len(MagicCookie)], MagicCookie) {
		return nil, fmt.Errorf("invalid magic cookie: %v", data[:len(MagicCookie)])
	}
	opts, err := OptionsFromBytesWithoutMagicCookie(data[len(MagicCookie):])
	if err != nil {
		return nil, err
	}
	return opts, nil
}

// OptionsFromBytesWithoutMagicCookie parses a sequence of bytes until the end
// and builds a list of options from it. The sequence should not contain the
// DHCP magic cookie. Returns an error if any invalid option or length is found.
func OptionsFromBytesWithoutMagicCookie(data []byte) ([]Option, error) {
	options := make([]Option, 0, 10)
	idx := 0
	for {
		if idx == len(data) {
			break
		}
		// This should never happen.
		if idx > len(data) {
			return nil, errors.New("read past the end of options")
		}
		opt, err := ParseOption(data[idx:])
		idx++
		if err != nil {
			return nil, err
		}
		options = append(options, opt)

		// Options with zero length have no length byte, so here we handle the
		// ones with nonzero length
		if opt.Length() > 0 {
			idx++
		}
		idx += opt.Length()
	}
	return options, nil
}
