package main

import (
	"testing"
)

func TestSniffer_afpacketComputeSize(t *testing.T) {
	var frame_size, block_size, num_blocks int
	var err error

	frame_size, block_size, num_blocks, err = afpacketComputeSize(30, 1514, 4096)
	if err != nil {
		t.Error(err)
	}
	if frame_size != 2048 || block_size != 2048*128 || num_blocks != 120 {
		t.Error("Bad result", frame_size, block_size, num_blocks)
	}
	if block_size*num_blocks > 30*1024*1024 {
		t.Error("Value too big", block_size, num_blocks)
	}

	frame_size, block_size, num_blocks, err = afpacketComputeSize(1, 1514, 4096)
	if err != nil {
		t.Error(err)
	}
	if frame_size != 2048 || block_size != 2048*128 || num_blocks != 4 {
		t.Error("Bad result", block_size, num_blocks)
	}
	if block_size*num_blocks > 1*1024*1024 {
		t.Error("Value too big", block_size, num_blocks)
	}

	frame_size, block_size, num_blocks, err = afpacketComputeSize(0, 1514, 4096)
	if err == nil {
		t.Error("Expected an error")
	}
}
