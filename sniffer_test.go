package main

import (
	"testing"
)

func TestSniffer_afpacketComputeSize(t *testing.T) {
	var block_size, num_blocks int
	var err error

	block_size, num_blocks, err = afpacketComputeSize(30, 1514)
	if err != nil {
		t.Error(err)
	}
	if block_size != 1514*128 || num_blocks != 162 {
		t.Error("Bad result", block_size, num_blocks)
	}
	if block_size*num_blocks > 30*1024*1024 {
		t.Error("Value too big", block_size, num_blocks)
	}

	block_size, num_blocks, err = afpacketComputeSize(1, 1514)
	if err != nil {
		t.Error(err)
	}
	if block_size != 1514*128 || num_blocks != 5 {
		t.Error("Bad result", block_size, num_blocks)
	}
	if block_size*num_blocks > 1*1024*1024 {
		t.Error("Value too big", block_size, num_blocks)
	}

	block_size, num_blocks, err = afpacketComputeSize(0, 1514)
	if err == nil {
		t.Error("Expected an error")
	}
}
