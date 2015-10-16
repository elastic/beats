package common

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBytes_Ntohs(t *testing.T) {
	type io struct {
		Input  []byte
		Output uint16
	}

	tests := []io{
		io{
			Input:  []byte{0, 1},
			Output: 1,
		},
		io{
			Input:  []byte{1, 0},
			Output: 256,
		},
		io{
			Input:  []byte{1, 2},
			Output: 258,
		},
		io{
			Input:  []byte{2, 3},
			Output: 515,
		},
	}

	for _, test := range tests {
		assert.Equal(t, test.Output, Bytes_Ntohs(test.Input))
	}
}

func TestBytes_Ntohl(t *testing.T) {
	type io struct {
		Input  []byte
		Output uint32
	}

	tests := []io{
		io{
			Input:  []byte{0, 0, 0, 1},
			Output: 1,
		},
		io{
			Input:  []byte{0, 0, 1, 0},
			Output: 256,
		},
		io{
			Input:  []byte{0, 1, 0, 0},
			Output: 1 << 16,
		},
		io{
			Input:  []byte{1, 0, 0, 0},
			Output: 1 << 24,
		},
		io{
			Input:  []byte{1, 0, 15, 0},
			Output: 0x01000f00,
		},
	}

	for _, test := range tests {
		assert.Equal(t, test.Output, Bytes_Ntohl(test.Input))
	}
}

func TestBytes_Htohl(t *testing.T) {
	type io struct {
		Input  []byte
		Output uint32
	}

	tests := []io{
		io{
			Input:  []byte{0, 0, 0, 1},
			Output: 1 << 24,
		},
		io{
			Input:  []byte{0, 0, 1, 0},
			Output: 1 << 16,
		},
		io{
			Input:  []byte{0, 1, 0, 0},
			Output: 256,
		},
		io{
			Input:  []byte{1, 0, 0, 0},
			Output: 1,
		},
		io{
			Input:  []byte{1, 0, 15, 0},
			Output: 0x000f0001,
		},
	}

	for _, test := range tests {
		assert.Equal(t, test.Output, Bytes_Htohl(test.Input))
	}
}

func TestBytes_Ntohll(t *testing.T) {
	type io struct {
		Input  []byte
		Output uint64
	}

	tests := []io{
		io{
			Input:  []byte{0, 0, 0, 0, 0, 0, 0, 1},
			Output: 1,
		},
		io{
			Input:  []byte{0, 0, 0, 0, 0, 0, 1, 0},
			Output: 256,
		},
		io{
			Input:  []byte{0, 0, 0, 0, 0, 1, 0, 0},
			Output: 1 << 16,
		},
		io{
			Input:  []byte{0, 0, 0, 0, 1, 0, 0, 0},
			Output: 1 << 24,
		},
		io{
			Input:  []byte{0, 0, 0, 1, 0, 0, 0, 0},
			Output: 1 << 32,
		},
		io{
			Input:  []byte{0, 0, 1, 0, 0, 0, 0, 0},
			Output: 1 << 40,
		},
		io{
			Input:  []byte{0, 1, 0, 0, 0, 0, 0, 0},
			Output: 1 << 48,
		},
		io{
			Input:  []byte{1, 0, 0, 0, 0, 0, 0, 0},
			Output: 1 << 56,
		},
		io{
			Input:  []byte{0, 1, 0, 0, 1, 0, 15, 0},
			Output: 0x0001000001000f00,
		},
	}

	for _, test := range tests {
		assert.Equal(t, test.Output, Bytes_Ntohll(test.Input))
	}
}

func TestIpv4_Ntoa(t *testing.T) {
	type io struct {
		Input  uint32
		Output string
	}

	tests := []io{
		io{
			Input:  0x7f000001,
			Output: "127.0.0.1",
		},
		io{
			Input:  0xc0a80101,
			Output: "192.168.1.1",
		},
		io{
			Input:  0,
			Output: "0.0.0.0",
		},
	}

	for _, test := range tests {
		assert.Equal(t, test.Output, Ipv4_Ntoa(test.Input))
	}
}

func TestReadString(t *testing.T) {
	type io struct {
		Input  []byte
		Output string
		Err    error
	}

	tests := []io{
		io{
			Input:  []byte{'a', 'b', 'c', 0, 'd', 'e', 'f'},
			Output: "abc",
			Err:    nil,
		},
		io{
			Input:  []byte{0},
			Output: "",
			Err:    nil,
		},
		io{
			Input:  []byte{'a', 'b', 'c'},
			Output: "",
			Err:    errors.New("No string found"),
		},
		io{
			Input:  []byte{},
			Output: "",
			Err:    errors.New("No string found"),
		},
	}

	for _, test := range tests {
		res, err := ReadString(test.Input)
		assert.Equal(t, test.Err, err)
		assert.Equal(t, test.Output, res)
	}
}
