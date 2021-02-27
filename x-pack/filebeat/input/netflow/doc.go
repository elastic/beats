// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package netflow

// Generate fields.yml for all Netflow fields.
//go:generate go run fields_gen.go --header _meta/fields.header.yml -output _meta/fields.yml decoder/fields/ipfix-information-elements.csv,2,3,true decoder/fields/cert_pen6871.csv,3,4,true decoder/fields/cisco.csv,1,4,true decoder/fields/assorted.csv,3,4,true
