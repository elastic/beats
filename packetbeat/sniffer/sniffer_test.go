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

// +build !integration

package sniffer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSniffer_afpacketComputeSize(t *testing.T) {
	var frameSize, blockSize, numBlocks int
	var err error

	frameSize, blockSize, numBlocks, err = afpacketComputeSize(30, 1514, 4096)
	if err != nil {
		t.Error(err)
	}
	if frameSize != 2048 || blockSize != 2048*128 || numBlocks != 120 {
		t.Error("Bad result", frameSize, blockSize, numBlocks)
	}
	if blockSize*numBlocks > 30*1024*1024 {
		t.Error("Value too big", blockSize, numBlocks)
	}

	frameSize, blockSize, numBlocks, err = afpacketComputeSize(1, 1514, 4096)
	if err != nil {
		t.Error(err)
	}
	if frameSize != 2048 || blockSize != 2048*128 || numBlocks != 4 {
		t.Error("Bad result", blockSize, numBlocks)
	}
	if blockSize*numBlocks > 1*1024*1024 {
		t.Error("Value too big", blockSize, numBlocks)
	}

	frameSize, blockSize, numBlocks, err = afpacketComputeSize(0, 1514, 4096)
	if err == nil {
		t.Error("Expected an error")
	}

	// 16436 is the default MTU size of the loopback interface
	frameSize, blockSize, numBlocks, err = afpacketComputeSize(30, 16436, 4096)
	if frameSize != 4096*5 || blockSize != 4096*5*128 || numBlocks != 12 {
		t.Error("Bad result", frameSize, blockSize, numBlocks)
	}

	frameSize, blockSize, numBlocks, err = afpacketComputeSize(3, 16436, 4096)
	if err != nil {
		t.Error(err)
	}
	if frameSize != 4096*5 || blockSize != 4096*5*128 || numBlocks != 1 {
		t.Error("Bad result", frameSize, blockSize, numBlocks)
	}
}

func Test_deviceNameFromIndex(t *testing.T) {
	devs := []string{"lo", "eth0", "eth1"}

	name, err := deviceNameFromIndex(0, devs)
	assert.Equal(t, "lo", name)
	assert.NoError(t, err)

	name, err = deviceNameFromIndex(1, devs)
	assert.Equal(t, "eth0", name)
	assert.NoError(t, err)

	name, err = deviceNameFromIndex(2, devs)
	assert.Equal(t, "eth1", name)
	assert.NoError(t, err)

	_, err = deviceNameFromIndex(3, devs)
	assert.Error(t, err)
}
