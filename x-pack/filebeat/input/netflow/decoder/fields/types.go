// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fields

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"strings"
	"time"
)

var (
	NtpEpoch = time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)

	ErrOutOfBounds = errors.New("excess bytes for decoding")
	ErrUnsupported = errors.New("unsupported data type")
)

type Decoder interface {
	Decode([]byte) (interface{}, error)
	MinLength() uint16
	MaxLength() uint16
}

type UnsignedDecoder uint8

func (u UnsignedDecoder) MinLength() uint16 {
	return 1
}

func (u UnsignedDecoder) MaxLength() uint16 {
	return uint16(u)
}

func (u UnsignedDecoder) Decode(data []byte) (interface{}, error) {
	n := len(data)
	if n > int(u) {
		return uint64(0), ErrOutOfBounds
	}
	switch n {
	case 0:
		return uint64(0), io.EOF
	case 1:
		return uint64(data[0]), nil
	case 2:
		return uint64(binary.BigEndian.Uint16(data)), nil
	case 4:
		return uint64(binary.BigEndian.Uint32(data)), nil
	case 8:
		return binary.BigEndian.Uint64(data), nil
	default:
		var value uint64
		for i := 0; i < n; i++ {
			value = (value << 8) | uint64(data[i])
		}
		return value, nil
	}
}

var _ Decoder = (*UnsignedDecoder)(nil)

type SignedDecoder uint8

func (u SignedDecoder) MinLength() uint16 {
	return 1
}

func (u SignedDecoder) MaxLength() uint16 {
	return uint16(u)
}

func (u SignedDecoder) Decode(data []byte) (interface{}, error) {
	n := len(data)
	if n > int(u) {
		return int64(0), ErrOutOfBounds
	}
	switch n {
	case 0:
		return int64(0), io.EOF
	case 1:
		return int64(int8(data[0])), nil
	case 2:
		return int64(int16(binary.BigEndian.Uint16(data))), nil
	case 4:
		return int64(int32(binary.BigEndian.Uint32(data))), nil
	case 8:
		return int64(binary.BigEndian.Uint64(data)), nil
	default:
		value := uint64(data[0])
		if value&0x80 != 0 {
			value |= ^uint64(0xFF)
		}
		for i := 1; i < n; i++ {
			value = (value << 8) | uint64(data[i])
		}
		return int64(value), nil
	}
}

var _ Decoder = (*SignedDecoder)(nil)

type FloatDecoder uint8

func (u FloatDecoder) MinLength() uint16 {
	return 4
}

func (u FloatDecoder) MaxLength() uint16 {
	return uint16(u)
}

func (u FloatDecoder) Decode(data []byte) (interface{}, error) {
	n := len(data)
	if n > int(u) {
		return float64(0), ErrOutOfBounds
	}
	switch n {
	case 0:
		return float64(0), io.EOF
	case 4:
		return float64(math.Float32frombits(binary.BigEndian.Uint32(data))), nil
	case 8:
		return float64(math.Float64frombits(binary.BigEndian.Uint64(data))), nil
	default:
		return float64(0), fmt.Errorf("wrong number of bytes in floating point decoding. have=%d want={4,8}", n)
	}
}

var _ Decoder = (*FloatDecoder)(nil)

type BooleanDecoder struct{}

func (u BooleanDecoder) MinLength() uint16 {
	return 1
}

func (u BooleanDecoder) MaxLength() uint16 {
	return 1
}

func (u BooleanDecoder) Decode(data []byte) (interface{}, error) {
	n := len(data)
	switch n {
	case 0:
		return false, io.EOF
	case 1:
		/* The boolean data type is specified according to the TruthValue in
		   [RFC2579].  It is encoded as a single-octet integer per
		   Section 6.1.1, with the value 1 for true and value 2 for false.
		   Every other value is undefined.
		*/
		switch data[0] {
		case 1:
			return true, nil
		case 2:
			return false, nil
		default:
			return false, fmt.Errorf("invalid value for boolean decoding. have=%d want={1,2}", data[0])
		}
	default:
		return false, ErrOutOfBounds
	}
}

var _ Decoder = (*BooleanDecoder)(nil)

type OctetArrayDecoder struct{}

func (u OctetArrayDecoder) MinLength() uint16 {
	return 0
}

func (u OctetArrayDecoder) MaxLength() uint16 {
	return 0xffff
}

func (u OctetArrayDecoder) Decode(data []byte) (interface{}, error) {
	return data, nil
}

var _ Decoder = (*OctetArrayDecoder)(nil)

type MacAddressDecoder struct{}

func (u MacAddressDecoder) MinLength() uint16 {
	return 6
}

func (u MacAddressDecoder) MaxLength() uint16 {
	return 6
}

func (u MacAddressDecoder) Decode(data []byte) (interface{}, error) {
	if len(data) != 6 {
		return net.HardwareAddr{}, ErrOutOfBounds
	}
	return net.HardwareAddr(data), nil
}

var _ Decoder = (*MacAddressDecoder)(nil)

type StringDecoder struct{}

func (u StringDecoder) MinLength() uint16 {
	return 0
}

func (u StringDecoder) MaxLength() uint16 {
	return 0xffff
}

func (u StringDecoder) Decode(data []byte) (interface{}, error) {
	return strings.TrimRightFunc(string(data), func(r rune) bool {
		return r == 0
	}), nil
}

var _ Decoder = (*StringDecoder)(nil)

type DateTimeSecondsDecoder struct{}

func (u DateTimeSecondsDecoder) MinLength() uint16 {
	return 4
}

func (u DateTimeSecondsDecoder) MaxLength() uint16 {
	return 4
}

func (u DateTimeSecondsDecoder) Decode(data []byte) (interface{}, error) {
	if len(data) != 4 {
		return time.Time{}, ErrOutOfBounds
	}
	return time.Unix(int64(binary.BigEndian.Uint32(data)), 0).UTC(), nil
}

var _ Decoder = (*DateTimeSecondsDecoder)(nil)

type DateTimeMillisecondsDecoder struct{}

func (u DateTimeMillisecondsDecoder) MinLength() uint16 {
	return 8
}

func (u DateTimeMillisecondsDecoder) MaxLength() uint16 {
	return 8
}

func (u DateTimeMillisecondsDecoder) Decode(data []byte) (interface{}, error) {
	if len(data) != 8 {
		return time.Time{}, ErrOutOfBounds
	}
	millis := binary.BigEndian.Uint64(data)
	return time.Unix(int64(millis/1000), int64(millis%1000)*1000000).UTC(), nil
}

var _ Decoder = (*DateTimeMillisecondsDecoder)(nil)

type NTPTimestampDecoder struct{}

func (u NTPTimestampDecoder) MinLength() uint16 {
	return 8
}

func (u NTPTimestampDecoder) MaxLength() uint16 {
	return 8
}

func (u NTPTimestampDecoder) Decode(data []byte) (interface{}, error) {
	if len(data) != 8 {
		return time.Time{}, ErrOutOfBounds
	}
	secs := binary.BigEndian.Uint32(data[:4])
	frac := binary.BigEndian.Uint32(data[4:])
	return NtpEpoch.Add(time.Duration(secs) * time.Second).Add(time.Duration(int64(frac)*int64(time.Second)/int64(0x100000000)) * time.Nanosecond), nil
}

var _ Decoder = (*NTPTimestampDecoder)(nil)

type IPAddressDecoder uint8

func (u IPAddressDecoder) MinLength() uint16 {
	return uint16(u)
}

func (u IPAddressDecoder) MaxLength() uint16 {
	return uint16(u)
}

func (u IPAddressDecoder) Decode(data []byte) (interface{}, error) {
	n := len(data)
	if n != int(u) {
		return net.IP{}, ErrOutOfBounds
	}
	if n == 4 {
		return net.IPv4(data[0], data[1], data[2], data[3]).To4(), nil
	}
	return net.IP(data), nil
}

var _ Decoder = (*IPAddressDecoder)(nil)

type UnsupportedDecoder struct{}

func (u UnsupportedDecoder) MinLength() uint16 {
	return 0
}

func (u UnsupportedDecoder) MaxLength() uint16 {
	return math.MaxUint16
}

func (u UnsupportedDecoder) Decode(data []byte) (interface{}, error) {
	return nil, ErrUnsupported
}

var _ Decoder = (*UnsupportedDecoder)(nil)

type ACLIDDecoder struct{}

const aclIDLength = 12

func (u ACLIDDecoder) MinLength() uint16 {
	return aclIDLength
}

func (u ACLIDDecoder) MaxLength() uint16 {
	return aclIDLength
}

func (u ACLIDDecoder) Decode(data []byte) (interface{}, error) {
	if len(data) != aclIDLength {
		return nil, ErrOutOfBounds
	}
	// Encode a [12]byte to a hex string in the form:
	// "11223344-55667788-99aabbcc"
	var result [aclIDLength*2 + 2]byte
	hex.Encode(result[:8], data[:4])
	hex.Encode(result[9:17], data[4:8])
	hex.Encode(result[18:], data[8:])
	result[8], result[17] = '-', '-'
	return string(result[:]), nil
}

var _ Decoder = (*OctetArrayDecoder)(nil)

// RFC5610 fields
var (
	OctetArray           = OctetArrayDecoder{}
	Unsigned8            = UnsignedDecoder(1)
	Unsigned16           = UnsignedDecoder(2)
	Unsigned32           = UnsignedDecoder(4)
	Unsigned64           = UnsignedDecoder(8)
	Signed8              = SignedDecoder(1)
	Signed16             = SignedDecoder(2)
	Signed32             = SignedDecoder(4)
	Signed64             = SignedDecoder(8)
	Float32              = FloatDecoder(4)
	Float64              = FloatDecoder(8)
	Boolean              = BooleanDecoder{}
	MacAddress           = MacAddressDecoder{}
	String               = StringDecoder{}
	DateTimeSeconds      = DateTimeSecondsDecoder{}
	DateTimeMilliseconds = DateTimeMillisecondsDecoder{}
	DateTimeMicroseconds = NTPTimestampDecoder{}
	DateTimeNanoseconds  = NTPTimestampDecoder{}
	Ipv4Address          = IPAddressDecoder(4)
	Ipv6Address          = IPAddressDecoder(16)
	BasicList            = UnsupportedDecoder{}
	SubTemplateList      = UnsupportedDecoder{}
	SubTemplateMultiList = UnsupportedDecoder{}
)

// ACLID field added for Cisco ASA devices
var ACLID = ACLIDDecoder{}
