// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"

	"github.com/elastic/elastic-agent-libs/logp"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/netflow/decoder"
)

func main() {
	logger := logp.L().Named("netflow")

	decoder, err := decoder.NewDecoder(decoder.NewConfig(logger).
		WithProtocols("v1", "v5", "v9", "ipfix"))
	if err != nil {
		logger.Fatal("Failed creating decoder:", err)
	}

	addr, err := net.ResolveUDPAddr("udp", ":2055")
	if err != nil {
		logger.Fatal("Failed to resolve address:", err)
	}

	server, err := net.ListenUDP("udp", addr)
	if err != nil {
		logger.Fatalf("Failed to listen on %v: %v", addr, err)
	}
	defer server.Close()

	if err = server.SetReadBuffer(1 << 16); err != nil {
		logger.Fatalf("Failed to set read buffer size for socket: %v", err)
	}

	logger.Debug("Listening on ", server.LocalAddr())
	buf := make([]byte, 8192)
	decBuf := new(bytes.Buffer)
	for {
		size, remote, err := server.ReadFromUDP(buf)
		if err != nil {
			logger.Debug("Error reading from socket:", err)
			continue
		}

		decBuf.Reset()
		decBuf.Write(buf[:size])
		records, err := decoder.Read(decBuf, remote)
		if err != nil {
			logger.Debugf("warn: Failed reading records from %v: %v\n", remote, err)
		}

		for _, r := range records {
			evt, err := json.Marshal(map[string]interface{}{
				"@timestamp": r.Timestamp,
				"type":       r.Type,
				"exporter":   r.Exporter,
				"data":       r.Fields,
			})
			if err != nil {
				logger.Fatal(err)
			}
			fmt.Println(string(evt))
		}
	}
}
