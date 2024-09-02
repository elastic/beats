// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package certificates

type Response struct {
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
