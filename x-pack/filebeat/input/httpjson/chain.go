// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

// chainConfig for chain request
// each step will be considered as seperate request call
type chainConfig struct {
	Step stepConfig `config:"step" validate:"required"`
}

type stepConfig struct {
	Auth     authConfig     `config:"auth,omitempty"`
	Request  requestConfig  `config:"request"`
	Response responseConfig `config:"response,omitempty"`
	Replace  string         `config:"replace,omitempty"`
}
