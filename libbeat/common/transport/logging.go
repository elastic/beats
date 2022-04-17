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

package transport

import (
	"io"
	"net"

	"github.com/menderesk/beats/v7/libbeat/logp"
)

type loggingConn struct {
	net.Conn
	logger *logp.Logger
}

func LoggingDialer(d Dialer, logger *logp.Logger) Dialer {
	return DialerFunc(func(network, addr string) (net.Conn, error) {
		logger := logger.With("network", network, "address", addr)
		c, err := d.Dial(network, addr)
		if err != nil {
			logger.Errorf("Error dialing %v", err)
			return nil, err
		}

		logger.Debugf("Completed dialing successfully")
		return &loggingConn{c, logger}, nil
	})
}

func (l *loggingConn) Read(b []byte) (int, error) {
	n, err := l.Conn.Read(b)
	if err != nil && err != io.EOF {
		l.logger.Debugf("Error reading from connection: %v", err)
	}
	return n, err
}

func (l *loggingConn) Write(b []byte) (int, error) {
	n, err := l.Conn.Write(b)
	if err != nil && err != io.EOF {
		l.logger.Debugf("Error writing to connection: %v", err)
	}
	return n, err
}
