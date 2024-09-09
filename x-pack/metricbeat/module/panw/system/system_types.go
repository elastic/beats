// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package system

import (
	"encoding/xml"
)

// system resources
type ResourceResponse struct {
	Status string `xml:"status,attr"`
	Result string `xml:"result"`
}

type SystemLoad struct {
	OneMinute     float64
	FiveMinute    float64
	FifteenMinute float64
}

type Uptime struct {
	Days    int
	Hours   int
	Minutes int
}

type SystemInfo struct {
	Uptime      Uptime
	UserCount   int
	LoadAverage SystemLoad
}

type TaskInfo struct {
	Total    int
	Running  int
	Sleeping int
	Stopped  int
	Zombie   int
}

type CPUInfo struct {
	User      float64
	System    float64
	Nice      float64
	Idle      float64
	Wait      float64
	Hi        float64
	SystemInt float64
	Steal     float64
}

type MemoryInfo struct {
	Total       float64
	Free        float64
	Used        float64
	BufferCache float64
}

type SwapInfo struct {
	Total     float64
	Free      float64
	Used      float64
	Available float64
}

// temperature
type ThermalResponse struct {
	Status string        `xml:"status,attr"`
	Result ThermalResult `xml:"result"`
}

type ThermalResult struct {
	Thermal Thermal `xml:"thermal"`
}

type Thermal struct {
	Slots []ThermalSlot `xml:",any"`
}

type ThermalSlot struct {
	Name    xml.Name       `xml:",any"`
	Entries []ThermalEntry `xml:"entry"`
}

type ThermalEntry struct {
	Slot           int     `xml:"slot"`
	Description    string  `xml:"description"`
	Alarm          bool    `xml:"alarm"`
	DegreesCelsius float64 `xml:"DegreesC"`
	MinimumTemp    float64 `xml:"min"`
	MaximumTemp    float64 `xml:"max"`
}

// power

type PowerResponse struct {
	Status string      `xml:"status,attr"`
	Result PowerResult `xml:"result"`
}

type PowerResult struct {
	Power Power `xml:"power"`
}

type Power struct {
	Slots []PowerSlot `xml:",any"`
}

type PowerSlot struct {
	Entries []PowerEntry `xml:"entry"`
}

type PowerEntry struct {
	Slot         int     `xml:"slot"`
	Description  string  `xml:"description"`
	Alarm        bool    `xml:"alarm"`
	Volts        float64 `xml:"Volts"`
	MinimumVolts float64 `xml:"min"`
	MaximumVolts float64 `xml:"max"`
}

// fans

type FanResponse struct {
	Status string    `xml:"status,attr"`
	Result FanResult `xml:"result"`
}

type FanResult struct {
	Fan Fan `xml:"fan"`
}

type Fan struct {
	Slots []FanSlot `xml:",any"`
}

type FanSlot struct {
	Entries []FanEntry `xml:"entry"`
}

type FanEntry struct {
	Slot        int    `xml:"slot"`
	Description string `xml:"description"`
	Alarm       string `xml:"alarm"`
	RPMs        int    `xml:"RPMs"`
	Min         int    `xml:"min"`
}

// filesystem

type FilesystemResponse struct {
	XMLName xml.Name         `xml:"response"`
	Status  string           `xml:"status,attr"`
	Result  FilesystemResult `xml:"result"`
}

type FilesystemResult struct {
	Data string `xml:",cdata"`
}

type Filesystem struct {
	Name    string
	Size    string
	Used    string
	Avail   string
	UsePerc string
	Mounted string
}

// licenses

type LicenseResponse struct {
	Status string        `xml:"status,attr"`
	Result LicenseResult `xml:"result"`
}

type LicenseResult struct {
	Licenses []License `xml:"licenses>entry"`
}

type License struct {
	Feature     string `xml:"feature"`
	Description string `xml:"description"`
	Serial      string `xml:"serial"`
	Issued      string `xml:"issued"`
	Expires     string `xml:"expires"`
	Expired     string `xml:"expired"`
	AuthCode    string `xml:"authcode"`
}

// certificates

type CertificateResponse struct {
	Status string `xml:"status,attr"`
	Result string `xml:"result"`
}

type Certificate struct {
	CertName          string
	Issuer            string
	IssuerSubjectHash string
	IssuerKeyHash     string
	DBType            string
	DBExpDate         string
	DBRevDate         string
	DBSerialNo        string
	DBFile            string
	DBName            string
	DBStatus          string
}
