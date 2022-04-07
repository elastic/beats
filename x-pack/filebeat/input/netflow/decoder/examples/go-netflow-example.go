// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/elastic/beats/v8/x-pack/filebeat/input/netflow/decoder"
)

func main() {
	decoder, err := decoder.NewDecoder(decoder.NewConfig().
		WithLogOutput(os.Stderr).
		WithProtocols("v1", "v5", "v9", "ipfix"))
	if err != nil {
		log.Fatal("Failed creating decoder:", err)
	}

	addr, err := net.ResolveUDPAddr("udp", ":2055")
	if err != nil {
		log.Fatal("Failed to resolve address:", err)
	}

	server, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatalf("Failed to listen on %v: %v", addr, err)
	}
	defer server.Close()

	if err = server.SetReadBuffer(1 << 16); err != nil {
		log.Fatalf("Failed to set read buffer size for socket: %v", err)
	}

	log.Println("Listening on ", server.LocalAddr())
	buf := make([]byte, 8192)
	decBuf := new(bytes.Buffer)
	for {
		size, remote, err := server.ReadFromUDP(buf)
		if err != nil {
			log.Println("Error reading from socket:", err)
			continue
		}

		decBuf.Reset()
		decBuf.Write(buf[:size])
		records, err := decoder.Read(decBuf, remote)
		if err != nil {
			log.Printf("warn: Failed reading records from %v: %v\n", remote, err)
		}

		for _, r := range records {
			evt, err := json.Marshal(map[string]interface{}{
				"@timestamp": r.Timestamp,
				"type":       r.Type,
				"exporter":   r.Exporter,
				"data":       r.Fields,
			})
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(string(evt))
		}
	}
}
