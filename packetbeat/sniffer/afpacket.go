// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package sniffer

import (
	"fmt"
	"time"
)

type afPacketConfig struct {
	// ID is the AF_PACKET identifier for metric collection.
	ID string
	// Device name (e.g. eth0). 'any' may be used to listen on all interfaces.
	Device string
	// Size of frame. A frame can be of any size with the only condition it can fit in a block.
	FrameSize int
	// Minimal size of contiguous block. Must be divisible by the FrameSize and OS page size.
	BlockSize       int
	NumBlocks       int           // Number of blocks.
	MetricsInterval time.Duration // Metrics polling interval.
	PollTimeout     time.Duration // Duration that poll() should block waiting for data.
	FanoutGroupID   *uint16       // Optional fanout group identifier.
	Promiscuous     bool          // Put device into promiscuous mode. Ignored when using 'any' device.
}

// afpacketComputeSize computes the block_size and the num_blocks in such a way
// that the allocated mmap buffer is close to but smaller than target_size_mb.
// The restriction is that the block_size must be divisible by both the frame
// size and page size.
func afpacketComputeSize(targetSizeMb, snaplen, pageSize int) (frameSize, blockSize, numBlocks int, err error) {
	if snaplen < pageSize {
		frameSize = pageSize / (pageSize / snaplen)
	} else {
		frameSize = (snaplen/pageSize + 1) * pageSize
	}

	// 128 is the default from the gopacket library so just use that
	blockSize = frameSize * 128
	numBlocks = (targetSizeMb * 1024 * 1024) / blockSize

	if numBlocks == 0 {
		return 0, 0, 0, fmt.Errorf("buffer size too small")
	}

	return frameSize, blockSize, numBlocks, nil
}
