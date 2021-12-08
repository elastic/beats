// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

type chainsConfig struct {
	Step stepsConfig `config:"step" validate:"required"`
}

type stepsConfig struct {
	Auth     authConfig     `config:"auth,omitempty"`
	Request  requestConfig  `config:"request"`
	Response responseConfig `config:"response,omitempty"`
	Replace  string         `config:"replace,omitempty"`
}
