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
	"errors"
	"testing"
	"time"

	"github.com/google/gopacket/layers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/packetbeat/config"
	"github.com/elastic/beats/v7/packetbeat/decoder"
	"github.com/elastic/beats/v7/packetbeat/flows"
	"github.com/elastic/beats/v7/packetbeat/protos"
	"github.com/elastic/elastic-agent-libs/logp"
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

	_, _, _, err = afpacketComputeSize(0, 1514, 4096)
	if err == nil {
		t.Error("Expected an error")
	}

	// 16436 is the default MTU size of the loopback interface
	frameSize, blockSize, numBlocks, err = afpacketComputeSize(30, 16436, 4096)
	if err != nil {
		t.Error(err)
	}
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

func TestEnsureDecoderLifecycle(t *testing.T) {
	var created int
	var firstCleanupCalls int
	var secondCleanupCalls int

	s := sniffer{
		log: logp.NewLogger("sniffer_test"),
		decoders: func(_ layers.LinkType, _ string, _ int) (*decoder.Decoder, func(), error) {
			created++
			switch created {
			case 1:
				return &decoder.Decoder{}, func() { firstCleanupCalls++ }, nil
			case 2:
				return &decoder.Decoder{}, func() { secondCleanupCalls++ }, nil
			default:
				t.Fatalf("unexpected decoder creation %d", created)
				return nil, nil, nil
			}
		},
	}

	var (
		last    layers.LinkType
		dec     *decoder.Decoder
		cleanup func()
		err     error
	)

	last, dec, cleanup, err = s.ensureDecoder(layers.LinkTypeEthernet, "eth0", last, dec, cleanup)
	require.NoError(t, err)
	require.NotNil(t, dec)
	require.NotNil(t, cleanup)
	assert.Equal(t, 1, created)
	assert.Equal(t, 0, firstCleanupCalls)

	firstDec := dec
	last, dec, cleanup, err = s.ensureDecoder(layers.LinkTypeEthernet, "eth1", last, dec, cleanup)
	require.NoError(t, err)
	assert.Same(t, firstDec, dec)
	assert.Equal(t, 1, created)
	assert.Equal(t, 0, firstCleanupCalls)

	last, dec, cleanup, err = s.ensureDecoder(layers.LinkTypeLinuxSLL, "any", last, dec, cleanup)
	require.NoError(t, err)
	require.NotNil(t, dec)
	require.NotNil(t, cleanup)
	assert.Equal(t, 2, created)
	assert.Equal(t, 1, firstCleanupCalls)
	assert.Equal(t, 0, secondCleanupCalls)

	cleanup()
	assert.Equal(t, 1, secondCleanupCalls)
}

func TestEnsureDecoderReplaceErrorKeepsCurrentCleanup(t *testing.T) {
	var created int
	var cleanupCalls int

	s := sniffer{
		log: logp.NewLogger("sniffer_test"),
		decoders: func(_ layers.LinkType, _ string, _ int) (*decoder.Decoder, func(), error) {
			created++
			if created == 1 {
				return &decoder.Decoder{}, func() { cleanupCalls++ }, nil
			}
			return nil, nil, errors.New("boom")
		},
	}

	last, dec, cleanup, err := s.ensureDecoder(layers.LinkTypeEthernet, "eth0", 0, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, dec)
	require.NotNil(t, cleanup)

	_, _, cleanup, err = s.ensureDecoder(layers.LinkTypeLinuxSLL, "any", last, dec, cleanup)
	require.Error(t, err)
	assert.Equal(t, 0, cleanupCalls)

	cleanup()
	assert.Equal(t, 1, cleanupCalls)
}

func TestSnifferStopRunLifecycle(t *testing.T) {
	t.Parallel()

	s := newTestFileSniffer(t)

	s.Stop()

	done := make(chan error, 1)
	go func() {
		done <- s.Run()
	}()

	select {
	case err := <-done:
		assert.NoError(t, err, "Run should return cleanly after Stop-before-Run")
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after Stop-before-Run")
	}

	for range 2 {
		s.Stop()
	}
}

func TestSnifferConcurrentStopRun(t *testing.T) {
	t.Parallel()

	const iterations = 50
	for i := range iterations {
		s := newTestFileSniffer(t)

		done := make(chan error, 1)
		go func() {
			done <- s.Run()
		}()

		for range 10 {
			s.Stop()
		}

		select {
		case err := <-done:
			assert.NoError(t, err, "Run should return cleanly after concurrent Stop calls")
		case <-time.After(5 * time.Second):
			t.Fatalf("Run did not return after concurrent Stop calls on iteration %d", i)
		}
	}
}

func newTestFileSniffer(t *testing.T) *Sniffer {
	t.Helper()

	logger := logp.NewLogger("sniffer_test")
	decoders := map[string]Decoders{
		"": func(dl layers.LinkType, _ string, _ int) (*decoder.Decoder, func(), error) {
			dec, err := decoder.New(nil, dl, discardICMP{}, discardICMP{}, discardTCP{}, discardUDP{}, false, logger)
			if err != nil {
				return nil, nil, err
			}
			return dec, func() {}, nil
		},
	}
	interfaces := []config.InterfaceConfig{{
		File: "../tests/system/pcaps/http_x_forwarded_for.pcap",
		Loop: 1000,
	}}
	s, err := New("test", true, "", decoders, interfaces, nil, logger)
	require.NoError(t, err, "creating a file-backed sniffer should succeed")
	return s
}

// The lifecycle tests only need a decoder that drains the pcap without
// panicking, so every protocol processor discards what it is handed.
type (
	discardTCP  struct{}
	discardUDP  struct{}
	discardICMP struct{}
)

func (discardTCP) Process(_ *flows.FlowID, _ *layers.TCP, _ *protos.Packet) {}

func (discardUDP) Process(_ *flows.FlowID, _ *protos.Packet) {}

func (discardICMP) ProcessICMPv4(_ *flows.FlowID, _ *layers.ICMPv4, _ *protos.Packet) {}

func (discardICMP) ProcessICMPv6(_ *flows.FlowID, _ *layers.ICMPv6, _ *protos.Packet) {}
