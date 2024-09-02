// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package globalprotect_stats

type Response struct {
	Status string `xml:"status,attr"`
	Result Result `xml:"result"`
}

type Result struct {
	Gateways           []Gateway `xml:"Gateway"`
	TotalCurrentUsers  int       `xml:"TotalCurrentUsers"`
	TotalPreviousUsers int       `xml:"TotalPreviousUsers"`
}

type Gateway struct {
	Name          string `xml:"name"`
	CurrentUsers  int    `xml:"CurrentUsers"`
	PreviousUsers int    `xml:"PreviousUsers"`
}
