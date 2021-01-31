// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package netflow

//go:generate go run fields_gen.go -output _meta/fields.yml --column-name=2 --column-type=3 --header _meta/fields.header.yml decoder/fields/ipfix-information-elements.csv
//go:generate go run fields_gen.go -output _meta/fields.yml --append --column-name=3 --column-type=4 --header _meta/fields.header.yml decoder/fields/cert_pen6871.csv
//go:generate go run fields_gen.go -output _meta/fields.yml --append --column-name=1 --column-type=4 --header _meta/fields.header.yml decoder/fields/cisco.csv
//go:generate go run fields_gen.go -output _meta/fields.yml --append --column-name=3 --column-type=4 --header _meta/fields.header.yml decoder/fields/assorted.csv
