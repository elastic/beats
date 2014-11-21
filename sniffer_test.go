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

	// 16436 is the default MTU size of the loopback interface
	frame_size, block_size, num_blocks, err = afpacketComputeSize(30, 16436, 4096)
	if frame_size != 4096*5 || block_size != 4096*5*128 || num_blocks != 12 {
		t.Error("Bad result", frame_size, block_size, num_blocks)
	}

	frame_size, block_size, num_blocks, err = afpacketComputeSize(3, 16436, 4096)
	if err != nil {
		t.Error(err)
	}
	if frame_size != 4096*5 || block_size != 4096*5*128 || num_blocks != 1 {
		t.Error("Bad result", frame_size, block_size, num_blocks)
	}
}
