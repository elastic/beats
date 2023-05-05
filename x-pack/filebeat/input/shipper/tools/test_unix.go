// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.
//go:build !windows

package tools

import (
	"net"
	"net/url"
	"path/filepath"
	"time"
)

// DialTestAddr dials the address with the operating specific function
func DialTestAddr(addr string, timeout time.Duration) (net.Conn, error) {
	dailer := net.Dialer{Timeout: timeout}
	return dailer.Dial("unix", addr)
}

// GenerateTestAddr creates a grpc address that is specific to the operating system
func GenerateTestAddr(path string) string {
	var socket url.URL
	socket.Scheme = "unix"
	socket.Path = filepath.Join(path, "grpc.sock")
	return socket.String()
}
