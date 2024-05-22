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

//go:build linux

package logp

import (
	"io"
	"testing"

	"go.uber.org/zap/zapcore"
	"gopkg.in/mcuadros/go-syslog.v2"
)

// TestSyslogOutputCanBeClosed instantiates a syslog output and ensures it
// implements `io.Close`.
//
// We call close to ensure it does not return an error.
//
// Our syslog is hardcoded to connect via Unix sockets. The container
// we use to run the tests does not have a syslog listening on Unix socket,
// so we instantiate our own.
func TestSyslogOutputCanBeClosed(t *testing.T) {
	channel := make(syslog.LogPartsChannel)
	handler := syslog.NewChannelHandler(channel)

	server := syslog.NewServer()
	server.SetFormat(syslog.RFC5424)
	server.SetHandler(handler)

	if err := server.ListenUnixgram("/var/run/syslog"); err != nil {
		t.Errorf("cannot configure syslog to listen on '/var/run/syslog/': %s", err)
		t.Log("You might already have a syslog running, this test assumes " +
			"there is no syslog running on the host. You can run this test " +
			"in a Docker container (adjust the image tag to the current " +
			"Go version):\n" +
			"'docker run --rm -it -v $PWD:$PWD -w $PWD golang:1.21.10 go test ./logp'")
	}

	if err := server.Boot(); err != nil {
		t.Fatalf("cannot start syslog server: %s", err)
	}

	t.Cleanup(func() {
		// ignore the error, we just need it to accept
		// connections from our syslog output
		_ = server.Kill()
	})

	cfg := DefaultConfig(DefaultEnvironment)
	cfg.ToFiles = true

	syslogOutput, err := makeSyslogOutput(cfg, zapcore.DebugLevel)
	if err != nil {
		t.Fatalf("cannot create syslog output: %s", err)
	}

	closer, ok := syslogOutput.(io.Closer)
	if !ok {
		t.Fatal("the 'Syslog Output' does not implement io.Closer")
	}
	if err := closer.Close(); err != nil {
		t.Fatalf("Close must not return any error, got: %s", err)
	}
}
