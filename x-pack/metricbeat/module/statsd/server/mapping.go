// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package server

import "regexp"

type StatsdMapping struct {
	Metric string
	Labels []Label
	Value  Value
	regex  *regexp.Regexp
}

type Value struct {
	Field string
}

type Label struct {
	Attr  string
	Field string
}
