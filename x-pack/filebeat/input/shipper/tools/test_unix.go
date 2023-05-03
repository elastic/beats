// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.
//go:build !windows

package tools

import (
	"net"
	"net/url"
	"path/filepath"
	"strings"
)

// DialTestAddr dials the address with the operating specific function
func DialTestAddr(addr string) (net.Conn, error) {
	return net.Dial("unix", strings.TrimPrefix(addr, "unix://"))
}

// GenerateTestAddr creates a grpc address that is specific to the operating system
func GenerateTestAddr(path string) string {
	var socket url.URL
	socket.Scheme = "unix"
	socket.Path = filepath.Join(path, "grpc.sock")
	return socket.String()
}
