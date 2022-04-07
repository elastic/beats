// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fuzz

import (
	cef2 "github.com/elastic/beats/v8/x-pack/filebeat/processors/decode_cef/cef"
)

// Fuzz is the entry point that go-fuzz uses to fuzz the parser.
func Fuzz(data []byte) int {
	var e cef2.Event
	if err := e.Unpack(string(data)); err != nil {
		return 1
	}
	return 0
}
