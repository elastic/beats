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

//go:build !linux

package sniffer

import (
	"errors"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

var errAFPacketLinuxOnly = errors.New("af_packet MMAP sniffing is only available on Linux")

type afpacketHandle struct{}

func newAfpacketHandle(_ afPacketConfig) (*afpacketHandle, error) {
	return nil, errAFPacketLinuxOnly
}

func (*afpacketHandle) ReadPacketData() ([]byte, gopacket.CaptureInfo, error) {
	return nil, gopacket.CaptureInfo{}, errAFPacketLinuxOnly
}

func (*afpacketHandle) SetBPFFilter(_ string) error {
	return errAFPacketLinuxOnly
}

func (*afpacketHandle) LinkType() layers.LinkType {
	return 0
}

func (*afpacketHandle) Close() {}

// isAfpacketErrTimeout returns whether the error is afpacket.ErrTimeout, always false on
// non-linux systems.
func isAfpacketErrTimeout(error) bool {
	return false
}
