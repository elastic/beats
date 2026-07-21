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

//go:build !integration

package sniffer

import (
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcapgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

func writeTestPcap(t *testing.T, path string, packets int) {
	t.Helper()
	f, err := os.Create(path)
	require.NoError(t, err, "create pcap file")
	defer f.Close()

	w := pcapgo.NewWriter(f)
	require.NoError(t, w.WriteFileHeader(65535, layers.LinkTypeEthernet), "write pcap header")

	// Minimal ethernet frame (dst+src+type).
	frame := make([]byte, 14)
	ts := time.Unix(1, 0)
	for i := 0; i < packets; i++ {
		ci := gopacket.CaptureInfo{
			Timestamp:     ts.Add(time.Duration(i) * time.Millisecond),
			CaptureLength: len(frame),
			Length:        len(frame),
		}
		require.NoError(t, w.WritePacket(ci, frame), "write packet %d", i)
	}
}

func TestFileHandlerLoopForever(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "loop.pcap")
	const packetsPerPass = 3
	writeTestPcap(t, path, packetsPerPass)

	h, err := newFileHandler(path, true, 0, logptest.NewTestingLogger(t, ""))
	require.NoError(t, err, "open file handler with loop forever")
	defer h.Close()

	// One pass plus the first packet of the reopen must succeed when maxLoopCount is 0.
	for i := 0; i < packetsPerPass+1; i++ {
		_, _, err := h.ReadPacketData()
		require.NoError(t, err, "packet %d should be readable with -l 0", i)
	}
}

func TestFileHandlerLoopOnce(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "once.pcap")
	const packetsPerPass = 2
	writeTestPcap(t, path, packetsPerPass)

	h, err := newFileHandler(path, true, 1, logptest.NewTestingLogger(t, ""))
	require.NoError(t, err, "open file handler with loop once")
	defer h.Close()

	for i := 0; i < packetsPerPass; i++ {
		_, _, err := h.ReadPacketData()
		require.NoError(t, err, "packet %d should be readable", i)
	}
	_, _, err = h.ReadPacketData()
	assert.ErrorIs(t, err, io.EOF, "should stop after one pass when -l 1")
}
