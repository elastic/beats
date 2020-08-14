// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fields

//go:generate go run gen.go -output zfields_ipfix.go -export IpfixFields --column-id=1 --column-name=2 --column-type=3 ipfix-information-elements.csv
//go:generate go run gen.go -output zfields_cert.go -export CertFields --column-pen=1 --column-id=2 --column-name=3 --column-type=4 cert_pen6871.csv
//go:generate go run gen.go -output zfields_cisco.go -export CiscoFields --column-pen=2 --column-id=3 --column-name=1 --column-type=4 cisco.csv
//go:generate go run gen.go -output zfields_assorted.go -export AssortedFields --column-pen=1 --column-id=2 --column-name=3 --column-type=4 assorted.csv
//go:generate go fmt
