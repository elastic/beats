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

package tracer

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
)

type Tracer interface {
	Start()
	Abort()
	Close()
}

type SockTracer struct {
	path string
	sock net.Conn
}

const (
	MSG_START = "start"
	MSG_STOP  = "stop"
	MSG_ABORT = "abort"
)

func NewSockTracer(path string, wait time.Duration) (st SockTracer, err error) {
	st.path = path
	delay := time.Millisecond * 250

	started := time.Now()
	for {
		elapsed := time.Since(started)
		if elapsed > wait {
			return st, fmt.Errorf("wait time for sock trace exceeded: %s", wait)
		}
		if _, err := os.Stat(st.path); err == nil {
			logp.L().Infof("socktracer found file for unix socket: %s, will attempt to connect", path)
			break
		} else {
			logp.L().Infof("socktracer could not find file for unix socket at: %s, will retry in %s", path, delay)
			time.Sleep(delay)
		}
	}

	st.sock, err = net.Dial("unix", path)
	if err != nil {
		return SockTracer{}, fmt.Errorf("could not create sock tracer at %s: %w", path, err)
	}

	return st, nil
}

func (st SockTracer) Start() {
	err := st.write(MSG_START)
	if err != nil {
		logp.L().Errorf("could not write start trace message: %s", err)
	}
}

func (st SockTracer) Abort() {
	err := st.write(MSG_ABORT)
	if err != nil {
		logp.L().Errorf("could not write abort trace message: %s", err)
	}
	st.closeSock()
}

func (st SockTracer) Close() {
	err := st.write(MSG_STOP)
	if err != nil {
		logp.L().Errorf("could not write stop trace message: %s", err)
	}
	st.closeSock()
}

func (st SockTracer) closeSock() {
	err := st.sock.Close()
	if err != nil {
		logp.L().Errorf("could not close trace sock: %s", err)
	}
}

func (st SockTracer) write(message string) error {
	// Note, we don't need to worry about partial writes here: https://pkg.go.dev/io?utm_source=godoc#Writer
	// an error will be returned here, which shouldn't really happen with unix sockets only
	_, err := st.sock.Write([]byte(message + "\n"))
	return err
}

type NoopTracer struct{}

func NewNoopTracer() NoopTracer {
	return NoopTracer{}
}

func (nt NoopTracer) Start() {
}

func (nt NoopTracer) Abort() {
}

func (nt NoopTracer) Close() {
}
