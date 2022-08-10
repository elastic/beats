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
	"io"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"

	"github.com/elastic/beats/v7/libbeat/logp"
)

type fileHandler struct {
	pcapHandle *pcap.Handle
	file       string

	loopCount, maxLoopCount int

	topSpeed bool
	lastTS   time.Time
}

func newFileHandler(file string, topSpeed bool, maxLoopCount int) (*fileHandler, error) {
	h := &fileHandler{
		file:         file,
		topSpeed:     topSpeed,
		maxLoopCount: maxLoopCount,
	}
	if err := h.open(); err != nil {
		return nil, err
	}

	return h, nil
}

func (h *fileHandler) open() error {
	tmp, err := pcap.OpenOffline(h.file)
	if err != nil {
		return err
	}

	h.pcapHandle = tmp
	return nil
}

func (h *fileHandler) ReadPacketData() ([]byte, gopacket.CaptureInfo, error) {
	data, ci, err := h.pcapHandle.ReadPacketData()
	if err != nil {
		if err != io.EOF {
			return data, ci, err
		}

		h.pcapHandle.Close()
		h.pcapHandle = nil

		h.loopCount++
		if h.loopCount >= h.maxLoopCount {
			return data, ci, err
		}

		logp.Debug("sniffer", "Reopening the file")
		if err = h.open(); err != nil {
			return nil, ci, fmt.Errorf("Error reopening file: %s", err)
		}

		data, ci, err = h.pcapHandle.ReadPacketData()
		h.lastTS = ci.Timestamp
		return data, ci, err
	}

	if h.topSpeed {
		return data, ci, nil
	}

	if !h.lastTS.IsZero() {
		sleep := ci.Timestamp.Sub(h.lastTS)
		if sleep > 0 {
			time.Sleep(sleep)
		} else {
			logp.Warn("Time in pcap went backwards: %d", sleep)
		}
	}

	h.lastTS = ci.Timestamp
	ci.Timestamp = time.Now()
	return data, ci, nil
}

func (h *fileHandler) LinkType() layers.LinkType {
	return h.pcapHandle.LinkType()
}

func (h *fileHandler) Close() {
	if h.pcapHandle != nil {
		h.pcapHandle.Close()
		h.pcapHandle = nil
	}
}
