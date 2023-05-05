// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.
//go:build windows

package tools

import (
	"crypto/sha256"
	"fmt"
	"net"
	"time"

	"github.com/Microsoft/go-winio"
)

// DialTestAddr dials the address with the operating specific function
func DialTestAddr(addr string, timeout time.Duration) (net.Conn, error) {
	return winio.DialPipe(addr, &timeout)
}

// GenerateTestAddr creates a grpc address that is specific to the operating system
func GenerateTestAddr(path string) string {
	// entire string cannot be longer than 256 characters, path
	// should be unique for each test
	return fmt.Sprintf(`\\.\pipe\shipper-%x-pipe`, sha256.Sum256([]byte(path)))
}
