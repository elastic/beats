// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package test

import (
	"bytes"
	"encoding/binary"
	"net"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/menderesk/beats/v7/x-pack/filebeat/input/netflow/decoder/record"
)

type TestLogWriter struct {
	testing.TB
}

func (t TestLogWriter) Write(buf []byte) (int, error) {
	t.Log(string(buf))
	return len(buf), nil
}

func MakeAddress(t testing.TB, ipPortPair string) net.Addr {
	ip, portS, err := net.SplitHostPort(ipPortPair)
	if err != nil {
		t.Fatal(err)
		return nil
	}
	port, err := strconv.Atoi(portS)
	if err != nil {
		t.Fatal(err)
		return nil
	}
	return &net.UDPAddr{
		IP:   net.ParseIP(ip),
		Port: port,
	}
}

func MakePacket(data []uint16) *bytes.Buffer {
	r := make([]byte, len(data)*2)
	for idx, val := range data {
		binary.BigEndian.PutUint16(r[idx*2:(idx+1)*2], val)
	}
	return bytes.NewBuffer(r)
}

func AssertMapEqual(t testing.TB, expected record.Map, actual record.Map) bool {
	for key, expectedValue := range expected {
		value, found := actual[key]
		if !assert.True(t, found, key) {
			return false
		}
		if !assert.Equal(t, expectedValue, value, key) {
			return false
		}
	}
	for key := range actual {
		_, found := expected[key]
		if !assert.True(t, found, key) {
			return false
		}
	}
	return true
}

func AssertRecordsEqual(t testing.TB, expected record.Record, actual record.Record) bool {
	if !assert.Equal(t, expected.Type, actual.Type) {
		return false
	}
	if !assert.Equal(t, expected.Timestamp, actual.Timestamp) {
		return false
	}
	if !AssertMapEqual(t, expected.Fields, actual.Fields) {
		return false
	}
	if !AssertMapEqual(t, expected.Exporter, actual.Exporter) {
		return false
	}
	return true
}
