// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package tunnels

type Response struct {
	Status string `xml:"status,attr"`
	Result Result `xml:"result"`
}

type Result struct {
	Entries []Entry `xml:"entries>entry"`
	NTun    int     `xml:"ntun"`
}

type Entry struct {
	ID        int    `xml:"id"`
	Name      string `xml:"name"`
	GW        string `xml:"gw"`
	TSiIP     string `xml:"TSi_ip"`
	TSiPrefix int    `xml:"TSi_prefix"`
	TSiProto  int    `xml:"TSi_proto"`
	TSiPort   int    `xml:"TSi_port"`
	TSrIP     string `xml:"TSr_ip"`
	TSrPrefix int    `xml:"TSr_prefix"`
	TSrProto  int    `xml:"TSr_proto"`
	TSrPort   int    `xml:"TSr_port"`
	Proto     string `xml:"proto"`
	Mode      string `xml:"mode"`
	DH        string `xml:"dh"`
	Enc       string `xml:"enc"`
	Hash      string `xml:"hash"`
	Life      int    `xml:"life"`
	KB        int    `xml:"kb"`
}
