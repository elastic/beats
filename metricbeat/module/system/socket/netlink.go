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

// +build linux

package socket

import (
	"os"
	"sync/atomic"

	"github.com/elastic/gosigar/sys/linux"
	"github.com/pkg/errors"
)

type NetlinkSession struct {
	readBuffer []byte
	seq        uint32
}

func NewNetlinkSession() *NetlinkSession {
	return &NetlinkSession{
		readBuffer: make([]byte, os.Getpagesize()),
	}
}

func (session *NetlinkSession) GetSocketList() ([]*linux.InetDiagMsg, error) {
	// Send request over netlink and parse responses.
	req := linux.NewInetDiagReq()
	req.Header.Seq = atomic.AddUint32(&session.seq, 1)
	sockets, err := linux.NetlinkInetDiagWithBuf(req, session.readBuffer, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed requesting socket dump")
	}
	return sockets, nil
}
