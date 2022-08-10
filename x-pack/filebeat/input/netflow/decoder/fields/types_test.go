// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fields

import (
	"encoding/binary"
	"fmt"
	"math"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	// UnixEpochInNTP represents the number of seconds between 1-Jan-1900
	// and 1-Jan-1970, that is, the UNIX Epoch as an NTP timestamp seconds.
	UnixEpochInNTP = uint32(2208988800)
)

func TestOctetArray(t *testing.T) {
	assert.Equal(t, uint16(0), OctetArray.MinLength())
	assert.Equal(t, ^uint16(0), OctetArray.MaxLength())
	for _, testCase := range [][]byte{
		{},
		{1},
		{1, 2, 3},
		make([]byte, 65535),
	} {
		t.Run(fmt.Sprintf("array of length %d", len(testCase)), func(t *testing.T) {
			value, err := OctetArray.Decode(testCase)
			assert.NoError(t, err)
			assert.Equal(t, testCase, value)
		})
	}
}

type testCase struct {
	title    string
	bytes    []byte
	value    interface{}
	err      bool
	strValue string
}

func (testCase testCase) Run(t *testing.T, decoder Decoder) {
	t.Run(testCase.title, func(t *testing.T) {
		value, err := decoder.Decode(testCase.bytes)
		assert.Equal(t, testCase.value, value)
		if testCase.err {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
		if len(testCase.strValue) > 0 {
			stringer, isStringer := value.(fmt.Stringer)
			assert.True(t, isStringer)
			assert.Equal(t, testCase.strValue, stringer.String())
		}
	})
}

func doTest(t *testing.T, decoder Decoder, min uint16, max uint16, testCases []testCase) {
	assert.Equal(t, min, decoder.MinLength(), "min length out of bounds")
	assert.Equal(t, max, decoder.MaxLength(), "max length out of bounds")
	for _, testCase := range testCases {
		testCase.Run(t, decoder)
	}
}

func TestUnsigned8(t *testing.T) {
	doTest(t, Unsigned8, 1, 1, []testCase{
		{
			title: "No data",
			value: uint64(0),
			err:   true,
		},
		{
			title: "Single byte 0",
			bytes: []byte{0},
			value: uint64(0),
		},
		{
			title: "Single byte 1",
			bytes: []byte{1},
			value: uint64(1),
		},
		{
			title: "Single byte 255",
			bytes: []byte{255},
			value: uint64(255),
		},
		{
			title: "Excess data",
			bytes: []byte{128, 129},
			value: uint64(0),
			err:   true,
		},
	})
}

func TestUnsigned16(t *testing.T) {
	doTest(t, Unsigned16, 1, 2, []testCase{
		{
			title: "No data",
			value: uint64(0),
			err:   true,
		},
		{
			title: "Single byte 0",
			bytes: []byte{0},
			value: uint64(0),
		},
		{
			title: "Single byte 1",
			bytes: []byte{1},
			value: uint64(1),
		},
		{
			title: "Single byte 255",
			bytes: []byte{255},
			value: uint64(255),
		},
		{
			title: "Two bytes",
			bytes: []byte{128, 129},
			value: uint64(128<<8 | 129),
		},
		{
			title: "Two bytes zero",
			bytes: []byte{0, 0},
			value: uint64(0),
		},
		{
			title: "Two bytes max",
			bytes: []byte{255, 255},
			value: uint64(0xFFFF),
		},
		{
			title: "Excess data",
			bytes: []byte{1, 255, 255},
			value: uint64(0),
			err:   true,
		},
	})
}

func TestUnsigned32(t *testing.T) {
	doTest(t, Unsigned32, 1, 4, []testCase{
		{
			title: "No data",
			value: uint64(0),
			err:   true,
		},
		{
			title: "Single byte 0",
			bytes: []byte{0},
			value: uint64(0),
		},
		{
			title: "Single byte 1",
			bytes: []byte{1},
			value: uint64(1),
		},
		{
			title: "Single byte 255",
			bytes: []byte{255},
			value: uint64(255),
		},
		{
			title: "Two bytes",
			bytes: []byte{128, 129},
			value: uint64(0x8081),
		},
		{
			title: "Two bytes zero",
			bytes: []byte{0, 0},
			value: uint64(0),
		},
		{
			title: "3 bytes",
			bytes: []byte{128, 129, 255},
			value: uint64(0x8081ff),
		},
		{
			title: "4 bytes",
			bytes: []byte{255, 1, 2, 3},
			value: uint64(0xff010203),
		},
		{
			title: "excess",
			bytes: []byte{10, 255, 1, 2, 3},
			value: uint64(0),
			err:   true,
		},
	})
}

func TestUnsigned64(t *testing.T) {
	doTest(t, Unsigned64, 1, 8, []testCase{
		{
			title: "No data",
			value: uint64(0),
			err:   true,
		},
		{
			title: "Single byte 0",
			bytes: []byte{0},
			value: uint64(0),
		},
		{
			title: "Single byte 1",
			bytes: []byte{1},
			value: uint64(1),
		},
		{
			title: "Single byte 255",
			bytes: []byte{255},
			value: uint64(255),
		},
		{
			title: "Two bytes",
			bytes: []byte{128, 129},
			value: uint64(0x8081),
		},
		{
			title: "Two bytes zero",
			bytes: []byte{0, 0},
			value: uint64(0),
		},
		{
			title: "3 bytes",
			bytes: []byte{128, 129, 255},
			value: uint64(0x8081ff),
		},
		{
			title: "4 bytes",
			bytes: []byte{255, 1, 2, 3},
			value: uint64(0xff010203),
		},
		{
			title: "5 bytes",
			bytes: []byte{10, 255, 1, 2, 3},
			value: uint64(0x0aff010203),
		},
		{
			title: "6 bytes",
			bytes: []byte{254, 10, 255, 1, 2, 3},
			value: uint64(0xfe0aff010203),
		},
		{
			title: "7 bytes",
			bytes: []byte{12, 254, 10, 255, 1, 2, 3},
			value: uint64(0x0cfe0aff010203),
		},
		{
			title: "8 bytes",
			bytes: []byte{240, 12, 254, 10, 255, 1, 2, 3},
			value: uint64(0xf00cfe0aff010203),
		},
		{
			title: "excess",
			bytes: []byte{1, 240, 12, 254, 10, 255, 1, 2, 3},
			value: uint64(0),
			err:   true,
		},
	})
}

func TestSigned8(t *testing.T) {
	doTest(t, Signed8, 1, 1, []testCase{
		{
			title: "No data",
			value: int64(0),
			err:   true,
		},
		{
			title: "Single byte 0",
			bytes: []byte{0},
			value: int64(0),
		},
		{
			title: "Single byte 1",
			bytes: []byte{1},
			value: int64(1),
		},
		{
			title: "Negative",
			bytes: []byte{255},
			value: int64(-1),
		},
		{
			title: "Negative 2",
			bytes: []byte{128},
			value: int64(-128),
		},
		{
			title: "Negative 3",
			bytes: []byte{240},
			value: int64(-16),
		},
		{
			title: "Excess data",
			bytes: []byte{128, 129},
			value: int64(0),
			err:   true,
		},
	})
}

func TestSigned16(t *testing.T) {
	doTest(t, Signed16, 1, 2, []testCase{
		{
			title: "No data",
			value: int64(0),
			err:   true,
		},
		{
			title: "Single byte 0",
			bytes: []byte{0},
			value: int64(0),
		},
		{
			title: "Single byte 1",
			bytes: []byte{1},
			value: int64(1),
		},
		{
			title: "Negative",
			bytes: []byte{255},
			value: int64(-1),
		},
		{
			title: "Negative 2",
			bytes: []byte{128},
			value: int64(-128),
		},
		{
			title: "Negative 3",
			bytes: []byte{240},
			value: int64(-16),
		},
		{
			title: "Two bytes positive",
			bytes: []byte{127, 129},
			value: int64(0x7f81),
		},
		{
			title: "Two bytes negative",
			bytes: []byte{128, 129},
			value: int64(-0x7f7f),
		},
		{
			title: "Minus one",
			bytes: []byte{0xff, 0xff},
			value: int64(-1),
		},
		{
			title: "excess",
			bytes: []byte{0x80, 0, 0},
			value: int64(0),
			err:   true,
		},
	})
}

func TestSigned32(t *testing.T) {
	doTest(t, Signed32, 1, 4, []testCase{
		{
			title: "No data",
			value: int64(0),
			err:   true,
		},
		{
			title: "Single byte 0",
			bytes: []byte{0},
			value: int64(0),
		},
		{
			title: "Single byte 1",
			bytes: []byte{1},
			value: int64(1),
		},
		{
			title: "Negative",
			bytes: []byte{255},
			value: int64(-1),
		},
		{
			title: "Negative 2",
			bytes: []byte{128},
			value: int64(-128),
		},
		{
			title: "Negative 3",
			bytes: []byte{240},
			value: int64(-16),
		},
		{
			title: "Two bytes positive",
			bytes: []byte{127, 129},
			value: int64(0x7f81),
		},
		{
			title: "Two bytes negative",
			bytes: []byte{128, 129},
			value: int64(-0x7f7f),
		},
		{
			title: "Minus one",
			bytes: []byte{0xff, 0xff},
			value: int64(-1),
		},
		{
			title: "3 bytes positive",
			bytes: []byte{127, 129, 255},
			value: int64(0x7f81ff),
		},
		{
			title: "3 bytes negative",
			bytes: []byte{128, 129, 255},
			value: int64(-0x7f7e01),
		},
		{
			title: "3 bytes Minus one",
			bytes: []byte{0xff, 0xff, 0xff},
			value: int64(-1),
		},
		{
			title: "4 bytes",
			bytes: []byte{0xff, 0xff, 0xff, 0xff},
			value: int64(-1),
		},
		{
			title: "4 bytes max positive",
			bytes: []byte{0x7f, 0xff, 0xff, 0xff},
			value: int64(1<<31 - 1),
		},
		{
			title: "4 bytes max negative",
			bytes: []byte{0x80, 0, 0, 0},
			value: int64(-(1 << 31)),
		},
		{
			title: "excess",
			bytes: []byte{0x80, 0, 0, 0, 0},
			value: int64(0),
			err:   true,
		},
	})
}

func TestSigned64(t *testing.T) {
	doTest(t, Signed64, 1, 8, []testCase{
		{
			title: "No data",
			value: int64(0),
			err:   true,
		},
		{
			title: "Single byte 0",
			bytes: []byte{0},
			value: int64(0),
		},
		{
			title: "Single byte 1",
			bytes: []byte{1},
			value: int64(1),
		},
		{
			title: "Negative",
			bytes: []byte{255},
			value: int64(-1),
		},
		{
			title: "Negative 2",
			bytes: []byte{128},
			value: int64(-128),
		},
		{
			title: "Negative 3",
			bytes: []byte{240},
			value: int64(-16),
		},
		{
			title: "Two bytes positive",
			bytes: []byte{127, 129},
			value: int64(0x7f81),
		},
		{
			title: "Two bytes negative",
			bytes: []byte{128, 129},
			value: int64(-0x7f7f),
		},
		{
			title: "Minus one",
			bytes: []byte{0xff, 0xff},
			value: int64(-1),
		},
		{
			title: "3 bytes positive",
			bytes: []byte{127, 129, 255},
			value: int64(0x7f81ff),
		},
		{
			title: "3 bytes negative",
			bytes: []byte{128, 129, 255},
			value: int64(-0x7f7e01),
		},
		{
			title: "3 bytes Minus one",
			bytes: []byte{0xff, 0xff, 0xff},
			value: int64(-1),
		},
		{
			title: "4 bytes",
			bytes: []byte{0xff, 0xff, 0xff, 0xff},
			value: int64(-1),
		},
		{
			title: "4 bytes max positive",
			bytes: []byte{0x7f, 0xff, 0xff, 0xff},
			value: int64(1<<31 - 1),
		},
		{
			title: "4 bytes max negative",
			bytes: []byte{0x80, 0, 0, 0},
			value: int64(-(1 << 31)),
		},
		{
			title: "5 bytes max positive",
			bytes: []byte{0x7f, 0xff, 0xff, 0xff, 0xff},
			value: int64(1<<39 - 1),
		},
		{
			title: "5 bytes max negative",
			bytes: []byte{0x80, 0, 0, 0, 0},
			value: int64(-(1 << 39)),
		},
		{
			title: "6 bytes max positive",
			bytes: []byte{0x7f, 0xff, 0xff, 0xff, 0xff, 0xff},
			value: int64(1<<47 - 1),
		},
		{
			title: "6 bytes max negative",
			bytes: []byte{0x80, 0, 0, 0, 0, 0},
			value: int64(-(1 << 47)),
		},
		{
			title: "7 bytes max positive",
			bytes: []byte{0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
			value: int64(1<<55 - 1),
		},
		{
			title: "7 bytes max negative",
			bytes: []byte{0x80, 0, 0, 0, 0, 0, 0},
			value: int64(-(1 << 55)),
		},
		{
			title: "8 bytes max positive",
			bytes: []byte{0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
			value: int64(1<<63 - 1),
		},
		{
			title: "8 bytes max negative",
			bytes: []byte{0x80, 0, 0, 0, 0, 0, 0, 0},
			value: int64(-(1 << 63)),
		},
		{
			title: "excess",
			bytes: []byte{0x80, 0, 0, 0, 0, 0, 0, 0, 1},
			value: int64(0),
			err:   true,
		},
	})
}

func makeFloat32(value float32) testCase {
	var bytes [4]byte
	binary.BigEndian.PutUint32(bytes[:], math.Float32bits(value))
	return testCase{
		title: fmt.Sprintf("expected float32 %v", value),
		value: float64(value),
		bytes: bytes[:],
	}
}

func makeFloat64(value float64) testCase {
	var bytes [8]byte
	binary.BigEndian.PutUint64(bytes[:], math.Float64bits(value))
	return testCase{
		title: fmt.Sprintf("expected float64 %v", value),
		value: float64(value),
		bytes: bytes[:],
	}
}

func TestFloat32(t *testing.T) {
	doTest(t, Float32, 4, 4, []testCase{
		{
			title: "No data",
			value: float64(0),
			err:   true,
		},
		{
			title: "No data 3",
			bytes: []byte{1, 2, 3},
			value: float64(0),
			err:   true,
		},
		{
			title: "No extra precision",
			bytes: []byte{1, 2, 3, 4, 5, 6, 7, 8},
			value: float64(0),
			err:   true,
		},
		makeFloat32(0.0),
		makeFloat32(-1.0),
		makeFloat32(1.0),
		makeFloat32(1.0 / 256.0),
		makeFloat32(-123.25),
		makeFloat32(math.Pi),
		makeFloat32(math.MaxFloat32),
	})
}

func TestFloat64(t *testing.T) {
	doTest(t, Float64, 4, 8, []testCase{
		{
			title: "No data",
			value: float64(0),
			err:   true,
		},
		{
			title: "No data 3",
			bytes: []byte{1, 2, 3},
			value: float64(0),
			err:   true,
		},
		{
			title: "No data 5",
			bytes: []byte{1, 2, 3, 4, 5},
			value: float64(0),
			err:   true,
		},
		makeFloat32(0.0),
		makeFloat32(-1.0),
		makeFloat32(1.0),
		makeFloat32(1.0 / 256.0),
		makeFloat32(-123.25),
		makeFloat32(math.Pi),
		makeFloat32(math.MaxFloat32),
		makeFloat64(0.0),
		makeFloat64(math.Pi),
		makeFloat64(math.MaxFloat64),
		makeFloat64(1.1),
	})
}

func TestBoolean(t *testing.T) {
	doTest(t, Boolean, 1, 1, []testCase{
		{
			title: "No data",
			value: false,
			err:   true,
		},
		{
			title: "Bad false 0",
			value: false,
			bytes: []byte{0},
			err:   true,
		},
		{
			title: "True",
			value: true,
			bytes: []byte{1},
		},
		{
			title: "false",
			value: false,
			bytes: []byte{2},
		},
		{
			title: "bad true",
			value: false,
			bytes: []byte{3},
			err:   true,
		},
		{
			title: "extra bytes",
			value: false,
			bytes: []byte{2, 2},
			err:   true,
		},
	})
}

func TestMacAddress(t *testing.T) {
	doTest(t, MacAddress, 6, 6, []testCase{
		{
			title: "No data",
			bytes: []byte{},
			value: net.HardwareAddr{},
			err:   true,
		},
		{
			title: "Not enough",
			bytes: []byte{0, 1, 2, 3, 4},
			value: net.HardwareAddr{},
			err:   true,
		},
		{
			title:    "Generic MAC",
			bytes:    []byte{1, 2, 3, 4, 5, 6},
			value:    net.HardwareAddr{0x1, 0x2, 0x3, 0x4, 0x5, 0x6},
			strValue: "01:02:03:04:05:06",
		},
		{
			title: "Excess",
			bytes: []byte{0, 1, 2, 3, 4, 5, 6},
			value: net.HardwareAddr{},
			err:   true,
		},
	})
}

func TestString(t *testing.T) {
	allAs := make([]byte, math.MaxUint16)
	for i := range allAs {
		allAs[i] = 'A'
	}
	doTest(t, String, 0, math.MaxUint16, []testCase{
		{
			title: "Empty string",
			bytes: []byte{},
			value: "",
		},
		{
			title: "Hello world",
			bytes: []byte("hello world"),
			value: "hello world",
		},
		{
			title: "Single char",
			bytes: []byte{49},
			value: "1",
		},
		{
			title: "Max length",
			bytes: allAs,
			value: string(allAs),
		},
		{
			title: "Zero byte stripped",
			bytes: []byte{0},
			value: "",
		},
		{
			title: "UTF-8",
			bytes: []byte{227, 128, 140, 230, 173, 187, 231, 165, 158, 227, 129, 175, 32, 227, 131, 170, 227, 131, 179, 227, 130, 180, 227, 129, 151, 227, 129, 139, 233, 163, 159, 227, 129, 185, 227, 129, 170, 227, 129, 132, 227, 128, 141},
			value: "「死神は リンゴしか食べない」",
		},
		{
			title: "Valid 2 Octet Sequence",
			bytes: []byte("\xc3\xb1"),
			value: "ñ",
		},
		{
			title: "Invalid 2 Octet Sequence",
			bytes: []byte("\xc3\x28"),
			value: "\xc3(",
		},
		{
			title: "Invalid Sequence Identifier",
			bytes: []byte("\xa0\xa1"),
			value: "\xa0\xa1",
		},
		{
			title: "Valid 3 Octet Sequence",
			bytes: []byte("\xe2\x82\xa1"),
			value: "₡",
		},
		{
			title: "Invalid 3 Octet Sequence (in 2nd Octet)",
			bytes: []byte("\xe2\x28\xa1"),
			value: "\xe2(\xa1",
		},
		{
			title: "Invalid 3 Octet Sequence (in 3rd Octet)",
			bytes: []byte("\xe2\x82\x28"),
			value: "\xe2\x82(",
		},
		{
			title: "Valid 4 Octet Sequence",
			bytes: []byte("\xf0\x90\x8c\xbc"),
			value: "𐌼",
		},
		{
			title: "Invalid 4 Octet Sequence (in 2nd Octet)",
			bytes: []byte("\xf0\x28\x8c\xbc"),
			value: "\xf0(\x8c\xbc",
		},
		{
			title: "Invalid 4 Octet Sequence (in 3rd Octet)",
			bytes: []byte("\xf0\x90\x28\xbc"),
			value: "\xf0\x90(\xbc",
		},
		{
			title: "Invalid 4 Octet Sequence (in 4th Octet)",
			bytes: []byte("\xf0\x28\x8c\x28"),
			value: "\xf0(\x8c(",
		},
		{
			title: "Valid 5 Octet Sequence (but not Unicode!)",
			bytes: []byte("\xf8\xa1\xa1\xa1\xa1"),
			value: "\xf8\xa1\xa1\xa1\xa1",
		},
		{
			title: "Valid 6 Octet Sequence (but not Unicode!)",
			bytes: []byte("\xfc\xa1\xa1\xa1\xa1\xa1"),
			value: "\xfc\xa1\xa1\xa1\xa1\xa1",
		},
		{
			title: "strip trailing nulls",
			bytes: []byte("Hello world\000\000\000\000\000"),
			value: "Hello world",
		},
		{
			title: "don't strip non-trailing nulls",
			bytes: []byte("\000Hello\000world\000"),
			value: "\000Hello\000world",
		},
	})
}

func TestDateTimeSeconds(t *testing.T) {
	timestamp := uint32(time.Now().Unix())
	var nowBytes [4]byte
	binary.BigEndian.PutUint32(nowBytes[:], timestamp)
	now := time.Unix(int64(timestamp), 0).UTC()

	doTest(t, DateTimeSeconds, 4, 4, []testCase{
		{
			title: "Empty",
			bytes: []byte{},
			value: time.Time{},
			err:   true,
		},
		{
			title: "Not enough",
			bytes: []byte{1, 2, 3},
			value: time.Time{},
			err:   true,
		},
		{
			title: "Too much",
			bytes: []byte{1, 2, 3, 4, 5},
			value: time.Time{},
			err:   true,
		},
		{
			title:    "UNIX Epoch",
			bytes:    []byte{0, 0, 0, 0},
			value:    time.Unix(0, 0).UTC(),
			strValue: "1970-01-01 00:00:00 +0000 UTC",
		},
		{
			title:    "Now",
			bytes:    nowBytes[:],
			value:    now,
			strValue: now.String(),
		},
		{
			title:    "Max value",
			bytes:    []byte{255, 255, 255, 255},
			value:    time.Unix(1<<32-1, 0).UTC(),
			strValue: "2106-02-07 06:28:15 +0000 UTC",
		},
	})
}

func TestDateTimeMilliseconds(t *testing.T) {
	timeMillis := time.Now().UnixNano() / int64(time.Millisecond)
	var nowBytes [8]byte
	binary.BigEndian.PutUint64(nowBytes[:], uint64(timeMillis))
	now := time.Unix(timeMillis*int64(time.Millisecond)/int64(time.Second), (timeMillis%1000)*int64(time.Millisecond/time.Nanosecond)).UTC()

	doTest(t, DateTimeMilliseconds, 8, 8, []testCase{
		{
			title: "Empty",
			bytes: []byte{},
			value: time.Time{},
			err:   true,
		},
		{
			title: "Not enough",
			bytes: []byte{1, 2, 3, 4, 5, 6, 7},
			value: time.Time{},
			err:   true,
		},
		{
			title: "Too much",
			bytes: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9},
			value: time.Time{},
			err:   true,
		},
		{
			title:    "UNIX Epoch",
			bytes:    []byte{0, 0, 0, 0, 0, 0, 0, 0},
			value:    time.Unix(0, 0).UTC(),
			strValue: "1970-01-01 00:00:00 +0000 UTC",
		},
		{
			title:    "Now",
			bytes:    nowBytes[:],
			value:    now,
			strValue: now.String(),
		},
		{
			title:    "Max value (63 bits)",
			bytes:    []byte{127, 255, 255, 255, 255, 255, 255, 255},
			value:    time.Unix(math.MaxInt64/1000, (math.MaxInt64%1000)*int64(time.Millisecond/time.Nanosecond)).UTC(),
			strValue: "292278994-08-17 07:12:55.807 +0000 UTC",
		},
		{
			title:    "Max value (64 bits)",
			bytes:    []byte{255, 255, 255, 255, 255, 255, 255, 255},
			value:    time.Unix(math.MaxUint64/1000, (math.MaxUint64%1000)*int64(time.Millisecond/time.Nanosecond)).UTC(),
			strValue: "584556019-04-03 14:25:51.615 +0000 UTC",
		},
	})
}

func TestNTPTimestamp(t *testing.T) {
	timeNow := time.Now().UTC()
	secsNTP := uint32(timeNow.Unix() + int64(UnixEpochInNTP))
	fracNTP := uint32(((timeNow.UnixNano() % int64(time.Second)) << 32) / int64(time.Second))

	// There is a small precision loss in the conversion between NTP and Time,
	// need to recalculate otherwise there's a nanosecond difference (rounding?)
	now := time.Unix(int64(secsNTP-UnixEpochInNTP), int64(fracNTP)*int64(time.Second)/(int64(0x100000000))).UTC()
	var nowBytes [8]byte
	binary.BigEndian.PutUint32(nowBytes[:4], secsNTP)
	binary.BigEndian.PutUint32(nowBytes[4:], fracNTP)

	var centuryBytes [8]byte
	binary.BigEndian.PutUint32(centuryBytes[:], 3155587200)

	doTest(t, DateTimeMicroseconds, 8, 8, []testCase{
		{
			title: "Empty",
			bytes: []byte{},
			value: time.Time{},
			err:   true,
		},
		{
			title: "Not enough",
			bytes: []byte{1, 2, 3, 4, 5, 6, 7},
			value: time.Time{},
			err:   true,
		},
		{
			title: "Too much",
			bytes: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9},
			value: time.Time{},
			err:   true,
		},
		{
			title:    "NTP Epoch",
			bytes:    []byte{0, 0, 0, 0, 0, 0, 0, 0},
			value:    NtpEpoch,
			strValue: "1900-01-01 00:00:00 +0000 UTC",
		},
		{
			title:    "Now",
			bytes:    nowBytes[:],
			value:    now,
			strValue: now.String(),
		},
		{
			title:    "Max value (64 bits)",
			bytes:    []byte{255, 255, 255, 255, 255, 255, 255, 255},
			value:    time.Unix(int64(math.MaxUint32-UnixEpochInNTP), int64(math.MaxUint32)*int64(time.Second)/int64(1<<32)).UTC(),
			strValue: "2036-02-07 06:28:15.999999999 +0000 UTC",
		},
		{
			title:    "Last day 20th century",
			bytes:    centuryBytes[:],
			value:    time.Date(1999, 12, 31, 0, 0, 0, 0, time.UTC),
			strValue: "1999-12-31 00:00:00 +0000 UTC",
		},
		{
			title:    "Random date from NTP server",
			bytes:    []byte{0xdf, 0x96, 0xd0, 0x2, 0x56, 0x67, 0xf8, 0xf3},
			value:    time.Date(2018, 11, 14, 16, 46, 58, 337523993, time.UTC),
			strValue: "2018-11-14 16:46:58.337523993 +0000 UTC",
		},
	})
}

func TestIPv4(t *testing.T) {
	doTest(t, Ipv4Address, 4, 4, []testCase{
		{
			title: "Empty",
			bytes: []byte{},
			value: net.IP{},
			err:   true,
		},
		{
			title: "Too little",
			bytes: []byte{1, 2, 3},
			value: net.IP{},
			err:   true,
		},
		{
			title: "Too much",
			bytes: []byte{1, 2, 3, 4, 5},
			value: net.IP{},
			err:   true,
		},
		{
			title:    "IP address",
			bytes:    []byte{192, 0, 2, 135},
			value:    net.IPv4(192, 0, 2, 135).To4(),
			strValue: "192.0.2.135",
		},
		{
			title:    "Zero address",
			bytes:    []byte{0, 0, 0, 0},
			value:    net.IPv4(0, 0, 0, 0).To4(),
			strValue: "0.0.0.0",
		},
		{
			title:    "Broadcast address",
			bytes:    []byte{255, 255, 255, 255},
			value:    net.IPv4(255, 255, 255, 255).To4(),
			strValue: "255.255.255.255",
		},
	})
}

func TestIPv6(t *testing.T) {
	doTest(t, Ipv6Address, 16, 16, []testCase{
		{
			title: "Empty",
			bytes: []byte{},
			value: net.IP{},
			err:   true,
		},
		{
			title: "Too little",
			bytes: make([]byte, 15),
			value: net.IP{},
			err:   true,
		},
		{
			title: "Too much",
			bytes: make([]byte, 17),
			value: net.IP{},
			err:   true,
		},
		{
			title:    "IPv6 address",
			bytes:    []byte{0x20, 0x01, 0x0d, 0xb8, 0x00, 0x00, 0x00, 0x00, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0},
			value:    net.ParseIP("2001:db8::1234:5678:9abc:def0"),
			strValue: "2001:db8::1234:5678:9abc:def0",
		},
		{
			title:    "Zero address",
			bytes:    make([]byte, 16),
			value:    net.ParseIP("::"),
			strValue: "::",
		},
	})
}

func TestUnsupported(t *testing.T) {
	doTest(t, BasicList, 0, math.MaxUint16, []testCase{
		{
			title: "Empty",
			bytes: []byte{},
			err:   true,
		},
		{
			title: "Any",
			bytes: make([]byte, 15),
			err:   true,
		},
	})
}

func TestACLID(t *testing.T) {
	doTest(t, ACLID, 12, 12, []testCase{
		{
			title: "Empty",
			bytes: []byte{},
			err:   true,
		},
		{
			title: "Sample",
			bytes: []byte{
				0x10, 0x21, 0x32, 0x43,
				0x54, 0x65, 0x76, 0x87,
				0x98, 0xA9, 0xBA, 0xCD,
			},
			value: "10213243-54657687-98a9bacd",
		},
		{
			title: "Short",
			bytes: []byte{
				0x10, 0x21, 0x32, 0x43,
				0x54, 0x65, 0x76, 0x87,
				0x98, 0xA9, 0xBA,
			},
			err: true,
		},
		{
			title: "Long",
			bytes: []byte{
				0x10, 0x21, 0x32, 0x43,
				0x54, 0x65, 0x76, 0x87,
				0x98, 0xA9, 0xBA, 0xCD,
				0xDF,
			},
			err: true,
		},
	})
}
