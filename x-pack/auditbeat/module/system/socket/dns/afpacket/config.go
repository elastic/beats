// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux
// +build linux

package afpacket

type config struct {
	// Interface to listen on. Defaults to "any".
	Interface string `config:"socket.dns.af_packet.interface"`
	// Snaplen is the packet snapshot size.
	Snaplen int `config:"socket.dns.af_packet.snaplen"`
}

func defaultConfig() config {
	return config{
		Interface: "any",
		Snaplen:   1024,
	}
}
