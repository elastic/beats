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

package testutil

import (
	"context"
	"fmt"
	"net"
	"time"
)

const defaultHost = "127.0.0.1"

// HostAddress returns host:port for net inputs.
func HostAddress(port uint16) string {
	return net.JoinHostPort(defaultHost, fmt.Sprintf("%d", port))
}

// EmitTCPMessages periodically sends a newline-terminated message over TCP
// until the context is cancelled. Connection failures are retried on each tick.
func EmitTCPMessages(ctx context.Context, address, message string) {
	payload := []byte(message + "\n")

	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				conn, err := net.DialTimeout("tcp", address, 2*time.Second)
				if err != nil {
					continue
				}
				_, _ = conn.Write(payload)
				_ = conn.Close()
			}
		}
	}()
}

// EmitUDPMessages periodically sends a newline-terminated message over UDP
// until the context is cancelled. Connection failures are retried on each tick.
func EmitUDPMessages(ctx context.Context, address, message string) {
	payload := []byte(message + "\n")

	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				conn, err := net.Dial("udp", address)
				if err != nil {
					continue
				}
				_, _ = conn.Write(payload)
				_ = conn.Close()
			}
		}
	}()
}
