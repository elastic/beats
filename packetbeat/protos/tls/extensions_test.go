// +build !integration

package tls

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSni(t *testing.T) {

	// Single element

	buf := mkBuf(t, "000d"+ // 13 bytes
		"00"+ // type host
		"000a"+ // 10 byte string
		"656c61737469632e636f", // elastic.co
		15)
	r := parseSni(*buf)
	assert.NotNil(t, r)
	assert.Equal(t, []string{"elastic.co"}, r.([]string))

	// 3 elements

	buf = mkBuf(t, "0027"+ // 39 bytes
		"00"+ // type host
		"000a"+ // 10 byte string
		"656c61737469632e636f"+ // elastic.co
		"00"+ // type host
		"000b"+ // 11 byte string
		"6578616d706c652e6e6574"+ // example.net
		"00"+ // type host
		"0009"+ // 9 byte string
		"6c6f63616c686f7374", // localhost
		41)
	r = parseSni(*buf)
	assert.NotNil(t, r)
	assert.Equal(t, []string{"elastic.co", "example.net", "localhost"}, r.([]string))

	// Unknown entry type

	buf = mkBuf(t, "0027"+ // 39 bytes
		"00"+ // type host
		"000a"+ // 10 byte string
		"656c61737469632e636f"+ // elastic.co
		"01"+ // type ???
		"000b"+ // 11 byte string
		"6578616d706c652e6e6574"+ // example.net
		"00"+ // type host
		"0009"+ // 9 byte string
		"6c6f63616c686f7374", // localhost
		41)
	r = parseSni(*buf)
	assert.NotNil(t, r)
	assert.Equal(t, []string{"elastic.co", "localhost"}, r.([]string))

	// Truncated

	buf = mkBuf(t, "0400"+ // 1024 bytes
		"00"+ // type host
		"000a"+ // 10 byte string
		"656c61737469632e636f"+ // elastic.co
		"00"+ // type host
		"000b"+ // 11 byte string
		"6578616d706c652e6e6574"+ // example.net
		"00"+ // type host
		"0009"+ // 9 byte string
		"6c6f63616c686f7374", // localhost
		41)
	r = parseSni(*buf)
	assert.NotNil(t, r)
	assert.Equal(t, []string{"elastic.co", "example.net", "localhost"}, r.([]string))

	// Out of bounds

	buf = mkBuf(t, "0026"+ // 38 bytes
		"00"+ // type host
		"000a"+ // 10 byte string
		"656c61737469632e636f"+ // elastic.co
		"00"+ // type host
		"000b"+ // 11 byte string
		"6578616d706c652e6e6574"+ // example.net
		"00"+ // type host
		"0009"+ // 9 byte string
		"6c6f63616c686f7374", // localhost
		41)
	r = parseSni(*buf)
	assert.NotNil(t, r)
	assert.Equal(t, []string{"elastic.co", "example.net"}, r.([]string))

	// Out of bounds

	buf = mkBuf(t, "001c"+ // 28 bytes
		"00"+ // type host
		"000a"+ // 10 byte string
		"656c61737469632e636f"+ // elastic.co
		"00"+ // type host
		"000b"+ // 11 byte string
		"6578616d706c652e6e6574"+ // example.net
		"00"+ // type host
		"0009"+ // 9 byte string
		"6c6f63616c686f7374", // localhost
		41)
	r = parseSni(*buf)
	assert.NotNil(t, r)
	assert.Equal(t, []string{"elastic.co", "example.net"}, r.([]string))
}

func TestParseMaxFragmentLength(t *testing.T) {

	r := parseMaxFragmentLen(*mkBuf(t, "01", 1))
	assert.Equal(t, "2^9", r.(string))
	r = parseMaxFragmentLen(*mkBuf(t, "04", 1))
	assert.Equal(t, "2^12", r.(string))
	r = parseMaxFragmentLen(*mkBuf(t, "00", 1))
	assert.Equal(t, "(unknown:0)", r.(string))
	r = parseMaxFragmentLen(*mkBuf(t, "FF", 1))
	assert.Equal(t, "(unknown:255)", r.(string))
	r = parseMaxFragmentLen(*mkBuf(t, "FF", 2))
	assert.Nil(t, r)
}

func TestParseCertType(t *testing.T) {
	r := parseCertType(*mkBuf(t, "00", 1))
	assert.Equal(t, []string{"X.509"}, r.([]string))
	r = parseCertType(*mkBuf(t, "01", 1))
	assert.Equal(t, []string{"OpenPGP"}, r.([]string))
	r = parseCertType(*mkBuf(t, "02", 1))
	assert.Equal(t, []string{"RawPubKey"}, r.([]string))
	r = parseCertType(*mkBuf(t, "3c", 1))
	assert.Equal(t, []string{"(unknown:60)"}, r.([]string))
	r = parseCertType(*mkBuf(t, "03020100", 4))
	assert.Equal(t, []string{"RawPubKey", "OpenPGP", "X.509"}, r.([]string))
}

func TestParseSrp(t *testing.T) {
	r := parseSrp(*mkBuf(t, "04726f6f74", 5))
	assert.Equal(t, "root", r.(string))
	r = parseSrp(*mkBuf(t, "FF726f6f74", 5))
	assert.Nil(t, r)
}
