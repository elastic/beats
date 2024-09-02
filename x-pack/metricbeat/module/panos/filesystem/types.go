// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package filesystem

import "encoding/xml"

type Response struct {
	XMLName xml.Name `xml:"response"`
	Status  string   `xml:"status,attr"`
	Result  Result   `xml:"result"`
}

type Result struct {
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
