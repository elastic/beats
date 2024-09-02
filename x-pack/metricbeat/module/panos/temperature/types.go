// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package temperature

import "encoding/xml"

type Response struct {
	Status string `xml:"status,attr"`
	Result Result `xml:"result"`
}

type Result struct {
	Thermal Thermal `xml:"thermal"`
}

type Thermal struct {
	Slots []Slot `xml:",any"`
}

type Slot struct {
	Name    xml.Name `xml:",any"`
	Entries []Entry  `xml:"entry"`
}

type Entry struct {
	Slot           int     `xml:"slot"`
	Description    string  `xml:"description"`
	Alarm          bool    `xml:"alarm"`
	DegreesCelsius float64 `xml:"DegreesC"`
	MinimumTemp    float64 `xml:"min"`
	MaximumTemp    float64 `xml:"max"`
}
