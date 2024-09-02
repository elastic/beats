// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fans

type Response struct {
	Status string `xml:"status,attr"`
	Result Result `xml:"result"`
}

type Result struct {
	Fan Fan `xml:"fan"`
}

type Fan struct {
	Slots []Slot `xml:",any"`
}

type Slot struct {
	Entries []Entry `xml:"entry"`
}

type Entry struct {
	Slot        int    `xml:"slot"`
	Description string `xml:"description"`
	Alarm       string `xml:"alarm"`
	RPMs        int    `xml:"RPMs"`
	Min         int    `xml:"min"`
}
