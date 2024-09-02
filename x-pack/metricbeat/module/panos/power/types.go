// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package power

type Response struct {
	Status string `xml:"status,attr"`
	Result Result `xml:"result"`
}

type Result struct {
	Power Power `xml:"power"`
}

type Power struct {
	Slots []Slot `xml:",any"`
}

type Slot struct {
	Entries []Entry `xml:"entry"`
}

type Entry struct {
	Slot         int     `xml:"slot"`
	Description  string  `xml:"description"`
	Alarm        bool    `xml:"alarm"`
	Volts        float64 `xml:"Volts"`
	MinimumVolts float64 `xml:"min"`
	MaximumVolts float64 `xml:"max"`
}
