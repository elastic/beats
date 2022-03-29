// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"log"
	"os"

	"github.com/elastic/beats/v7/x-pack/filebeat/processors/decode_cef/cef"
)

var (
	fullExtensionNames bool
)

func init() {
	flag.BoolVar(&fullExtensionNames, "full", true, "Use full extension key names.")
}

var cefMarker = []byte("CEF:")

func main() {
	log.SetFlags(0)
	flag.Parse()

	var opts []cef.Option
	if fullExtensionNames {
		opts = append(opts, cef.WithFullExtensionNames())
	}

	s := bufio.NewScanner(os.Stdin)
	for s.Scan() {
		line := s.Bytes()
		if len(line) == 0 {
			continue
		}

		begin := bytes.Index(line, cefMarker)
		if begin == -1 {
			continue
		}
		line = line[begin:]

		var e cef.Event
		if err := e.Unpack(string(line), opts...); err != nil {
			log.Println("ERROR:", err, "in:", string(line))
		}

		jsonData, err := json.Marshal(e)
		if err != nil {
			log.Println("ERROR:", err)
			continue
		}

		// Written as writes to avoid forbidigo.
		os.Stdout.Write(jsonData)
		os.Stdout.Write([]byte{'\n'})
	}
}
