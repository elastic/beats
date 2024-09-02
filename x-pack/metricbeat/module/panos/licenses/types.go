// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package licenses

type Response struct {
	Status string `xml:"status,attr"`
	Result Result `xml:"result"`
}

type Result struct {
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
